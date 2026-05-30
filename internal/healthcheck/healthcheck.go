package healthcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/domain/orchestration"
)

// Checker performs health checks on services
type Checker struct {
	httpClient *http.Client
	dnsClient  *net.Resolver
}

// NewChecker creates a new health check checker
func NewChecker() *Checker {
	return &Checker{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		dnsClient: &net.Resolver{
			PreferGo: true,
		},
	}
}

// Check performs a health check based on the configuration
func (c *Checker) Check(ctx context.Context, config *orchestration.HealthCheck) error {
	switch config.Type {
	case "http":
		return c.checkHTTP(ctx, config)
	case "tcp":
		return c.checkTCP(ctx, config)
	case "docker":
		return c.checkDocker(ctx, config)
	case "dns":
		return c.checkDNS(ctx, config)
	case "custom":
		return c.checkCustom(ctx, config)
	default:
		return fmt.Errorf("unknown health check type: %s", config.Type)
	}
}

// checkHTTP performs an HTTP health check
func (c *Checker) checkHTTP(ctx context.Context, config *orchestration.HealthCheck) error {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", config.Endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http health check failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	if config.Condition != "" {
		expectedStatus := config.Condition
		actualStatus := fmt.Sprintf("%d", resp.StatusCode)

		if actualStatus != expectedStatus {
			return fmt.Errorf("http health check failed: expected status %s, got %s", expectedStatus, actualStatus)
		}
	} else {
		// Default: check if status is 2xx
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("http health check failed: status code %d", resp.StatusCode)
		}
	}

	return nil
}

// checkTCP performs a TCP health check
func (c *Checker) checkTCP(ctx context.Context, config *orchestration.HealthCheck) error {
	dialer := &net.Dialer{
		Timeout: config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", config.Endpoint)
	if err != nil {
		return fmt.Errorf("tcp health check failed: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	return nil
}

// checkDocker performs a Docker container health check
func (c *Checker) checkDocker(ctx context.Context, config *orchestration.HealthCheck) error {
	// Use docker inspect to check container health
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", config.Container)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker health check failed: %w", err)
	}

	status := strings.TrimSpace(string(output))
	if status != "healthy" && status != "starting" {
		return fmt.Errorf("docker health check failed: container status is %s", status)
	}

	return nil
}

// checkDNS performs a DNS resolution health check
func (c *Checker) checkDNS(ctx context.Context, config *orchestration.HealthCheck) error {
	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Resolve domain
	var ips []string

	switch config.RecordType {
	case "A", "":
		addrs, resolveErr := c.dnsClient.LookupHost(ctx, config.Domain)
		if resolveErr != nil {
			return fmt.Errorf("DNS resolution failed: %w", resolveErr)
		}
		ips = addrs
	case "AAAA":
		addrs, resolveErr := c.dnsClient.LookupHost(ctx, config.Domain)
		if resolveErr != nil {
			return fmt.Errorf("DNS resolution failed: %w", resolveErr)
		}
		// Filter for IPv6 addresses
		for _, addr := range addrs {
			if strings.Contains(addr, ":") {
				ips = append(ips, addr)
			}
		}
	case "CNAME":
		cname, resolveErr := c.dnsClient.LookupCNAME(ctx, config.Domain)
		if resolveErr != nil {
			return fmt.Errorf("DNS CNAME resolution failed: %w", resolveErr)
		}
		ips = []string{cname}
	case "MX":
		mxRecords, resolveErr := c.dnsClient.LookupMX(ctx, config.Domain)
		if resolveErr != nil {
			return fmt.Errorf("DNS MX resolution failed: %w", resolveErr)
		}
		for _, mx := range mxRecords {
			ips = append(ips, mx.Host)
		}
	default:
		return fmt.Errorf("unsupported DNS record type: %s", config.RecordType)
	}

	// Check if we got any results
	if len(ips) == 0 {
		return fmt.Errorf("DNS resolution returned no results")
	}

	// Validate expected IP if specified
	if config.ExpectedIP != "" {
		found := false
		for _, ip := range ips {
			if ip == config.ExpectedIP {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("DNS resolution did not return expected IP %s (got: %v)", config.ExpectedIP, ips)
		}
	}

	// Validate expected IPs if specified
	if len(config.ExpectedIPs) > 0 {
		found := false
		for _, ip := range ips {
			for _, expectedIP := range config.ExpectedIPs {
				if ip == expectedIP {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("DNS resolution did not return any expected IPs %v (got: %v)", config.ExpectedIPs, ips)
		}
	}

	return nil
}

// checkCustom performs a custom command-based health check
func (c *Checker) checkCustom(ctx context.Context, config *orchestration.HealthCheck) error {
	// Parse command
	parts := strings.Fields(config.Command)
	if len(parts) == 0 {
		return fmt.Errorf("custom health check command is empty")
	}

	// Create command
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set working directory if specified
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	// Run command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("custom health check failed: %w", err)
	}

	return nil
}

// CheckWithRetries performs a health check with retries
func (c *Checker) CheckWithRetries(ctx context.Context, config *orchestration.HealthCheck) error {
	var lastErr error

	for i := 0; i <= config.Retries; i++ {
		// Wait for start period on first check
		if i == 0 && config.StartPeriod > 0 {
			select {
			case <-time.After(config.StartPeriod):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Perform check
		err := c.Check(ctx, config)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't wait after last retry
		if i < config.Retries {
			select {
			case <-time.After(config.Interval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", config.Retries, lastErr)
}

// Monitor continuously monitors a service's health
func (c *Checker) Monitor(ctx context.Context, service *orchestration.Service, callback func(bool, error)) {
	ticker := time.NewTicker(service.HealthCheck.Interval)
	defer ticker.Stop()

	// Perform initial check after start period
	if service.HealthCheck.StartPeriod > 0 {
		select {
		case <-time.After(service.HealthCheck.StartPeriod):
		case <-ctx.Done():
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.CheckWithRetries(ctx, service.HealthCheck)
			callback(err == nil, err)
		}
	}
}
