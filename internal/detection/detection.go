package detection

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Detector handles smart detection of tools, frameworks, and environments
type Detector struct {
	// Cache for detection results to avoid repeated checks
	cache map[string]interface{}
}

// NewDetector creates a new detector instance
func NewDetector() *Detector {
	return &Detector{
		cache: make(map[string]interface{}),
	}
}

// DetectionResult represents the result of a detection operation
type DetectionResult struct {
	Available bool
	Version   string
	Path      string
	Details   map[string]string
}

// IsToolAvailable checks if a tool is available in the system
func (d *Detector) IsToolAvailable(tool string) bool {
	cacheKey := "tool_" + tool
	if cached, exists := d.cache[cacheKey]; exists {
		return cached.(bool)
	}

	available := false
	switch strings.ToLower(tool) {
	case "docker":
		available = d.isCommandAvailable("docker")
	case "docker-buildx", "docker buildx":
		available = d.isDockerBuildxAvailable()
	case "docker-compose", "docker compose":
		available = d.isDockerComposeAvailable()
	case "git":
		available = d.isCommandAvailable("git")
	case "node", "nodejs":
		available = d.isCommandAvailable("node")
	case "npm":
		available = d.isCommandAvailable("npm")
	case "yarn":
		available = d.isCommandAvailable("yarn")
	case "python", "python3":
		available = d.isCommandAvailable("python") || d.isCommandAvailable("python3")
	case "pip", "pip3":
		available = d.isCommandAvailable("pip") || d.isCommandAvailable("pip3")
	case "go", "golang":
		available = d.isCommandAvailable("go")
	case "java":
		available = d.isCommandAvailable("java")
	case "ruby":
		available = d.isCommandAvailable("ruby")
	case "php":
		available = d.isCommandAvailable("php")
	case "rust", "cargo":
		available = d.isCommandAvailable("cargo")
	case "kubectl":
		available = d.isCommandAvailable("kubectl")
	case "helm":
		available = d.isCommandAvailable("helm")
	case "terraform":
		available = d.isCommandAvailable("terraform")
	case "aws":
		available = d.isCommandAvailable("aws")
	default:
		available = d.isCommandAvailable(tool)
	}

	d.cache[cacheKey] = available
	return available
}

// GetToolVersion gets the version of a tool
func (d *Detector) GetToolVersion(tool string) string {
	cacheKey := "version_" + tool
	if cached, exists := d.cache[cacheKey]; exists {
		return cached.(string)
	}

	version := ""
	switch strings.ToLower(tool) {
	case "docker":
		version = d.getCommandVersion("docker", "--version", `Docker version (\d+\.\d+\.\d+)`)
	case "docker-buildx", "docker buildx":
		version = d.getDockerBuildxVersion()
	case "docker-compose", "docker compose":
		version = d.getDockerComposeVersion()
	case "git":
		version = d.getCommandVersion("git", "--version", `git version (\d+\.\d+\.\d+)`)
	case "node", "nodejs":
		version = d.getCommandVersion("node", "--version", `v(\d+\.\d+\.\d+)`)
	case "npm":
		version = d.getCommandVersion("npm", "--version", `(\d+\.\d+\.\d+)`)
	case "yarn":
		version = d.getCommandVersion("yarn", "--version", `(\d+\.\d+\.\d+)`)
	case "python":
		version = d.getCommandVersion("python", "--version", `Python (\d+\.\d+\.\d+)`)
		if version == "" {
			version = d.getCommandVersion("python3", "--version", `Python (\d+\.\d+\.\d+)`)
		}
	case "go", "golang":
		version = d.getCommandVersion("go", "version", `go version go(\d+\.\d+\.\d+)`)
	case "java":
		version = d.getCommandVersion("java", "-version", `version "(\d+\.\d+\.\d+)`)
	case "ruby":
		version = d.getCommandVersion("ruby", "--version", `ruby (\d+\.\d+\.\d+)`)
	case "php":
		version = d.getCommandVersion("php", "--version", `PHP (\d+\.\d+\.\d+)`)
	case "rust", "cargo":
		version = d.getCommandVersion("cargo", "--version", `cargo (\d+\.\d+\.\d+)`)
	case "kubectl":
		version = d.getCommandVersion("kubectl", "version", `GitVersion:"v(\d+\.\d+\.\d+)"`)
	case "helm":
		version = d.getCommandVersion("helm", "version", `Version:"v(\d+\.\d+\.\d+)"`)
	case "terraform":
		version = d.getCommandVersion("terraform", "version", `Terraform v(\d+\.\d+\.\d+)`)
	}

	d.cache[cacheKey] = version
	return version
}

