package provisioning

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/phillarmonic/drun/v2/internal/cache"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/remote"
	"gopkg.in/yaml.v3"
)

const (
	manifestVersion      = "1"
	defaultManifestName  = "provisionings.yaml"
	defaultFetchTimeout  = 30 * time.Second
	defaultCacheDuration = time.Minute
	officialManifestRef  = "github:phillarmonic/drun-provisionings/provisionings.yaml@master"
)

var ErrNoProvisioningMatch = errors.New("no provisioning entry found")

type githubFetcher interface {
	Fetch(ctx context.Context, path, ref string) ([]byte, error)
}

type httpsFetcher interface {
	Fetch(ctx context.Context, path, ref string) ([]byte, error)
}

type gitFetcher interface {
	FetchManifest(ctx context.Context, repoURL, manifestPath, ref string) ([]byte, error)
}

type Resolver struct {
	workingDir  string
	currentOS   string
	currentArch string

	cacheManager *cache.Manager
	github       githubFetcher
	https        httpsFetcher
	git          gitFetcher

	embedded []EmbeddedSource
	builtin  []string

	fetchTimeout  time.Duration
	cacheDuration time.Duration

	mu            sync.Mutex
	manifestCache map[string]*Manifest
}

type EmbeddedSource struct {
	Name    string
	Content []byte
}

type SourceSet struct {
	Project []string
	User    []string
}

type Resolution struct {
	Source               string
	Entry                Entry
	MatchedName          string
	Target               Target
	ExactVersion         string
	UsesVersionedInstall bool
}

func (r Resolution) InstallCommand() string {
	if r.UsesVersionedInstall && r.ExactVersion != "" && r.Target.InstallVersioned != "" {
		return strings.ReplaceAll(r.Target.InstallVersioned, "{version}", r.ExactVersion)
	}
	return r.Target.Install
}

type Option func(*Resolver)

func WithCacheManager(cm *cache.Manager) Option {
	return func(r *Resolver) {
		r.cacheManager = cm
	}
}

func WithEmbeddedSources(sources []EmbeddedSource) Option {
	return func(r *Resolver) {
		r.embedded = append([]EmbeddedSource(nil), sources...)
	}
}

func WithBuiltinSources(sources []string) Option {
	return func(r *Resolver) {
		r.builtin = append([]string(nil), sources...)
	}
}

func WithCurrentPlatform(osName, arch string) Option {
	return func(r *Resolver) {
		r.currentOS = normalizeTargetOS(osName)
		r.currentArch = normalizeTargetArch(arch)
	}
}

func WithGitHubFetcher(fetcher githubFetcher) Option {
	return func(r *Resolver) {
		r.github = fetcher
	}
}

func WithHTTPSFetcher(fetcher httpsFetcher) Option {
	return func(r *Resolver) {
		r.https = fetcher
	}
}

func WithGitFetcher(fetcher gitFetcher) Option {
	return func(r *Resolver) {
		r.git = fetcher
	}
}

func WithFetchTimeout(timeout time.Duration) Option {
	return func(r *Resolver) {
		r.fetchTimeout = timeout
	}
}

func WithCacheDuration(duration time.Duration) Option {
	return func(r *Resolver) {
		r.cacheDuration = duration
	}
}

func NewResolver(workingDir string, opts ...Option) *Resolver {
	resolver := &Resolver{
		workingDir:    workingDir,
		currentOS:     normalizeTargetOS(runtime.GOOS),
		currentArch:   normalizeTargetArch(runtime.GOARCH),
		github:        remote.NewGitHubFetcher(),
		https:         remote.NewHTTPSFetcher(),
		git:           gitCommandFetcher{},
		fetchTimeout:  defaultFetchTimeout,
		cacheDuration: defaultCacheDuration,
		manifestCache: make(map[string]*Manifest),
		builtin:       []string{officialManifestRef},
	}

	for _, opt := range opts {
		opt(resolver)
	}

	return resolver
}

