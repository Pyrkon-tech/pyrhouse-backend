package equipment_requests

import (
	"context"
	"log"
	"sync"
	"time"
)

// Scheduler handles automatic synchronization of equipment requests from Google Sheets
type Scheduler struct {
	service      *Service
	interval     time.Duration
	enabled      bool
	stopChan     chan struct{}
	stoppedChan  chan struct{}
	mu           sync.RWMutex
	errorHandler func(error)
	lastSync     time.Time
	lastError    error
}

// NewScheduler creates a new scheduler instance
func NewScheduler(service *Service, interval time.Duration) *Scheduler {
	return &Scheduler{
		service:     service,
		interval:    interval,
		enabled:     false,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// SetErrorHandler sets a custom error handler for sync failures
func (s *Scheduler) SetErrorHandler(handler func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorHandler = handler
}

// Start begins the automatic sync loop
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.enabled {
		s.mu.Unlock()
		log.Println("[WARN] Equipment request scheduler is already running")
		return
	}
	s.enabled = true
	s.mu.Unlock()

	log.Printf("[INFO] Starting equipment request scheduler (interval: %v)", s.interval)
	go s.syncLoop()
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.enabled {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	log.Println("[INFO] Stopping equipment request scheduler...")
	close(s.stopChan)

	// Wait for sync loop to finish
	<-s.stoppedChan

	s.mu.Lock()
	s.enabled = false
	s.mu.Unlock()

	log.Println("[INFO] Equipment request scheduler stopped")
}

// IsEnabled returns whether the scheduler is currently running
func (s *Scheduler) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// GetInterval returns the current sync interval
func (s *Scheduler) GetInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interval
}

// GetLastSync returns the timestamp of the last successful sync
func (s *Scheduler) GetLastSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}

// GetLastError returns the last error encountered during sync
func (s *Scheduler) GetLastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// syncLoop is the main scheduler loop
func (s *Scheduler) syncLoop() {
	defer close(s.stoppedChan)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run initial sync immediately
	s.performSync()

	for {
		select {
		case <-ticker.C:
			s.performSync()

		case <-s.stopChan:
			log.Println("[INFO] Scheduler received stop signal")
			return
		}
	}
}

// performSync executes a single sync operation
func (s *Scheduler) performSync() {
	start := time.Now()

	// Create context with timeout (2 minutes should be enough for sync)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("[INFO] Auto-sync: Starting equipment request sync...")

	result, err := s.service.SyncQuestsToDatabase(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[ERROR] Auto-sync failed after %v: %v", duration, err)

		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()

		// Call error handler if set
		s.mu.RLock()
		handler := s.errorHandler
		s.mu.RUnlock()

		if handler != nil {
			handler(err)
		}
		return
	}

	// Success
	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastError = nil
	s.mu.Unlock()

	log.Printf("[INFO] Auto-sync completed in %v: %d created, %d updated, %d unchanged, %d items added, %d items removed",
		duration,
		result.Stats.Created,
		result.Stats.Updated,
		result.Stats.Unchanged,
		result.Stats.ItemsAdded,
		result.Stats.ItemsRemoved,
	)
}

// TriggerManualSync performs an immediate sync without waiting for the next scheduled run
// This does not reset the scheduler timer
func (s *Scheduler) TriggerManualSync(ctx context.Context) (*SyncResult, error) {
	log.Println("[INFO] Manual sync trigger requested")

	start := time.Now()
	result, err := s.service.SyncQuestsToDatabase(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[ERROR] Manual sync failed after %v: %v", duration, err)
		return nil, err
	}

	log.Printf("[INFO] Manual sync completed in %v: %d created, %d updated, %d unchanged",
		duration,
		result.Stats.Created,
		result.Stats.Updated,
		result.Stats.Unchanged,
	)

	return result, nil
}
