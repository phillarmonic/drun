# 📚 drun Examples

This directory contains comprehensive examples showcasing all of drun's powerful features. Each example is designed to be educational and demonstrates real-world usage patterns.

## 🎯 Quick Start

```bash
# Try the comprehensive feature showcase
./bin/drun -f examples/feature-showcase.yml showcase-all

# Test matrix execution across multiple configurations
./bin/drun -f examples/matrix-working-demo.yml test-matrix

# Explore remote includes and caching
./bin/drun -f examples/remote-includes-showcase.yml show-remote-capabilities

# See smart template functions in action
./bin/drun -f examples/next-level-features.yml smart-detection
```

## 📖 Example Files

### 🚀 **Core Features**

#### [`simple.yml`](simple.yml)
**Basic recipes and fundamental patterns**
- Simple recipe definitions
- Positional arguments
- Basic templating
- Environment variables

```bash
# Run basic examples
./bin/drun -f examples/simple.yml hello
./bin/drun -f examples/simple.yml greet Alice
```

#### [`prerun-demo.yml`](prerun-demo.yml) ⭐ **UPDATED**
**Recipe-prerun and recipe-postrun snippets for DRY common setup**
- Universal ANSI color codes for any project type
- Generic helper functions (log_info, log_success, etc.)
- Common shell settings and error handling
- Per-recipe completion logging and cleanup
- Language-agnostic project workflow examples

```bash
# See recipe-prerun and recipe-postrun snippets in action
./bin/drun -f examples/prerun-demo.yml setup --dry-run
./bin/drun -f examples/prerun-demo.yml build --target=production
./bin/drun -f examples/prerun-demo.yml test --coverage
./bin/drun -f examples/prerun-demo.yml deploy --environment=staging --dry-run
```

#### [`lifecycle-demo.yml`](lifecycle-demo.yml) ⭐ **NEW v1.4.0**
**Complete execution lifecycle with before/after blocks**
- **Before blocks**: Run once before any recipe execution (setup, validation)
- **After blocks**: Run once after all recipes complete (cleanup, reporting)
- **State management**: Shared state across lifecycle phases
- **Error handling**: After blocks run even when recipes fail
- **Pipeline reporting**: JSON reports and execution metrics

```bash
# See lifecycle management in action
./bin/drun -f examples/lifecycle-demo.yml build
./bin/drun -f examples/lifecycle-demo.yml deploy staging
./bin/drun -f examples/lifecycle-demo.yml fail  # After blocks still run!
```

#### [`namespacing-demo.yml`](namespacing-demo.yml) ⭐ **NEW v1.4.0**
**Recipe namespacing to prevent collisions**
- **Namespace syntax**: `namespace::path` in include statements
- **Collision prevention**: Multiple files can have recipes with same names
- **Organization**: Clear separation by source (docker:build, k8s:deploy, etc.)
- **Mixed dependencies**: Recipes can depend on namespaced recipes

```bash
# Explore namespacing features
./bin/drun -f examples/namespacing-demo.yml list-recipes
./bin/drun -f examples/namespacing-demo.yml build        # Local recipe
./bin/drun -f examples/namespacing-demo.yml docker:build # Namespaced recipe
./bin/drun -f examples/namespacing-demo.yml full-build   # Mixed dependencies
```

#### [`complete-demo.yml`](complete-demo.yml) ⭐ **NEW v1.4.0**
**Production-ready pipeline combining lifecycle + namespacing**
- **Full CI pipeline**: Complete build, test, package, deploy workflow
- **Mixed dependencies**: Local and namespaced recipe dependencies
- **Pipeline reporting**: JSON reports and artifact archiving
- **Real-world patterns**: Production-ready configuration examples

```bash
# Run complete CI pipeline
./bin/drun -f examples/complete-demo.yml full-ci
./bin/drun -f examples/complete-demo.yml deploy production
./bin/drun -f examples/complete-demo.yml status
```

#### [`snippets-showcase.yml`](snippets-showcase.yml) ⭐ **NEW**
**Comprehensive snippet library and patterns**
- Advanced snippet patterns and best practices
- Prerun snippet calls with `{{ snippet "name" }}`
- Conditional snippet usage with flags
- Environment detection and tool checking
- Professional logging and error handling
- Real-world development workflow examples

```bash
# Explore snippet capabilities
./bin/drun -f examples/snippets-showcase.yml demo-snippets --dry-run
./bin/drun -f examples/snippets-showcase.yml hello
./bin/drun -f examples/snippets-showcase.yml setup --tools --node
./bin/drun -f examples/snippets-showcase.yml test --unit --coverage
./bin/drun -f examples/snippets-showcase.yml build --docker --push
```

#### [`docker-devops.yml`](docker-devops.yml)
**Docker workflows with intelligent auto-detection**
- Auto-detect Docker Compose vs docker-compose
- Auto-detect Docker Buildx vs docker build
- Multi-stage builds and deployments
- Environment-specific configurations

