# Examples

## Examples

### Simple Task

```drun
task "hello":
  info "Hello, drun v2!"
```

### Task with Parameters

```drun
task "greet" means "Greet someone by name":
  requires name
  given title defaults to "friend"

  info "Hello, {title} {name}!"
```

### Docker Build and Deploy

```drun
project "webapp" version "1.0.0":
  set registry to "ghcr.io/company"

task "build" means "Build Docker image":
  given tag defaults to "{current git commit}"

  step "Building application image"
  build docker image "{registry}/webapp:{tag}"
  success "Build completed: {registry}/webapp:{tag}"

task "deploy" means "Deploy to Kubernetes":
  requires environment from ["dev", "staging", "production"]
  given replicas defaults to 3
  depends on build

  step "Deploying to {environment}"

  when environment is "production":
    require manual approval "Deploy to production?"
    ensure git repo is clean

  deploy webapp:latest to kubernetes namespace {environment} with {replicas} replicas
  wait for rollout to complete

  success "Deployment to {environment} completed"
```

### Git Branch Operations  *New*

```drun
version: 2.0

task "git branch operations" means "Demonstrate git branch builtin and pipe operations":
  info " Testing git branch operations..."

  # Basic git branch builtin
  set $branch to {current git branch}
  info "Current branch: {$branch}"

  # Git branch with pipe operations
  set $safe_branch to {current git branch | replace "/" by "-"}
  info "Safe branch name: {$safe_branch}"

  # Use in parameter defaults
  given $deployment_branch defaults to "{current git branch | replace '/' by '-' | lowercase}"
  info "Deployment branch: {$deployment_branch}"

  success " Git branch operations completed!"

task "parameter defaults with builtins" means "Demonstrate builtin functions in parameter defaults":
  given $commit defaults to "{current git commit}"
  given $branch defaults to "{current git branch}"
  given $safe_branch defaults to "{current git branch | replace '/' by '-'}"

  info " Parameter values:"
  info "  Commit: {$commit}"
  info "  Branch: {$branch}"
  info "  Safe branch: {$safe_branch}"

  success " Parameter defaults test completed!"
```

### Complex CI/CD Pipeline

```drun
project "microservices":
  set registry to "ghcr.io/company"
  set environments to ["dev", "staging", "production"]
  set services to ["api", "web", "worker"]

task "test matrix" means "Run tests across multiple configurations":
  for each service in {services}:
    for each env in ["test", "integration"]:
      step "Testing {service} in {env} environment"
      run tests for {service} in {env} mode

task "build all" means "Build all service images":
  for each service in {services} in parallel:
    step "Building {service}"
    build docker image "{registry}/{service}:latest"
    push image "{registry}/{service}:latest"

task "deploy pipeline" means "Full deployment pipeline":
  requires target_env from {environments}
  depends on test_matrix and build_all

  let deployment_id be "deploy-{now.unix}"
  let failed_services be empty list

  step "Starting deployment {deployment_id} to {target_env}"

  for each service in {services}:
    try:
      deploy {service}:latest to kubernetes namespace {target_env}
      wait for {service} rollout to complete
      check health of {service} in {target_env}
      success "{service} deployed successfully"
    catch deployment_error:
      error "{service} deployment failed: {deployment_error}"
      add {service} to {failed_services}

  if {failed_services} is not empty:
    error "Deployment failed for services: {failed_services}"

    for each service in {failed_services}:
      rollback {service} in {target_env}

    fail "Deployment {deployment_id} failed"
  else:
    success "Deployment {deployment_id} completed successfully"
    notify slack " All services deployed to {target_env}"
```

### Advanced Features Example

```drun
project "advanced-example":
  set notification_webhook to secret "slack_webhook"

  before any task:
    capture start_time from now
    info "Starting task execution at {start_time}"

  after any task:
    capture end_time from now
    let duration be {end_time} - {start_time}
    info "Task completed in {duration}"

  # Tool-level lifecycle hooks (run once per drun execution)
  on drun setup:
    info " Starting drun execution pipeline"
    info " Tool version: {$globals.drun_version}"
    capture pipeline_start_time from now

  on drun teardown:
    capture pipeline_end_time from now
    let total_time be {pipeline_end_time} - {pipeline_start_time}
    info " Drun execution pipeline completed"
    info " Total execution time: {total_time}"

task "smart deployment" means "Intelligent deployment with auto-detection":
  requires environment from ["dev", "staging", "production"]
  given force_deploy defaults to false

  # Smart project detection
  when symfony is detected:
    let app_type be "symfony"
    let health_endpoint be "/health"
  when laravel is detected:
    let app_type be "laravel"
    let health_endpoint be "/api/health"
  when node project exists:
    let app_type be "node"
    let health_endpoint be "/healthz"
  else:
    let app_type be "generic"
    let health_endpoint be "/"

  step "Detected {app_type} application"

  # Environment-specific validation
  when environment is "production":
    if not force_deploy and git repo is dirty:
      error "Cannot deploy dirty repository to production"
      fail

    if not git tag exists for current commit:
      warn "No git tag found for current commit"
      require manual approval "Deploy untagged commit to production?"

  # Smart build detection
  if file "Dockerfile" exists:
    step "Building containerized application"
    build docker image "myapp:latest"

    if kubernetes is available:
      deploy myapp:latest to kubernetes namespace {environment}
    else:
      run container "myapp:latest" on port 8080
  else:
    step "Deploying non-containerized application"

    when app_type is "symfony":
      run "composer install --no-dev --optimize-autoloader"
      run "php bin/console cache:clear --env=prod"
    when app_type is "laravel":
      run "composer install --no-dev --optimize-autoloader"
      run "php artisan config:cache"
      run "php artisan route:cache"
    when app_type is "node":
      install dependencies
      run "npm run build"

  # Health check
  step "Performing health check"

  for attempt from 1 to 5:
    try:
      check health of service at "https://app-{environment}.example.com{health_endpoint}"
      success "Health check passed"
      break
    catch health_check_error:
      if attempt == 5:
        error "Health check failed after 5 attempts"
        fail
      warn "Health check attempt {attempt} failed, retrying in {attempt * 2} seconds..."
      wait {attempt * 2} seconds

  # Notification
  let message be " {app_type} application deployed to {environment}"
  send POST request to {notification_webhook} with data {
    text: message,
    username: "drun-bot"
  }

  success "Deployment completed successfully"
```

