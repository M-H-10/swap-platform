package domain

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Role is what the user does in a team.
type Role string

const (
	RoleDeveloper Role = "developer"
	RoleDesigner  Role = "designer"
	RoleManager   Role = "manager"
	RoleOther     Role = "other"
)

// InitialReputationBP is the reputation every new account starts with
// (5000 bp = 50 out of 100), the same default as the old project.
const InitialReputationBP = 5000

// User is a registered account.
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string // bcrypt hash, never the password itself
	Role         Role
	ReputationBP int // latest reputation score, 0..10000
	CreatedAt    time.Time
}

// RefreshToken is one login session stored in the database.
type RefreshToken struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time // nil while the session is still valid
}

// RegisterInput is what a new user sends to create an account.
type RegisterInput struct {
	Username string
	Email    string
	Password string
	Role     Role
}

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// Normalize trims spaces and lower-cases the email, so " Ali@Mail.com "
// and "ali@mail.com" are treated as the same address.
func (in *RegisterInput) Normalize() {
	in.Username = strings.TrimSpace(in.Username)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Role == "" {
		in.Role = RoleDeveloper
	}
}

// Validate checks every field and returns the first problem it finds.
func (in RegisterInput) Validate() error {
	if !usernamePattern.MatchString(in.Username) {
		return &ValidationError{"username", "must be 3-32 characters: letters, digits or _"}
	}
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email {
		return &ValidationError{"email", "is not a valid email address"}
	}
	if utf8.RuneCountInString(in.Password) < 8 {
		return &ValidationError{"password", "must be at least 8 characters"}
	}
	if len(in.Password) > 72 { // bcrypt only uses the first 72 bytes
		return &ValidationError{"password", "must be at most 72 bytes"}
	}
	switch in.Role {
	case RoleDeveloper, RoleDesigner, RoleManager, RoleOther:
	default:
		return &ValidationError{"role", "must be developer, designer, manager or other"}
	}
	return nil
}
