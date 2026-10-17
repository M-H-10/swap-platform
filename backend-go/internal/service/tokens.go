// Package service holds the business logic: what happens on register,
// login, project creation, swaps and so on.
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

const tokenIssuer = "swap-platform"

// TokenManager creates and checks the two kinds of tokens:
//   - access token: a signed JWT, short-lived (15 minutes), sent in the
//     Authorization header with every request;
//   - refresh token: a long random string, long-lived (7 days), kept in an
//     HttpOnly cookie and used only to get a new access token.
type TokenManager struct {
	secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// NewTokenManager needs the secret used to sign access tokens.
func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), AccessTTL: accessTTL, RefreshTTL: refreshTTL}
}

// NewAccessToken signs a JWT that says "this is user <userID>" until it expires.
func (m *TokenManager) NewAccessToken(userID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.AccessTTL)
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing access token: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseAccessToken checks the signature, the algorithm, the issuer and the
// expiry, and returns the user ID stored inside the token.
func (m *TokenManager) ParseAccessToken(token string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return "", domain.ErrInvalidToken
	}
	return claims.Subject, nil
}

// NewRefreshToken returns a random token for the cookie and its hash for
// the database. The plain token is never stored.
func NewRefreshToken() (plain, hash string, err error) {
	b := make([]byte, 32) // 256 random bits
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashRefreshToken(plain), nil
}

// HashRefreshToken turns a refresh token into the value we store and search by.
func HashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
