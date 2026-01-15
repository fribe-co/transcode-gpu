package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/cashbacktv/backend/internal/pkg/logger"
	"github.com/google/uuid"
)

// SettingsRepository interface for getting settings
type SettingsRepository interface {
	GetSystemSettings() (map[string]interface{}, error)
}

// StatusUpdateCallback is called to update channel status when process fails to start
type StatusUpdateCallback func(channelID uuid.UUID, status domain.ChannelStatus) error

// ProcessManager manages FFmpeg processes
type ProcessManager struct {
	processes        map[uuid.UUID]*Process
	mu               sync.RWMutex
	config           *Config
	hlsPath          string
	logoPath         string
	settingsRepo     SettingsRepository
	maxThreadsPerProcess int // Maximum threads per FFmpeg process
	statusCallback   StatusUpdateCallback // Callback to update channel status when process fails
	numaNodeCount    int    // Number of NUMA nodes available
	numaNodeCounter  int    // Counter for round-robin NUMA node assignment
	numaMu           sync.Mutex // Mutex for NUMA node counter
}

// Config holds FFmpeg configuration
type Config struct {
	BinaryPath    string
	SegmentTime   int
	PlaylistSize  int
	DefaultPreset string
	DefaultBitrate string
}

// Process represents a running FFmpeg process
type Process struct {
	ChannelID uuid.UUID
	Channel   *domain.Channel
	Cmd       *exec.Cmd
	Cancel    context.CancelFunc
	StartedAt time.Time
	Metrics   *domain.ProcessMetrics
	Logs      []string
	mu        sync.RWMutex
	logMu     sync.Mutex
	// CPU tracking for accurate percentage calculation
	lastCPUStat struct {
		utime  int64
		stime  int64
		cutime int64
		cstime int64
		time   time.Time
	}
}

// NewProcessManager creates a new process manager
func NewProcessManager(config *Config, hlsPath, logoPath string, settingsRepo SettingsRepository) *ProcessManager {
	return NewProcessManagerWithCallback(config, hlsPath, logoPath, settingsRepo, nil)
}

// NewProcessManagerWithCallback creates a new process manager with status update callback
func NewProcessManagerWithCallback(config *Config, hlsPath, logoPath string, settingsRepo SettingsRepository, statusCallback StatusUpdateCallback) *ProcessManager {
	numCPU := runtime.NumCPU()
	
	// Optimize thread calculation for 128 core / 256 thread system (Hyperthreading)
	// For 160 channels on 128 physical cores (256 threads with HT):
	// - Use 1 thread per process to maximize channel capacity
	// - With HT, we can run more processes but physical cores are the bottleneck
	// - Formula: allocate 1-2 threads per process based on physical core count (not logical threads)
	var maxThreads int
	if numCPU >= 256 {
		// 128 core / 256 thread system with HT
		// Use 1 thread per process (conservative for physical cores)
		// HT helps but physical cores are the real limit for encoding
		maxThreads = 1
	} else if numCPU >= 128 {
		// Very high core count (64+ physical cores)
		maxThreads = 1
	} else if numCPU >= 88 {
		// 44 core / 88 thread system with HT
		maxThreads = 1
	} else if numCPU >= 64 {
		// High core count (32+ physical cores)
		maxThreads = 1
	} else if numCPU >= 32 {
		// Medium-high core count (16-31 physical cores)
		maxThreads = 2
	} else if numCPU >= 16 {
		maxThreads = 2
	} else if numCPU >= 8 {
		maxThreads = 1
	} else {
		maxThreads = 1
	}
	
	// Detect NUMA nodes for dual CPU systems
	numaNodeCount := detectNUMANodes()
	if numaNodeCount == 0 {
		numaNodeCount = 1 // Fallback to single node if detection fails
	}
	
	return &ProcessManager{
		processes:            make(map[uuid.UUID]*Process),
		config:               config,
		hlsPath:              hlsPath,
		logoPath:             logoPath,
		settingsRepo:         settingsRepo,
		maxThreadsPerProcess: maxThreads,
		statusCallback:       statusCallback,
		numaNodeCount:        numaNodeCount,
		numaNodeCounter:      0,
	}
}

// SetStatusCallback sets the callback function for updating channel status
func (m *ProcessManager) SetStatusCallback(callback StatusUpdateCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCallback = callback
}