type Manifest struct {
	Version       string  `yaml:"version"`
	Provisionings []Entry `yaml:"-"`
}

type Entry struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Aliases     []string `yaml:"aliases,omitempty"`
	Targets     []Target `yaml:"targets"`
}

type Target struct {
	OS               string `yaml:"os,omitempty"`
	Arch             string `yaml:"arch,omitempty"`
	Install          string `yaml:"install"`
	InstallVersioned string `yaml:"install_versioned,omitempty"`
}

func (m *Manifest) UnmarshalYAML(node *yaml.Node) error {
	var seq struct {
		Version       string  `yaml:"version"`
		Provisionings []Entry `yaml:"provisionings"`
	}
	if err := node.Decode(&seq); err == nil && len(seq.Provisionings) > 0 {
		m.Version = seq.Version
		m.Provisionings = seq.Provisionings
		return nil
	}

	var mp struct {
		Version       string               `yaml:"version"`
		Provisionings map[string]yaml.Node `yaml:"provisionings"`
	}
	if err := node.Decode(&mp); err != nil {
		return err
	}

	m.Version = mp.Version
	m.Provisionings = make([]Entry, 0, len(mp.Provisionings))
	for name, raw := range mp.Provisionings {
		var entry Entry
		if err := raw.Decode(&entry); err != nil {
			return fmt.Errorf("invalid provisioning %q: %w", name, err)
		}
		if strings.TrimSpace(entry.Name) == "" {
			entry.Name = name
		}
		m.Provisionings = append(m.Provisionings, entry)
	}

	return nil
}

