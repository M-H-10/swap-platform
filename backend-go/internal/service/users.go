package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// UserDirectory is what the user service needs from the database.
type UserDirectory interface {
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	SearchUsers(ctx context.Context, query, excludeID string, limit int) ([]domain.User, error)
}

// UserService lets users find each other, for example to pick a partner
// for a swap.
type UserService struct {
	users UserDirectory
}

// NewUserService builds the service.
func NewUserService(users UserDirectory) *UserService {
	return &UserService{users: users}
}

// Search finds other users by part of their username.
func (s *UserService) Search(ctx context.Context, currentUserID, query string, limit int) ([]domain.User, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 32 {
		return nil, &domain.ValidationError{Field: "q", Message: "must be at most 32 characters"}
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 50 {
		return nil, &domain.ValidationError{Field: "limit", Message: "must be 1-50"}
	}
	return s.users.SearchUsers(ctx, query, currentUserID, limit)
}

// Get returns one user's public profile.
func (s *UserService) Get(ctx context.Context, id string) (domain.User, error) {
	if !domain.IsUUID(id) {
		return domain.User{}, domain.ErrNotFound
	}
	return s.users.GetUserByID(ctx, id)
}