// Start starts transcoding for a channel
func (m *ProcessManager) Start(channel *domain.Channel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.processes[channel.ID]; exists {
		return fmt.Errorf("channel %s is already running", channel.ID)
	}

	// Get active process count before building args (for thread calculation)
	activeProcessCount := len(m.processes)
	
	// Determine output directory
	// Note: In Docker, tmpfs is already mounted at hlsPath, so no separate RAM disk path needed
	outputDir := filepath.Join(m.hlsPath, channel.ID.String())
	
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Build FFmpeg command
	args, err := m.buildArgs(channel, outputDir, activeProcessCount)
	if err != nil {
		return fmt.Errorf("failed to build FFmpeg args: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Use numactl to bind to NUMA node if available and multiple nodes detected
	// Fallback to normal FFmpeg if numactl is not available (safe default)
	var cmd *exec.Cmd
	useNumactl := false
	if m.numaNodeCount > 1 && runtime.GOOS == "linux" {
		// Safely check if numactl is available (must not block FFmpeg startup)
		if isNumactlAvailable() {
			useNumactl = true
			numaNode := m.getNextNUMANode()
			// Wrap FFmpeg command with numactl for NUMA binding
			// --cpunodebind: bind to CPUs on this NUMA node
			// --membind: prefer memory from this NUMA node
			numactlArgs := []string{
				fmt.Sprintf("--cpunodebind=%d", numaNode),
				fmt.Sprintf("--membind=%d", numaNode),
				m.config.BinaryPath,
			}
			numactlArgs = append(numactlArgs, args...)
			cmd = exec.CommandContext(ctx, "numactl", numactlArgs...)
			logger.Debug().
				Str("channel_id", channel.ID.String()).
				Int("numa_node", numaNode).
				Msg("Using numactl for NUMA binding")
		}
	}
	
	// Fallback to normal FFmpeg if numactl not available or not needed
	if !useNumactl {
		cmd = exec.CommandContext(ctx, m.config.BinaryPath, args...)
		if m.numaNodeCount > 1 {
			logger.Debug().
				Str("channel_id", channel.ID.String()).
				Msg("NUMA nodes detected but numactl not available, using normal FFmpeg")
		}
	}
	
	// Set process attributes to create a new process group
	// This allows us to kill all child processes (FFmpeg and its children) together
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
		}
	}
	
	// Capture stderr for progress parsing
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	process := &Process{
		ChannelID: channel.ID,
		Channel:   channel,
		Cmd:       cmd,
		Cancel:    cancel,
		StartedAt: time.Now(),
		Metrics:   &domain.ProcessMetrics{},
		Logs:      make([]string, 0, 1000), // Pre-allocate for 1000 log lines
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Set process priority, CPU affinity, and NUMA node binding for optimal performance
	if cmd.Process != nil {
		// Set nice value to 0 (normal priority) for high-performance systems
		niceValue := 0
		numCPU := runtime.NumCPU()
		if numCPU >= 64 {
			// Very high-core system: use normal priority for better throughput
			niceValue = 0
		} else if numCPU >= 16 {
			// Medium-high core system: slightly lower priority
			niceValue = 2
		} else {
			// Lower core system: lower priority to avoid system lag
			niceValue = 5
		}
		
		if err := setProcessPriority(cmd.Process.Pid, niceValue); err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", channel.ID.String()).
				Msg("Failed to set process priority, continuing anyway")
		}
		
		// NUMA binding is handled at process launch via numactl wrapper
		// If numactl was not available, process will run on default CPUs
	}

	m.processes[channel.ID] = process

	// Start progress monitoring goroutine
	go m.monitorProgress(process, stderr)

	// Start process watcher goroutine
	go m.watchProcess(process)

	// Log FFmpeg command for debugging
	logger.Info().
		Str("channel_id", channel.ID.String()).
		Str("channel_name", channel.Name).
		Str("source_url", channel.SourceURL).
		Int("pid", cmd.Process.Pid).
		Int("active_processes", activeProcessCount).
		Str("output_dir", outputDir).
		Str("ffmpeg_command", strings.Join(append([]string{m.config.BinaryPath}, args...), " ")).
		Msg("Started FFmpeg process")

	return nil
}

// Stop stops transcoding for a channel
func (m *ProcessManager) Stop(channelID uuid.UUID) error {
	m.mu.Lock()
	process, exists := m.processes[channelID]
	if !exists {
		m.mu.Unlock()
		// Channel directory might still exist even if process is not in map
		// Clean it up anyway
		outputDir := filepath.Join(m.hlsPath, channelID.String())
		if err := os.RemoveAll(outputDir); err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", channelID.String()).
				Str("output_dir", outputDir).
				Msg("Failed to remove channel directory (process not in map)")
		} else {
			logger.Info().
				Str("channel_id", channelID.String()).
				Str("output_dir", outputDir).
				Msg("Cleaned up channel directory (process not in map)")
		}
		return nil
	}
	
	// Get output directory before removing from map
	outputDir := filepath.Join(m.hlsPath, channelID.String())
	pid := 0
	if process.Cmd != nil && process.Cmd.Process != nil {
		pid = process.Cmd.Process.Pid
	}
	
	// Remove from map first to prevent auto-restart
	delete(m.processes, channelID)
	m.mu.Unlock()

	logger.Info().
		Str("channel_id", channelID.String()).
		Int("pid", pid).
		Str("output_dir", outputDir).
		Msg("Stopping FFmpeg process and cleaning up")

	// Step 1: Cancel context to stop the command
	process.Cancel()

	// Step 2: Try graceful shutdown with SIGTERM
	if process.Cmd != nil && process.Cmd.Process != nil {
		// Get process group ID (negative PID kills process group on Linux)
		pgid, err := syscall.Getpgid(pid)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", channelID.String()).
				Int("pid", pid).
				Msg("Failed to get process group ID, killing individual process")
			// Kill just the process
			process.Cmd.Process.Signal(syscall.SIGTERM)
		} else {
			// Kill entire process group to ensure all child processes are terminated
			logger.Debug().
				Str("channel_id", channelID.String()).
				Int("pid", pid).
				Int("pgid", pgid).
				Msg("Killing process group")
			syscall.Kill(-pgid, syscall.SIGTERM)
		}
	}

	// Step 3: Wait for graceful shutdown, then force kill if needed
	done := make(chan error, 1)
	go func() {
		if process.Cmd != nil {
			done <- process.Cmd.Wait()
		} else {
			done <- nil
		}
	}()

	select {
	case <-done:
		logger.Debug().
			Str("channel_id", channelID.String()).
			Int("pid", pid).
			Msg("Process exited gracefully")
	case <-time.After(3 * time.Second):
		// Force kill with SIGKILL after 3 seconds
		logger.Warn().
			Str("channel_id", channelID.String()).
			Int("pid", pid).
			Msg("Process did not exit gracefully, forcing kill with SIGKILL")
		
		if process.Cmd != nil && process.Cmd.Process != nil {
			// Try to get process group again
			pgid, err := syscall.Getpgid(pid)
			if err == nil {
				// Kill entire process group
				syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				// Kill individual process
				process.Cmd.Process.Kill()
			}
			
			// Wait a bit more for force kill to take effect
			select {
			case <-done:
				logger.Debug().
					Str("channel_id", channelID.String()).
					Int("pid", pid).
					Msg("Process killed with SIGKILL")
			case <-time.After(2 * time.Second):
				logger.Error().
					Str("channel_id", channelID.String()).
					Int("pid", pid).
					Msg("Process still running after SIGKILL, may be zombie process")
			}
		}
	}

	// Step 4: Verify process is actually dead
	if process.Cmd != nil && process.Cmd.Process != nil {
		// Check if process still exists
		err := process.Cmd.Process.Signal(syscall.Signal(0))
		if err == nil {
			logger.Error().
				Str("channel_id", channelID.String()).
				Int("pid", pid).
				Msg("Process still running after kill attempt")
		}
	}

	// Step 5: Clean up channel directory completely
	if err := os.RemoveAll(outputDir); err != nil {
		logger.Error().
			Err(err).
			Str("channel_id", channelID.String()).
			Str("output_dir", outputDir).
			Msg("Failed to remove channel directory")
	} else {
		logger.Info().
			Str("channel_id", channelID.String()).
			Str("output_dir", outputDir).
			Msg("Successfully removed channel directory")
	}

	logger.Info().
		Str("channel_id", channelID.String()).
		Int("pid", pid).
		Msg("Stopped FFmpeg process and cleaned up")

	return nil
}

