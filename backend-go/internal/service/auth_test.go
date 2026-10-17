package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// fakeStore keeps users and sessions in memory instead of PostgreSQL.
type fakeStore struct {
	users  map[string]domain.User          // by ID
	tokens map[string]*domain.RefreshToken // by hash
	nextID int
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]domain.User{}, tokens: map[string]*domain.RefreshToken{}}
}

func (f *fakeStore) CreateUser(_ context.Context, in domain.RegisterInput, hash string) (domain.User, error) {
	for _, u := range f.users {
		if u.Email == in.Email {
			return domain.User{}, &domain.ConflictError{Message: "email is already registered"}
		}
	}
	f.nextID++
	u := domain.User{ID: fmt.Sprintf("user-%d", f.nextID), Username: in.Username, Email: in.Email,
		PasswordHash: hash, Role: in.Role, ReputationBP: domain.InitialReputationBP}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) GetUserByID(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) SaveRefreshToken(_ context.Context, userID, hash string, exp time.Time) error {
	f.tokens[hash] = &domain.RefreshToken{ID: hash, UserID: userID, ExpiresAt: exp}
	return nil
}

func (f *fakeStore) GetRefreshToken(_ context.Context, hash string) (domain.RefreshToken, error) {
	t, ok := f.tokens[hash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	return *t, nil
}

func (f *fakeStore) RevokeRefreshToken(_ context.Context, id string) (bool, error) {
	t := f.tokens[id]
	if t == nil || t.RevokedAt != nil {
		return false, nil
	}
	now := time.Now()
	t.RevokedAt = &now
	return true, nil
}

func (f *fakeStore) RevokeAllRefreshTokens(_ context.Context, userID string) error {
	now := time.Now()
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func newTestService(t *testing.T) *AuthService {
	t.Helper()
	tokens := NewTokenManager("test-secret-that-is-long-enough-123", 15*time.Minute, 7*24*time.Hour)
	svc, err := NewAuthService(newFakeStore(), tokens, bcrypt.MinCost) // fast hashing for tests
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

var ali = domain.RegisterInput{Username: "ali", Email: "Ali@Example.com", Password: "secret123", Role: domain.RoleDesigner}

func TestRegisterThenLogin(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	reg, err := svc.Register(ctx, ali)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if reg.User.PasswordHash == ali.Password {
		t.Fatal("password was stored in plain text")
	}
	if reg.AccessToken == "" || reg.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}

	// The email is normalized, so a different case must still log in.
	if _, err := svc.Login(ctx, "ALI@example.com", "secret123"); err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, ali); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(ctx, ali)
	var ce *domain.ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
}

func TestLoginWrongPasswordAndUnknownEmailLookTheSame(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, ali); err != nil {
		t.Fatal(err)
	}

	_, errWrong := svc.Login(ctx, "ali@example.com", "wrong-password")
	_, errUnknown := svc.Login(ctx, "nobody@example.com", "secret123")
	if !errors.Is(errWrong, domain.ErrInvalidCredentials) || !errors.Is(errUnknown, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials twice, got %v and %v", errWrong, errUnknown)
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	svc := newTestService(t)
	reg, err := svc.Register(context.Background(), ali)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := svc.tokens.ParseAccessToken(reg.AccessToken)
	if err != nil || userID != reg.User.ID {
		t.Fatalf("expected user %q, got %q (err %v)", reg.User.ID, userID, err)
	}

	// A token signed with another secret must be rejected.
	other := NewTokenManager("a-completely-different-secret-456", time.Minute, time.Hour)
	forged, _, _ := other.NewAccessToken(reg.User.ID)
	if _, err := svc.tokens.ParseAccessToken(forged); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("forged token was accepted: %v", err)
	}
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg, err := svc.Register(ctx, ali)
	if err != nil {
		t.Fatal(err)
	}

	// First refresh works and gives a new refresh token.
	second, err := svc.Refresh(ctx, reg.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if second.RefreshToken == reg.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	// Using the old token again is a reuse: it fails...
	if _, err := svc.Refresh(ctx, reg.RefreshToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("reused token was accepted: %v", err)
	}
	// ...and it also kills the newest session.
	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("session survived a reuse attack: %v", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	reg, err := svc.Register(ctx, ali)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, reg.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, reg.RefreshToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("refresh after logout should fail, got %v", err)
	}
}
