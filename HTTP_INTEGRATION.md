# 🌐 HTTP Integration in drun

drun provides powerful HTTP client capabilities that can be used directly in your templates and recipes. This allows you to integrate with APIs, send notifications, fetch data, and perform complex HTTP-based workflows.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [HTTP Endpoint Configuration](#http-endpoint-configuration)
- [Template Functions](#template-functions)
- [Authentication](#authentication)
- [Caching](#caching)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Best Practices](#best-practices)

## 🚀 Quick Start

### 1. Define HTTP Endpoints

Add HTTP endpoint definitions to your `drun.yml`:

```yaml
version: "1.4"

http:
  api:
    url: "https://api.example.com"
    headers:
      Content-Type: "application/json"
    timeout: 30s
    auth:
      type: "bearer"
      token: "{{ secret \"api_token\" }}"

secrets:
  api_token:
    source: "env://API_TOKEN"
    required: true
```

### 2. Use in Templates

Make HTTP calls in your recipes:

```yaml
recipes:
  fetch-data:
    run: |
      {{ step "Fetching data from API" }}
      
      # Make HTTP call using predefined endpoint
      {{ $data := httpCallJSON "api" (dict "url" "/users") }}
      
      {{ info (printf "Found %d users" (len $data)) }}
      
      {{ success "Data fetched successfully" }}
```

## 🔧 HTTP Endpoint Configuration

### Basic Configuration

```yaml
http:
  endpoint_name:
    url: "https://api.example.com"          # Base URL (required)
    method: "GET"                           # Default HTTP method
    headers:                                # Default headers
      Content-Type: "application/json"
      User-Agent: "drun/1.4"
    timeout: 30s                            # Request timeout
    description: "Example API endpoint"     # Documentation
```

### Advanced Configuration

```yaml
http:
  advanced_api:
    url: "https://api.example.com"
    method: "POST"
    headers:
      Content-Type: "application/json"
      Accept: "application/json"
    
    # Authentication
    auth:
      type: "bearer"                        # basic, bearer, api-key, oauth2
      token: "{{ secret \"api_token\" }}"   # Can reference secrets
    
    # Timeout configuration
    timeout: 30s
    
    # Retry configuration
    retry:
      max_attempts: 3                       # Maximum retry attempts
      backoff: "exponential"                # exponential, linear, fixed
      base_delay: 1s                        # Base delay between retries
      max_delay: 30s                        # Maximum delay
    
    # Caching configuration
    cache:
      ttl: 5m                               # Cache time-to-live
      key: "api-{{ .endpoint }}"            # Custom cache key template
```

## 🔐 Authentication

### Bearer Token Authentication

```yaml
http:
  api:
    url: "https://api.example.com"
    auth:
      type: "bearer"
      token: "{{ secret \"api_token\" }}"
```

### Basic Authentication

```yaml
http:
  api:
    url: "https://api.example.com"
    auth:
      type: "basic"
      user: "{{ secret \"api_user\" }}"
      pass: "{{ secret \"api_pass\" }}"
```

### API Key Authentication

```yaml
http:
  api:
    url: "https://api.example.com"
    auth:
      type: "api-key"
      header: "X-API-Key"                   # Header name
      token: "{{ secret \"api_key\" }}"     # API key value

  # Or as query parameter
  api_query:
    url: "https://api.example.com"
    auth:
      type: "api-key"
      query: "api_key"                      # Query parameter name
      token: "{{ secret \"api_key\" }}"
```

### OAuth2 Authentication

```yaml
http:
  api:
    url: "https://api.example.com"
    auth:
      type: "oauth2"
      token: "{{ secret \"access_token\" }}"
```

## 📡 Template Functions

### `httpCall` - Make HTTP Call

Make an HTTP call using a predefined endpoint:

```yaml
# Basic call
{{ $response := httpCall "endpoint_name" }}

# With options
{{ $response := httpCall "endpoint_name" (dict "url" "/specific/path" "query" (dict "limit" "10")) }}
```

### `httpCallJSON` - Make HTTP Call and Parse JSON

Make an HTTP call and automatically parse JSON response:

```yaml
{{ $data := httpCallJSON "api" (dict "url" "/users") }}
{{ range $user := $data }}
echo "User: {{ $user.name }}"
{{ end }}
```

### `httpGet` - Direct GET Request

Make a direct GET request to any URL:

```yaml
{{ $response := httpGet "https://api.example.com/status" }}
```

### `httpPost` - Direct POST Request

Make a direct POST request:

```yaml
# With JSON body
{{ $data := dict "name" "John" "email" "john@example.com" }}
{{ $response := httpPost "https://api.example.com/users" $data }}

# With string body
{{ $response := httpPost "https://api.example.com/webhook" "Hello, World!" }}
```

### `httpPut` - Direct PUT Request

```yaml
{{ $data := dict "name" "Updated Name" }}
{{ $response := httpPut "https://api.example.com/users/123" $data }}
```

### `httpDelete` - Direct DELETE Request

```yaml
{{ $response := httpDelete "https://api.example.com/users/123" }}
```

### Function Options

All HTTP functions accept an optional map of options:

```yaml
{{ $options := dict 
     "headers" (dict "X-Custom-Header" "value")
     "query" (dict "param1" "value1" "param2" "value2")
     "timeout" "15s"
     "body" (dict "key" "value")
}}

{{ $response := httpGet "https://api.example.com/data" $options }}
```

## 💾 Caching

### Automatic Caching

Configure caching in endpoint definitions:

```yaml
http:
  api:
    url: "https://api.example.com"
    cache:
      ttl: 5m                               # Cache for 5 minutes
      key: "api-{{ .endpoint }}-{{ .query }}" # Custom cache key
```

### Manual Cache Control

Control caching per request:

```yaml
# Cache this specific call for 10 minutes
{{ $data := httpCallJSON "api" (dict "url" "/users" "cache_ttl" "10m") }}
```

## 🚨 Error Handling

### Automatic Retries

Configure retries in endpoint definitions:

```yaml
http:
  unreliable_api:
    url: "https://unreliable-api.example.com"
    retry:
      max_attempts: 5
      backoff: "exponential"
      base_delay: 1s
      max_delay: 30s
```

### Manual Error Handling

```yaml
recipes:
  safe-api-call:
    run: |
      {{ step "Making safe API call" }}
      
      # Check if the call succeeds
      {{ $result := httpGet "https://api.example.com/health" }}
      
      {{ if $result }}
      {{ success "API is healthy" }}
      {{ else }}
      {{ error "API health check failed" }}
      exit 1
      {{ end }}
```

## 📚 Examples

### GitHub Integration

```yaml
http:
  github:
    url: "https://api.github.com"
    headers:
      Accept: "application/vnd.github.v3+json"
      Authorization: "token {{ secret \"github_token\" }}"
    timeout: 30s

recipes:
  create-issue:
    positionals:
      - name: repo
        required: true
      - name: title
        required: true
      - name: body
        required: true
    run: |
      {{ step "Creating GitHub issue" }}
      
      {{ $issue := dict "title" .title "body" .body }}
      {{ $response := httpPost (printf "https://api.github.com/repos/%s/issues" .repo) $issue }}
      
      {{ success "Issue created successfully" }}
```

### Slack Notifications

```yaml
http:
  slack:
    url: "{{ env \"SLACK_WEBHOOK_URL\" }}"
    method: "POST"
    headers:
      Content-Type: "application/json"

recipes:
  notify-deployment:
    run: |
      {{ $message := dict 
           "text" "🚀 Deployment started"
           "username" "drun"
           "icon_emoji" ":rocket:"
      }}
      
      {{ httpPost (env "SLACK_WEBHOOK_URL") $message }}
```

### API Data Processing

```yaml
http:
  api:
    url: "https://jsonplaceholder.typicode.com"
    cache:
      ttl: 2m

recipes:
  process-posts:
    run: |
      {{ step "Processing posts" }}
      
      # Fetch all posts (cached for 2 minutes)
      {{ $posts := httpCallJSON "api" (dict "url" "/posts") }}
      
      {{ info (printf "Processing %d posts" (len $posts)) }}
      
      # Process each post
      {{ range $post := $posts }}
      echo "Post {{ $post.id }}: {{ $post.title }}"
      
      # Fetch comments for each post
      {{ $comments := httpCallJSON "api" (dict "url" (printf "/posts/%d/comments" $post.id)) }}
      echo "  Comments: {{ len $comments }}"
      {{ end }}
      
      {{ success "Post processing completed" }}
```

### Health Check Workflow

```yaml
http:
  service:
    url: "https://my-service.example.com"
    timeout: 5s
    retry:
      max_attempts: 3
      backoff: "fixed"
      base_delay: 2s

recipes:
  health-check:
    run: |
      {{ step "Checking service health" }}
      
      # Check main service
      {{ $health := httpCallJSON "service" (dict "url" "/health") }}
      
      {{ if eq $health.status "healthy" }}
      {{ success "Service is healthy" }}
      
      # Check dependencies
      {{ range $dep := $health.dependencies }}
      {{ if eq $dep.status "healthy" }}
      {{ info (printf "✅ %s: healthy" $dep.name) }}
      {{ else }}
      {{ warn (printf "⚠️  %s: %s" $dep.name $dep.status) }}
      {{ end }}
      {{ end }}
      
      {{ else }}
      {{ error "Service is unhealthy" }}
      exit 1
      {{ end }}
```

## 🎯 Best Practices

### 1. Use Endpoint Definitions

Define reusable endpoints instead of hardcoding URLs:

```yaml
# ✅ Good
http:
  api:
    url: "https://api.example.com"
    auth:
      type: "bearer"
      token: "{{ secret \"api_token\" }}"

recipes:
  fetch-data:
    run: |
      {{ $data := httpCallJSON "api" (dict "url" "/users") }}

# ❌ Avoid
recipes:
  fetch-data:
    run: |
      {{ $data := httpGet "https://api.example.com/users" }}
```

### 2. Use Secrets for Authentication

Never hardcode credentials:

```yaml
# ✅ Good
secrets:
  api_token:
    source: "env://API_TOKEN"
    required: true

http:
  api:
    auth:
      type: "bearer"
      token: "{{ secret \"api_token\" }}"

# ❌ Avoid
http:
  api:
    auth:
      type: "bearer"
      token: "hardcoded-token-123"
```

### 3. Configure Appropriate Timeouts

Set reasonable timeouts for different types of operations:

```yaml
http:
  # Quick health checks
  health:
    url: "https://service.example.com/health"
    timeout: 5s
  
  # Data processing APIs
  data_api:
    url: "https://api.example.com"
    timeout: 30s
  
  # File uploads
  upload_api:
    url: "https://upload.example.com"
    timeout: 5m
```

### 4. Use Caching Wisely

Cache stable data to improve performance:

```yaml
http:
  # Cache configuration data
  config_api:
    url: "https://config.example.com"
    cache:
      ttl: 10m
  
  # Don't cache dynamic data
  metrics_api:
    url: "https://metrics.example.com"
    # No caching for real-time metrics
```

### 5. Handle Errors Gracefully

Always check for errors and provide meaningful feedback:

```yaml
recipes:
  robust-api-call:
    run: |
      {{ step "Making API call" }}
      
      {{ $response := httpCallJSON "api" (dict "url" "/data") }}
      
      {{ if $response }}
      {{ success "Data retrieved successfully" }}
      # Process the data
      {{ else }}
      {{ error "Failed to retrieve data from API" }}
      {{ warn "Check API credentials and network connectivity" }}
      exit 1
      {{ end }}
```

### 6. Use Structured Logging

Provide clear, structured output:

```yaml
recipes:
  api-workflow:
    run: |
      {{ step "Starting API workflow" }}
      
      {{ info "Fetching user data..." }}
      {{ $users := httpCallJSON "api" (dict "url" "/users") }}
      {{ info (printf "Found %d users" (len $users)) }}
      
      {{ info "Processing users..." }}
      # Process users
      
      {{ success "API workflow completed successfully" }}
```

## 🔗 Integration with Other Features

### Matrix Builds with HTTP

```yaml
recipes:
  test-environments:
    matrix:
      env: ["dev", "staging", "prod"]
    run: |
      {{ step (printf "Testing %s environment" .matrix_env) }}
      
      # Environment-specific health check
      {{ $health := httpCallJSON "api" (dict "url" (printf "/%s/health" .matrix_env)) }}
      
      {{ if eq $health.status "healthy" }}
      {{ success (printf "%s environment is healthy" .matrix_env) }}
      {{ else }}
      {{ error (printf "%s environment is unhealthy" .matrix_env) }}
      {{ end }}
```

### Conditional HTTP Calls

```yaml
recipes:
  conditional-deploy:
    run: |
      {{ if eq (env "ENVIRONMENT") "production" }}
      # Production deployment notification
      {{ $msg := dict "text" "🚀 Production deployment started" "channel" "#alerts" }}
      {{ httpPost (env "SLACK_WEBHOOK_URL") $msg }}
      {{ else }}
      # Development deployment
      {{ $msg := dict "text" "🔧 Development deployment started" }}
      {{ httpPost (env "DEV_WEBHOOK_URL") $msg }}
      {{ end }}
```

## 📖 More Information

- **Main Documentation**: [README.md](README.md)
- **Template Functions**: [TEMPLATE_FUNCTIONS.md](TEMPLATE_FUNCTIONS.md)
- **YAML Specification**: [YAML_SPEC.md](YAML_SPEC.md)
- **Examples**: [examples/](examples/) directory

The HTTP integration in drun provides a powerful way to connect your build and deployment workflows with external services, APIs, and notification systems. Use it to create sophisticated, connected automation workflows that can interact with your entire development ecosystem.