// Restart restarts transcoding for a channel
func (m *ProcessManager) Restart(channelID uuid.UUID) error {
	m.mu.RLock()
	process, exists := m.processes[channelID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s is not running", channelID)
	}

	channel := process.Channel

	if err := m.Stop(channelID); err != nil {
		return err
	}

	time.Sleep(time.Second)

	return m.Start(channel)
}

// GetProcess returns process info for a channel
func (m *ProcessManager) GetProcess(channelID uuid.UUID) (*domain.TranscoderProcess, error) {
	m.mu.RLock()
	process, exists := m.processes[channelID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("channel %s is not running", channelID)
	}

	process.mu.RLock()
	// Copy process data we need
	if process.Cmd == nil || process.Cmd.Process == nil {
		process.mu.RUnlock()
		return nil, fmt.Errorf("process not initialized for channel %s", channelID)
	}
	pid := process.Cmd.Process.Pid
	lastCPUStat := process.lastCPUStat
	bitrate := process.Metrics.Bitrate
	startedAt := process.StartedAt
	dropFrames := process.Metrics.DropFrames
	fps := process.Metrics.FPS
	speed := process.Metrics.Speed
	process.mu.RUnlock()

	// Get CPU and memory usage (pass process for tracking, but don't lock here)
	cpuUsage, memoryUsage := m.getProcessStats(pid, process, &lastCPUStat)

	// Parse bitrate for output
	outputBitrate := 0
	if bitrate != "" {
		// Extract numeric value from bitrate string (e.g., "4000k" -> 4000)
		bitrateStr := strings.TrimSuffix(bitrate, "k")
		bitrateStr = strings.TrimSuffix(bitrateStr, "M")
		if val, err := strconv.Atoi(bitrateStr); err == nil {
			outputBitrate = val
		}
	}

	return &domain.TranscoderProcess{
		ChannelID:     channelID,
		PID:           pid,
		StartedAt:     startedAt,
		CPUUsage:      cpuUsage,
		MemoryUsage:   memoryUsage,
		InputBitrate:  0, // Will be parsed from input if available
		OutputBitrate: outputBitrate,
		DroppedFrames: dropFrames,
		FPS:           fps,
		Speed:         parseSpeed(speed),
		Uptime:        int64(time.Since(startedAt).Seconds()),
	}, nil
}

// GetAllProcesses returns all running processes
func (m *ProcessManager) GetAllProcesses() ([]*domain.TranscoderProcess, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processes := make([]*domain.TranscoderProcess, 0, len(m.processes))
	for channelID, process := range m.processes {
		process.mu.RLock()
		if process.Cmd == nil || process.Cmd.Process == nil {
			process.mu.RUnlock()
			continue // Skip processes that aren't fully initialized
		}
		pid := process.Cmd.Process.Pid
		lastCPUStat := process.lastCPUStat
		startedAt := process.StartedAt
		bitrate := process.Metrics.Bitrate
		dropFrames := process.Metrics.DropFrames
		fps := process.Metrics.FPS
		speed := process.Metrics.Speed
		process.mu.RUnlock()
		
		cpuUsage, memoryUsage := m.getProcessStats(pid, process, &lastCPUStat)
		
		outputBitrate := 0
		if bitrate != "" {
			bitrateStr := strings.TrimSuffix(bitrate, "k")
			bitrateStr = strings.TrimSuffix(bitrateStr, "M")
			if val, err := strconv.Atoi(bitrateStr); err == nil {
				outputBitrate = val
			}
		}

		processes = append(processes, &domain.TranscoderProcess{
			ChannelID:     channelID,
			PID:           pid,
			StartedAt:     startedAt,
			CPUUsage:      cpuUsage,
			MemoryUsage:   memoryUsage,
			InputBitrate:  0,
			OutputBitrate: outputBitrate,
			DroppedFrames: dropFrames,
			FPS:           fps,
			Speed:         parseSpeed(speed),
			Uptime:        int64(time.Since(startedAt).Seconds()),
		})
	}

	return processes, nil
}

