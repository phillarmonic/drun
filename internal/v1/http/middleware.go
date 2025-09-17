package http

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Common middleware implementations

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(req *Request) error {
		start := time.Now()

		// Log request
		if logger != nil {
			logger.Printf("HTTP %s %s", req.method, req.url)
		}

		// Store start time for response logging (would need interceptor for full logging)
		req.Header("X-Request-Start", start.Format(time.RFC3339Nano))

		return nil
	}
}

// UserAgentMiddleware sets a custom User-Agent header
func UserAgentMiddleware(userAgent string) Middleware {
	return func(req *Request) error {
		req.Header("User-Agent", userAgent)
		return nil
	}
}

// TimeoutMiddleware sets a timeout for the request
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(req *Request) error {
		req.Timeout(timeout)
		return nil
	}
}

// RateLimitMiddleware implements basic rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() Middleware {
	return func(req *Request) error {
		key := req.url // Use URL as key, could be more sophisticated
		now := time.Now()

		// Clean old requests
		if times, exists := rl.requests[key]; exists {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					validTimes = append(validTimes, t)
				}
			}
			rl.requests[key] = validTimes
		}

		// Check if limit exceeded
		if len(rl.requests[key]) >= rl.limit {
			return fmt.Errorf("rate limit exceeded: %d requests per %v", rl.limit, rl.window)
		}

		// Add current request
		rl.requests[key] = append(rl.requests[key], now)

		return nil
	}
}

// CompressionMiddleware adds compression headers
func CompressionMiddleware() Middleware {
	return func(req *Request) error {
		req.Header("Accept-Encoding", "gzip, deflate")
		return nil
	}
}

// CacheControlMiddleware sets cache control headers
func CacheControlMiddleware(cacheControl string) Middleware {
	return func(req *Request) error {
		req.Header("Cache-Control", cacheControl)
		return nil
	}
}

// ConditionalMiddleware applies middleware based on a condition
func ConditionalMiddleware(condition func(*Request) bool, middleware Middleware) Middleware {
	return func(req *Request) error {
		if condition(req) {
			return middleware(req)
		}
		return nil
	}
}

// ChainMiddleware chains multiple middleware together
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(req *Request) error {
		for _, middleware := range middlewares {
			if err := middleware(req); err != nil {
				return err
			}
		}
		return nil
	}
}

// Common interceptors for response processing

// LoggingInterceptor logs HTTP responses
func LoggingInterceptor(logger *log.Logger) Interceptor {
	return func(resp *Response) error {
		if logger != nil {
			duration := resp.Duration()
			logger.Printf("HTTP %d %s (%v)", resp.StatusCode, resp.Request.URL.String(), duration)
		}
		return nil
	}
}

// StatusCodeInterceptor returns an error for specific status codes
func StatusCodeInterceptor(errorCodes ...int) Interceptor {
	return func(resp *Response) error {
		for _, code := range errorCodes {
			if resp.StatusCode == code {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			}
		}
		return nil
	}
}

// ClientErrorInterceptor returns an error for 4xx status codes
func ClientErrorInterceptor() Interceptor {
	return func(resp *Response) error {
		if resp.IsClientError() {
			return fmt.Errorf("client error: HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return nil
	}
}

// ServerErrorInterceptor returns an error for 5xx status codes
func ServerErrorInterceptor() Interceptor {
	return func(resp *Response) error {
		if resp.IsServerError() {
			return fmt.Errorf("server error: HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return nil
	}
}

// JSONErrorInterceptor parses JSON error responses
func JSONErrorInterceptor() Interceptor {
	return func(resp *Response) error {
		if !resp.IsSuccess() && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			var errorResp struct {
				Error   string `json:"error"`
				Message string `json:"message"`
				Code    int    `json:"code"`
			}

			if err := resp.JSON(&errorResp); err == nil {
				if errorResp.Error != "" {
					return fmt.Errorf("API error: %s", errorResp.Error)
				}
				if errorResp.Message != "" {
					return fmt.Errorf("API error: %s", errorResp.Message)
				}
			}
		}
		return nil
	}
}

// RetryInterceptor marks responses for retry based on conditions
func RetryInterceptor(shouldRetry func(*Response) bool) Interceptor {
	return func(resp *Response) error {
		// This is more of a marker - actual retry logic is in the client
		if shouldRetry(resp) {
			return fmt.Errorf("retry condition met")
		}
		return nil
	}
}

// MetricsInterceptor collects metrics about HTTP requests
type MetricsCollector struct {
	RequestCount  int64
	TotalDuration time.Duration
	StatusCounts  map[int]int64
	ErrorCount    int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		StatusCounts: make(map[int]int64),
	}
}

// Interceptor returns the metrics collection interceptor
func (mc *MetricsCollector) Interceptor() Interceptor {
	return func(resp *Response) error {
		mc.RequestCount++
		mc.TotalDuration += resp.Duration()
		mc.StatusCounts[resp.StatusCode]++

		if !resp.IsSuccess() {
			mc.ErrorCount++
		}

		return nil
	}
}

// AverageResponseTime returns the average response time
func (mc *MetricsCollector) AverageResponseTime() time.Duration {
	if mc.RequestCount == 0 {
		return 0
	}
	return mc.TotalDuration / time.Duration(mc.RequestCount)
}

// ErrorRate returns the error rate as a percentage
func (mc *MetricsCollector) ErrorRate() float64 {
	if mc.RequestCount == 0 {
		return 0
	}
	return float64(mc.ErrorCount) / float64(mc.RequestCount) * 100
}
