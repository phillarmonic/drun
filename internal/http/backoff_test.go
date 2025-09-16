package http

import (
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:  time.Second,
		Multiplier: 2.0,
		Jitter:     false, // Disable jitter for predictable testing
	}

	// Test first attempt (should be base delay)
	delay0 := backoff.NextDelay(0)
	if delay0 != time.Second {
		t.Errorf("Expected delay of 1s for attempt 0, got %v", delay0)
	}

	// Test second attempt (should be 2s)
	delay1 := backoff.NextDelay(1)
	if delay1 != 2*time.Second {
		t.Errorf("Expected delay of 2s for attempt 1, got %v", delay1)
	}

	// Test third attempt (should be 4s)
	delay2 := backoff.NextDelay(2)
	if delay2 != 4*time.Second {
		t.Errorf("Expected delay of 4s for attempt 2, got %v", delay2)
	}

	// Ensure delays are increasing
	if delay1 <= delay0 {
		t.Error("Expected delays to increase exponentially")
	}
	if delay2 <= delay1 {
		t.Error("Expected delays to increase exponentially")
	}
}

func TestExponentialBackoffWithMaxDelay(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:  time.Second,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     false,
	}

	// Test that delay doesn't exceed max
	delay := backoff.NextDelay(10) // Should be way over max
	if delay > 5*time.Second {
		t.Errorf("Expected delay to be capped at 5s, got %v", delay)
	}
}

func TestExponentialBackoffWithoutJitter(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:  time.Second,
		Multiplier: 2.0,
		Jitter:     false,
	}

	// Without jitter, delays should be exact
	delay0 := backoff.NextDelay(0)
	if delay0 != time.Second {
		t.Errorf("Expected exact 1s delay for attempt 0, got %v", delay0)
	}

	delay1 := backoff.NextDelay(1)
	if delay1 != 2*time.Second {
		t.Errorf("Expected exact 2s delay for attempt 1, got %v", delay1)
	}
}

func TestLinearBackoff(t *testing.T) {
	backoff := &LinearBackoff{
		BaseDelay: time.Second,
		Increment: time.Second,
		Jitter:    false, // Disable jitter for predictable testing
	}

	// Test linear progression
	delay0 := backoff.NextDelay(0)
	delay1 := backoff.NextDelay(1)
	delay2 := backoff.NextDelay(2)

	// Should increase linearly
	if delay1-delay0 != delay2-delay1 {
		t.Errorf("Expected linear increase in delays. Got: %v, %v, %v", delay0, delay1, delay2)
	}

	// Check specific values
	if delay0 != time.Second {
		t.Errorf("Expected delay0 to be 1s, got %v", delay0)
	}
	if delay1 != 2*time.Second {
		t.Errorf("Expected delay1 to be 2s, got %v", delay1)
	}
	if delay2 != 3*time.Second {
		t.Errorf("Expected delay2 to be 3s, got %v", delay2)
	}
}

func TestLinearBackoffWithMaxDelay(t *testing.T) {
	backoff := &LinearBackoff{
		BaseDelay: time.Second,
		MaxDelay:  5 * time.Second,
		Increment: time.Second,
		Jitter:    false,
	}

	// Test that delay doesn't exceed max
	delay := backoff.NextDelay(10) // Should be way over max
	if delay > 5*time.Second {
		t.Errorf("Expected delay to be capped at 5s, got %v", delay)
	}
}

func TestLinearBackoffWithoutJitter(t *testing.T) {
	backoff := &LinearBackoff{
		BaseDelay: time.Second,
		Increment: 500 * time.Millisecond,
		Jitter:    false,
	}

	// Without jitter, delays should be exact
	delay0 := backoff.NextDelay(0)
	if delay0 != time.Second {
		t.Errorf("Expected exact 1s delay for attempt 0, got %v", delay0)
	}

	delay1 := backoff.NextDelay(1)
	expected := time.Second + 500*time.Millisecond
	if delay1 != expected {
		t.Errorf("Expected exact %v delay for attempt 1, got %v", expected, delay1)
	}
}

func TestFixedBackoff(t *testing.T) {
	backoff := NewFixedBackoff(2 * time.Second)

	// All delays should be the same
	delay0 := backoff.NextDelay(0)
	delay1 := backoff.NextDelay(1)
	delay2 := backoff.NextDelay(10)

	if delay0 != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay0)
	}

	if delay1 != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay1)
	}

	if delay2 != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay2)
	}
}

func TestFixedBackoffWithJitter(t *testing.T) {
	backoff := &FixedBackoff{
		Delay:  2 * time.Second,
		Jitter: true,
	}

	// With jitter, delays should vary but be around 2s
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = backoff.NextDelay(0)
	}

	// Check that not all delays are exactly the same (jitter working)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Expected jitter to create variation in delays")
	}

	// Check that all delays are reasonable (within jitter range)
	for i, delay := range delays {
		if delay < time.Second || delay > 3*time.Second {
			t.Errorf("Delay %d out of expected range: %v", i, delay)
		}
	}
}

func TestCustomBackoff(t *testing.T) {
	// Custom backoff that returns attempt number as seconds
	backoff := NewCustomBackoff(func(attempt int) time.Duration {
		return time.Duration(attempt+1) * time.Second
	})

	delay0 := backoff.NextDelay(0)
	if delay0 != time.Second {
		t.Errorf("Expected 1s delay for attempt 0, got %v", delay0)
	}

	delay1 := backoff.NextDelay(1)
	if delay1 != 2*time.Second {
		t.Errorf("Expected 2s delay for attempt 1, got %v", delay1)
	}

	delay5 := backoff.NextDelay(5)
	if delay5 != 6*time.Second {
		t.Errorf("Expected 6s delay for attempt 5, got %v", delay5)
	}
}

func TestBackoffStrategiesArePositive(t *testing.T) {
	strategies := []BackoffStrategy{
		NewExponentialBackoff(time.Second),
		NewLinearBackoff(time.Second),
		NewFixedBackoff(time.Second),
		NewCustomBackoff(func(attempt int) time.Duration {
			return time.Duration(attempt+1) * time.Second
		}),
	}

	for i, strategy := range strategies {
		for attempt := 0; attempt < 5; attempt++ {
			delay := strategy.NextDelay(attempt)
			if delay <= 0 {
				t.Errorf("Strategy %d returned non-positive delay %v for attempt %d", i, delay, attempt)
			}
		}
	}
}