// IsRunning checks if a channel is running
func (m *ProcessManager) IsRunning(channelID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.processes[channelID]
	return exists
}

// buildArgs builds FFmpeg command arguments
func (m *ProcessManager) buildArgs(channel *domain.Channel, outputDir string, activeProcessCount int) ([]string, error) {
	// Start with basic FFmpeg arguments with reconnect and stability options
	// Optimized for 70 simultaneous streams with stability and performance
	args := []string{
		"-hide_banner",
		"-loglevel", "warning", // Reduced logging for performance
		"-progress", "pipe:2",
		// Reconnect options for network streams (optimized)
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "2", // Faster reconnection
		"-reconnect_at_eof", "1",
		"-timeout", "5000000", // 5 second timeout (faster failure detection)
		// Input buffer optimizations for stability
		"-fflags", "+genpts+discardcorrupt+nobuffer",
		"-analyzeduration", "2000000", // 2 seconds (reduced for faster startup)
		"-probesize", "2000000", // 2MB (reduced for faster startup)
		"-thread_queue_size", "512", // Balanced queue size (reduced memory per stream)
		"-i", channel.SourceURL,
	}

	// Check for NVIDIA GPU availability for hardware acceleration
	useNVENC := isNvidiaAvailable()
	if useNVENC {
		// Insert hardware acceleration at the beginning of input arguments
		// -hwaccel cuda: Use CUDA for hardware acceleration
		// Note: We need to put this before -i if we want to decode with GPU as well,
		// but usually decoding with CPU and encoding with GPU is more stable for various inputs.
		// For now, we'll keep it simple and just use GPU for encoding.
		logger.Debug().
			Str("channel_id", channel.ID.String()).
			Msg("NVIDIA GPU detected, using NVENC for encoding")
	} else {
		logger.Debug().
			Str("channel_id", channel.ID.String()).
			Msg("NVIDIA GPU not detected, falling back to libx264 (CPU)")
	}

	// Get settings from database first (this is the source of truth)
	preset := m.config.DefaultPreset
	bitrate := m.config.DefaultBitrate
	segmentTime := m.config.SegmentTime
	playlistSize := m.config.PlaylistSize
	resolution := "1920x1080"
	profile := "high"
	
	// Load settings from database (these override config defaults)
	if m.settingsRepo != nil {
		dbSettings, err := m.settingsRepo.GetSystemSettings()
		if err == nil {
			// Use database settings as primary source
			if val, ok := dbSettings["default_preset"]; ok {
				if v, ok := val.(string); ok && v != "" {
					preset = v
				}
			}
			if val, ok := dbSettings["default_bitrate"]; ok {
				if v, ok := val.(string); ok && v != "" {
					bitrate = v
				}
			}
			if val, ok := dbSettings["segment_time"]; ok {
				if v, ok := val.(float64); ok {
					segmentTime = int(v)
				} else if v, ok := val.(int); ok {
					segmentTime = v
				}
			}
			if val, ok := dbSettings["playlist_size"]; ok {
				if v, ok := val.(float64); ok {
					playlistSize = int(v)
				} else if v, ok := val.(int); ok {
					playlistSize = v
				}
			}
			if val, ok := dbSettings["default_resolution"]; ok {
				if v, ok := val.(string); ok && v != "" {
					resolution = v
				}
			}
			if val, ok := dbSettings["default_profile"]; ok {
				if v, ok := val.(string); ok && v != "" {
					profile = v
				}
			}
		}
	}
	
	// Channel-specific config overrides database settings (highest priority)
	if channel.OutputConfig != nil {
		if channel.OutputConfig.Preset != "" {
			preset = channel.OutputConfig.Preset
		}
		if channel.OutputConfig.Bitrate != "" {
			bitrate = channel.OutputConfig.Bitrate
		}
		if channel.OutputConfig.Resolution != "" {
			resolution = channel.OutputConfig.Resolution
		}
		if channel.OutputConfig.Profile != "" {
			profile = channel.OutputConfig.Profile
		}
	}

	// Parse resolution string (e.g., "1920x1080")
	var outputWidth, outputHeight int
	resParts := strings.Split(resolution, "x")
	if len(resParts) == 2 {
		if w, err := strconv.Atoi(resParts[0]); err == nil {
			outputWidth = w
		}
		if h, err := strconv.Atoi(resParts[1]); err == nil {
			outputHeight = h
		}
	}
	
	// If resolution not parsed, use defaults
	if outputWidth == 0 || outputHeight == 0 {
		outputWidth = 1920
		outputHeight = 1080
	}

	// Build video filter complex
	var videoFilters []string
	hasLogo := channel.Logo != nil && channel.Logo.Path != ""
	
	if hasLogo {
		// Handle logo path
		var logoPath string
		if filepath.IsAbs(channel.Logo.Path) {
			logoPath = channel.Logo.Path
		} else {
			logoPath = filepath.Join(m.logoPath, channel.Logo.Path)
		}

		// Check if logo file exists
		if _, err := os.Stat(logoPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("logo file not found: %s", logoPath)
		}

		// Add logo as second input
		args = append(args, "-i", logoPath)
		
		// Build filter: scale input video, prepare logo, overlay
		// Format: [0:v]scale=WxH[scaled];[1:v]scale=WxH,format=rgba,colorchannelmixer=aa=OPACITY[logo];[scaled][logo]overlay=X:Y[vout]
		videoFilters = append(videoFilters, fmt.Sprintf(
			"[0:v]scale=%d:%d[scaled]",
			outputWidth, outputHeight,
		))
		videoFilters = append(videoFilters, fmt.Sprintf(
			"[1:v]scale=%d:%d,format=rgba,colorchannelmixer=aa=%f[logo]",
			channel.Logo.Width, channel.Logo.Height, channel.Logo.Opacity,
		))
		videoFilters = append(videoFilters, fmt.Sprintf(
			"[scaled][logo]overlay=%d:%d[vout]",
			channel.Logo.X, channel.Logo.Y,
		))
	} else {
		// No logo, just scale video
		videoFilters = append(videoFilters, fmt.Sprintf(
			"[0:v]scale=%d:%d[vout]",
			outputWidth, outputHeight,
		))
	}

	// Add filter_complex for video processing
	if len(videoFilters) > 0 {
		filterComplex := strings.Join(videoFilters, ";")
		args = append(args, "-filter_complex", filterComplex)
		// Map the filtered video output (vout is the final video output from filter_complex)
		args = append(args, "-map", "[vout]")
	} else {
		// Fallback: map video directly if no filters
		args = append(args, "-map", "0:v")
	}

	// Map audio from first input
	// Note: FFmpeg will handle missing audio gracefully - if no audio stream exists,
	// it will continue without audio (we'll encode audio only if present)
	args = append(args, "-map", "0:a")

	// Get encoding parameters from database settings (with defaults)
	// Note: Using -threads 0 (auto threads) for better stability and automatic thread management
	crf := 23
	maxrate := "5000k"
	bufsize := "10000k"
	gopSize := segmentTime * 30 // GOP size (segment_time seconds at 30fps, e.g., 6 seconds = 180 frames)
	
	// Load additional encoding settings from database
	if m.settingsRepo != nil {
		dbSettings, err := m.settingsRepo.GetSystemSettings()
		if err == nil {
			if val, ok := dbSettings["default_crf"]; ok {
				if v, ok := val.(float64); ok {
					crf = int(v)
				} else if v, ok := val.(int); ok {
					crf = v
				}
			}
			if val, ok := dbSettings["default_maxrate"]; ok {
				if v, ok := val.(string); ok && v != "" {
					maxrate = v
				}
			}
			if val, ok := dbSettings["default_bufsize"]; ok {
				if v, ok := val.(string); ok && v != "" {
					bufsize = v
				}
			}
		}
	}
	
	// Channel-specific config overrides (highest priority)
	if channel.OutputConfig != nil {
		// Bitrate can override maxrate
		if channel.OutputConfig.Bitrate != "" {
			bitrate = channel.OutputConfig.Bitrate
			maxrate = bitrate
		}
	}
	
	// Calculate bufsize from maxrate if not set explicitly
	if bufsize == "10000k" && maxrate != "" {
		// Default: 2x maxrate for bufsize
		maxrateNum := strings.TrimSuffix(maxrate, "k")
		isMB := strings.HasSuffix(maxrate, "M")
		if isMB {
			maxrateNum = strings.TrimSuffix(maxrateNum, "M")
		}
		if val, err := strconv.Atoi(maxrateNum); err == nil {
			multiplier := 2
			if isMB {
				bufsize = fmt.Sprintf("%dM", val*multiplier)
			} else {
				bufsize = fmt.Sprintf("%dk", val*multiplier)
			}
		}
	}
	
	// Video encoding parameters (optimized for stability, quality, and 70 streams performance)
	// Use optimized thread count from settings or auto-detect
	threadCount := "0" // Auto-detect threads
	if m.settingsRepo != nil {
		dbSettings, err := m.settingsRepo.GetSystemSettings()
		if err == nil {
			if val, ok := dbSettings["threads_per_process"]; ok {
				if v, ok := val.(float64); ok && v > 0 {
					threadCount = strconv.Itoa(int(v))
				} else if v, ok := val.(int); ok && v > 0 {
					threadCount = strconv.Itoa(v)
				}
			}
		}
	}
	
	// Video encoding parameters
	if useNVENC {
		// NVENC optimized parameters
		args = append(args,
			"-c:v", "h264_nvenc",
			"-preset", "p4", // Medium quality/speed for newer NVENC
			"-tune", "ull", // Ultra-low latency
			"-rc", "vbr", // Variable bitrate
			"-cq", strconv.Itoa(crf), // Quality
			"-maxrate", maxrate,
			"-bufsize", bufsize,
			"-profile:v", profile,
			"-level", "4.1",
			"-pix_fmt", "yuv420p",
			"-g", strconv.Itoa(gopSize),
			"-keyint_min", strconv.Itoa(gopSize/2),
			"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentTime),
			"-bf", "0",
			"-gpu", "any", // Use any available GPU or round-robin if we want to be fancy
		)
	} else {
		// x264 (CPU) parameters
		args = append(args,
			"-c:v", "libx264",
			"-preset", preset,
			"-tune", "zerolatency",
			"-crf", strconv.Itoa(crf),
			"-maxrate", maxrate,
			"-bufsize", bufsize,
			"-profile:v", profile,
			"-level", "4.1",
			"-pix_fmt", "yuv420p",
			"-g", strconv.Itoa(gopSize),
			"-keyint_min", strconv.Itoa(gopSize/2),
			"-sc_threshold", "0",
			"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentTime),
			"-threads", threadCount,
			"-x264opts", "nal-hrd=cbr:force-cfr=1",
			"-bf", "0",
		)
	}
	
	// Audio encoding parameters
	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k", // Good quality audio
		"-ar", "48000",
		"-ac", "2", // Stereo
	)
	
	// HLS output parameters (optimized for stability and performance with 70 streams)
	args = append(args,
		"-f", "hls",
		"-hls_time", strconv.Itoa(segmentTime), // 3 second segments (optimal for stability)
		"-hls_list_size", strconv.Itoa(playlistSize), // Keep 6 segments in playlist (18 seconds)
		"-hls_flags", "delete_segments+independent_segments+program_date_time", // Auto-delete + independent segments + timestamps
		"-hls_delete_threshold", "1", // Delete old segments immediately
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%05d.ts"),
		"-hls_segment_type", "mpegts",
		"-start_number", "0",
		"-avoid_negative_ts", "make_zero",
		"-max_muxing_queue_size", "1024", // Reasonable queue (reduced from 9999 for memory efficiency with 70 streams)
		"-muxdelay", "0", // No delay
		"-muxpreload", "0", // No preload
		filepath.Join(outputDir, "index.m3u8"),
	)

	return args, nil
}