// DetectEnvironment detects the current environment
func (d *Detector) DetectEnvironment() string {
	cacheKey := "environment"
	if cached, exists := d.cache[cacheKey]; exists {
		return cached.(string)
	}

	env := "local" // default

	// Check for CI environments
	if d.isCIEnvironment() {
		env = "ci"
	} else if d.isProductionEnvironment() {
		env = "production"
	} else if d.isStagingEnvironment() {
		env = "staging"
	} else if d.isDevelopmentEnvironment() {
		env = "development"
	}

	d.cache[cacheKey] = env
	return env
}

// DetectProjectType detects the project type based on files
func (d *Detector) DetectProjectType() []string {
	cacheKey := "project_type"
	if cached, exists := d.cache[cacheKey]; exists {
		return cached.([]string)
	}

	var types []string

	// Check for various project indicators
	if d.fileExists("package.json") {
		types = append(types, "node")

		// Check for specific frameworks
		if d.packageJSONContains("react") {
			types = append(types, "react")
		}
		if d.packageJSONContains("vue") {
			types = append(types, "vue")
		}
		if d.packageJSONContains("@angular/core") {
			types = append(types, "angular")
		}
		if d.packageJSONContains("express") {
			types = append(types, "express")
		}
	}

	if d.fileExists("go.mod") || d.fileExists("go.sum") {
		types = append(types, "go")
	}

	if d.fileExists("requirements.txt") || d.fileExists("setup.py") || d.fileExists("pyproject.toml") {
		types = append(types, "python")

		if d.fileExists("manage.py") || d.requirementsContains("Django") {
			types = append(types, "django")
		}
	}

	if d.fileExists("Gemfile") || d.fileExists("Rakefile") {
		types = append(types, "ruby")

		if d.fileExists("config/application.rb") {
			types = append(types, "rails")
		}
	}

	if d.fileExists("composer.json") {
		types = append(types, "php")

		if d.fileExists("artisan") {
			types = append(types, "laravel")
		}
	}

	if d.fileExists("pom.xml") || d.fileExists("build.gradle") {
		types = append(types, "java")

		if d.pomXMLContains("spring") {
			types = append(types, "spring")
		}
	}

	if d.fileExists("Cargo.toml") {
		types = append(types, "rust")
	}

	if d.fileExists("Dockerfile") {
		types = append(types, "docker")
	}

	if d.fileExists("docker-compose.yml") || d.fileExists("docker-compose.yaml") {
		types = append(types, "docker-compose")
	}

	d.cache[cacheKey] = types
	return types
}

// CompareVersion compares two version strings
func (d *Detector) CompareVersion(version1, operator, version2 string) bool {
	v1 := d.parseVersion(version1)
	v2 := d.parseVersion(version2)

	switch operator {
	case ">=", "gte":
		return d.compareVersions(v1, v2) >= 0
	case ">", "gt":
		return d.compareVersions(v1, v2) > 0
	case "<=", "lte":
		return d.compareVersions(v1, v2) <= 0
	case "<", "lt":
		return d.compareVersions(v1, v2) < 0
	case "==", "=", "eq":
		return d.compareVersions(v1, v2) == 0
	case "!=", "ne":
		return d.compareVersions(v1, v2) != 0
	default:
		return false
	}
}

// Helper methods

