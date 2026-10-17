package handler

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// refreshCookieName is the cookie that holds the refresh token.
const refreshCookieName = "refresh_token"

// AuthHandler exposes register, login, refresh, logout and "who am I".
type AuthHandler struct {
	auth         *service.AuthService
	cookieSecure bool // true in production (HTTPS only)
}

// NewAuthHandler creates the handler.
func NewAuthHandler(auth *service.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{auth: auth, cookieSecure: cookieSecure}
}

// ----- request and response shapes (the JSON the frontend sees) -----

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	ReputationBP int       `json:"reputation_bp"`
	CreatedAt    time.Time `json:"created_at"`
}

type authResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"` // seconds until the access token expires
	User        userResponse `json:"user"`
}

func toUserResponse(u domain.User) userResponse {
	return userResponse{
		ID: u.ID, Username: u.Username, Email: u.Email, Role: string(u.Role),
		ReputationBP: u.ReputationBP, CreatedAt: u.CreatedAt,
	}
}

// ----- handlers -----

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "request body must be valid JSON")
	}

	result, err := h.auth.Register(c.Context(), domain.RegisterInput{
		Username: req.Username, Email: req.Email, Password: req.Password, Role: domain.Role(req.Role),
	})
	if err != nil {
		return err // ErrorHandler turns it into the right status code
	}
	return h.sendAuth(c, fiber.StatusCreated, result)
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "request body must be valid JSON")
	}

	result, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	return h.sendAuth(c, fiber.StatusOK, result)
}

// Refresh handles POST /api/auth/refresh. The refresh token comes from the
// HttpOnly cookie, never from the body, so JavaScript never touches it.
func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	result, err := h.auth.Refresh(c.Context(), c.Cookies(refreshCookieName))
	if err != nil {
		h.clearRefreshCookie(c)
		return err
	}
	return h.sendAuth(c, fiber.StatusOK, result)
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	if err := h.auth.Logout(c.Context(), c.Cookies(refreshCookieName)); err != nil {
		return err
	}
	h.clearRefreshCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// Me handles GET /api/auth/me (needs a valid access token).
func (h *AuthHandler) Me(c fiber.Ctx) error {
	user, err := h.auth.CurrentUser(c.Context(), middleware.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(toUserResponse(user))
}

// ----- helpers -----

// sendAuth puts the refresh token in the cookie and the access token in the body.
func (h *AuthHandler) sendAuth(c fiber.Ctx, status int, r service.AuthResult) error {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    r.RefreshToken,
		Path:     "/api/auth", // the browser sends it only to the auth routes
		Expires:  r.RefreshExpiresAt,
		HTTPOnly: true,                           // JavaScript cannot read it (protects against XSS)
		Secure:   h.cookieSecure,                 // HTTPS only in production
		SameSite: fiber.CookieSameSiteStrictMode, // never sent from other websites
	})
	return c.Status(status).JSON(authResponse{
		AccessToken: r.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(r.AccessExpiresAt).Seconds()),
		User:        toUserResponse(r.User),
	})
}

// clearRefreshCookie tells the browser to delete the cookie.
func (h *AuthHandler) clearRefreshCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}
