package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChannelStatus represents the current state of a channel
type ChannelStatus string

const (
	ChannelStatusStopped  ChannelStatus = "stopped"
	ChannelStatusStarting ChannelStatus = "starting"
	ChannelStatusRunning  ChannelStatus = "running"
	ChannelStatusError    ChannelStatus = "error"
	ChannelStatusStopping ChannelStatus = "stopping"
)

// LogoConfig represents logo overlay configuration
type LogoConfig struct {
	Path    string  `json:"path"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Opacity float64 `json:"opacity"`
}

// OutputConfig represents encoding output configuration
type OutputConfig struct {
	Codec      string `json:"codec"`
	Bitrate    string `json:"bitrate"`
	Resolution string `json:"resolution"`
	Preset     string `json:"preset"`
	Profile    string `json:"profile"`
}

// Channel represents a video channel entity
type Channel struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	SourceURL      string        `json:"source_url"`
	OutputURL      string        `json:"output_url,omitempty"`
	Logo           *LogoConfig   `json:"logo,omitempty"`
	OutputConfig   *OutputConfig `json:"output_config,omitempty"`
	Status         ChannelStatus `json:"status"`
	AutoRestart    bool          `json:"auto_restart"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// NewChannel creates a new channel with default values
func NewChannel(name, sourceURL string) *Channel {
	now := time.Now()
	return &Channel{
		ID:        uuid.New(),
		Name:      name,
		SourceURL: sourceURL,
		Status:    ChannelStatusStopped,
		OutputConfig: &OutputConfig{
			Codec:      "libx264",
			Bitrate:    "5000k",
			Resolution: "1920x1080",
			Preset:     "ultrafast",
			Profile:    "high",
		},
		AutoRestart: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ChannelRepository defines the interface for channel persistence
type ChannelRepository interface {
	Create(channel *Channel) error
	GetByID(id uuid.UUID) (*Channel, error)
	GetAll() ([]*Channel, error)
	Update(channel *Channel) error
	Delete(id uuid.UUID) error
	UpdateStatus(id uuid.UUID, status ChannelStatus) error
}

