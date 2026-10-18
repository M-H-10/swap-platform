package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// UserStore is everything the auth service needs from the database.
// The repository implements it; tests use a fake in memory.
type UserStore interface {
	CreateUser(ctx context.Context, in domain.RegisterInput, passwordHash string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) (bool, error)
	RevokeAllRefreshTokens(ctx context.Context, userID string) error
}

// AuthResult is what a successful register, login or refresh returns.
type AuthResult struct {
	User             domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// AuthService holds the rules for accounts and sessions.
type AuthService struct {
	store      UserStore
	tokens     *TokenManager
	bcryptCost int
	dummyHash  []byte // used to make "unknown email" as slow as "wrong password"
}

// NewAuthService builds the service. Use bcrypt cost 12 in production
// (about 0.3 s per hash, which slows down password-guessing attacks).
func NewAuthService(store UserStore, tokens *TokenManager, bcryptCost int) (*AuthService, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("preparing dummy hash: %w", err)
	}
	return &AuthService{store: store, tokens: tokens, bcryptCost: bcryptCost, dummyHash: dummy}, nil
}

// Register creates the account (with its wallet and starting reputation)
// and logs the user in straight away.
func (s *AuthService) Register(ctx context.Context, in domain.RegisterInput) (AuthResult, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return AuthResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hashing password: %w", err)
	}

	user, err := s.store.CreateUser(ctx, in, string(hash))
	if err != nil {
		return AuthResult{}, err
	}
	return s.issueTokens(ctx, user)
}

// Login checks the email and password. Both "unknown email" and "wrong
// password" return the same error, so an attacker cannot learn which
// emails are registered.
func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		// Spend the same time as a real check before answering.
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	return s.issueTokens(ctx, user)
}

// Refresh trades a valid refresh token for a new pair of tokens. The old
// refresh token is revoked (rotation). If someone presents a token that was
// already used, we assume it was stolen and end every session of that user.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	if refreshToken == "" {
		return AuthResult{}, domain.ErrInvalidToken
	}

	stored, err := s.store.GetRefreshToken(ctx, HashRefreshToken(refreshToken))
	if errors.Is(err, domain.ErrNotFound) {
		return AuthResult{}, domain.ErrInvalidToken
	}
	if err != nil {
		return AuthResult{}, err
	}

	if stored.RevokedAt != nil {
		// Reuse of an old token: possible theft.
		if err := s.store.RevokeAllRefreshTokens(ctx, stored.UserID); err != nil {
			return AuthResult{}, err
		}
		return AuthResult{}, domain.ErrInvalidToken
	}
	if time.Now().After(stored.ExpiresAt) {
		return AuthResult{}, domain.ErrInvalidToken
	}

	// Revoke the old token. If another request revoked it a moment ago,
	// treat this one as a reuse as well.
	revoked, err := s.store.RevokeRefreshToken(ctx, stored.ID)
	if err != nil {
		return AuthResult{}, err
	}
	if !revoked {
		if err := s.store.RevokeAllRefreshTokens(ctx, stored.UserID); err != nil {
			return AuthResult{}, err
		}
		return AuthResult{}, domain.ErrInvalidToken
	}

	user, err := s.store.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return AuthResult{}, err
	}
	return s.issueTokens(ctx, user)
}

// Logout ends one session. Logging out with an unknown token is not an error.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	stored, err := s.store.GetRefreshToken(ctx, HashRefreshToken(refreshToken))
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.store.RevokeRefreshToken(ctx, stored.ID)
	return err
}

// CurrentUser returns the account behind an access token.
func (s *AuthService) CurrentUser(ctx context.Context, userID string) (domain.User, error) {
	return s.store.GetUserByID(ctx, userID)
}

// issueTokens creates an access token and a new stored refresh token.
func (s *AuthService) issueTokens(ctx context.Context, user domain.User) (AuthResult, error) {
	access, accessExp, err := s.tokens.NewAccessToken(user.ID)
	if err != nil {
		return AuthResult{}, err
	}

	plain, hash, err := NewRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	refreshExp := time.Now().Add(s.tokens.RefreshTTL)
	if err := s.store.SaveRefreshToken(ctx, user.ID, hash, refreshExp); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:             user,
		AccessToken:      access,
		AccessExpiresAt:  accessExp,
		RefreshToken:     plain,
		RefreshExpiresAt: refreshExp,
	}, nil
}
