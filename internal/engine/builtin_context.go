package engine

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/builtins"
	"github.com/phillarmonic/drun/v2/internal/platform"
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

// GetTaskNames returns all user-defined tasks available to the current
// execution. Local tasks retain declaration order; included task names are
// appended in lexical order because their backing store is a map.
func (bc *BuiltinContext) GetTaskNames() []string {
	if bc.execCtx == nil {
		return nil
	}

	names := make([]string, 0)
	seen := make(map[string]struct{})
	add := func(name string) {
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	if bc.execCtx.Program != nil {
		for _, task := range bc.execCtx.Program.Tasks {
			if taskMatchesCurrentPlatform(task.Name, task.Annotations) {
				add(task.Name)
			}
		}
	}

	if bc.execCtx.Project != nil {
		includedNames := make([]string, 0, len(bc.execCtx.Project.IncludedTasks))
		for name, variants := range bc.execCtx.Project.IncludedTasks {
			for _, variant := range variants {
				if taskMatchesCurrentPlatform(name, variant.Annotations) {
					includedNames = append(includedNames, name)
					break
				}
			}
		}
		sort.Strings(includedNames)
		for _, name := range includedNames {
			add(name)
		}
	}

	return names
}

func taskMatchesCurrentPlatform(name string, annotations []ast.Annotation) bool {
	metadata, err := platform.ValidateAnnotations("task", name, annotations)
	return err == nil && platform.MatchesCurrent(metadata.Platforms)
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
