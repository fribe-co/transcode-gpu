package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/cashbacktv/backend/internal/application"
	"github.com/cashbacktv/backend/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ChannelHandler handles HTTP requests for channels
type ChannelHandler struct {
	service   *application.ChannelService
	hlsPath   string
	logoPath  string
	ffmpegBin string
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(service *application.ChannelService) *ChannelHandler {
	return &ChannelHandler{service: service}
}

// NewChannelHandlerWithPaths creates a new channel handler with paths
func NewChannelHandlerWithPaths(service *application.ChannelService, hlsPath, logoPath string) *ChannelHandler {
	return &ChannelHandler{
		service:  service,
		hlsPath:  hlsPath,
		logoPath: logoPath,
		ffmpegBin: "/usr/bin/ffmpeg", // Default FFmpeg path
	}
}

// NewChannelHandlerWithFFmpeg creates a new channel handler with paths and FFmpeg binary
func NewChannelHandlerWithFFmpeg(service *application.ChannelService, hlsPath, logoPath, ffmpegBin string) *ChannelHandler {
	return &ChannelHandler{
		service:   service,
		hlsPath:   hlsPath,
		logoPath:  logoPath,
		ffmpegBin: ffmpegBin,
	}
}

// CreateChannelRequest represents channel creation request
type CreateChannelRequest struct {
	Name         string              `json:"name" validate:"required"`
	SourceURL    string              `json:"source_url" validate:"required,url"`
	Logo         *domain.LogoConfig  `json:"logo,omitempty"`
	OutputConfig *domain.OutputConfig `json:"output_config,omitempty"`
}

// UpdateChannelRequest represents channel update request
type UpdateChannelRequest struct {
	Name         string              `json:"name,omitempty"`
	SourceURL    string              `json:"source_url,omitempty"`
	Logo         *domain.LogoConfig  `json:"logo,omitempty"`
	OutputConfig *domain.OutputConfig `json:"output_config,omitempty"`
}

// List returns all channels
func (h *ChannelHandler) List(c *fiber.Ctx) error {
	channels, err := h.service.ListChannels()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": channels,
	})
}

// Get returns a single channel
func (h *ChannelHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	channel, err := h.service.GetChannel(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": channel,
	})
}

// Create creates a new channel
func (h *ChannelHandler) Create(c *fiber.Ctx) error {
	var req CreateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	channel, err := h.service.CreateChannel(req.Name, req.SourceURL, req.Logo, req.OutputConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": channel,
	})
}

// Update updates an existing channel
func (h *ChannelHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	var req UpdateChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	channel, err := h.service.UpdateChannel(id, req.Name, req.SourceURL, req.Logo, req.OutputConfig)
	if err != nil {
		if err == application.ErrChannelNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err == application.ErrChannelRunning {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "çalışan kanal güncellenemez",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": channel,
	})
}

// Delete removes a channel
func (h *ChannelHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	if err := h.service.DeleteChannel(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"message": "kanal silindi",
		},
	})
}

// Start starts transcoding for a channel
func (h *ChannelHandler) Start(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	if err := h.service.StartChannel(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"message": "kanal başlatıldı",
		},
	})
}

// Stop stops transcoding for a channel
func (h *ChannelHandler) Stop(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	if err := h.service.StopChannel(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"message": "kanal durduruldu",
		},
	})
}

// Restart restarts transcoding for a channel
func (h *ChannelHandler) Restart(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	if err := h.service.RestartChannel(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"message": "kanal yeniden başlatıldı",
		},
	})
}

// Metrics returns transcoding metrics for a channel
func (h *ChannelHandler) Metrics(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	metrics, err := h.service.GetChannelMetrics(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "kanal çalışmıyor veya metrikler mevcut değil",
		})
	}

	return c.JSON(fiber.Map{
		"data": metrics,
	})
}

