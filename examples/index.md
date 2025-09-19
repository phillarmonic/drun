# drun v2 Examples Index

## 📚 Learning Path

Follow this recommended order to learn drun v2:

### 🎯 Beginner (Start Here)
1. **[01-hello-world.drun](01-hello-world.drun)**
   - Your first drun v2 tasks
   - Basic syntax introduction
   - Status messages

2. **[02-parameters.drun](02-parameters.drun)**
   - Required and optional parameters
   - Parameter validation and constraints
   - Variable interpolation

3. **[03-control-flow.drun](03-control-flow.drun)**
   - If/when statements and conditions
   - Loops and iteration
   - Error handling with try/catch

### 🐳 Infrastructure
4. **[04-docker-basics.drun](04-docker-basics.drun)**
   - Docker image building and management
   - Container operations
   - Multi-architecture builds

5. **[05-kubernetes.drun](05-kubernetes.drun)**
   - Kubernetes deployments
   - Scaling and rollbacks
   - Health checks and monitoring

### 🚀 Advanced
6. **[06-cicd-pipeline.drun](06-cicd-pipeline.drun)**
   - Complete CI/CD pipeline
   - Blue-green deployments
   - Production safety checks
   - Parallel execution

7. **[07-smart-detection.drun](07-smart-detection.drun)**
   - Framework detection (Symfony, Laravel, etc.)
   - Tool detection (Docker, Kubernetes, etc.)
   - Intelligent builds and deployments

## 🎨 By Feature

### Parameters & Variables
- [02-parameters.drun](02-parameters.drun) - Parameter declaration and validation
- [03-control-flow.drun](03-control-flow.drun) - Variable assignment and scoping

### Control Flow
- [03-control-flow.drun](03-control-flow.drun) - Comprehensive control flow examples
- [06-cicd-pipeline.drun](06-cicd-pipeline.drun) - Complex conditional logic

### Docker & Containers
- [04-docker-basics.drun](04-docker-basics.drun) - Docker operations
- [06-cicd-pipeline.drun](06-cicd-pipeline.drun) - Container-based CI/CD

### Kubernetes
- [05-kubernetes.drun](05-kubernetes.drun) - Kubernetes operations
- [06-cicd-pipeline.drun](06-cicd-pipeline.drun) - K8s deployments in pipelines

### Smart Detection
- [07-smart-detection.drun](07-smart-detection.drun) - All smart detection features
- [06-cicd-pipeline.drun](06-cicd-pipeline.drun) - Detection in real workflows

## 🔍 By Use Case

### Web Development
- **Symfony/Laravel**: [07-smart-detection.drun](07-smart-detection.drun)
- **Node.js**: [06-cicd-pipeline.drun](06-cicd-pipeline.drun), [07-smart-detection.drun](07-smart-detection.drun)
- **Docker**: [04-docker-basics.drun](04-docker-basics.drun)

### DevOps & Infrastructure
- **CI/CD Pipelines**: [06-cicd-pipeline.drun](06-cicd-pipeline.drun)
- **Kubernetes**: [05-kubernetes.drun](05-kubernetes.drun)
- **Monitoring**: [05-kubernetes.drun](05-kubernetes.drun), [06-cicd-pipeline.drun](06-cicd-pipeline.drun)

### Automation
- **Deployment**: [05-kubernetes.drun](05-kubernetes.drun), [06-cicd-pipeline.drun](06-cicd-pipeline.drun)
- **Testing**: [06-cicd-pipeline.drun](06-cicd-pipeline.drun), [07-smart-detection.drun](07-smart-detection.drun)
- **Build**: [04-docker-basics.drun](04-docker-basics.drun), [07-smart-detection.drun](07-smart-detection.drun)

## 🎯 Quick Reference

### Essential Syntax
```
# Task definition
task "name" means "description":
  requires param from ["option1", "option2"]
  given default_param defaults to "value"
  depends on other_task
  
  step "Doing something"
  info "Information message"
  success "Completed successfully"

# Control flow
if condition:
  do_something
else:
  do_something_else

when variable:
  is "value1": action1
  is "value2": action2
  else: default_action

for each item in collection:
  process item
```

### Common Actions
```
# Docker
build docker image "name:tag"
push image "name:tag" to "registry"
run container "name:tag" on port 8080

# Kubernetes  
deploy app:tag to kubernetes namespace env
scale deployment "app" to 5 replicas
rollback deployment "app"

# Git
commit changes with message "Update"
push to branch "main"
create tag "v1.0.0"

# Files
copy "src" to "dest"
backup "file" as "backup-{now.date}"
remove "old-files"
```

## 🚀 Getting Started

1. **Read the basics**: Start with [01-hello-world.drun](01-hello-world.drun)
2. **Try parameters**: Move to [02-parameters.drun](02-parameters.drun)  
3. **Learn control flow**: Study [03-control-flow.drun](03-control-flow.drun)
4. **Pick your path**: Choose Docker, Kubernetes, or CI/CD examples
5. **Go advanced**: Explore smart detection and complex pipelines

## 📖 Documentation

- **[README.md](README.md)** - Comprehensive overview and guide
- **[../DRUN_V2_SPECIFICATION.md](../DRUN_V2_SPECIFICATION.md)** - Complete language specification
- **Language Reference** - Detailed syntax documentation (coming soon)

---

**Happy automating with drun v2!** 🎉
