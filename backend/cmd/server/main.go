package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cashbacktv/backend/internal/application"
	"github.com/cashbacktv/backend/internal/domain"
	"github.com/cashbacktv/backend/internal/infrastructure/ffmpeg"
	"github.com/cashbacktv/backend/internal/infrastructure/repository/postgres"
	"github.com/cashbacktv/backend/internal/interfaces/http"
	"github.com/cashbacktv/backend/internal/interfaces/http/handlers"
	"github.com/cashbacktv/backend/internal/interfaces/http/middleware"
	"github.com/cashbacktv/backend/internal/pkg/config"
	"github.com/cashbacktv/backend/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func main() {
	// Initialize logger
	logger.Init("info", true)
	log := logger.Get()

	log.Info().Msg("Starting CashbackTV Backend...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to PostgreSQL
	dbPool, err := connectDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer dbPool.Close()

	log.Info().Msg("Connected to PostgreSQL")

	// Run migrations to ensure database schema is up to date (without dropping existing data)
	runMigrations(dbPool, log)

	// Initialize repositories
	channelRepo := postgres.NewChannelRepository(dbPool)
	userRepo := postgres.NewUserRepository(dbPool)
	settingsRepo := postgres.NewSettingsRepository(dbPool)

	// Initialize FFmpeg process manager
	ffmpegConfig := &ffmpeg.Config{
		BinaryPath:    cfg.FFmpeg.BinaryPath,
		SegmentTime:   cfg.FFmpeg.SegmentTime,
		PlaylistSize:  cfg.FFmpeg.PlaylistSize,
		DefaultPreset: cfg.FFmpeg.DefaultPreset,
		DefaultBitrate: cfg.FFmpeg.DefaultBitrate,
	}
	processManager := ffmpeg.NewProcessManager(ffmpegConfig, cfg.Storage.HLSPath, cfg.Storage.LogoPath, settingsRepo)

	// Initialize services
	channelService := application.NewChannelService(channelRepo, processManager)
	
	// Set status callback for ProcessManager to update channel status when FFmpeg fails to start
	processManager.SetStatusCallback(func(channelID uuid.UUID, status domain.ChannelStatus) error {
		return channelRepo.UpdateStatus(channelID, status)
	})
	authService := application.NewAuthService(
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.ExpirationHours,
		cfg.JWT.RefreshHours,
	)
	settingsService := application.NewSettingsService(channelService, settingsRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	channelHandler := handlers.NewChannelHandlerWithFFmpeg(channelService, cfg.Storage.HLSPath, cfg.Storage.LogoPath, cfg.FFmpeg.BinaryPath)
	uploadHandler := handlers.NewUploadHandler(cfg.Storage.LogoPath, cfg.Storage.UploadPath)
	settingsHandler := handlers.NewSettingsHandler(settingsService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Setup router with server config for performance optimizations
	router := http.NewRouter(authHandler, channelHandler, uploadHandler, settingsHandler, authMiddleware, cfg.Storage.LogoPath, cfg.Storage.HLSPath, &cfg.Server)
	router.SetupRoutes()

	// Initialize startup tasks
	log.Info().Msg("Running startup initialization tasks...")
	
	// Clean HLS history (remove old segments)
	cleanHLSHistory(cfg.Storage.HLSPath, log)
	
	// Create default admin user if not exists
	createDefaultAdmin(authService)

	// Stop all running channels on startup (prevent auto-start)
	stopAllRunningChannels(channelRepo, log)

	// Start server in goroutine
	serverAddr := cfg.Server.Addr()
	go func() {
		log.Info().Str("address", serverAddr).Msg("Starting HTTP server")
		if err := router.Start(serverAddr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	if err := router.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error during shutdown")
	}

	log.Info().Msg("Server stopped")
}

func connectDB(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

func runMigrations(dbPool *pgxpool.Pool, log *zerolog.Logger) {
	log.Info().Msg("Running database migrations (preserving channels/users data, resetting settings)...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if tables exist by querying information_schema
	var tableCount int
	err := dbPool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('users', 'channels', 'channel_logs', 'settings')
	`).Scan(&tableCount)
	
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check existing tables, will attempt to create schema")
		tableCount = 0
	}

	// If tables don't exist, create them
	if tableCount < 4 {
		log.Info().Msg("Creating database schema...")

		// Migration SQL (only creates if not exists, preserves existing data)
		migrationSQL := `
			-- Enable UUID extension
			CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

			-- Users table
			CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				email VARCHAR(255) UNIQUE NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				name VARCHAR(255) NOT NULL,
				role VARCHAR(50) NOT NULL DEFAULT 'viewer',
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);

			-- Create index on email for faster lookups (if not exists)
			CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

			-- Channels table
			CREATE TABLE IF NOT EXISTS channels (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				name VARCHAR(255) NOT NULL,
				source_url TEXT NOT NULL,
				logo JSONB,
				output_config JSONB,
				status VARCHAR(50) NOT NULL DEFAULT 'stopped',
				auto_restart BOOLEAN DEFAULT true,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);

			-- Create index on status for filtering (if not exists)
			CREATE INDEX IF NOT EXISTS idx_channels_status ON channels(status);

			-- Channel logs table (for storing FFmpeg output history)
			CREATE TABLE IF NOT EXISTS channel_logs (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
				level VARCHAR(20) NOT NULL,
				message TEXT NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);

			-- Create indexes on channel_logs (if not exists)
			CREATE INDEX IF NOT EXISTS idx_channel_logs_channel_id ON channel_logs(channel_id);
			CREATE INDEX IF NOT EXISTS idx_channel_logs_created_at ON channel_logs(created_at);

			-- System settings table
			CREATE TABLE IF NOT EXISTS settings (
				key VARCHAR(255) PRIMARY KEY,
				value JSONB NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);

			-- Function to update updated_at timestamp
			CREATE OR REPLACE FUNCTION update_updated_at_column()
			RETURNS TRIGGER AS $$
			BEGIN
				NEW.updated_at = NOW();
				RETURN NEW;
			END;
			$$ language 'plpgsql';

			-- Triggers for updated_at (drop and recreate to ensure they exist)
			DROP TRIGGER IF EXISTS update_users_updated_at ON users;
			CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
				FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

			DROP TRIGGER IF EXISTS update_channels_updated_at ON channels;
			CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels
				FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

			DROP TRIGGER IF EXISTS update_settings_updated_at ON settings;
			CREATE TRIGGER update_settings_updated_at BEFORE UPDATE ON settings
				FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		`

		// Execute migration
		_, err = dbPool.Exec(ctx, migrationSQL)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to run database migrations")
		}

		log.Info().Msg("Database schema created successfully")
	} else {
		log.Info().Msg("Database schema already exists")
	}

	// Always reset settings to defaults on startup (preserve other data)
	// Settings are reset to optimized values for 70 streams on 2-node NUMA system
	log.Info().Msg("Resetting settings to optimized default values...")
	resetSettingsSQL := `
		-- Delete all existing settings
		DELETE FROM settings;

		-- Insert optimized default settings (for 70 streams on 2-node NUMA system)
		INSERT INTO settings (key, value) VALUES
			('encoding_presets', '[
				{"name": "High Quality", "preset": "slow", "bitrate": "6000k", "resolution": "1920x1080"},
				{"name": "Standard", "preset": "veryfast", "bitrate": "4000k", "resolution": "1920x1080"},
				{"name": "Low Bandwidth", "preset": "veryfast", "bitrate": "2000k", "resolution": "1280x720"}
			]'::jsonb),
			('system', '{
				"max_channels": 80,
				"segment_time": 3,
				"playlist_size": 6,
				"log_retention": 1,
				"default_preset": "veryfast",
				"default_bitrate": "3500k",
				"default_resolution": "1920x1080",
				"default_profile": "high",
				"default_crf": 23,
				"default_maxrate": "3800k",
				"default_bufsize": "7600k",
				"auto_restart_enabled": true,
				"use_ramdisk": true,
				"threads_per_process": 1
			}'::jsonb);
	`

	_, err = dbPool.Exec(ctx, resetSettingsSQL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to reset settings to defaults")
	}

	log.Info().Msg("Database migrations completed successfully (settings reset to defaults, channels/users data preserved)")
}

func createDefaultAdmin(authService *application.AuthService) {
	log := logger.Get()

	// Try to create default admin
	_, err := authService.CreateUser(
		"admin@cashbacktv.local",
		"C@shb@ckTV2024!L1ve",
		"Admin",
		"admin",
	)
	if err != nil {
		// User might already exist
		log.Debug().Err(err).Msg("Default admin user creation skipped (may already exist)")
	} else {
		log.Info().Msg("Created default admin user (admin@cashbacktv.local)")
	}
}

func stopAllRunningChannels(repo *postgres.ChannelRepository, log *zerolog.Logger) {
	// Get all channels
	channels, err := repo.GetAll()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get channels for startup cleanup")
		return
	}

	if len(channels) == 0 {
		log.Info().Msg("No channels found on startup")
		return
	}

	// Reset all channels' output_config to optimized default values and stop running ones
	// Use optimized values for 70 streams on 2-node NUMA system
	stoppedCount := 0
	resetCount := 0
	defaultOutputConfig := &domain.OutputConfig{
		Codec:      "libx264",
		Bitrate:    "3500k",
		Resolution: "1920x1080",
		Preset:     "veryfast",
		Profile:    "high",
	}

	for _, channel := range channels {
		// Reset output_config to defaults for all channels
		channel.OutputConfig = defaultOutputConfig
		if err := repo.Update(channel); err != nil {
			log.Warn().
				Str("channel_id", channel.ID.String()).
				Str("channel_name", channel.Name).
				Err(err).
				Msg("Failed to reset channel output_config on startup")
		} else {
			resetCount++
			log.Debug().
				Str("channel_id", channel.ID.String()).
				Str("channel_name", channel.Name).
				Msg("Reset channel output_config to defaults")
		}

		// Stop all running channels
		if channel.Status == domain.ChannelStatusRunning || 
		   channel.Status == domain.ChannelStatusStarting {
			err := repo.UpdateStatus(channel.ID, domain.ChannelStatusStopped)
			if err != nil {
				log.Warn().
					Str("channel_id", channel.ID.String()).
					Str("channel_name", channel.Name).
					Err(err).
					Msg("Failed to stop channel on startup")
			} else {
				stoppedCount++
				log.Info().
					Str("channel_id", channel.ID.String()).
					Str("channel_name", channel.Name).
					Msg("Stopped channel on startup")
			}
		}
	}

	if resetCount > 0 {
		log.Info().Int("count", resetCount).Msg("Reset all channels' output_config to defaults on startup")
	}
	if stoppedCount > 0 {
		log.Info().Int("count", stoppedCount).Msg("Stopped all running channels on startup")
	} else {
		log.Info().Msg("No running channels found on startup")
	}
}

func cleanHLSHistory(hlsPath string, log *zerolog.Logger) {
	log.Info().Str("hls_path", hlsPath).Msg("Cleaning HLS history...")
	
	// Remove all HLS segment directories (they will be recreated when channels start)
	err := filepath.Walk(hlsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the root directory
		if path == hlsPath {
			return nil
		}
		
		// Remove all files and directories in HLS path
		if info.IsDir() {
			// Remove directory and all its contents
			return os.RemoveAll(path)
		}
		
		// Remove file
		return os.Remove(path)
	})
	
	if err != nil {
		log.Warn().Err(err).Msg("Failed to clean HLS history (may not exist yet)")
	} else {
		log.Info().Msg("HLS history cleaned successfully")
	}
}