// AllMetrics returns transcoding metrics for all running channels (optimized batch endpoint)
func (h *ChannelHandler) AllMetrics(c *fiber.Ctx) error {
	metricsMap, err := h.service.GetAllChannelMetrics()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "metrikler alınamadı: " + err.Error(),
		})
	}

	// Convert map to array for easier frontend consumption
	metricsList := make([]*domain.TranscoderProcess, 0, len(metricsMap))
	for _, metrics := range metricsMap {
		metricsList = append(metricsList, metrics)
	}

	return c.JSON(fiber.Map{
		"data": metricsList,
	})
}

// Logs returns transcoding logs for a channel
func (h *ChannelHandler) Logs(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz kanal ID",
		})
	}

	logs, err := h.service.GetChannelLogs(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": logs,
	})
}

// BatchStartRequest represents batch start request
type BatchStartRequest struct {
	ChannelIDs []string `json:"channel_ids" validate:"required,min=1"`
}

// BatchStopRequest represents batch stop request
type BatchStopRequest struct {
	ChannelIDs []string `json:"channel_ids" validate:"required,min=1"`
}

// BatchRestartRequest represents batch restart request
type BatchRestartRequest struct {
	ChannelIDs []string `json:"channel_ids" validate:"required,min=1"`
}

// BatchDeleteRequest represents batch delete request
type BatchDeleteRequest struct {
	ChannelIDs []string `json:"channel_ids" validate:"required,min=1"`
}

// BatchStart starts multiple channels
func (h *ChannelHandler) BatchStart(c *fiber.Ctx) error {
	var req BatchStartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	if len(req.ChannelIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "en az bir kanal ID gerekli",
		})
	}

	// Parse UUIDs
	ids := make([]uuid.UUID, 0, len(req.ChannelIDs))
	for _, idStr := range req.ChannelIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("geçersiz kanal ID: %s", idStr),
			})
		}
		ids = append(ids, id)
	}

	result, err := h.service.BatchStartChannels(ids)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// BatchStop stops multiple channels
func (h *ChannelHandler) BatchStop(c *fiber.Ctx) error {
	var req BatchStopRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	if len(req.ChannelIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "en az bir kanal ID gerekli",
		})
	}

	// Parse UUIDs
	ids := make([]uuid.UUID, 0, len(req.ChannelIDs))
	for _, idStr := range req.ChannelIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("geçersiz kanal ID: %s", idStr),
			})
		}
		ids = append(ids, id)
	}

	result, err := h.service.BatchStopChannels(ids)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// BatchRestart restarts multiple channels
func (h *ChannelHandler) BatchRestart(c *fiber.Ctx) error {
	var req BatchRestartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	if len(req.ChannelIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "en az bir kanal ID gerekli",
		})
	}

	// Parse UUIDs
	ids := make([]uuid.UUID, 0, len(req.ChannelIDs))
	for _, idStr := range req.ChannelIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("geçersiz kanal ID: %s", idStr),
			})
		}
		ids = append(ids, id)
	}

	result, err := h.service.BatchRestartChannels(ids)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// BatchDelete deletes multiple channels
func (h *ChannelHandler) BatchDelete(c *fiber.Ctx) error {
	var req BatchDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz istek gövdesi",
		})
	}

	if len(req.ChannelIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "en az bir kanal ID gerekli",
		})
	}

	// Parse UUIDs
	ids := make([]uuid.UUID, 0, len(req.ChannelIDs))
	for _, idStr := range req.ChannelIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("geçersiz kanal ID: %s", idStr),
			})
		}
		ids = append(ids, id)
	}

	result, err := h.service.BatchDeleteChannels(ids)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": result,
	})
}

// ServeStream handles HLS stream requests
func (h *ChannelHandler) ServeStream(c *fiber.Ctx) error {
	channelIDStr := c.Params("channelId")
	
	// Check if regular m3u8 file exists (live stream)
	if h.hlsPath != "" {
		m3u8Path := filepath.Join(h.hlsPath, channelIDStr, "index.m3u8")
		if _, err := os.Stat(m3u8Path); err == nil {
			// File exists, serve it directly
			return c.SendFile(m3u8Path)
		}
	}

	// Stream not available
	return c.Status(fiber.StatusNotFound).SendString("Stream not available")
}