func (r *Resolver) ResolveRequirement(ctx context.Context, req statement.ToolRequirement, sources SourceSet) (*Resolution, error) {
	exactVersion, hasExactVersion, err := deriveExactVersion(req.Constraints)
	if err != nil {
		return nil, fmt.Errorf("determine exact version for %q: %w", req.Name, err)
	}

	for _, source := range sources.Project {
		resolution, err := r.resolveFromReference(ctx, req.Name, source, exactVersion, hasExactVersion)
		if err == nil {
			return resolution, nil
		}
		if !errors.Is(err, ErrNoProvisioningMatch) {
			return nil, err
		}
	}

	for _, source := range sources.User {
		resolution, err := r.resolveFromReference(ctx, req.Name, source, exactVersion, hasExactVersion)
		if err == nil {
			return resolution, nil
		}
		if !errors.Is(err, ErrNoProvisioningMatch) {
			return nil, err
		}
	}

	for _, source := range r.builtin {
		resolution, err := r.resolveFromReference(ctx, req.Name, source, exactVersion, hasExactVersion)
		if err == nil {
			return resolution, nil
		}
		// Built-in first-party catalogs are opportunistic fallbacks. If the
		// remote source is unavailable, continue to the bundled embedded
		// defaults instead of failing the whole requirement.
		if err != nil && !errors.Is(err, ErrNoProvisioningMatch) {
			continue
		}
		if !errors.Is(err, ErrNoProvisioningMatch) {
			return nil, err
		}
	}

	for _, source := range r.embedded {
		manifest, err := parseManifest(source.Name, source.Content)
		if err != nil {
			return nil, fmt.Errorf("load embedded provisioning source %q: %w", source.Name, err)
		}

		resolution, err := r.resolveFromManifest(req.Name, exactVersion, hasExactVersion, source.Name, manifest)
		if err == nil {
			return resolution, nil
		}
		if !errors.Is(err, ErrNoProvisioningMatch) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("%w for tool %q", ErrNoProvisioningMatch, req.Name)
}

func (r *Resolver) resolveFromReference(ctx context.Context, toolName, sourceRef, exactVersion string, hasExactVersion bool) (*Resolution, error) {
	manifest, resolvedSource, err := r.loadManifest(ctx, sourceRef)
	if err != nil {
		return nil, fmt.Errorf("load provisioning source %q: %w", sourceRef, err)
	}

	return r.resolveFromManifest(toolName, exactVersion, hasExactVersion, resolvedSource, manifest)
}

func (r *Resolver) resolveFromManifest(toolName, exactVersion string, hasExactVersion bool, source string, manifest *Manifest) (*Resolution, error) {
	entry, matchedName, ok := manifest.findEntry(toolName)
	if !ok {
		return nil, ErrNoProvisioningMatch
	}

	target, err := selectTarget(entry.Targets, r.currentOS, r.currentArch)
	if err != nil {
		return nil, fmt.Errorf("source %q entry %q: %w", source, entry.Name, err)
	}

	resolution := &Resolution{
		Source:       source,
		Entry:        *entry,
		MatchedName:  matchedName,
		Target:       *target,
		ExactVersion: exactVersion,
	}
	if hasExactVersion && target.InstallVersioned != "" {
		resolution.UsesVersionedInstall = true
	}
	return resolution, nil
}

func (m *Manifest) findEntry(toolName string) (*Entry, string, bool) {
	needle := normalizeLookupName(toolName)
	for i := range m.Provisionings {
		entry := &m.Provisionings[i]
		if normalizeLookupName(entry.Name) == needle {
			return entry, entry.Name, true
		}
		for _, alias := range entry.Aliases {
			if normalizeLookupName(alias) == needle {
				return entry, alias, true
			}
		}
	}
	return nil, "", false
}

func (r *Resolver) loadManifest(ctx context.Context, sourceRef string) (*Manifest, string, error) {
	normalizedRef, err := r.normalizeSourceRef(sourceRef)
	if err != nil {
		return nil, "", err
	}

	r.mu.Lock()
	if manifest, ok := r.manifestCache[normalizedRef]; ok {
		r.mu.Unlock()
		return manifest, normalizedRef, nil
	}
	r.mu.Unlock()

	content, err := r.fetchManifestContent(ctx, normalizedRef)
	if err != nil {
		return nil, "", err
	}

	manifest, err := parseManifest(normalizedRef, content)
	if err != nil {
		return nil, "", err
	}

	r.mu.Lock()
	r.manifestCache[normalizedRef] = manifest
	r.mu.Unlock()

	return manifest, normalizedRef, nil
}

func (r *Resolver) normalizeSourceRef(sourceRef string) (string, error) {
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" {
		return "", fmt.Errorf("source is empty")
	}

	if isGitSource(sourceRef) || remote.IsRemoteURL(sourceRef) {
		return sourceRef, nil
	}

	expanded, err := expandUserPath(sourceRef)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(r.workingDir, expanded)
	}
	expanded = filepath.Clean(expanded)

	info, err := os.Stat(expanded)
	if err == nil && info.IsDir() {
		return filepath.Join(expanded, defaultManifestName), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect local provisioning source %q: %w", expanded, err)
	}
	return expanded, nil
}

func (r *Resolver) fetchManifestContent(ctx context.Context, sourceRef string) ([]byte, error) {
	if !isGitSource(sourceRef) && !remote.IsRemoteURL(sourceRef) {
		return os.ReadFile(sourceRef)
	}

	cacheKey := cache.GenerateKey(sourceRef, "")
	if r.cacheManager != nil {
		if content, hit, err := r.cacheManager.Get(cacheKey); err == nil && hit {
			return content, nil
		}
	}

	fetchCtx, cancel := context.WithTimeout(ctx, r.fetchTimeout)
	defer cancel()

	var (
		content []byte
		err     error
	)

	switch {
	case isGitSource(sourceRef):
		repoURL, manifestPath, ref, parseErr := parseGitSource(sourceRef)
		if parseErr != nil {
			return nil, parseErr
		}
		content, err = r.git.FetchManifest(fetchCtx, repoURL, manifestPath, ref)
	case remote.IsRemoteURL(sourceRef):
		protocol, path, ref, parseErr := remote.ParseRemoteURL(sourceRef)
		if parseErr != nil {
			return nil, parseErr
		}
		switch protocol {
		case "github":
			content, err = r.github.Fetch(fetchCtx, path, ref)
		case "https":
			content, err = r.https.Fetch(fetchCtx, path, ref)
		default:
			return nil, fmt.Errorf("unsupported provisioning protocol %q", protocol)
		}
	default:
		return nil, fmt.Errorf("unsupported provisioning source %q", sourceRef)
	}
	if err != nil {
		if r.cacheManager != nil {
			if stale, ok := r.cacheManager.GetStale(cacheKey); ok {
				return stale, nil
			}
		}
		return nil, err
	}

	if r.cacheManager != nil {
		if err := r.cacheManager.Set(cacheKey, content, r.cacheDuration); err != nil {
			return nil, fmt.Errorf("cache provisioning source %q: %w", sourceRef, err)
		}
	}

	return content, nil
}