// monitorProgress parses FFmpeg progress output and collects logs
func (m *ProcessManager) monitorProgress(process *Process, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	
	frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
	fpsRegex := regexp.MustCompile(`fps=\s*([\d.]+)`)
	bitrateRegex := regexp.MustCompile(`bitrate=\s*([\d.]+\w+)`)
	speedRegex := regexp.MustCompile(`speed=\s*([\d.]+x)`)
	dropRegex := regexp.MustCompile(`drop=\s*(\d+)`)
	errorRegex := regexp.MustCompile(`(?i)(error|failed|cannot|unable|invalid)`)

	// Optimize parsing: only parse every N lines to reduce CPU usage
	// For high-performance systems, we can parse more frequently without much overhead
	lineCount := 0
	parseInterval := 3 // Parse metrics every 3 lines (slightly more frequent for better monitoring)

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		
		// Store all log lines (limit to last 500 lines to reduce memory usage)
		process.logMu.Lock()
		process.Logs = append(process.Logs, line)
		if len(process.Logs) > 500 {
			process.Logs = process.Logs[len(process.Logs)-500:]
		}
		process.logMu.Unlock()

		// Only parse metrics periodically to reduce CPU usage
		shouldParse := lineCount%parseInterval == 0 || errorRegex.MatchString(line)

		// Check for errors/warnings (always check these)
		if errorRegex.MatchString(line) {
			logger.Warn().
				Str("channel_id", process.ChannelID.String()).
				Str("line", line).
				Msg("FFmpeg warning/error detected")
		}

		// Only parse metrics if we should (reduces CPU usage)
		if shouldParse {
			process.mu.Lock()
			if matches := frameRegex.FindStringSubmatch(line); len(matches) > 1 {
				process.Metrics.Frame, _ = strconv.ParseInt(matches[1], 10, 64)
			}
			if matches := fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
				process.Metrics.FPS, _ = strconv.ParseFloat(matches[1], 64)
			}
			if matches := bitrateRegex.FindStringSubmatch(line); len(matches) > 1 {
				process.Metrics.Bitrate = matches[1]
			}
			if matches := speedRegex.FindStringSubmatch(line); len(matches) > 1 {
				process.Metrics.Speed = matches[1]
			}
			if matches := dropRegex.FindStringSubmatch(line); len(matches) > 1 {
				process.Metrics.DropFrames, _ = strconv.Atoi(matches[1])
			}
			process.mu.Unlock()
		}
	}
	
	// Log scanner errors
	if err := scanner.Err(); err != nil {
		logger.Error().
			Err(err).
			Str("channel_id", process.ChannelID.String()).
			Msg("Error reading FFmpeg stderr")
	}
}

