package rate_limiter

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Uruchom goroutine do czyszczenia starych IP
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for ip, times := range rl.requests {
			var validTimes []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) IsAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Usuń stare requesty
	if times, exists := rl.requests[ip]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(windowStart) {
				validTimes = append(validTimes, t)
			}
		}
		rl.requests[ip] = validTimes
	}

	// Sprawdź czy limit został przekroczony
	if len(rl.requests[ip]) >= rl.limit {
		return false
	}

	// Dodaj nowy request
	rl.requests[ip] = append(rl.requests[ip], now)
	return true
}

// GetRemainingRequests zwraca liczbę pozostałych requestów dla danego IP
func (rl *RateLimiter) GetRemainingRequests(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	if times, exists := rl.requests[ip]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(windowStart) {
				validTimes = append(validTimes, t)
			}
		}
		return rl.limit - len(validTimes)
	}
	return rl.limit
}
