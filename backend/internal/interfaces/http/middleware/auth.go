package middleware

import (
	"strings"

	"github.com/cashbacktv/backend/internal/application"
	"github.com/cashbacktv/backend/internal/domain"
	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService *application.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *application.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// Authenticate validates JWT token
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		// Store claims in context for use in handlers
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}

// RequireRole checks if user has required role
func (m *AuthMiddleware) RequireRole(requiredRole domain.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("user_role").(domain.UserRole)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		roleHierarchy := map[domain.UserRole]int{
			domain.UserRoleViewer:   1,
			domain.UserRoleOperator: 2,
			domain.UserRoleAdmin:    3,
		}

		if roleHierarchy[role] < roleHierarchy[requiredRole] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}

		return c.Next()
	}
}





