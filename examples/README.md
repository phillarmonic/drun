# drun Examples

This directory contains comprehensive examples of `drun` configurations for different types of projects and use cases.

## 📁 Available Examples

### 🚀 **[simple.yml](simple.yml)**
Perfect for getting started with drun. Shows basic concepts like:
- Simple recipes
- Positional arguments
- Dependencies
- Environment variables
- Conditional templating

```bash
drun -f examples/simple.yml hello
drun -f examples/simple.yml greet Alice
drun -f examples/simple.yml work
```

### 🐹 **[go-project.yml](go-project.yml)**
Complete Go project workflow including:
- Building and testing
- Code formatting and vetting
- Cross-compilation
- Dependency management

```bash
drun -f examples/go-project.yml build
drun -f examples/go-project.yml test
drun -f examples/go-project.yml build-all
```

### 📦 **[nodejs-project.yml](nodejs-project.yml)**
Node.js/JavaScript project with:
- Development server
- Testing with different modes
- Linting and formatting
- Docker operations
- Deployment workflows

```bash
drun -f examples/nodejs-project.yml dev
drun -f examples/nodejs-project.yml test --set coverage=true
drun -f examples/nodejs-project.yml deploy staging
```

### ⚛️ **[frontend-react.yml](frontend-react.yml)**
Frontend/React project featuring:
- Development and build processes
- Testing (unit, E2E, visual)
- Code quality tools
- Performance auditing
- Deployment strategies

```bash
drun -f examples/frontend-react.yml dev
drun -f examples/frontend-react.yml test-e2e chromium
drun -f examples/frontend-react.yml lighthouse
```

### 🐳 **[docker-devops.yml](docker-devops.yml)**
Docker and DevOps operations:
- Multi-arch image building
- Container testing and security scanning
- Registry operations
- Kubernetes deployment
- CI/CD pipelines

```bash
drun -f examples/docker-devops.yml build latest
drun -f examples/docker-devops.yml deploy production latest
drun -f examples/docker-devops.yml ci
```

### 🐍 **[python-project.yml](python-project.yml)**
Python development workflow:
- Virtual environment management
- Testing with pytest
- Code quality (linting, formatting, type checking)
- Documentation building
- Package building and deployment

```bash
drun -f examples/python-project.yml setup
drun -f examples/python-project.yml test --set coverage=true
drun -f examples/python-project.yml check
```

### 🗄️ **[database-ops.yml](database-ops.yml)**
Database operations and maintenance:
- Backup and restore
- Migrations
- Data seeding
- Performance monitoring
- Maintenance tasks

```bash
drun -f examples/database-ops.yml backup
drun -f examples/database-ops.yml migrate up
drun -f examples/database-ops.yml seed development
```

### 🏢 **[monorepo.yml](monorepo.yml)**
Monorepo/multi-service management:
- Service-specific operations
- Cross-service coordination
- Docker Compose orchestration
- Deployment coordination
- Health monitoring

```bash
drun -f examples/monorepo.yml build-service api
drun -f examples/monorepo.yml test
drun -f examples/monorepo.yml deploy staging latest
```

## 🎯 Key Concepts Demonstrated

### **Positional Arguments**
```yaml
positionals:
  - name: version
    required: true
  - name: arch
    one_of: ["amd64", "arm64", "both"]
  - name: files
    variadic: true
```

### **Templating**
```yaml
run: |
  echo "Building {{ .app_name }} version {{ .version }}"
  {{ if eq .environment "production" }}
  echo "Production build with optimizations"
  {{ else }}
  echo "Development build"
  {{ end }}
```

### **Dependencies**
```yaml
deploy:
  deps: [test, build, security-scan]
  parallel_deps: true
  run: |
    echo "All checks passed, deploying..."
```

### **Environment Variables**
```yaml
env:
  NODE_ENV: production
  BUILD_DATE: "{{ now \"2006-01-02T15:04:05Z\" }}"
  VERSION: "{{ .version }}"
```

### **Snippets**
```yaml
snippets:
  docker_login: |
    echo "$REGISTRY_TOKEN" | docker login {{ .registry }} --username "$REGISTRY_USER" --password-stdin

recipes:
  deploy:
    run: |
      {{ snippet "docker_login" }}
      docker push {{ .image }}
```

## 🚀 Getting Started

1. **Try the simple example first:**
   ```bash
   drun -f examples/simple.yml --list
   drun -f examples/simple.yml hello
   ```

2. **Explore project-specific examples:**
   ```bash
   # For Go projects
   drun -f examples/go-project.yml --list
   
   # For Node.js projects  
   drun -f examples/nodejs-project.yml --list
   ```

3. **Use dry-run to understand what commands would execute:**
   ```bash
   drun -f examples/docker-devops.yml build --dry-run
   ```

4. **Copy and adapt examples for your projects:**
   ```bash
   cp examples/go-project.yml ./drun.yml
   # Edit drun.yml to match your project structure
   ```

## 💡 Tips for Creating Your Own Configurations

1. **Start Simple**: Begin with basic recipes and add complexity gradually
2. **Use Descriptive Names**: Make recipe names and help text clear
3. **Leverage Dependencies**: Break complex workflows into smaller, reusable pieces
4. **Template Everything**: Use variables and templates for flexibility
5. **Add Validation**: Use `one_of` and `required` for positional arguments
6. **Document Well**: Good help text makes recipes self-documenting
7. **Test Thoroughly**: Use `--dry-run` and `--explain` during development

## 🔗 Related Resources

- [Main README](../README.md) - Getting started with drun
- [Specification](../spec.md) - Complete feature specification
- [Template Functions](../README.md#template-functions) - Available template functions

## 🤝 Contributing Examples

Have a great example for a specific use case? We'd love to include it! Consider contributing examples for:

- Rust projects
- Mobile development (React Native, Flutter)
- Infrastructure as Code (Terraform, Pulumi)
- Machine Learning workflows
- Game development
- Embedded systems
- And more!

Each example should be self-contained, well-documented, and demonstrate best practices for that domain.