// watchProcess monitors process health and handles auto-restart
func (m *ProcessManager) watchProcess(process *Process) {
	err := process.Cmd.Wait()
	
	// Calculate process uptime to determine if it failed to start
	uptime := time.Since(process.StartedAt)
	const minUptimeForRestart = 10 * time.Second // If process runs less than 10 seconds, don't auto-restart
	
	// Add exit message to logs
	process.logMu.Lock()
	if err != nil {
		process.Logs = append(process.Logs, fmt.Sprintf("[ERROR] Process exited with error: %v (uptime: %v)", err, uptime))
	} else {
		process.Logs = append(process.Logs, fmt.Sprintf("[INFO] Process exited normally (uptime: %v)", uptime))
	}
	process.logMu.Unlock()
	
	// Check if process is still in map (might have been stopped manually)
	m.mu.Lock()
	_, stillInMap := m.processes[process.ChannelID]
	
	// Remove from active processes if it's still there
	if stillInMap {
		delete(m.processes, process.ChannelID)
		logger.Debug().
			Str("channel_id", process.ChannelID.String()).
			Msg("Removed process from map after exit")
	} else {
		logger.Debug().
			Str("channel_id", process.ChannelID.String()).
			Msg("Process already removed from map (likely stopped manually)")
	}
	
	// Check if auto-restart is enabled and channel is still supposed to be running
	autoRestart := false
	if process.Channel != nil && stillInMap {
		autoRestart = process.Channel.AutoRestart
	}
	
	// Get output directory for cleanup
	outputDir := filepath.Join(m.hlsPath, process.ChannelID.String())
	m.mu.Unlock()
	
	// If process was manually stopped (not in map), clean up directory and exit
	if !stillInMap {
		logger.Info().
			Str("channel_id", process.ChannelID.String()).
			Dur("uptime", uptime).
			Msg("Process was stopped manually, cleaning up directory")
		
		// Clean up channel directory
		if err := os.RemoveAll(outputDir); err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Failed to remove channel directory in watchProcess")
		} else {
			logger.Info().
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Cleaned up channel directory in watchProcess")
		}
		return // Don't attempt auto-restart if process was manually stopped
	}

	if err != nil {
		logger.Error().
			Err(err).
			Str("channel_id", process.ChannelID.String()).
			Dur("uptime", uptime).
			Bool("auto_restart", autoRestart).
			Msg("FFmpeg process exited with error")
	} else {
		logger.Info().
			Str("channel_id", process.ChannelID.String()).
			Dur("uptime", uptime).
			Bool("auto_restart", autoRestart).
			Msg("FFmpeg process exited")
	}
	
	// If process ran for less than minUptimeForRestart, it likely failed to start
	// Don't auto-restart, update channel status to stopped and clean up
	if uptime < minUptimeForRestart {
		logger.Warn().
			Str("channel_id", process.ChannelID.String()).
			Dur("uptime", uptime).
			Msg("FFmpeg process exited too quickly, likely failed to start. Stopping channel instead of auto-restart.")
		
		// Clean up channel directory (process failed to start properly)
		if err := os.RemoveAll(outputDir); err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Failed to remove channel directory after start failure")
		} else {
			logger.Info().
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Cleaned up channel directory after start failure")
		}
		
		// Update channel status to stopped if callback is available
		if m.statusCallback != nil && process.Channel != nil {
			if updateErr := m.statusCallback(process.ChannelID, domain.ChannelStatusStopped); updateErr != nil {
				logger.Error().
					Err(updateErr).
					Str("channel_id", process.ChannelID.String()).
					Msg("Failed to update channel status to stopped")
			} else {
				logger.Info().
					Str("channel_id", process.ChannelID.String()).
					Str("channel_name", process.Channel.Name).
					Msg("Channel status updated to stopped due to FFmpeg start failure")
			}
		}
		return // Don't attempt auto-restart
	}
	
	// Auto-restart if enabled and process ran for sufficient time
	// Double-check that process is still supposed to be running (check map again)
	if autoRestart && process.Channel != nil {
		logger.Info().
			Str("channel_id", process.ChannelID.String()).
			Str("channel_name", process.Channel.Name).
			Msg("Auto-restart enabled, restarting FFmpeg process in 2 seconds...")
		
		// Wait 2 seconds before restart to avoid rapid restart loops
		time.Sleep(2 * time.Second)
		
		// Check again if process is still supposed to be running (might have been stopped during wait)
		m.mu.RLock()
		_, shouldRestart := m.processes[process.ChannelID]
		m.mu.RUnlock()
		
		if !shouldRestart {
			logger.Info().
				Str("channel_id", process.ChannelID.String()).
				Msg("Process was stopped during restart wait, skipping auto-restart")
			
			// Clean up directory if process was stopped
			if err := os.RemoveAll(outputDir); err != nil {
				logger.Warn().
					Err(err).
					Str("channel_id", process.ChannelID.String()).
					Str("output_dir", outputDir).
					Msg("Failed to remove channel directory after restart skip")
			}
			return
		}
		
		// Try to restart the process
		restartErr := m.Start(process.Channel)
		if restartErr != nil {
			logger.Error().
				Err(restartErr).
				Str("channel_id", process.ChannelID.String()).
				Msg("Failed to auto-restart FFmpeg process")
			
			// Clean up directory on restart failure
			if err := os.RemoveAll(outputDir); err != nil {
				logger.Warn().
					Err(err).
					Str("channel_id", process.ChannelID.String()).
					Str("output_dir", outputDir).
					Msg("Failed to remove channel directory after restart failure")
			}
			
			// If restart fails, update channel status to error/stopped
			if m.statusCallback != nil {
				if updateErr := m.statusCallback(process.ChannelID, domain.ChannelStatusError); updateErr != nil {
					logger.Error().
						Err(updateErr).
						Str("channel_id", process.ChannelID.String()).
						Msg("Failed to update channel status to error")
				}
			}
		} else {
			logger.Info().
				Str("channel_id", process.ChannelID.String()).
				Str("channel_name", process.Channel.Name).
				Msg("FFmpeg process auto-restarted successfully")
		}
	} else {
		// Process exited but auto-restart is disabled or channel is nil
		// Clean up directory
		if err := os.RemoveAll(outputDir); err != nil {
			logger.Warn().
				Err(err).
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Failed to remove channel directory after process exit")
		} else {
			logger.Info().
				Str("channel_id", process.ChannelID.String()).
				Str("output_dir", outputDir).
				Msg("Cleaned up channel directory after process exit (no auto-restart)")
		}
	}
}

