# 🛠️ drun Template Functions Reference

Quick reference for all built-in template functions in drun.

💡 **Pro Tip**: Use [prerun snippets](YAML_SPEC.md#prerun-snippets-new-feature) to define common functions and colors that are automatically available in all recipes!

## 🐳 Docker Integration

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ dockerCompose }}` | Auto-detect Docker Compose command | `{{ dockerCompose }} up` | `docker compose up` or `docker-compose up` |
| `{{ (dockerCompose).IsRunning }}` | Check if Docker Compose services are running | `{{ if (dockerCompose).IsRunning }}...{{ end }}` | `true` or `false` |
| `{{ dockerBuildx }}` | Auto-detect Docker Buildx command | `{{ dockerBuildx }} build .` | `docker buildx build .` or `docker build .` |
| `{{ hasCommand "kubectl" }}` | Check if command exists in PATH | `{{ if hasCommand "kubectl" }}...{{ end }}` | `true` or `false` |

## 🔗 Git Integration

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ gitBranch }}` | Current Git branch name | `Branch: {{ gitBranch }}` | `Branch: main` |
| `{{ gitCommit }}` | Full commit hash (40 chars) | `{{ gitCommit }}` | `a1b2c3d4e5f6...` (40 chars) |
| `{{ gitShortCommit }}` | Short commit hash (7 chars) | `v{{ gitShortCommit }}` | `va1b2c3d` |
| `{{ isDirty }}` | Working directory has changes | `{{ if isDirty }}dirty{{ end }}` | `dirty` or empty |

## 📦 Project Detection

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ packageManager }}` | Auto-detect package manager | `Using {{ packageManager }}` | `npm`, `yarn`, `go`, `pip`, etc. |
| `{{ hasFile "go.mod" }}` | Check if file exists | `{{ if hasFile "Dockerfile" }}...{{ end }}` | `true` or `false` |
| `{{ isCI }}` | Detect CI environment | `{{ if isCI }}CI Mode{{ end }}` | `CI Mode` or empty |

## 📊 Status Messages

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ step "message" }}` | Step indicator | `{{ step "Building project" }}` | `echo "🚀 Building project"` |
| `{{ info "message" }}` | Information message | `{{ info "Processing files" }}` | `echo "ℹ️  Processing files"` |
| `{{ warn "message" }}` | Warning message | `{{ warn "Deprecated feature" }}` | `echo "⚠️  Deprecated feature"` |
| `{{ error "message" }}` | Error message (non-fatal) | `{{ error "Failed to connect" }}` | `echo "❌ Failed to connect"` |
| `{{ success "message" }}` | Success message | `{{ success "Deploy completed" }}` | `echo "✅ Deploy completed"` |

## 🔐 Secrets Management

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ secret "name" }}` | Access secret value | `TOKEN={{ secret "api_key" }}` | `TOKEN=actual_secret_value` |
| `{{ hasSecret "name" }}` | Check secret availability | `{{ if hasSecret "token" }}...{{ end }}` | `true` or `false` |

## 🌐 HTTP Functions

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ httpCall "endpoint" }}` | Call predefined HTTP endpoint | `{{ httpCall "api" }}` | Raw response string |
| `{{ httpCallJSON "endpoint" }}` | Call endpoint and parse JSON | `{{ $data := httpCallJSON "api" }}` | Parsed JSON object |
| `{{ httpGet "url" }}` | Direct GET request | `{{ httpGet "https://api.example.com/status" }}` | Response body |
| `{{ httpPost "url" data }}` | Direct POST request | `{{ httpPost "https://api.example.com/users" (dict "name" "John") }}` | Response body |
| `{{ httpPut "url" data }}` | Direct PUT request | `{{ httpPut "https://api.example.com/users/1" (dict "name" "Jane") }}` | Response body |
| `{{ httpDelete "url" }}` | Direct DELETE request | `{{ httpDelete "https://api.example.com/users/1" }}` | Response body |

**HTTP Options**: All HTTP functions accept optional parameters:
```yaml
{{ $options := dict 
     "headers" (dict "Authorization" "Bearer token")
     "query" (dict "limit" "10")
     "timeout" "30s"
}}
{{ httpGet "https://api.example.com/data" $options }}
```

