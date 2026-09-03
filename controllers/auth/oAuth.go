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
	"net/url"
	"os"
	"strings"
	"time"

	"ccs-forms/db"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
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
	// Keep provider credentials server-side. The redirect URL must exactly match
	// one registered in Google Cloud; 0.0.0.0 is only a development fallback.
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, errOAuthNotConfigured
	}

	redirect := os.Getenv("OAUTH_REDIRECT_URL")
	if redirect == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		redirect = "http://0.0.0.0:" + port + "/auth/google/callback"
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirect,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}, nil
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GoogleLogin(c *gin.Context) {
	// OAuth state binds the browser that started the flow to the callback and
	// prevents an attacker from injecting their own authorization response.
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

	// The state cookie is short-lived, HttpOnly, and SameSite=Lax: it must be
	// sent on Google's top-level redirect but not exposed to page JavaScript.
	secure := requestUsesHTTPS(c)
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
	// Validate state before exchanging the authorization code. The callback is
	// otherwise an untrusted public request, even though Google redirects to it.
	cfg, err := googleConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google sign-in is not configured"})
		return
	}

	want, cookieErr := c.Cookie(oauthStateCookie)
	got := c.Query("state")
	// Consume the state before any token exchange so it cannot be replayed.
	c.SetCookie(oauthStateCookie, "", -1, "/", "", requestUsesHTTPS(c), true)

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
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	if email == "" || !profile.VerifiedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "Google account has no verified email"})
		return
	}

	name := profile.Name
	if name == "" {
		name = email
	}

	var userID int64
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (gmail, name)
		VALUES ($1, $2)
		ON CONFLICT (gmail) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, email, name).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign in user"})
		return
	}

	signed, err := middleware.SignToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	// A browser reaches this endpoint by top-level redirect from Google, so a
	// JSON body would leave the user staring at raw text. When a frontend is
	// configured, hand the token back to it instead. The token travels in the
	// URL fragment because fragments are never sent to a server, so it stays
	// out of proxy and access logs and out of any Referer header.
	if target, ok := frontendCallbackURL(); ok {
		params := url.Values{}
		params.Set("token", signed)
		params.Set("email", email)
		c.Redirect(http.StatusFound, target+"#"+params.Encode())
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed, "email": email})
}

// frontendCallbackURL is where the browser is sent once sign-in succeeds. It is
// built from FRONTEND_URL so the same backend can serve a local dev server, an
// ngrok tunnel, or a deployed frontend without a rebuild. FRONTEND_CALLBACK_PATH
// overrides the default route for frontends that use a different one.
func frontendCallbackURL() (string, bool) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_URL")), "/")
	if base == "" {
		return "", false
	}

	path := strings.TrimSpace(os.Getenv("FRONTEND_CALLBACK_PATH"))
	if path == "" {
		path = "/auth/callback"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path, true
}

// requestUsesHTTPS also handles the common TLS-terminating reverse-proxy
// setup. X-Forwarded-Proto must only be trusted when the proxy is controlled
// by the deployment; direct internet clients must not be able to set it.
func requestUsesHTTPS(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}