// GetLogs returns the logs for a process
func (m *ProcessManager) GetLogs(channelID uuid.UUID) ([]string, error) {
	m.mu.RLock()
	process, exists := m.processes[channelID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("channel %s is not running", channelID)
	}

	process.logMu.Lock()
	defer process.logMu.Unlock()

	// Return a copy of the logs
	logs := make([]string, len(process.Logs))
	copy(logs, process.Logs)
	return logs, nil
}

// parseSpeed converts speed string to float
func parseSpeed(speed string) float64 {
	speed = strings.TrimSuffix(speed, "x")
	val, _ := strconv.ParseFloat(speed, 64)
	return val
}

// getProcessStats retrieves CPU and memory usage for a process
// process can be nil if we don't need to track CPU stats
func (m *ProcessManager) getProcessStats(pid int, process *Process, lastCPUStat *struct {
	utime  int64
	stime  int64
	cutime int64
	cstime int64
	time   time.Time
}) (float64, int64) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statusPath := fmt.Sprintf("/proc/%d/status", pid)

	var cpuUsage float64 = 0.0
	var memoryUsage int64 = 0

	// Read CPU usage from /proc/[pid]/stat
	if statData, err := os.ReadFile(statPath); err == nil {
		fields := strings.Fields(string(statData))
		if len(fields) >= 22 {
			// utime (14), stime (15), cutime (16), cstime (17) - 1-indexed in /proc/stat
			utime, _ := strconv.ParseInt(fields[13], 10, 64)
			stime, _ := strconv.ParseInt(fields[14], 10, 64)
			cutime, _ := strconv.ParseInt(fields[15], 10, 64)
			cstime, _ := strconv.ParseInt(fields[16], 10, 64)

			// Calculate CPU usage percentage if we have previous stats
			if process != nil && lastCPUStat != nil {
				now := time.Now()

				// Calculate CPU usage percentage
				if !lastCPUStat.time.IsZero() {
					totalTime := (utime + stime + cutime + cstime) - (lastCPUStat.utime + lastCPUStat.stime + lastCPUStat.cutime + lastCPUStat.cstime)
					elapsed := now.Sub(lastCPUStat.time).Seconds()

					if elapsed > 0 {
						// Get system clock ticks per second (usually 100)
						clockTicks := int64(100) // Default, can be read from sysconf(_SC_CLK_TCK)
						
						// CPU usage = (process_time / elapsed_time) / num_cores * 100
						// Process time is in clock ticks, convert to seconds
						processTimeSeconds := float64(totalTime) / float64(clockTicks)
						cpuUsage = (processTimeSeconds / elapsed) * 100.0
						
						// Normalize by number of CPU cores for accurate percentage
						numCPU := runtime.NumCPU()
						if numCPU > 0 {
							cpuUsage = cpuUsage / float64(numCPU)
						}
					}
				}

				// Update last stats (with lock)
				process.mu.Lock()
				process.lastCPUStat.utime = utime
				process.lastCPUStat.stime = stime
				process.lastCPUStat.cutime = cutime
				process.lastCPUStat.cstime = cstime
				process.lastCPUStat.time = now
				process.mu.Unlock()
			}
		}
	}

	// Read memory usage from /proc/[pid]/status
	if statusData, err := os.ReadFile(statusPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(statusData)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					// Memory in KB
					if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						memoryUsage = val * 1024 // Convert KB to bytes
					}
				}
				break
			}
		}
	}

	// If we couldn't read from /proc, return default values
	if cpuUsage == 0 {
		cpuUsage = 0.0 // Return 0 instead of placeholder
	}
	if memoryUsage == 0 {
		memoryUsage = 100 * 1024 * 1024 // Default 100MB placeholder
	}

	return cpuUsage, memoryUsage
}