## 🛠️ Standard Functions

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ now "2006-01-02" }}` | Current time formatting | `{{ now "2006-01-02 15:04:05" }}` | `2024-01-15 14:30:45` |
| `{{ env "HOME" }}` | Environment variables | `Home: {{ env "HOME" }}` | `Home: /home/user` |
| `{{ .version }}` | Positional arguments | `Version: {{ .version }}` | `Version: v1.0.0` |
| `{{ snippet "name" }}` | Include reusable snippets | `{{ snippet "docker-login" }}` | Snippet content |
| `{{ shellquote .arg }}` | Shell-safe quoting | `echo {{ shellquote .message }}` | `echo "safe message"` |
| `{{ truncate 50 .text }}` | Truncate text to length | `{{ truncate 20 "This is a long message" }}` | `This is a long messa` |
| `{{ stringContains .text "substring" }}` | Check if string contains substring | `{{ if stringContains .env "prod" }}...{{ end }}` | `true` or `false` |

## 🎯 Usage Examples

### Smart Docker Workflow
```yaml
env:
  DOCKER_COMPOSE: "{{ dockerCompose }}"
  DOCKER_BUILDX: "{{ dockerBuildx }}"

recipes:
  build:
    run: |
      {{ step "Building with auto-detected commands" }}
      {{ info "Using: {{ dockerCompose }} and {{ dockerBuildx }}" }}
      
      $DOCKER_BUILDX build -t app:{{ gitShortCommit }} .
      $DOCKER_COMPOSE up -d
      
      {{ success "Build completed!" }}

  status:
    run: |
      {{ step "Checking Docker Compose status" }}
      
      {{ if dockerCompose.IsRunning }}
      {{ success "Services are running" }}
      {{ dockerCompose }} ps
      {{ else }}
      {{ warn "No services are running" }}
      {{ info "Use '{{ dockerCompose }} up -d' to start services" }}
      {{ end }}

  restart:
    run: |
      {{ step "Restarting Docker Compose services" }}
      
      {{ if dockerCompose.IsRunning }}
      {{ info "Stopping running services..." }}
      {{ dockerCompose }} down
      {{ end }}
      
      {{ info "Starting services..." }}
      {{ dockerCompose }} up -d
      
      {{ success "Services restarted!" }}
```

### Git-Aware Deployment
```yaml
recipes:
  deploy:
    run: |
      {{ step "Deploying {{ gitBranch }}@{{ gitShortCommit }}" }}
      
      {{ if isDirty }}
      {{ error "Cannot deploy with uncommitted changes" }}
      exit 1
      {{ end }}
      
      {{ info "Clean working directory - proceeding" }}
      
      # Deploy with git info
      kubectl set image deployment/app \
        app=myapp:{{ gitShortCommit }}
      
      {{ success "Deployed {{ gitBranch }}@{{ gitShortCommit }}" }}
```

### Project-Aware Build
```yaml
recipes:
  build:
    run: |
      {{ step "Building {{ packageManager }} project" }}
      
      {{ if eq (packageManager) "npm" }}
      {{ info "Node.js project detected" }}
      npm ci && npm run build
      {{ else if eq (packageManager) "go" }}
      {{ info "Go project detected" }}
      go build ./...
      {{ else if eq (packageManager) "python" }}
      {{ info "Python project detected" }}
      pip install -r requirements.txt
      {{ else }}
      {{ warn "Unknown project type" }}
      {{ end }}
      
      {{ success "Build completed for {{ packageManager }}" }}
```

### Secure Secrets Usage
```yaml
secrets:
  api_key:
    source: "env://API_KEY"
    required: true
  optional_token:
    source: "env://OPTIONAL_TOKEN"
    required: false

recipes:
  deploy:
    run: |
      {{ step "Secure deployment" }}
      
      {{ if not (hasSecret "api_key") }}
      {{ error "API_KEY is required" }}
      exit 1
      {{ end }}
      
      # Use secrets securely
      curl -H "Authorization: Bearer {{ secret "api_key" }}" \
        https://api.example.com/deploy
      
      {{ if hasSecret "optional_token" }}
      {{ info "Using optional authentication" }}
      {{ else }}
      {{ warn "Using default authentication" }}
      {{ end }}
      
      {{ success "Deployment completed securely" }}
