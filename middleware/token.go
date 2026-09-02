package middleware

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrJWTSecretNotConfigured = errors.New("JWT_SECRET is not configured")

// jwtSecret is deliberately shared by token issuance and verification. A
// deployment with no secret must fail closed instead of issuing tokens that
// anybody could reproduce with an empty key.
func jwtSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, ErrJWTSecretNotConfigured
	}
	return []byte(secret), nil
}

// SignToken creates the short-lived bearer token used by the API. The token
// contains only the user's database ID; authorization decisions remain in the
// handlers and database queries.
func SignToken(userID int64) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseUserID verifies the signature and standard claims before returning the
// subject. WithValidMethods is important: accepting an algorithm supplied by
// the token would weaken the HS256-only contract used when signing tokens.
func ParseUserID(tokenString string) (int64, error) {
	secret, err := jwtSecret()
	if err != nil {
		return 0, err
	}

	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || token == nil || !token.Valid || claims.Subject == "" {
		return 0, errors.New("invalid token")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID < 1 {
		return 0, errors.New("invalid token subject")
	}
	return userID, nil
}
