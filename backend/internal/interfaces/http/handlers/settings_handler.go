package handlers

import (
	"github.com/cashbacktv/backend/internal/application"
	"github.com/gofiber/fiber/v2"
)

// SettingsHandler handles HTTP requests for settings
type SettingsHandler struct {
	service *application.SettingsService
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(service *application.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// GetSettingsRequest represents settings retrieval
type GetSettingsResponse struct {
	MaxChannels      int    `json:"max_channels"`
	SegmentTime      int    `json:"segment_time"`
	PlaylistSize     int    `json:"playlist_size"`
	LogRetention     int    `json:"log_retention"`
	DefaultPreset    string `json:"default_preset"`
	DefaultBitrate   string `json:"default_bitrate"`
	DefaultResolution string `json:"default_resolution"`
	DefaultProfile   string `json:"default_profile"`
	DefaultCRF       int    `json:"default_crf"`
	DefaultMaxrate   string `json:"default_maxrate"`
	DefaultBufsize   string `json:"default_bufsize"`
}

// UpdateSettingsRequest represents settings update request
type UpdateSettingsRequest struct {
	MaxChannels      *int    `json:"max_channels,omitempty"`
	SegmentTime      *int    `json:"segment_time,omitempty"`
	PlaylistSize     *int    `json:"playlist_size,omitempty"`
	LogRetention     *int    `json:"log_retention,omitempty"`
	DefaultPreset    *string `json:"default_preset,omitempty"`
	DefaultBitrate   *string `json:"default_bitrate,omitempty"`
	DefaultResolution *string `json:"default_resolution,omitempty"`
	DefaultProfile   *string `json:"default_profile,omitempty"`
	DefaultCRF       *int    `json:"default_crf,omitempty"`
	DefaultMaxrate   *string `json:"default_maxrate,omitempty"`
	DefaultBufsize   *string `json:"default_bufsize,omitempty"`
}

// Get returns current settings
func (h *SettingsHandler) Get(c *fiber.Ctx) error {
	settings, err := h.service.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": settings,
	})
}

// Update updates settings
func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi: " + err.Error(),
		})
	}

	// Check if any channel is running
	if err := h.service.CheckRunningChannels(); err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "ayarlar güncellenemez: " + err.Error(),
		})
	}

	settings, err := h.service.UpdateSettings(
		req.MaxChannels,
		req.SegmentTime,
		req.PlaylistSize,
		req.LogRetention,
		req.DefaultPreset,
		req.DefaultBitrate,
		req.DefaultResolution,
		req.DefaultProfile,
		req.DefaultCRF,
		req.DefaultMaxrate,
		req.DefaultBufsize,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": settings,
	})
}

