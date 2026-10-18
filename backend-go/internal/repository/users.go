package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// UserRepository reads and writes users and their login sessions.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates the repository on top of the shared pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// CreateUser inserts the user, an empty wallet and the starting reputation
// in ONE transaction: either all three rows are saved, or none of them.
func (r *UserRepository) CreateUser(ctx context.Context, in domain.RegisterInput, passwordHash string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin transaction: %w", err)
	}
	// If anything below fails, Rollback undoes everything. After a
	// successful Commit, Rollback does nothing.
	defer tx.Rollback(ctx)

	user := domain.User{
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: passwordHash,
		Role:         in.Role,
		ReputationBP: domain.InitialReputationBP,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		in.Username, in.Email, passwordHash, in.Role,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return domain.User{}, translateUserError(err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO wallets (user_id) VALUES ($1)`, user.ID); err != nil {
		return domain.User{}, fmt.Errorf("create wallet: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO reputation_snapshots (user_id, score_bp, reason)
		VALUES ($1, $2, 'registration')`,
		user.ID, domain.InitialReputationBP,
	); err != nil {
		return domain.User{}, fmt.Errorf("create reputation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit: %w", err)
	}
	return user, nil
}

// selectUser reads a user together with the latest reputation snapshot.
const selectUser = `
	SELECT u.id, u.username, u.email, u.password_hash, u.role, u.created_at,
	       COALESCE(r.score_bp, 0)
	FROM users u
	LEFT JOIN LATERAL (
		SELECT score_bp FROM reputation_snapshots
		WHERE user_id = u.id
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	) r ON true`

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.ReputationBP)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("read user: %w", err)
	}
	return u, nil
}

// GetUserByEmail finds a user by email, ignoring upper/lower case.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, selectUser+` WHERE lower(u.email) = lower($1)`, email))
}

// GetUserByID finds a user by ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, selectUser+` WHERE u.id = $1`, id))
}

// SaveRefreshToken stores the hash of a new refresh token.
func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken finds a session by the hash of its token.
func (r *UserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, revoked_at
		FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("read refresh token: %w", err)
	}
	return t, nil
}

// RevokeRefreshToken ends one session. It returns false if the session was
// already revoked, so two requests can never both use the same token.
func (r *UserRepository) RevokeRefreshToken(ctx context.Context, id string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return false, fmt.Errorf("revoke refresh token: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// RevokeAllRefreshTokens ends every session of a user ("log out everywhere").
func (r *UserRepository) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("revoke all refresh tokens: %w", err)
	}
	return nil
}

// translateUserError turns a PostgreSQL "unique violation" into a clear
// domain error, so the user sees "email is already registered" instead of
// a database message.
func translateUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 = unique_violation
		switch pgErr.ConstraintName {
		case "users_email_lower_key":
			return &domain.ConflictError{Message: "email is already registered"}
		case "users_username_lower_key":
			return &domain.ConflictError{Message: "username is already taken"}
		}
	}
	return fmt.Errorf("insert user: %w", err)
}

// SearchUsers finds users whose username contains the text, best
// reputation first. excludeID leaves out one user (the one searching).
func (r *UserRepository) SearchUsers(ctx context.Context, query, excludeID string, limit int) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, selectUser+`
		WHERE u.username ILIKE $1 AND u.id <> $2
		ORDER BY COALESCE(r.score_bp, 0) DESC, lower(u.username)
		LIMIT $3`,
		"%"+escapeLike(query)+"%", excludeID, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	return users, nil
}