```bash
# Smart Docker operations
./bin/drun -f examples/docker-devops.yml build
./bin/drun -f examples/docker-devops.yml deploy production
```

**Features demonstrated:**
- `{{ dockerCompose }}` and `{{ dockerBuildx }}` functions
- Conditional Docker command usage
- Multi-environment deployments

### 🌟 **Advanced Features**

#### [`matrix-working-demo.yml`](matrix-working-demo.yml)
**Matrix execution across multiple configurations**
- Multi-dimensional matrix builds
- OS, version, and architecture combinations
- Matrix variable access in templates
- Conditional logic based on matrix values

```bash
# Run matrix tests (generates multiple jobs)
./bin/drun -f examples/matrix-working-demo.yml test-matrix

# Matrix with dependencies
./bin/drun -f examples/matrix-working-demo.yml build-matrix
```

**Matrix expansion:**
- `test-matrix` → 18 jobs (3 OS × 3 versions × 2 arch)
- `build-matrix` → 4 jobs (2 arch × 2 variants)

#### [`secrets-demo.yml`](secrets-demo.yml)
**Secure secrets management**
- Environment variable secrets (`env://`)
- File-based secrets (`file://`)
- Required vs optional secrets
- Secure usage patterns

```bash
# Set up secrets
export API_KEY="your-api-key"
echo "deploy-token-123" > ~/.secrets/deploy-token

# Run with secrets
./bin/drun -f examples/secrets-demo.yml secure-deploy
```

**Security features:**
- Secrets not logged in plain text
- Multiple source types
- Validation and error handling

#### [`includes-demo.yml`](includes-demo.yml)
**Local and remote recipe includes**
- Local file includes with glob patterns
- Shared recipe libraries
- Modular configuration management

```bash
# Demonstrate local includes
./bin/drun -f examples/includes-demo.yml deploy-with-shared
```

#### [`remote-includes-showcase.yml`](remote-includes-showcase.yml)
**Remote includes deep dive**
- HTTP/HTTPS includes with caching
- Git repository includes with branch/tag references
- Performance optimization through intelligent caching
- Enterprise-grade recipe sharing

```bash
# Explore remote capabilities
./bin/drun -f examples/remote-includes-showcase.yml show-remote-capabilities

# Test HTTP includes
./bin/drun -f examples/remote-includes-showcase.yml test-http-includes

# Test Git includes with versioning
./bin/drun -f examples/remote-includes-showcase.yml test-git-includes
```

**Remote sources:**
- Raw GitHub URLs
- Git repositories with refs
- Intelligent caching system

### 📊 **Developer Experience**

#### [`logging-demo.yml`](logging-demo.yml)
**Advanced logging and metrics**
- Structured status messages with emojis
- Performance tracking and metrics
- Progress indicators
- Error handling patterns

```bash
# See beautiful logging in action
./bin/drun -f examples/logging-demo.yml performance-test
./bin/drun -f examples/logging-demo.yml status-showcase
```

**Logging features:**
- 🚀 Step indicators
- ℹ️ Info messages
- ⚠️ Warnings
- ❌ Errors
- ✅ Success messages

#### [`next-level-features.yml`](next-level-features.yml)
**Smart detection and automation**
- Auto-detect project types (npm, go, python, etc.)
- Git integration (branch, commit, dirty status)
- CI environment detection
- Intelligent command selection

```bash
# Smart project detection
./bin/drun -f examples/next-level-features.yml smart-detection

# Git integration
./bin/drun -f examples/next-level-features.yml git-info

# CI detection
./bin/drun -f examples/next-level-features.yml ci-check
```

#### [`feature-showcase.yml`](feature-showcase.yml)
**Comprehensive feature demonstration**
- All features in one place
- Real-world usage patterns
- Best practices examples
- Performance demonstrations

```bash
# Complete feature tour
./bin/drun -f examples/feature-showcase.yml showcase-all

# Individual feature demos
./bin/drun -f examples/feature-showcase.yml smart-build
./bin/drun -f examples/feature-showcase.yml comprehensive-workflow
```

## 🎓 Learning Path

### 1. **Start Here** - Basic Concepts
```bash
# Learn the fundamentals
./bin/drun -f examples/simple.yml hello
./bin/drun -f examples/simple.yml greet Alice
```

### 2. **New Features (v1.4.0)** - Lifecycle & Namespacing ⭐
```bash
# Try lifecycle management
./bin/drun -f examples/lifecycle-demo.yml build

# Explore recipe namespacing
./bin/drun -f examples/namespacing-demo.yml list-recipes
./bin/drun -f examples/namespacing-demo.yml docker:build

# Complete production pipeline
./bin/drun -f examples/complete-demo.yml full-ci
```

### 3. **Docker Integration** - Smart Detection
```bash
# See auto-detection in action
./bin/drun -f examples/docker-devops.yml build
```