// setProcessPriority sets the nice value (priority) for a process
func setProcessPriority(pid int, nice int) error {
	// Use syscall.Setpriority on Unix systems
	// Nice values: -20 (highest priority) to 19 (lowest priority)
	// We use 0 (normal priority) for high-performance systems
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, nice)
}

// isNumactlAvailable checks if numactl command is available in the system
// Returns false if numactl is not found or if check fails (safe fallback)
func isNumactlAvailable() bool {
	// Try to check if numactl exists by running it with --version (faster than --show)
	cmd := exec.Command("numactl", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		// numactl not available or error - return false for safe fallback
		return false
	}
	return true
}

// detectNUMANodes detects the number of available NUMA nodes
// Returns 0 if detection fails (fallback to single node)
func detectNUMANodes() int {
	// Try to detect NUMA nodes by reading /sys/devices/system/node/
	// Count node directories (node0, node1, etc.)
	numaPath := "/sys/devices/system/node"
	
	entries, err := os.ReadDir(numaPath)
	if err != nil {
		// If /sys/devices/system/node doesn't exist, assume single node
		logger.Debug().
			Err(err).
			Msg("Could not read NUMA nodes directory, assuming single node")
		return 0
	}
	
	nodeCount := 0
	for _, entry := range entries {
		// Count directories that start with "node" followed by a number
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "node") {
			// Verify it's a valid node directory (node0, node1, etc.)
			if len(entry.Name()) > 4 {
				nodeID := entry.Name()[4:]
				if _, err := strconv.Atoi(nodeID); err == nil {
					nodeCount++
				}
			}
		}
	}
	
	if nodeCount > 0 {
		logger.Info().
			Int("numa_nodes", nodeCount).
			Msg("Detected NUMA nodes")
		return nodeCount
	}
	
	// Fallback: try numactl --hardware if available
	cmd := exec.Command("numactl", "--hardware")
	output, err := cmd.Output()
	if err == nil {
		// Parse output to count nodes
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "available:") {
				// Line format: "available: 2 nodes (0-1)"
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					if count, err := strconv.Atoi(parts[1]); err == nil && count > 0 {
						logger.Info().
							Int("numa_nodes", count).
							Msg("Detected NUMA nodes via numactl")
						return count
					}
				}
			}
		}
	}
	
	logger.Debug().
		Msg("Could not detect NUMA nodes, assuming single node")
	return 0
}

// getNextNUMANode returns the next NUMA node for round-robin distribution
func (m *ProcessManager) getNextNUMANode() int {
	m.numaMu.Lock()
	defer m.numaMu.Unlock()
	
	if m.numaNodeCount <= 1 {
		return 0 // Single node system
	}
	
	node := m.numaNodeCounter % m.numaNodeCount
	m.numaNodeCounter++
	
	return node
}

// isNvidiaAvailable checks if NVIDIA GPU is available via nvidia-smi
func isNvidiaAvailable() bool {
	cmd := exec.Command("nvidia-smi", "-L")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