func parseManifest(source string, content []byte) (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := validateManifest(&manifest); err != nil {
		return nil, fmt.Errorf("validate manifest: %w", err)
	}
	return &manifest, nil
}

func validateManifest(manifest *Manifest) error {
	if len(manifest.Provisionings) == 0 {
		return fmt.Errorf("manifest contains no provisionings")
	}
	if manifest.Version != "" && manifest.Version != manifestVersion {
		return fmt.Errorf("unsupported manifest version %q", manifest.Version)
	}

	seen := make(map[string]string)
	for i := range manifest.Provisionings {
		entry := &manifest.Provisionings[i]
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" {
			return fmt.Errorf("manifest contains a provisioning without a name")
		}
		if len(entry.Targets) == 0 {
			return fmt.Errorf("provisioning %q has no targets", entry.Name)
		}

		names := append([]string{entry.Name}, entry.Aliases...)
		for _, name := range names {
			normalized := normalizeLookupName(name)
			if normalized == "" {
				return fmt.Errorf("provisioning %q has an empty alias", entry.Name)
			}
			if previous, ok := seen[normalized]; ok {
				return fmt.Errorf("duplicate provisioning name or alias %q in %q and %q", name, previous, entry.Name)
			}
			seen[normalized] = entry.Name
		}

		for j := range entry.Targets {
			target := &entry.Targets[j]
			target.OS = normalizeTargetOS(target.OS)
			target.Arch = normalizeTargetArch(target.Arch)
			target.Install = strings.TrimSpace(target.Install)
			target.InstallVersioned = strings.TrimSpace(target.InstallVersioned)

			if target.Install == "" {
				return fmt.Errorf("provisioning %q target %d is missing install", entry.Name, j+1)
			}
			if target.Arch != "" && target.OS == "" {
				return fmt.Errorf("provisioning %q target %d sets arch without os", entry.Name, j+1)
			}
		}
	}

	return nil
}

func selectTarget(targets []Target, currentOS, currentArch string) (*Target, error) {
	type rankedTarget struct {
		index int
		rank  int
	}

	candidates := make([]rankedTarget, 0, len(targets))
	for i := range targets {
		rank, ok := targetRank(targets[i], currentOS, currentArch)
		if ok {
			candidates = append(candidates, rankedTarget{index: i, rank: rank})
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no matching target for %s/%s", currentOS, currentArch)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].rank > candidates[j].rank
	})

	best := candidates[0]
	if len(candidates) > 1 && candidates[1].rank == best.rank {
		return nil, fmt.Errorf("ambiguous targets with equal specificity for %s/%s", currentOS, currentArch)
	}

	target := targets[best.index]
	return &target, nil
}

func targetRank(target Target, currentOS, currentArch string) (int, bool) {
	switch {
	case target.OS == currentOS && target.Arch == currentArch:
		return 3, true
	case target.OS == currentOS && target.Arch == "":
		return 2, true
	case target.OS == "" && target.Arch == "":
		return 1, true
	default:
		return 0, false
	}
}

