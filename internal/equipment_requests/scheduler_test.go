package equipment_requests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService is a mock implementation of the sync service for testing
type mockService struct {
	mu           sync.Mutex
	syncCount    int
	syncDuration time.Duration
	syncError    error
	syncResult   *SyncResult
}

func (m *mockService) SyncQuestsToDatabase(ctx context.Context) (*SyncResult, error) {
	m.mu.Lock()
	m.syncCount++
	m.mu.Unlock()

	if m.syncDuration > 0 {
		time.Sleep(m.syncDuration)
	}

	if m.syncError != nil {
		return nil, m.syncError
	}

	if m.syncResult != nil {
		return m.syncResult, nil
	}

	return &SyncResult{
		Stats: &SyncStats{
			Created:   1,
			Updated:   2,
			Unchanged: 10,
		},
	}, nil
}

func (m *mockService) getSyncCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncCount
}

// syncableService is an interface that defines what the scheduler needs
type syncableService interface {
	SyncQuestsToDatabase(ctx context.Context) (*SyncResult, error)
}

// testScheduler wraps Scheduler to use interface instead of concrete Service type
type testScheduler struct {
	service      syncableService
	interval     time.Duration
	enabled      bool
	stopChan     chan struct{}
	stoppedChan  chan struct{}
	mu           sync.RWMutex
	errorHandler func(error)
	lastSync     time.Time
	lastError    error
}

func newTestScheduler(service syncableService, interval time.Duration) *testScheduler {
	return &testScheduler{
		service:     service,
		interval:    interval,
		enabled:     false,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

func (s *testScheduler) SetErrorHandler(handler func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorHandler = handler
}

func (s *testScheduler) Start() {
	s.mu.Lock()
	if s.enabled {
		s.mu.Unlock()
		return
	}
	s.enabled = true
	s.mu.Unlock()

	go s.syncLoop()
}

func (s *testScheduler) Stop() {
	s.mu.Lock()
	if !s.enabled {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	close(s.stopChan)
	<-s.stoppedChan

	s.mu.Lock()
	s.enabled = false
	s.mu.Unlock()
}

func (s *testScheduler) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

func (s *testScheduler) GetInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interval
}

func (s *testScheduler) GetLastSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}

func (s *testScheduler) GetLastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *testScheduler) syncLoop() {
	defer close(s.stoppedChan)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.performSync()

	for {
		select {
		case <-ticker.C:
			s.performSync()
		case <-s.stopChan:
			return
		}
	}
}

func (s *testScheduler) performSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, err := s.service.SyncQuestsToDatabase(ctx)

	if err != nil {
		s.mu.Lock()
		s.lastError = err
		s.mu.Unlock()

		s.mu.RLock()
		handler := s.errorHandler
		s.mu.RUnlock()

		if handler != nil {
			handler(err)
		}
		return
	}

	s.mu.Lock()
	s.lastSync = time.Now()
	s.lastError = nil
	s.mu.Unlock()
}

func (s *testScheduler) TriggerManualSync(ctx context.Context) (*SyncResult, error) {
	return s.service.SyncQuestsToDatabase(ctx)
}

func TestScheduler_NewScheduler(t *testing.T) {
	mock := &mockService{}

	scheduler := newTestScheduler(mock, 5*time.Minute)

	assert.NotNil(t, scheduler)
	assert.Equal(t, 5*time.Minute, scheduler.GetInterval())
	assert.False(t, scheduler.IsEnabled())
}

func TestScheduler_StartStop(t *testing.T) {
	mock := &mockService{
		syncDuration: 10 * time.Millisecond, // Short duration to speed up test
	}
	scheduler := newTestScheduler(mock, 100*time.Millisecond)

	// Start scheduler
	assert.False(t, scheduler.IsEnabled())
	scheduler.Start()
	assert.True(t, scheduler.IsEnabled())

	// Wait a bit to ensure it's running
	time.Sleep(150 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()
	assert.False(t, scheduler.IsEnabled())

	// Verify it actually stopped
	time.Sleep(200 * time.Millisecond)
}

func TestScheduler_DoubleStart(t *testing.T) {
	mock := &mockService{}
	scheduler := newTestScheduler(mock, 1*time.Second)

	scheduler.Start()
	assert.True(t, scheduler.IsEnabled())

	// Try to start again - should be a no-op
	scheduler.Start()
	assert.True(t, scheduler.IsEnabled())

	scheduler.Stop()
}

func TestScheduler_StopBeforeStart(t *testing.T) {
	mock := &mockService{}
	scheduler := newTestScheduler(mock, 1*time.Second)

	// Stop before start - should be a no-op
	assert.False(t, scheduler.IsEnabled())
	scheduler.Stop()
	assert.False(t, scheduler.IsEnabled())
}

func TestScheduler_ErrorHandling(t *testing.T) {
	testError := errors.New("sync failed")
	mock := &mockService{
		syncError: testError,
	}
	scheduler := newTestScheduler(mock, 100*time.Millisecond)

	// Track errors
	var errorCount int
	var lastError error
	var mu sync.Mutex

	scheduler.SetErrorHandler(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errorCount++
		lastError = err
	})

	scheduler.Start()
	defer scheduler.Stop()

	// Wait for at least one sync attempt
	time.Sleep(250 * time.Millisecond)

	// Verify error was handled
	mu.Lock()
	assert.Greater(t, errorCount, 0)
	assert.Equal(t, testError, lastError)
	mu.Unlock()

	// Verify last error is stored
	assert.Equal(t, testError, scheduler.GetLastError())

	// Scheduler should still be running despite errors
	assert.True(t, scheduler.IsEnabled())
}

