package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	logoPath   string
	uploadPath string
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(logoPath, uploadPath string) *UploadHandler {
	// Ensure directories exist
	os.MkdirAll(logoPath, 0755)
	os.MkdirAll(uploadPath, 0755)

	return &UploadHandler{
		logoPath:   logoPath,
		uploadPath: uploadPath,
	}
}

// UploadLogoResponse represents the response for logo upload
type UploadLogoResponse struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// UploadLogo handles logo file upload
func (h *UploadHandler) UploadLogo(c *fiber.Ctx) error {
	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "logo dosyası gerekli",
		})
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "sadece PNG, JPG, GIF veya WebP formatları desteklenir",
		})
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "dosya boyutu maksimum 5MB olabilir",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(h.logoPath, filename)

	// Save file
	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "dosya kaydedilemedi",
		})
	}

	// Return relative path for logo (just filename, will be joined with logoPath in FFmpeg)
	return c.JSON(fiber.Map{
		"data": UploadLogoResponse{
			Path:     filename, // Store just filename, not full path
			Filename: filename,
			URL:      "/logos/" + filename,
		},
	})
}

// DeleteLogo removes a logo file
func (h *UploadHandler) DeleteLogo(c *fiber.Ctx) error {
	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "dosya adı gerekli",
		})
	}

	// Security: prevent directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "geçersiz dosya adı",
		})
	}

	filePath := filepath.Join(h.logoPath, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "dosya bulunamadı",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "dosya silinemedi",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

