package domain

import (
	"time"

	"github.com/google/uuid"
)

// TranscoderProcess represents an active FFmpeg process
type TranscoderProcess struct {
	ChannelID     uuid.UUID `json:"channel_id"`
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"started_at"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   int64     `json:"memory_usage"`
	InputBitrate  int       `json:"input_bitrate"`
	OutputBitrate int       `json:"output_bitrate"`
	DroppedFrames int       `json:"dropped_frames"`
	FPS           float64   `json:"fps"`
	Speed         float64   `json:"speed"`
	LastError     string    `json:"last_error,omitempty"`
	Uptime        int64     `json:"uptime"`
}

// ProcessMetrics holds real-time metrics from FFmpeg
type ProcessMetrics struct {
	Frame         int64   `json:"frame"`
	FPS           float64 `json:"fps"`
	Bitrate       string  `json:"bitrate"`
	TotalSize     int64   `json:"total_size"`
	OutTimeMs     int64   `json:"out_time_ms"`
	DupFrames     int     `json:"dup_frames"`
	DropFrames    int     `json:"drop_frames"`
	Speed         string  `json:"speed"`
	Progress      string  `json:"progress"`
}

// SystemMetrics holds system-wide metrics
type SystemMetrics struct {
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryTotal    int64     `json:"memory_total"`
	MemoryUsed     int64     `json:"memory_used"`
	MemoryPercent  float64   `json:"memory_percent"`
	DiskTotal      int64     `json:"disk_total"`
	DiskUsed       int64     `json:"disk_used"`
	DiskPercent    float64   `json:"disk_percent"`
	NetworkRxBytes int64     `json:"network_rx_bytes"`
	NetworkTxBytes int64     `json:"network_tx_bytes"`
	LoadAverage    []float64 `json:"load_average"`
	Uptime         int64     `json:"uptime"`
	Timestamp      time.Time `json:"timestamp"`
}

// StreamInfo holds HLS stream information
type StreamInfo struct {
	ChannelID       uuid.UUID `json:"channel_id"`
	PlaylistURL     string    `json:"playlist_url"`
	SegmentCount    int       `json:"segment_count"`
	LastSegmentTime time.Time `json:"last_segment_time"`
	Viewers         int       `json:"viewers"`
}

// GPUInfo holds information about a single GPU
type GPUInfo struct {
	ID          string  `json:"id"`           // GPU ID (e.g., 0)
	Name        string  `json:"name"`         // GPU name (e.g., NVIDIA GeForce RTX 3060)
	Utilization float64 `json:"utilization"`  // GPU utilization percentage
	MemoryUsed  int64   `json:"memory_used"`  // Used GPU memory in bytes
	MemoryTotal int64   `json:"memory_total"` // Total GPU memory in bytes
	Temperature int     `json:"temperature"`  // GPU temperature in Celsius
}

// SystemInfo holds system hardware and resource information
type SystemInfo struct {
	CPUCores        int       `json:"cpu_cores"`         // Total CPU cores
	CPUThreads      int       `json:"cpu_threads"`       // Total CPU threads (with HT)
	CPUUsage        float64   `json:"cpu_usage"`         // Current CPU usage percentage
	MemoryTotal     int64     `json:"memory_total"`      // Total memory in bytes
	MemoryUsed      int64     `json:"memory_used"`       // Used memory in bytes
	MemoryAvailable int64     `json:"memory_available"`  // Available memory in bytes
	MemoryPercent   float64   `json:"memory_percent"`   // Memory usage percentage
	LoadAverage1    float64   `json:"load_average_1"`   // 1-minute load average
	LoadAverage5    float64   `json:"load_average_5"`    // 5-minute load average
	LoadAverage15   float64   `json:"load_average_15"`   // 15-minute load average
	Uptime          int64     `json:"uptime"`           // System uptime in seconds
	GPUs            []GPUInfo `json:"gpus"`              // GPU information
}

// TranscoderManager defines the interface for transcoder operations
type TranscoderManager interface {
	Start(channel *Channel) error
	Stop(channelID uuid.UUID) error
	Restart(channelID uuid.UUID) error
	GetProcess(channelID uuid.UUID) (*TranscoderProcess, error)
	GetAllProcesses() ([]*TranscoderProcess, error)
	IsRunning(channelID uuid.UUID) bool
	GetLogs(channelID uuid.UUID) ([]string, error)
}

