package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrChannelRunning  = errors.New("channel is running")
	ErrInvalidChannel  = errors.New("invalid channel data")
)

// ChannelService handles channel business logic
type ChannelService struct {
	repo       domain.ChannelRepository
	transcoder domain.TranscoderManager
}

// NewChannelService creates a new channel service
func NewChannelService(repo domain.ChannelRepository, transcoder domain.TranscoderManager) *ChannelService {
	return &ChannelService{
		repo:       repo,
		transcoder: transcoder,
	}
}

// CreateChannel creates a new channel
func (s *ChannelService) CreateChannel(name, sourceURL string, logo *domain.LogoConfig, output *domain.OutputConfig) (*domain.Channel, error) {
	if name == "" || sourceURL == "" {
		return nil, ErrInvalidChannel
	}

	channel := domain.NewChannel(name, sourceURL)
	if logo != nil {
		channel.Logo = logo
	}
	if output != nil {
		channel.OutputConfig = output
	}

	if err := s.repo.Create(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(id uuid.UUID) (*domain.Channel, error) {
	channel, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrChannelNotFound
	}
	// Set output URL dynamically with CDN
	channel.OutputURL = fmt.Sprintf("https://cdn.cashbacktv.live/streams/%s/index.m3u8", channel.ID.String())
	return channel, nil
}

// ListChannels retrieves all channels
func (s *ChannelService) ListChannels() ([]*domain.Channel, error) {
	channels, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	// Set output URL dynamically for all channels with CDN
	for _, channel := range channels {
		channel.OutputURL = fmt.Sprintf("https://cdn.cashbacktv.live/streams/%s/index.m3u8", channel.ID.String())
	}
	return channels, nil
}

// UpdateChannel updates an existing channel
func (s *ChannelService) UpdateChannel(id uuid.UUID, name, sourceURL string, logo *domain.LogoConfig, output *domain.OutputConfig) (*domain.Channel, error) {
	channel, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	if s.transcoder.IsRunning(id) {
		return nil, ErrChannelRunning
	}

	if name != "" {
		channel.Name = name
	}
	if sourceURL != "" {
		channel.SourceURL = sourceURL
	}
	// Handle logo: if logo is explicitly set (including nil), update it
	// This allows removing logo by sending null
	channel.Logo = logo
	if output != nil {
		channel.OutputConfig = output
	}
	channel.UpdatedAt = time.Now()

	if err := s.repo.Update(channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// DeleteChannel deletes a channel
func (s *ChannelService) DeleteChannel(id uuid.UUID) error {
	// Check if channel exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return ErrChannelNotFound
	}

	// Stop the channel if running (best effort - don't fail delete if stop fails)
	if s.transcoder.IsRunning(id) {
		// Try to stop, but don't fail delete if stop fails
		// The process will be cleaned up eventually
		s.transcoder.Stop(id)
		// Give a brief moment for cleanup
		time.Sleep(200 * time.Millisecond)
	}

	return s.repo.Delete(id)
}

// StartChannel starts transcoding for a channel
func (s *ChannelService) StartChannel(id uuid.UUID) error {
	channel, err := s.repo.GetByID(id)
	if err != nil {
		return ErrChannelNotFound
	}

	// Check if already running - if so, ensure status is correct and return success
	if s.transcoder.IsRunning(id) {
		// Ensure status is set to running (might be out of sync)
		s.repo.UpdateStatus(id, domain.ChannelStatusRunning)
		return nil
	}

	if err := s.repo.UpdateStatus(id, domain.ChannelStatusStarting); err != nil {
		return err
	}

	if err := s.transcoder.Start(channel); err != nil {
		s.repo.UpdateStatus(id, domain.ChannelStatusError)
		return err
	}

	return s.repo.UpdateStatus(id, domain.ChannelStatusRunning)
}

// StopChannel stops transcoding for a channel
func (s *ChannelService) StopChannel(id uuid.UUID) error {
	// Check if channel exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return ErrChannelNotFound
	}

	// If not running, ensure status is correct and return success
	if !s.transcoder.IsRunning(id) {
		// Ensure status is set to stopped (might be out of sync)
		s.repo.UpdateStatus(id, domain.ChannelStatusStopped)
		return nil
	}

	if err := s.repo.UpdateStatus(id, domain.ChannelStatusStopping); err != nil {
		return err
	}

	if err := s.transcoder.Stop(id); err != nil {
		// If stop fails, try to set status back to running or error
		s.repo.UpdateStatus(id, domain.ChannelStatusError)
		return err
	}

	return s.repo.UpdateStatus(id, domain.ChannelStatusStopped)
}

// RestartChannel restarts transcoding for a channel
func (s *ChannelService) RestartChannel(id uuid.UUID) error {
	// Check if channel exists
	_, err := s.repo.GetByID(id)
	if err != nil {
		return ErrChannelNotFound
	}

	// Stop the channel (ignore error if not running)
	wasRunning := s.transcoder.IsRunning(id)
	if wasRunning {
		if err := s.StopChannel(id); err != nil {
			// If stop fails, try to continue anyway (might be in inconsistent state)
			// But log the error
			s.repo.UpdateStatus(id, domain.ChannelStatusError)
		}
		// Give a brief moment for cleanup
		time.Sleep(500 * time.Millisecond)
	}

	// Start the channel
	return s.StartChannel(id)
}

// GetChannelMetrics retrieves transcoding metrics for a channel
func (s *ChannelService) GetChannelMetrics(id uuid.UUID) (*domain.TranscoderProcess, error) {
	return s.transcoder.GetProcess(id)
}

// GetAllChannelMetrics retrieves transcoding metrics for all running channels (optimized batch operation)
func (s *ChannelService) GetAllChannelMetrics() (map[uuid.UUID]*domain.TranscoderProcess, error) {
	processes, err := s.transcoder.GetAllProcesses()
	if err != nil {
		return nil, err
	}
	
	metricsMap := make(map[uuid.UUID]*domain.TranscoderProcess, len(processes))
	for _, process := range processes {
		metricsMap[process.ChannelID] = process
	}
	
	return metricsMap, nil
}

// GetChannelLogs retrieves logs for a channel
func (s *ChannelService) GetChannelLogs(id uuid.UUID) ([]string, error) {
	if !s.transcoder.IsRunning(id) {
		return nil, fmt.Errorf("channel %s is not running", id)
	}
	return s.transcoder.GetLogs(id)
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Success []uuid.UUID `json:"success"`
	Failed  []BatchError `json:"failed"`
}

// BatchError represents an error for a specific channel in batch operation
type BatchError struct {
	ChannelID uuid.UUID `json:"channel_id"`
	Error     string    `json:"error"`
}

// BatchStartChannels starts multiple channels with rate limiting
// Processes channels in batches to avoid overwhelming the system
func (s *ChannelService) BatchStartChannels(ids []uuid.UUID) (*BatchResult, error) {
	return s.batchProcess(ids, func(id uuid.UUID) error {
		return s.StartChannel(id)
	}, 5, 100*time.Millisecond) // 5 concurrent, 100ms delay between batches
}

// BatchStopChannels stops multiple channels with rate limiting
func (s *ChannelService) BatchStopChannels(ids []uuid.UUID) (*BatchResult, error) {
	return s.batchProcess(ids, func(id uuid.UUID) error {
		return s.StopChannel(id)
	}, 5, 100*time.Millisecond) // 5 concurrent, 100ms delay between batches
}

// BatchRestartChannels restarts multiple channels with rate limiting
func (s *ChannelService) BatchRestartChannels(ids []uuid.UUID) (*BatchResult, error) {
	return s.batchProcess(ids, func(id uuid.UUID) error {
		return s.RestartChannel(id)
	}, 3, 200*time.Millisecond) // 3 concurrent (restart is heavier), 200ms delay
}

// BatchDeleteChannels deletes multiple channels with rate limiting
func (s *ChannelService) BatchDeleteChannels(ids []uuid.UUID) (*BatchResult, error) {
	return s.batchProcess(ids, func(id uuid.UUID) error {
		return s.DeleteChannel(id)
	}, 5, 100*time.Millisecond) // 5 concurrent, 100ms delay
}

// batchProcess processes channels in batches with concurrency control and rate limiting
func (s *ChannelService) batchProcess(
	ids []uuid.UUID,
	processFunc func(uuid.UUID) error,
	concurrentLimit int,
	delayBetweenBatches time.Duration,
) (*BatchResult, error) {
	result := &BatchResult{
		Success: make([]uuid.UUID, 0),
		Failed:   make([]BatchError, 0),
	}

	if len(ids) == 0 {
		return result, nil
	}

	// Process in batches to avoid overwhelming the system
	type job struct {
		id  uuid.UUID
		err error
	}

	jobs := make(chan uuid.UUID, len(ids))
	results := make(chan job, len(ids))

	// Start worker goroutines
	for i := 0; i < concurrentLimit; i++ {
		go func() {
			for id := range jobs {
				err := processFunc(id)
				results <- job{id: id, err: err}
			}
		}()
	}

	// Send all jobs
	for _, id := range ids {
		jobs <- id
	}
	close(jobs)

	// Collect results
	for i := 0; i < len(ids); i++ {
		job := <-results
		if job.err != nil {
			result.Failed = append(result.Failed, BatchError{
				ChannelID: job.id,
				Error:     job.err.Error(),
			})
		} else {
			result.Success = append(result.Success, job.id)
		}

		// Add delay between batches to avoid overwhelming the system
		if i > 0 && i%concurrentLimit == 0 {
			time.Sleep(delayBetweenBatches)
		}
	}

	return result, nil
}
