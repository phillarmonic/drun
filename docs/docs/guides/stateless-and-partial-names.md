# Stateless Drun and Partial Task Name Matching

This document describes two new features added to drun: stateless configuration and partial task name matching.

## Stateless Drun

### Overview

Stateless drun allows you to mark directories where drun configurations should be stored in your home directory instead of in the repository itself. This is useful for:

- Repositories where you can't commit drun configs (e.g., third-party code)
- Personal automation scripts for public repositories
- Development workflows that shouldn't be version controlled

### How It Works

When a directory is marked as stateless:
1. drun stores the configuration in `~/.drun/stateless/<hash>/spec.drun`
2. The hash is generated from the absolute path of the directory
3. Multiple developers can have different configs for the same repository
4. The stateless configuration file is tracked in `~/.drun/stateless.yml`

### Commands

#### Mark a directory as stateless

```bash
# Mark current directory
xdrun cmd:stateless add

# Mark specific directory
xdrun cmd:stateless add /path/to/dir

# Mark and create a template config
xdrun cmd:stateless add --create
```

#### Remove stateless marking

```bash
# Remove marking (keeps config file)
xdrun cmd:stateless remove

# Remove and delete config file
xdrun cmd:stateless remove --delete
```

#### List all stateless directories

```bash
xdrun cmd:stateless list
```

Output:
```text
Stateless directories:
  ✓ /tmp/drun-test-stateless
    → /home/user/.drun/stateless/909e63477a73559a/spec.drun
```

#### Check current directory status

```bash
xdrun cmd:stateless info
```

Output for stateless directory:
```text
 Current directory is marked as STATELESS
   Config location: /home/user/.drun/stateless/909e63477a73559a/spec.drun
   Status:  Config file exists
```

Output for normal directory:
```text
 Current directory is NOT marked as stateless
   Using local configuration (.drun/spec.drun)
   Run 'xdrun cmd:stateless add' to mark as stateless
```

### Example Workflow

```bash
# Clone a third-party repository
git clone https://github.com/example/repo.git
cd repo

# Mark as stateless (config won't go into the repo)
xdrun cmd:stateless add --create

# Edit the config file
vim ~/.drun/stateless/<hash>/spec.drun

# Use drun normally
xdrun build
xdrun test
```

## Partial Task Name Matching

### Overview

Partial task name matching allows you to run tasks using just the first few characters of their name, as long as the partial name uniquely identifies a single task.

### How It Works

When you provide a partial task name:
1. drun first checks for an exact match
2. If no exact match, it finds all tasks starting with the partial name
3. If exactly one match is found, that task is executed
4. If multiple matches are found, an error is shown with disambiguation suggestions
5. If no matches are found, fuzzy matching suggests similar task names

### Examples

Given these tasks: `build`, `beta`, `benchmark`, `test`, `deploy`

#### Unique Match

```bash
# 'bu' matches only 'build'
xdrun bu
```

Output:
```text
  Building the project
Building project...
```

With verbose mode:
```bash
xdrun -v bet
```

Output:
```text
 Resolved 'bet' → 'beta'
  Running beta task
...
```

#### Ambiguous Match

```bash
# 'b' matches multiple: build, beta, benchmark
xdrun b
```

Output:
```text
Error: ambiguous task name 'b' - matches multiple tasks:
  - benchmark (use: xdrun ben)
  - beta (use: xdrun bet)
  - build (use: xdrun bu)

Please use more characters to disambiguate

Run 'xdrun --list' to see all available tasks
```

#### No Match with Suggestions

```bash
# Typo: 'tst' instead of 'test'
xdrun tst
```

Output:
```text
Error: task 'tst' not found

Did you mean one of these?
  - test

Run 'xdrun --list' to see all available tasks
```

#### Full Name Still Works

```bash
# Full task names continue to work exactly as before
xdrun build
```

### Disambiguation Guide

The error message shows the shortest unique prefix for each matching task:

```text
  - benchmark (use: xdrun ben)
  - beta (use: xdrun bet)
  - build (use: xdrun bu)
```

This tells you:
- Use `ben` or longer to match `benchmark`
- Use `bet` or longer to match `beta`
- Use `bu` or longer to match `build`

### Fuzzy Matching

When no tasks start with the partial name, drun uses fuzzy matching to suggest similar tasks based on:
- Substring matching (contains the partial name)
- Common prefixes (first 2-3 characters)
- Levenshtein distance (edit distance ≤ 2)

This helps catch typos and find tasks you might be looking for.

## Implementation Details

### File Structure

New files added:
- `cmd/drun/app/stateless.go` - Stateless configuration management
- `cmd/drun/app/task_matcher.go` - Partial task name matching and fuzzy search

Modified files:
- `cmd/drun/app/config.go` - Updated `FindConfigFile` to check stateless directories
- `cmd/drun/app/runner.go` - Updated `ExecuteTask` to resolve partial names
- `cmd/drun/app/cli.go` - Added `cmd:stateless` subcommands

### Configuration Storage

Stateless configuration is stored at:
- Config file: `~/.drun/stateless.yml`
- Format:
  ```yaml
  directories:
    /absolute/path/to/dir: /home/user/.drun/stateless/<hash>/spec.drun
  ```

Task configuration files are stored at:
- `~/.drun/stateless/<hash>/spec.drun`
- Hash: First 16 characters of SHA-256 hash of absolute directory path

### Algorithm Complexity

Partial task name matching:
- Exact match check: O(n) where n = number of tasks
- Prefix matching: O(n)
- Fuzzy matching (when needed): O(n*m) where m = average task name length
- Total: O(n*m) worst case, typically very fast for typical task counts

## Benefits

### Stateless Drun
-  Use drun with third-party repositories
-  Keep personal automation private
-  No need to fork repositories just to add drun configs
-  Different team members can have different workflows

### Partial Task Names
-  Faster task execution with fewer keystrokes
-  Better developer experience
-  Smart error messages guide you to the right task
-  Fuzzy matching helps with typos
-  Backwards compatible - full names still work

## Future Enhancements

Potential improvements for future versions:

1. **Stateless Drun**
   - Sync stateless configs across machines
   - Import/export stateless configurations
   - Template management for common setups

2. **Partial Task Names**
   - Tab completion integration
   - Learning from usage patterns (most used tasks)
   - Context-aware suggestions based on git branch or directory

## Testing

Both features have been tested with:

### Stateless Drun Tests
-  Adding directories as stateless
-  Creating template configurations
-  Listing stateless directories
-  Removing stateless marking
-  Deleting configurations
-  Info command shows correct status
-  Task execution from stateless config

### Partial Name Tests
-  Exact match resolution
-  Unique partial match resolution
-  Ambiguous match error with suggestions
-  No match with fuzzy suggestions
-  Full task names still work
-  Verbose mode shows resolution
