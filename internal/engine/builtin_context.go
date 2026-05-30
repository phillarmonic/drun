package engine

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/builtins"
)

// BuiltinContext implements builtins.Context interface for the engine
type BuiltinContext struct {
	execCtx        *ExecutionContext
	secretsManager SecretsManager
	dryRun         bool
}

// GetProjectName returns the current project name
func (bc *BuiltinContext) GetProjectName() string {
	if bc.execCtx != nil && bc.execCtx.Project != nil {
		return bc.execCtx.Project.Name
	}
	return ""
}

// GetSecretsManager returns the secrets manager
func (bc *BuiltinContext) GetSecretsManager() builtins.SecretsManager {
	// Return nil if no secrets manager
	if bc.secretsManager == nil {
		return nil
	}
	// The SecretsManager interface matches, so we can return it directly
	return bc.secretsManager
}

// IsDryRun returns whether we're in dry-run mode
func (bc *BuiltinContext) IsDryRun() bool {
	return bc.dryRun
}

// DNSResolve resolves a domain for DNS-oriented builtins.
func (bc *BuiltinContext) DNSResolve(domain string) (string, error) {
	if bc.dryRun {
		return "127.0.0.1", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("dns lookup failed for %q: %w", domain, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("dns lookup returned no results for %q", domain)
	}

	return addrs[0], nil
}

// DNSCheck validates that a DNS record resolves.
func (bc *BuiltinContext) DNSCheck(domain, recordType string) (bool, error) {
	if bc.dryRun {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch strings.ToUpper(strings.TrimSpace(recordType)) {
	case "", "A":
		addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
		return len(addrs) > 0, err
	case "AAAA":
		addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
		if err != nil {
			return false, err
		}
		for _, addr := range addrs {
			if strings.Contains(addr, ":") {
				return true, nil
			}
		}
		return false, nil
	case "CNAME":
		_, err := net.DefaultResolver.LookupCNAME(ctx, domain)
		return err == nil, err
	case "MX":
		records, err := net.DefaultResolver.LookupMX(ctx, domain)
		return len(records) > 0, err
	default:
		return false, fmt.Errorf("unsupported DNS record type: %s", recordType)
	}
}

// DNSValidate checks whether a domain resolves to an expected IP.
func (bc *BuiltinContext) DNSValidate(domain, expectedIP string) (bool, error) {
	if bc.dryRun {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupHost(ctx, domain)
	if err != nil {
		return false, err
	}
	for _, addr := range addrs {
		if addr == expectedIP {
			return true, nil
		}
	}
	return false, nil
}