func deriveExactVersion(constraints []statement.VersionConstraint) (string, bool, error) {
	if len(constraints) == 0 {
		return "", false, nil
	}

	var (
		lowerInclusive string
		upperInclusive string
		exact          string
	)

	for _, constraint := range constraints {
		version := strings.TrimSpace(constraint.Version)
		switch constraint.Operator {
		case "=", "==":
			if exact != "" && compareVersions(exact, version) != 0 {
				return "", false, fmt.Errorf("conflicting exact versions %q and %q", exact, version)
			}
			exact = version
		case ">=":
			if lowerInclusive == "" || compareVersions(version, lowerInclusive) > 0 {
				lowerInclusive = version
			}
		case "<=":
			if upperInclusive == "" || compareVersions(version, upperInclusive) < 0 {
				upperInclusive = version
			}
		case ">", "<":
			return "", false, nil
		default:
			return "", false, fmt.Errorf("unsupported version operator %q", constraint.Operator)
		}
	}

	if exact != "" {
		if lowerInclusive != "" && compareVersions(exact, lowerInclusive) < 0 {
			return "", false, fmt.Errorf("exact version %q is below lower bound %q", exact, lowerInclusive)
		}
		if upperInclusive != "" && compareVersions(exact, upperInclusive) > 0 {
			return "", false, fmt.Errorf("exact version %q is above upper bound %q", exact, upperInclusive)
		}
		return exact, true, nil
	}

	if lowerInclusive != "" && upperInclusive != "" && compareVersions(lowerInclusive, upperInclusive) == 0 {
		return lowerInclusive, true, nil
	}

	return "", false, nil
}

func compareVersions(a, b string) int {
	left := parseVersion(a)
	right := parseVersion(b)
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}

	for i := 0; i < maxLen; i++ {
		li := 0
		ri := 0
		if i < len(left) {
			li = left[i]
		}
		if i < len(right) {
			ri = right[i]
		}
		switch {
		case li < ri:
			return -1
		case li > ri:
			return 1
		}
	}
	return 0
}

func parseVersion(version string) []int {
	parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(version, "v")), ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		num := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			num = num*10 + int(r-'0')
		}
		values = append(values, num)
	}
	return values
}

func normalizeLookupName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeTargetOS(osName string) string {
	switch strings.ToLower(strings.TrimSpace(osName)) {
	case "", "linux", "windows":
		return strings.ToLower(strings.TrimSpace(osName))
	case "darwin", "mac", "macos", "osx":
		return "darwin"
	default:
		return strings.ToLower(strings.TrimSpace(osName))
	}
}

func normalizeTargetArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

func expandUserPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func isGitSource(source string) bool {
	return strings.HasPrefix(source, "ssh://") || strings.HasPrefix(source, "git+ssh://")
}

func parseGitSource(source string) (string, string, string, error) {
	raw := strings.TrimPrefix(source, "git+")
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("parse git source: %w", err)
	}

	trimmedPath := strings.TrimPrefix(parsed.Path, "/")
	split := strings.SplitN(trimmedPath, "//", 2)
	if len(split) != 2 || strings.TrimSpace(split[0]) == "" {
		return "", "", "", fmt.Errorf("git provisioning source must include repo and manifest path: %q", source)
	}

	repoURL := &url.URL{
		Scheme: parsed.Scheme,
		User:   parsed.User,
		Host:   parsed.Host,
		Path:   "/" + split[0],
	}
	manifestPath := strings.TrimSpace(split[1])
	if manifestPath == "" {
		manifestPath = defaultManifestName
	}

	return repoURL.String(), filepath.FromSlash(manifestPath), parsed.Query().Get("ref"), nil
}

type gitCommandFetcher struct{}

func (gitCommandFetcher) FetchManifest(ctx context.Context, repoURL, manifestPath, ref string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "drun-provisioning-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, repoURL, tempDir)

	// #nosec G204 -- provisioning sources intentionally invoke git with a validated remote URL and ref.
	cmd := exec.CommandContext(ctx, "git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	return os.ReadFile(filepath.Join(tempDir, manifestPath))
}