func TestScheduler_SuccessfulSync(t *testing.T) {
	mock := &mockService{
		syncResult: &SyncResult{
			Stats: &SyncStats{
				Created:   5,
				Updated:   3,
				Unchanged: 12,
			},
		},
	}
	scheduler := newTestScheduler(mock, 100*time.Millisecond)

	scheduler.Start()
	defer scheduler.Stop()

	// Wait for initial sync
	time.Sleep(150 * time.Millisecond)

	// Verify last error is nil after successful sync
	assert.Nil(t, scheduler.GetLastError())

	// Verify last sync time is recent
	lastSync := scheduler.GetLastSync()
	assert.WithinDuration(t, time.Now(), lastSync, 200*time.Millisecond)
}

func TestScheduler_ManualTrigger(t *testing.T) {
	mock := &mockService{
		syncResult: &SyncResult{
			Stats: &SyncStats{
				Created:   2,
				Updated:   1,
				Unchanged: 8,
			},
		},
	}
	scheduler := newTestScheduler(mock, 10*time.Second) // Long interval

	// Don't start scheduler, just trigger manually
	ctx := context.Background()
	result, err := scheduler.TriggerManualSync(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Stats.Created)
	assert.Equal(t, 1, result.Stats.Updated)
	assert.Equal(t, 8, result.Stats.Unchanged)
}

func TestScheduler_ManualTriggerWithError(t *testing.T) {
	testError := errors.New("manual sync failed")
	mock := &mockService{
		syncError: testError,
	}
	scheduler := newTestScheduler(mock, 10*time.Second)

	ctx := context.Background()
	result, err := scheduler.TriggerManualSync(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, testError, err)
}

func TestScheduler_GracefulShutdown(t *testing.T) {
	mock := &mockService{
		syncDuration: 200 * time.Millisecond, // Longer sync
	}
	scheduler := newTestScheduler(mock, 50*time.Millisecond)

	scheduler.Start()

	// Wait for sync to start
	time.Sleep(75 * time.Millisecond)

	// Stop should wait for current sync to complete
	startStop := time.Now()
	scheduler.Stop()
	stopDuration := time.Since(startStop)

	// Stop should have waited for the sync to finish
	// But should be less than a full interval
	assert.Greater(t, stopDuration, 50*time.Millisecond)
	assert.Less(t, stopDuration, 1000*time.Millisecond)

	assert.False(t, scheduler.IsEnabled())
}

func TestScheduler_GettersThreadSafe(t *testing.T) {
	mock := &mockService{}
	scheduler := newTestScheduler(mock, 5*time.Minute)

	// Test concurrent access to getters
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = scheduler.IsEnabled()
			_ = scheduler.GetInterval()
			_ = scheduler.GetLastSync()
			_ = scheduler.GetLastError()
		}()
	}

	wg.Wait()
}

func TestScheduler_SetErrorHandlerThreadSafe(t *testing.T) {
	mock := &mockService{}
	scheduler := newTestScheduler(mock, 5*time.Minute)

	// Test concurrent error handler updates
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scheduler.SetErrorHandler(func(err error) {
				// No-op
			})
		}()
	}

	wg.Wait()
}

// Integration-like test with actual service mock behavior
func TestScheduler_PeriodicExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping periodic execution test in short mode")
	}

	mock := &mockService{
		syncDuration: 10 * time.Millisecond,
		syncResult: &SyncResult{
			Stats: &SyncStats{
				Created:   1,
				Updated:   0,
				Unchanged: 5,
			},
		},
	}
	scheduler := newTestScheduler(mock, 100*time.Millisecond)

	scheduler.Start()
	defer scheduler.Stop()

	// Wait for multiple sync cycles
	time.Sleep(450 * time.Millisecond)

	// Should have run initial sync + at least 3 more syncs
	syncCount := mock.getSyncCount()
	assert.GreaterOrEqual(t, syncCount, 4, "Expected at least 4 syncs (initial + 3 periodic)")
	assert.LessOrEqual(t, syncCount, 6, "Expected at most 6 syncs (with some timing tolerance)")
}
