package controllers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"ccs-forms/db"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthStateCookie = "oauth_state"
	oauthStateTTL    = 10 * time.Minute
	userInfoURL      = "https://www.googleapis.com/oauth2/v2/userinfo"
)

var errOAuthNotConfigured = errors.New("google oauth is not configured")

func googleConfig() (*oauth2.Config, error) {
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, errOAuthNotConfigured
	}

	redirect := os.Getenv("OAUTH_REDIRECT_URL")
	if redirect == "" {
		redirect = "http://localhost:8080/auth/google/callback"
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirect,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}, nil
}

func signToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GoogleLogin(c *gin.Context) {
	cfg, err := googleConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google sign-in is not configured"})
		return
	}

	state, err := randomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start Google sign-in"})
		return
	}

	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookie, state, int(oauthStateTTL.Seconds()), "/", "", secure, true)

	c.Redirect(http.StatusTemporaryRedirect, cfg.AuthCodeURL(state))
}

type googleProfile struct {
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
}

func fetchGoogleProfile(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token) (googleProfile, error) {
	var profile googleProfile

	client := cfg.Client(ctx, tok)
	client.Timeout = 10 * time.Second

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return profile, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return profile, errors.New("userinfo returned " + resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
		return profile, err
	}
	return profile, nil
}

func GoogleCallback(c *gin.Context) {
	cfg, err := googleConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google sign-in is not configured"})
		return
	}

	want, cookieErr := c.Cookie(oauthStateCookie)
	got := c.Query("state")
	c.SetCookie(oauthStateCookie, "", -1, "/", "", c.Request.TLS != nil, true)

	if cookieErr != nil || want == "" || got == "" ||
		subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired sign-in state"})
		return
	}

	if reason := c.Query("error"); reason != "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Google sign-in was denied"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	ctx := c.Request.Context()
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Could not complete Google sign-in"})
		return
	}

	profile, err := fetchGoogleProfile(ctx, cfg, tok)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not read Google profile"})
		return
	}
	if profile.Email == "" || !profile.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Google account has no verified email"})
		return
	}

	name := profile.Name
	if name == "" {
		name = profile.Email
	}

	var userID int64
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (gmail, name)
		VALUES ($1, $2)
		ON CONFLICT (gmail) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, profile.Email, name).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign in user"})
		return
	}

	signed, err := signToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed, "email": profile.Email})
}
