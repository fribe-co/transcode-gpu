package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepository handles settings persistence
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// GetSystemSettings retrieves system settings from database
func (r *SettingsRepository) GetSystemSettings() (map[string]interface{}, error) {
	ctx := context.Background()
	
	var value json.RawMessage
	err := r.pool.QueryRow(ctx, "SELECT value FROM settings WHERE key = $1", "system").Scan(&value)
	if err != nil {
		// Return defaults if not found (optimized for 70 streams on 2-node NUMA system)
		return map[string]interface{}{
			"max_channels":        80,
			"segment_time":        3,  // 3 seconds - optimal for stability and latency
			"playlist_size":       6,  // 6 segments (18 seconds buffer)
			"log_retention":       1,
			"default_preset":      "veryfast",  // Better quality/stability balance than ultrafast
			"default_bitrate":     "3500k",
			"default_resolution":  "1920x1080",
			"default_profile":     "high",
			"default_crf":         23,
			"default_maxrate":     "3800k",
			"default_bufsize":     "7600k",
			"use_ramdisk":         true,
			"threads_per_process": 1,
		}, nil
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(value, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return settings, nil
}

// UpdateSystemSettings updates system settings in database
func (r *SettingsRepository) UpdateSystemSettings(settings map[string]interface{}) error {
	ctx := context.Background()

	valueJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) 
		DO UPDATE SET value = $2, updated_at = NOW()
	`

	_, err = r.pool.Exec(ctx, query, "system", valueJSON)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}

// GetEncodingPresets retrieves encoding presets from database
func (r *SettingsRepository) GetEncodingPresets() ([]map[string]interface{}, error) {
	ctx := context.Background()
	
	var value json.RawMessage
	err := r.pool.QueryRow(ctx, "SELECT value FROM settings WHERE key = $1", "encoding_presets").Scan(&value)
	if err != nil {
		return []map[string]interface{}{}, nil
	}

	var presets []map[string]interface{}
	if err := json.Unmarshal(value, &presets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal presets: %w", err)
	}

	return presets, nil
}

// UpdateEncodingPresets updates encoding presets in database
func (r *SettingsRepository) UpdateEncodingPresets(presets []map[string]interface{}) error {
	ctx := context.Background()

	valueJSON, err := json.Marshal(presets)
	if err != nil {
		return fmt.Errorf("failed to marshal presets: %w", err)
	}

	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) 
		DO UPDATE SET value = $2, updated_at = NOW()
	`

	_, err = r.pool.Exec(ctx, query, "encoding_presets", valueJSON)
	if err != nil {
		return fmt.Errorf("failed to update presets: %w", err)
	}

	return nil
}

