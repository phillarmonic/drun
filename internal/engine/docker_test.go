package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_DockerBuildImage(t *testing.T) {
	input := `version: 2.0

task "build":
  docker build image "myapp:latest" from "Dockerfile"
  success "Image built successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "build", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker build image myapp:latest --file Dockerfile",
		"✅ Image built successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerPushImage(t *testing.T) {
	input := `version: 2.0

task "push":
  docker push image "myapp:latest" to "registry.example.com"
  success "Image pushed successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "push", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker push image myapp:latest registry.example.com",
		"✅ Image pushed successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerRunContainer(t *testing.T) {
	input := `version: 2.0

task "run":
  docker run container "webapp" from "myapp:latest"
  success "Container started successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "run", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker run container webapp myapp:latest",
		"✅ Container started successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerCompose(t *testing.T) {
	input := `version: 2.0

task "start_services":
  docker compose up
  success "Services started successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "start_services", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker compose up",
		"✅ Services started successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerTagImage(t *testing.T) {
	input := `version: 2.0

task "tag":
  docker tag image "myapp:latest" as "myapp:v1.0.0"
  success "Image tagged successfully!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "tag", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker tag image myapp:latest myapp:v1.0.0",
		"✅ Image tagged successfully!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerWithVariableInterpolation(t *testing.T) {
	input := `version: 2.0

project "webapp" version "1.0.0":
  set registry to "ghcr.io/company"

task "build":
  docker build image "{project}:{version}" from "Dockerfile"
  docker push image "{project}:{version}" to "{registry}"
  success "Build and push completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "build", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker build image webapp:1.0.0 --file Dockerfile",
		"🐳 Running Docker: docker push image webapp:1.0.0 ghcr.io/company",
		"✅ Build and push completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerDryRun(t *testing.T) {
	input := `version: 2.0

task "deploy":
  docker build image "myapp:latest" from "Dockerfile"
  docker push image "myapp:latest" to "registry.example.com"
  docker run container "webapp" from "myapp:latest"
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would execute Docker command: docker build image myapp:latest from Dockerfile",
		"[DRY RUN] Would execute Docker command: docker push image myapp:latest to registry.example.com",
		"[DRY RUN] Would execute Docker command: docker run container webapp from myapp:latest",
		"[DRY RUN] success: Deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerMultipleOperations(t *testing.T) {
	input := `version: 2.0

task "full_deploy":
  docker build image "myapp:latest" from "Dockerfile"
  docker tag image "myapp:latest" as "myapp:v1.0.0"
  docker push image "myapp:v1.0.0" to "registry.example.com"
  docker run container "webapp" from "myapp:v1.0.0"
  success "Full deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "full_deploy", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🐳 Running Docker: docker build image myapp:latest --file Dockerfile",
		"🐳 Running Docker: docker tag image myapp:latest myapp:v1.0.0",
		"🐳 Running Docker: docker push image myapp:v1.0.0 registry.example.com",
		"🐳 Running Docker: docker run container webapp myapp:v1.0.0",
		"✅ Full deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DockerWithDependencies(t *testing.T) {
	input := `version: 2.0

task "build_image":
  docker build image "myapp:latest" from "Dockerfile"
  success "Build completed!"

task "deploy":
  depends on build_image
  docker run container "webapp" from "myapp:latest"
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that build runs before deploy
	buildIdx := strings.Index(outputStr, "docker build image")
	deployIdx := strings.Index(outputStr, "docker run container")

	if buildIdx == -1 {
		t.Errorf("Expected build Docker command to run")
	}
	if deployIdx == -1 {
		t.Errorf("Expected deploy Docker command to run")
	}
	if buildIdx >= deployIdx {
		t.Errorf("Build Docker command should run before deploy Docker command")
	}

	expectedParts := []string{
		"🐳 Running Docker: docker build image myapp:latest --file Dockerfile",
		"✅ Build completed!",
		"🐳 Running Docker: docker run container webapp myapp:latest",
		"✅ Deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
