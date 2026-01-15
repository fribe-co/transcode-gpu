package application

import (
	"errors"
	"fmt"
)

var (
	ErrSettingsNotFound = errors.New("settings not found")
	ErrChannelsRunning  = errors.New("aktif yayın var, ayarlar güncellenemez")
)

// SettingsRepository defines the interface for settings persistence
type SettingsRepository interface {
	GetSystemSettings() (map[string]interface{}, error)
	UpdateSystemSettings(settings map[string]interface{}) error
}

// SettingsService handles settings business logic
type SettingsService struct {
	channelService *ChannelService
	repo           SettingsRepository
}

// NewSettingsService creates a new settings service
func NewSettingsService(channelService *ChannelService, repo SettingsRepository) *SettingsService {
	return &SettingsService{
		channelService: channelService,
		repo:           repo,
	}
}

// Settings represents system settings
type Settings struct {
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

// GetSettings retrieves current settings
func (s *SettingsService) GetSettings() (*Settings, error) {
	dbSettings, err := s.repo.GetSystemSettings()
	if err != nil {
		return nil, err
	}

	settings := &Settings{
		MaxChannels:      80,
		SegmentTime:      3,  // 3 seconds - optimal for stability and latency (70 streams)
		PlaylistSize:     6,  // 6 segments (18 seconds buffer)
		LogRetention:     1,
		DefaultPreset:    "veryfast",  // Better quality/stability balance than ultrafast
		DefaultBitrate:   "3500k",
		DefaultResolution: "1920x1080",
		DefaultProfile:   "high",
		DefaultCRF:       23,
		DefaultMaxrate:   "3800k",
		DefaultBufsize:   "7600k",
	}

	// Map database values to settings struct
	if val, ok := dbSettings["max_channels"]; ok {
		if v, ok := val.(float64); ok {
			settings.MaxChannels = int(v)
		} else if v, ok := val.(int); ok {
			settings.MaxChannels = v
		}
	}
	if val, ok := dbSettings["segment_time"]; ok {
		if v, ok := val.(float64); ok {
			settings.SegmentTime = int(v)
		} else if v, ok := val.(int); ok {
			settings.SegmentTime = v
		}
	}
	if val, ok := dbSettings["playlist_size"]; ok {
		if v, ok := val.(float64); ok {
			settings.PlaylistSize = int(v)
		} else if v, ok := val.(int); ok {
			settings.PlaylistSize = v
		}
	}
	if val, ok := dbSettings["log_retention"]; ok {
		if v, ok := val.(float64); ok {
			settings.LogRetention = int(v)
		} else if v, ok := val.(int); ok {
			settings.LogRetention = v
		}
	}
	if val, ok := dbSettings["default_preset"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultPreset = v
		}
	}
	if val, ok := dbSettings["default_bitrate"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultBitrate = v
		}
	}
	if val, ok := dbSettings["default_resolution"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultResolution = v
		}
	}
	if val, ok := dbSettings["default_profile"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultProfile = v
		}
	}
	if val, ok := dbSettings["default_crf"]; ok {
		if v, ok := val.(float64); ok {
			settings.DefaultCRF = int(v)
		} else if v, ok := val.(int); ok {
			settings.DefaultCRF = v
		}
	}
	if val, ok := dbSettings["default_maxrate"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultMaxrate = v
		}
	}
	if val, ok := dbSettings["default_bufsize"]; ok {
		if v, ok := val.(string); ok {
			settings.DefaultBufsize = v
		}
	}

	return settings, nil
}

// CheckRunningChannels checks if any channel is currently running
func (s *SettingsService) CheckRunningChannels() error {
	channels, err := s.channelService.ListChannels()
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.Status == "running" {
			return ErrChannelsRunning
		}
	}

	return nil
}

// UpdateSettings updates system settings
func (s *SettingsService) UpdateSettings(
	maxChannels *int,
	segmentTime *int,
	playlistSize *int,
	logRetention *int,
	defaultPreset *string,
	defaultBitrate *string,
	defaultResolution *string,
	defaultProfile *string,
	defaultCRF *int,
	defaultMaxrate *string,
	defaultBufsize *string,
) (*Settings, error) {
	// Check if any channel is running
	if err := s.CheckRunningChannels(); err != nil {
		return nil, err
	}

	// Get current settings
	current, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	// Update only provided fields
	if maxChannels != nil {
		if *maxChannels < 1 || *maxChannels > 1000 {
			return nil, fmt.Errorf("maksimum kanal sayısı 1-1000 arasında olmalıdır")
		}
		current.MaxChannels = *maxChannels
	}
	if segmentTime != nil {
		if *segmentTime < 1 || *segmentTime > 30 {
			return nil, fmt.Errorf("segment süresi 1-30 saniye arasında olmalıdır")
		}
		current.SegmentTime = *segmentTime
	}
	if playlistSize != nil {
		if *playlistSize < 1 || *playlistSize > 100 {
			return nil, fmt.Errorf("playlist boyutu 1-100 arasında olmalıdır")
		}
		current.PlaylistSize = *playlistSize
	}
	if logRetention != nil {
		if *logRetention < 1 || *logRetention > 365 {
			return nil, fmt.Errorf("log saklama süresi 1-365 gün arasında olmalıdır")
		}
		current.LogRetention = *logRetention
	}
	if defaultPreset != nil {
		validPresets := map[string]bool{
			"ultrafast": true, "superfast": true, "veryfast": true,
			"faster": true, "fast": true, "medium": true,
			"slow": true, "slower": true, "veryslow": true,
		}
		if !validPresets[*defaultPreset] {
			return nil, fmt.Errorf("geçersiz preset: %s", *defaultPreset)
		}
		current.DefaultPreset = *defaultPreset
	}
	if defaultBitrate != nil {
		current.DefaultBitrate = *defaultBitrate
	}
	if defaultResolution != nil {
		current.DefaultResolution = *defaultResolution
	}
	if defaultProfile != nil {
		validProfiles := map[string]bool{
			"baseline": true, "main": true, "high": true,
		}
		if !validProfiles[*defaultProfile] {
			return nil, fmt.Errorf("geçersiz profil: %s", *defaultProfile)
		}
		current.DefaultProfile = *defaultProfile
	}
	if defaultCRF != nil {
		if *defaultCRF < 0 || *defaultCRF > 51 {
			return nil, fmt.Errorf("CRF değeri 0-51 arasında olmalıdır")
		}
		current.DefaultCRF = *defaultCRF
	}
	if defaultMaxrate != nil {
		current.DefaultMaxrate = *defaultMaxrate
	}
	if defaultBufsize != nil {
		current.DefaultBufsize = *defaultBufsize
	}

	// Save to database
	dbSettings := map[string]interface{}{
		"max_channels":       current.MaxChannels,
		"segment_time":       current.SegmentTime,
		"playlist_size":      current.PlaylistSize,
		"log_retention":      current.LogRetention,
		"default_preset":     current.DefaultPreset,
		"default_bitrate":    current.DefaultBitrate,
		"default_resolution": current.DefaultResolution,
		"default_profile":    current.DefaultProfile,
		"default_crf":        current.DefaultCRF,
		"default_maxrate":    current.DefaultMaxrate,
		"default_bufsize":    current.DefaultBufsize,
	}

	if err := s.repo.UpdateSystemSettings(dbSettings); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	return current, nil
}