### 4. **Advanced Features** - Matrix & Secrets
```bash
# Try matrix execution
./bin/drun -f examples/matrix-working-demo.yml test-matrix

# Set up and use secrets
export API_KEY="test-key"
./bin/drun -f examples/secrets-demo.yml secure-deploy
```

### 5. **Remote Includes** - Collaboration
```bash
# Explore remote recipe sharing
./bin/drun -f examples/remote-includes-showcase.yml show-remote-capabilities
```

### 6. **Complete Tour** - Everything Together
```bash
# See all features working together
./bin/drun -f examples/feature-showcase.yml showcase-all
```

## 🛠️ Template Functions Reference

All examples demonstrate these powerful template functions:

### 🐳 **Docker Integration**
- `{{ dockerCompose }}` - Auto-detect Docker Compose command
- `{{ dockerBuildx }}` - Auto-detect Docker Buildx command
- `{{ hasCommand "kubectl" }}` - Check command availability

### 🔗 **Git Integration**
- `{{ gitBranch }}` - Current branch name
- `{{ gitCommit }}` - Full commit hash
- `{{ gitShortCommit }}` - Short commit hash (7 chars)
- `{{ isDirty }}` - Working directory status

### 📦 **Project Detection**
- `{{ packageManager }}` - Auto-detect npm, yarn, go, pip, etc.
- `{{ hasFile "go.mod" }}` - File existence check
- `{{ isCI }}` - CI environment detection

### 📊 **Status Messages**
- `{{ step "message" }}` - 🚀 Step indicator
- `{{ info "message" }}` - ℹ️ Information
- `{{ warn "message" }}` - ⚠️ Warning
- `{{ error "message" }}` - ❌ Error (non-fatal)
- `{{ success "message" }}` - ✅ Success

### 🔐 **Secrets Management**
- `{{ secret "name" }}` - Access secret securely
- `{{ hasSecret "name" }}` - Check secret availability

## 🏗️ Real-World Patterns

### **Enterprise Workflow**
```yaml
# Complete CI/CD pipeline
matrix:
  environment: ["dev", "staging", "prod"]
  arch: ["amd64", "arm64"]

secrets:
  deploy_token:
    source: "env://DEPLOY_TOKEN"
    required: true

include:
  - "git+https://company.com/drun-recipes.git@v1.0.0:ci/common.yml"

recipes:
  deploy:
    deps: [test, build]
    run: |
      {{ step "Deploying to {{ .matrix_environment }}/{{ .matrix_arch }}" }}
      {{ if eq .matrix_environment "prod" }}
      {{ warn "Production deployment - extra validation" }}
      {{ end }}
      # Deploy using shared recipes and secrets
```

### **Multi-Project Monorepo**
```yaml
# Smart project detection
recipes:
  build-all:
    run: |
      for dir in */; do
        cd "$dir"
        {{ step "Building $dir ({{ packageManager }})" }}
        
        {{ if eq (packageManager) "npm" }}
        npm ci && npm run build
        {{ else if eq (packageManager) "go" }}
        go build ./...
        {{ else if eq (packageManager) "python" }}
        pip install -r requirements.txt
        {{ end }}
        
        cd ..
      done
```

### **Docker Multi-Architecture**
```yaml
# Cross-platform builds
matrix:
  arch: ["amd64", "arm64"]
  variant: ["alpine", "debian"]

recipes:
  docker-build:
    run: |
      {{ step "Building for {{ .matrix_arch }}/{{ .matrix_variant }}" }}
      
      {{ dockerBuildx }} build \
        --platform linux/{{ .matrix_arch }} \
        -f Dockerfile.{{ .matrix_variant }} \
        -t myapp:{{ gitShortCommit }}-{{ .matrix_arch }}-{{ .matrix_variant }} \
        .
```

## 🚀 Performance Tips

1. **Use Matrix Execution** for parallel builds across configurations
2. **Leverage Remote Includes** for shared recipes (cached automatically)
3. **Smart Detection Functions** reduce conditional complexity
4. **Secrets Management** keeps sensitive data secure
5. **Status Messages** provide clear feedback and debugging

## 🤝 Contributing Examples

Have a great drun pattern? Add it to the examples!

1. Create a new `.yml` file with clear naming
2. Add comprehensive comments explaining the pattern
3. Include usage examples in comments
4. Update this README with your example
5. Test thoroughly with `./bin/drun -f your-example.yml recipe-name`

## 📚 Additional Resources

- **Main README**: [`../README.md`](../README.md) - Complete drun documentation
- **Specification**: [`../spec.md`](../spec.md) - Detailed YAML format reference
- **Roadmap**: [`../ROADMAP.md`](../ROADMAP.md) - Future features and enhancements

---

**Happy automating with drun!** 🎉

These examples demonstrate that drun isn't just a task runner—it's a **comprehensive automation platform** that scales from simple scripts to enterprise-grade CI/CD pipelines.