func (d *Detector) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func (d *Detector) getCommandVersion(command, flag, pattern string) string {
	cmd := exec.Command(command, flag)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (d *Detector) getCommandVersionWithArgs(command string, args []string, pattern string) string {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (d *Detector) isCIEnvironment() bool {
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TRAVIS", "CIRCLECI"}
	for _, env := range ciVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

func (d *Detector) isProductionEnvironment() bool {
	env := strings.ToLower(os.Getenv("NODE_ENV"))
	if env == "production" {
		return true
	}
	env = strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}

func (d *Detector) isStagingEnvironment() bool {
	env := strings.ToLower(os.Getenv("NODE_ENV"))
	if env == "staging" {
		return true
	}
	env = strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "staging" || env == "stage"
}

func (d *Detector) isDevelopmentEnvironment() bool {
	env := strings.ToLower(os.Getenv("NODE_ENV"))
	if env == "development" || env == "dev" {
		return true
	}
	env = strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "development" || env == "dev"
}

func (d *Detector) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (d *Detector) packageJSONContains(dependency string) bool {
	// This is a simplified check - in a real implementation,
	// you would parse the package.json file and check dependencies
	return false // Placeholder
}

func (d *Detector) requirementsContains(dependency string) bool {
	// This is a simplified check - in a real implementation,
	// you would read and parse requirements.txt
	return false // Placeholder
}

func (d *Detector) pomXMLContains(dependency string) bool {
	// This is a simplified check - in a real implementation,
	// you would parse the pom.xml file
	return false // Placeholder
}

func (d *Detector) parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	var nums []int
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err == nil {
			nums = append(nums, num)
		}
	}
	return nums
}

func (d *Detector) compareVersions(v1, v2 []int) int {
	maxLen := len(v1)
	if len(v2) > maxLen {
		maxLen = len(v2)
	}

	for i := 0; i < maxLen; i++ {
		n1, n2 := 0, 0
		if i < len(v1) {
			n1 = v1[i]
		}
		if i < len(v2) {
			n2 = v2[i]
		}

		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}
	return 0
}

// isDockerBuildxAvailable checks for Docker Buildx availability
// Supports both "docker buildx" and "docker-buildx" commands
func (d *Detector) isDockerBuildxAvailable() bool {
	// First try "docker buildx" (modern Docker installations)
	if d.isCommandAvailable("docker") {
		cmd := exec.Command("docker", "buildx", "version")
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	// Fallback to standalone "docker-buildx" command
	return d.isCommandAvailable("docker-buildx")
}

// isDockerComposeAvailable checks for Docker Compose availability
// Supports both "docker compose" and "docker-compose" commands
func (d *Detector) isDockerComposeAvailable() bool {
	// First try "docker compose" (Docker Compose V2)
	if d.isCommandAvailable("docker") {
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	// Fallback to standalone "docker-compose" command (V1)
	return d.isCommandAvailable("docker-compose")
}

// getDockerBuildxVersion gets the Docker Buildx version
func (d *Detector) getDockerBuildxVersion() string {
	// Try "docker buildx version" first
	if d.isCommandAvailable("docker") {
		version := d.getCommandVersionWithArgs("docker", []string{"buildx", "version"}, `buildx v(\d+\.\d+\.\d+)`)
		if version != "" {
			return version
		}
		// Alternative pattern for different output formats
		version = d.getCommandVersionWithArgs("docker", []string{"buildx", "version"}, `github.com/docker/buildx v(\d+\.\d+\.\d+)`)
		if version != "" {
			return version
		}
	}

	// Fallback to standalone docker-buildx
	return d.getCommandVersion("docker-buildx", "version", `buildx v(\d+\.\d+\.\d+)`)
}

// getDockerComposeVersion gets the Docker Compose version
func (d *Detector) getDockerComposeVersion() string {
	// Try "docker compose version" first (V2)
	if d.isCommandAvailable("docker") {
		version := d.getCommandVersionWithArgs("docker", []string{"compose", "version"}, `Docker Compose version (\d+\.\d+\.\d+)`)
		if version != "" {
			return version
		}
		// Alternative pattern for different output formats
		version = d.getCommandVersionWithArgs("docker", []string{"compose", "version"}, `version v?(\d+\.\d+\.\d+)`)
		if version != "" {
			return version
		}
	}

	// Fallback to standalone docker-compose (V1)
	version := d.getCommandVersion("docker-compose", "version", `docker-compose version (\d+\.\d+\.\d+)`)
	if version != "" {
		return version
	}

	// Alternative pattern for V1
	return d.getCommandVersion("docker-compose", "version", `version (\d+\.\d+\.\d+)`)
}
