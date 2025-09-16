# 🛠️ drun Template Functions Reference

Quick reference for all built-in template functions in drun.

💡 **Pro Tip**: Use [prerun snippets](YAML_SPEC.md#prerun-snippets-new-feature) to define common functions and colors that are automatically available in all recipes!

## 🐳 Docker Integration

| Function | Description | Example | Output |
|----------|-------------|---------|---------|
| `{{ dockerCompose }}` | Auto-detect Docker Compose command | `{{ dockerCompose }} up` | `docker compose up` or `docker-compose up` |
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