```

### Matrix with Smart Detection
```yaml
recipes:
  test-matrix:
    matrix:
      os: ["ubuntu", "macos", "windows"]
      version: ["16", "18", "20"]
    run: |
      {{ step "Testing {{ .matrix_os }}/{{ .matrix_version }}" }}
      
      {{ info "Project: {{ packageManager }}" }}
      {{ info "Git: {{ gitBranch }}@{{ gitShortCommit }}" }}
      {{ info "CI: {{ isCI }}" }}
      
      # Matrix-specific logic
      {{ if eq .matrix_os "windows" }}
      {{ info "Windows-specific setup" }}
      {{ else }}
      {{ info "Unix-like setup" }}
      {{ end }}
      
      {{ success "Test completed for {{ .matrix_os }}/{{ .matrix_version }}" }}
```

### String Matching and Conditional Logic
```yaml
recipes:
  environment-check:
    run: |
      {{ step "Environment-based configuration" }}
      
      # Check environment type using stringContains
      {{ if stringContains .env "prod" }}
      {{ success "Production environment detected" }}
      export NODE_ENV=production
      export LOG_LEVEL=warn
      {{ else if stringContains .env "dev" }}
      {{ info "Development environment detected" }}
      export NODE_ENV=development
      export LOG_LEVEL=debug
      {{ else }}
      {{ warn "Unknown environment: {{ .env }}" }}
      export NODE_ENV=development
      {{ end }}

  file-type-handler:
    run: |
      {{ step "Processing files based on type" }}
      
      # Handle different file types
      {{ range $file := .files }}
      {{ if stringContains $file ".js" }}
      {{ info "Processing JavaScript file: {{ $file }}" }}
      npm run lint {{ $file }}
      {{ else if stringContains $file ".go" }}
      {{ info "Processing Go file: {{ $file }}" }}
      go fmt {{ $file }}
      gofmt -s -w {{ $file }}
      {{ else if stringContains $file ".py" }}
      {{ info "Processing Python file: {{ $file }}" }}
      black {{ $file }}
      {{ else }}
      {{ warn "Unknown file type: {{ $file }}" }}
      {{ end }}
      {{ end }}

  smart-deployment:
    run: |
      {{ step "Smart deployment based on branch and environment" }}
      
      # Check branch and environment for deployment strategy
      {{ $branch := gitBranch }}
      {{ $env := .env }}
      
      {{ if and (stringContains $branch "main") (stringContains $env "prod") }}
      {{ success "Production deployment on main branch" }}
      {{ info "Using blue-green deployment strategy" }}
      kubectl apply -f k8s/production/
      {{ else if and (stringContains $branch "develop") (stringContains $env "staging") }}
      {{ info "Staging deployment on develop branch" }}
      kubectl apply -f k8s/staging/
      {{ else if stringContains $branch "feature/" }}
      {{ info "Feature branch deployment" }}
      # Create temporary namespace for feature testing
      kubectl create namespace feature-{{ $branch | replace "feature/" "" }}
      {{ else }}
      {{ error "Invalid deployment configuration" }}
      {{ error "Branch: {{ $branch }}, Environment: {{ $env }}" }}
      exit 1
      {{ end }}

  negation-examples:
    run: |
      {{ step "Using negation with stringContains" }}
      
      # Check if NOT a test file
      {{ if not (stringContains .filename "test") }}
      {{ info "Processing non-test file: {{ .filename }}" }}
      npm run build {{ .filename }}
      {{ else }}
      {{ info "Skipping test file: {{ .filename }}" }}
      {{ end }}
      
      # Check if NOT in CI environment
      {{ if not (stringContains (env "CI") "true") }}
      {{ warn "Not running in CI - enabling development mode" }}
      export DEBUG=true
      {{ else }}
      {{ info "CI environment detected - using production settings" }}
      {{ end }}
      
      # Multiple negation conditions
      {{ $file := .filename }}
      {{ if and (not (stringContains $file ".test.")) (not (stringContains $file ".spec.")) }}
      {{ success "Processing production file: {{ $file }}" }}
      {{ else }}
      {{ info "Skipping test/spec file: {{ $file }}" }}
      {{ end }}
