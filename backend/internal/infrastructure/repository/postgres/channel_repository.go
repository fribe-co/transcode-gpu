package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChannelRepository implements domain.ChannelRepository with PostgreSQL
type ChannelRepository struct {
	db *pgxpool.Pool
}

// NewChannelRepository creates a new PostgreSQL channel repository
func NewChannelRepository(db *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// Create inserts a new channel
func (r *ChannelRepository) Create(channel *domain.Channel) error {
	ctx := context.Background()

	logoJSON, _ := json.Marshal(channel.Logo)
	outputJSON, _ := json.Marshal(channel.OutputConfig)

	query := `
		INSERT INTO channels (id, name, source_url, logo, output_config, status, auto_restart, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		channel.ID,
		channel.Name,
		channel.SourceURL,
		logoJSON,
		outputJSON,
		channel.Status,
		channel.AutoRestart,
		channel.CreatedAt,
		channel.UpdatedAt,
	)

	return err
}

// GetByID retrieves a channel by ID
func (r *ChannelRepository) GetByID(id uuid.UUID) (*domain.Channel, error) {
	ctx := context.Background()

	query := `
		SELECT id, name, source_url, logo, output_config, status, auto_restart, created_at, updated_at
		FROM channels WHERE id = $1
	`

	var channel domain.Channel
	var logoJSON, outputJSON sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&channel.ID,
		&channel.Name,
		&channel.SourceURL,
		&logoJSON,
		&outputJSON,
		&channel.Status,
		&channel.AutoRestart,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	if logoJSON.Valid {
		json.Unmarshal([]byte(logoJSON.String), &channel.Logo)
	}
	if outputJSON.Valid {
		json.Unmarshal([]byte(outputJSON.String), &channel.OutputConfig)
	}

	return &channel, nil
}

// GetAll retrieves all channels
func (r *ChannelRepository) GetAll() ([]*domain.Channel, error) {
	ctx := context.Background()

	query := `
		SELECT id, name, source_url, logo, output_config, status, auto_restart, created_at, updated_at
		FROM channels ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*domain.Channel
	for rows.Next() {
		var channel domain.Channel
		var logoJSON, outputJSON sql.NullString

		err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.SourceURL,
			&logoJSON,
			&outputJSON,
			&channel.Status,
			&channel.AutoRestart,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if logoJSON.Valid {
			json.Unmarshal([]byte(logoJSON.String), &channel.Logo)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &channel.OutputConfig)
		}

		channels = append(channels, &channel)
	}

	return channels, nil
}

// Update updates an existing channel
func (r *ChannelRepository) Update(channel *domain.Channel) error {
	ctx := context.Background()

	logoJSON, _ := json.Marshal(channel.Logo)
	outputJSON, _ := json.Marshal(channel.OutputConfig)

	query := `
		UPDATE channels 
		SET name = $1, source_url = $2, logo = $3, output_config = $4, auto_restart = $5, updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.Exec(ctx, query,
		channel.Name,
		channel.SourceURL,
		logoJSON,
		outputJSON,
		channel.AutoRestart,
		time.Now(),
		channel.ID,
	)

	return err
}

// Delete removes a channel
func (r *ChannelRepository) Delete(id uuid.UUID) error {
	ctx := context.Background()
	query := `DELETE FROM channels WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// UpdateStatus updates channel status
func (r *ChannelRepository) UpdateStatus(id uuid.UUID, status domain.ChannelStatus) error {
	ctx := context.Background()
	query := `UPDATE channels SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, time.Now(), id)
	return err
}





