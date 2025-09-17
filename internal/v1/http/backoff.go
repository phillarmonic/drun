package http

import (
	"math"
	"math/rand"
	"time"
)

// ExponentialBackoff implements exponential backoff strategy
type ExponentialBackoff struct {
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

// NextDelay calculates the next delay for exponential backoff
func (e *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	multiplier := e.Multiplier
	if multiplier == 0 {
		multiplier = 2.0
	}

	delay := float64(e.BaseDelay) * math.Pow(multiplier, float64(attempt))

	if e.MaxDelay > 0 && time.Duration(delay) > e.MaxDelay {
		delay = float64(e.MaxDelay)
	}

	if e.Jitter {
		// Add random jitter (±25%)
		jitter := delay * 0.25 * (rand.Float64()*2 - 1)
		delay += jitter
	}

	if delay < 0 {
		delay = float64(e.BaseDelay)
	}

	return time.Duration(delay)
}

// LinearBackoff implements linear backoff strategy
type LinearBackoff struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Increment time.Duration
	Jitter    bool
}

// NextDelay calculates the next delay for linear backoff
func (l *LinearBackoff) NextDelay(attempt int) time.Duration {
	increment := l.Increment
	if increment == 0 {
		increment = l.BaseDelay
	}

	delay := l.BaseDelay + time.Duration(attempt)*increment

	if l.MaxDelay > 0 && delay > l.MaxDelay {
		delay = l.MaxDelay
	}

	if l.Jitter {
		// Add random jitter (±25%)
		jitter := time.Duration(float64(delay) * 0.25 * (rand.Float64()*2 - 1))
		delay += jitter
	}

	if delay < 0 {
		delay = l.BaseDelay
	}

	return delay
}

// FixedBackoff implements fixed delay backoff strategy
type FixedBackoff struct {
	Delay  time.Duration
	Jitter bool
}

// NextDelay returns a fixed delay
func (f *FixedBackoff) NextDelay(attempt int) time.Duration {
	delay := f.Delay

	if f.Jitter {
		// Add random jitter (±25%)
		jitter := time.Duration(float64(delay) * 0.25 * (rand.Float64()*2 - 1))
		delay += jitter
	}

	if delay < 0 {
		delay = f.Delay
	}

	return delay
}

// CustomBackoff allows custom backoff logic
type CustomBackoff struct {
	DelayFunc func(attempt int) time.Duration
}

// NextDelay uses the custom delay function
func (c *CustomBackoff) NextDelay(attempt int) time.Duration {
	return c.DelayFunc(attempt)
}

// Helper functions for creating backoff strategies

// NewExponentialBackoff creates a new exponential backoff strategy
func NewExponentialBackoff(baseDelay time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{
		BaseDelay:  baseDelay,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// NewLinearBackoff creates a new linear backoff strategy
func NewLinearBackoff(baseDelay time.Duration) *LinearBackoff {
	return &LinearBackoff{
		BaseDelay: baseDelay,
		MaxDelay:  30 * time.Second,
		Increment: baseDelay,
		Jitter:    true,
	}
}

// NewFixedBackoff creates a new fixed backoff strategy
func NewFixedBackoff(delay time.Duration) *FixedBackoff {
	return &FixedBackoff{
		Delay:  delay,
		Jitter: false,
	}
}

// NewCustomBackoff creates a new custom backoff strategy
func NewCustomBackoff(delayFunc func(int) time.Duration) *CustomBackoff {
	return &CustomBackoff{DelayFunc: delayFunc}
}
