package handlers

import (
	"github.com/cashbacktv/backend/internal/infrastructure/system"
	"github.com/gofiber/fiber/v2"
)

// SystemHandler handles system information requests
type SystemHandler struct{}

// NewSystemHandler creates a new system handler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// GetSystemInfo returns current system information
func (h *SystemHandler) GetSystemInfo(c *fiber.Ctx) error {
	info, err := system.GetSystemInfo()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "sistem bilgileri alınamadı: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": info,
	})
}
