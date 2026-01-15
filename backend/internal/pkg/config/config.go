package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	FFmpeg   FFmpegConfig   `mapstructure:"ffmpeg"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int    `mapstructure:"max_conns"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	ExpirationHours int    `mapstructure:"expiration_hours"`
	RefreshHours    int    `mapstructure:"refresh_hours"`
}

// FFmpegConfig holds FFmpeg configuration
type FFmpegConfig struct {
	BinaryPath     string `mapstructure:"binary_path"`
	WorkerCount    int    `mapstructure:"worker_count"`
	SegmentTime    int    `mapstructure:"segment_time"`
	PlaylistSize   int    `mapstructure:"playlist_size"`
	DefaultPreset  string `mapstructure:"default_preset"`
	DefaultBitrate string `mapstructure:"default_bitrate"`
}

// StorageConfig holds storage paths configuration
type StorageConfig struct {
	HLSPath    string `mapstructure:"hls_path"`
	LogoPath   string `mapstructure:"logo_path"`
	UploadPath string `mapstructure:"upload_path"`
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/cashbacktv")

	// Environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 60)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "cashbacktv")
	viper.SetDefault("database.password", "cashbacktv")
	viper.SetDefault("database.dbname", "cashbacktv")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_conns", 50)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// JWT defaults
	viper.SetDefault("jwt.secret", "your-super-secret-key-change-in-production")
	viper.SetDefault("jwt.expiration_hours", 24)
	viper.SetDefault("jwt.refresh_hours", 168)

	// FFmpeg defaults
	viper.SetDefault("ffmpeg.binary_path", "/usr/bin/ffmpeg")
	viper.SetDefault("ffmpeg.worker_count", 10)
	viper.SetDefault("ffmpeg.segment_time", 6)
	viper.SetDefault("ffmpeg.playlist_size", 10)
	viper.SetDefault("ffmpeg.default_preset", "ultrafast")
	viper.SetDefault("ffmpeg.default_bitrate", "5000k")

	// Storage defaults
	viper.SetDefault("storage.hls_path", "/var/lib/cashbacktv/streams")
	viper.SetDefault("storage.logo_path", "/var/lib/cashbacktv/logos")
	viper.SetDefault("storage.upload_path", "/var/lib/cashbacktv/uploads")
}

// DSN returns PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// Addr returns Redis address
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Addr returns server address
func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}





