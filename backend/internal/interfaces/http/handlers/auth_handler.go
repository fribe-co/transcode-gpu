package handlers

import (
	"strings"

	"github.com/cashbacktv/backend/internal/application"
	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles HTTP requests for authentication
type AuthHandler struct {
	service *application.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(service *application.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RefreshRequest represents token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Login authenticates a user
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	tokens, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if err == application.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "geçersiz e-posta veya şifre",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": tokens,
	})
}

// Logout invalidates the current session
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// In a stateless JWT setup, logout is handled client-side
	// For stateful sessions, we would invalidate the token here
		return c.JSON(fiber.Map{
			"message": "başarıyla çıkış yapıldı",
		})
}

// Refresh generates new token pair
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	tokens, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "geçersiz veya süresi dolmuş refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"data": tokens,
	})
}

// Me returns current user information
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "yetkilendirme başlığı eksik",
		})
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	user, err := h.service.GetCurrentUser(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "geçersiz veya süresi dolmuş token",
		})
	}

	return c.JSON(fiber.Map{
		"data": user,
	})
}