```

### HTTP API Integration
```yaml
# Define HTTP endpoints
http:
  github:
    url: "https://api.github.com"
    headers:
      Accept: "application/vnd.github.v3+json"
      Authorization: "token {{ secret \"github_token\" }}"
    timeout: 30s
    cache:
      ttl: 5m

  slack:
    url: "{{ env \"SLACK_WEBHOOK_URL\" }}"
    method: "POST"
    headers:
      Content-Type: "application/json"

secrets:
  github_token:
    source: "env://GITHUB_TOKEN"
    required: true

recipes:
  github-status:
    run: |
      {{ step "Checking GitHub API status" }}
      
      # Get user info using predefined endpoint
      {{ $user := httpCallJSON "github" (dict "url" "/user") }}
      {{ info (printf "Authenticated as: %s" $user.login) }}
      
      # Get repository info
      {{ $repo := httpCallJSON "github" (dict "url" "/repos/owner/repo") }}
      {{ info (printf "Repository: %s (%d stars)" $repo.name $repo.stargazers_count) }}
      
      # Send notification to Slack
      {{ $message := dict "text" (printf "✅ GitHub check completed for %s" $user.login) }}
      {{ httpPost (env "SLACK_WEBHOOK_URL") $message }}
      
      {{ success "GitHub integration completed" }}

  api-workflow:
    run: |
      {{ step "API workflow with error handling" }}
      
      # Direct HTTP calls with options
      {{ $options := dict 
           "headers" (dict "User-Agent" "drun/1.4")
           "query" (dict "per_page" "5")
           "timeout" "10s"
      }}
      
      {{ $repos := httpCallJSON "github" (merge $options (dict "url" "/user/repos")) }}
      
      {{ info (printf "Found %d repositories" (len $repos)) }}
      {{ range $repo := $repos }}
      echo "- {{ $repo.name }}: {{ $repo.description }}"
      {{ end }}
      
      {{ success "API workflow completed" }}
```

## 🔗 Sprig Functions

drun includes all [Sprig](https://masterminds.github.io/sprig/) functions (150+ additional functions):

### String Functions
- `{{ upper "hello" }}` → `HELLO`
- `{{ lower "WORLD" }}` → `world`
- `{{ title "hello world" }}` → `Hello World`
- `{{ trim " text " }}` → `text`

### Math Functions
- `{{ add 1 2 }}` → `3`
- `{{ sub 5 3 }}` → `2`
- `{{ mul 4 3 }}` → `12`
- `{{ div 10 2 }}` → `5`

### Date Functions
- `{{ now }}` → Current time
- `{{ date "2006-01-02" .timestamp }}` → Formatted date
- `{{ dateInZone "2006-01-02" .timestamp "UTC" }}` → Date in timezone

### List Functions
- `{{ list "a" "b" "c" }}` → `[a b c]`
- `{{ join "," (list "a" "b" "c") }}` → `a,b,c`
- `{{ split "," "a,b,c" }}` → `[a b c]`

### Conditional Functions
- `{{ if eq .env "prod" }}production{{ else }}development{{ end }}`
- `{{ default "fallback" .optional_value }}`
- `{{ empty .value }}` → Check if empty

## 💡 Pro Tips

1. **Combine Functions**: `{{ step "Deploying {{ gitBranch }}@{{ gitShortCommit }}" }}`
2. **Conditional Logic**: Use `{{ if hasFile "Dockerfile" }}` for smart detection
3. **Error Handling**: Use `{{ error }}` for non-fatal warnings
4. **Status Updates**: Provide clear feedback with status functions
5. **Security**: Never log secrets in plain text - use `{{ secret }}` function
6. **Performance**: Functions are cached - use liberally without performance concerns
7. **HTTP Integration**: Define endpoints in YAML, use `{{ httpCallJSON }}` for APIs
8. **API Error Handling**: Check response status and provide meaningful error messages
9. **HTTP Caching**: Use endpoint-level caching for frequently accessed APIs
10. **Authentication**: Store API tokens as secrets, reference with `{{ secret "name" }}`

## 📚 More Information

- **Main Documentation**: [README.md](README.md)
- **HTTP Integration**: [HTTP_INTEGRATION.md](HTTP_INTEGRATION.md)
- **Examples**: [examples/](examples/) directory
- **Sprig Documentation**: https://masterminds.github.io/sprig/
- **Go Templates**: https://pkg.go.dev/text/template
