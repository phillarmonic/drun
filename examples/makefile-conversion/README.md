# Makefile to drun Conversion

This folder contains examples for converting Makefiles to drun format.

## Convert Command

The `xdrun cmd:from makefile` command converts Makefiles into equivalent drun v2 task files.

The `cmd:` prefix is reserved for built-in commands to avoid conflicts with user-defined tasks.

### Usage

```bash
xdrun cmd:from makefile [flags]

Flags:
  -i, --input <file>    Path to input Makefile (default: Makefile)
  -o, --output <file>   Path to output .drun file (default: <input>.drun)
  -h, --help            Help for convert command
```

### Examples

Convert the default Makefile in current directory:
```bash
xdrun cmd:from makefile
```

Convert a specific Makefile:
```bash
xdrun cmd:from makefile -i myproject.mk -o myproject.drun
```

Convert with explicit paths:
```bash
xdrun cmd:from makefile --input Makefile --output tasks.drun
```

Convert the example Makefile:
```bash
xdrun cmd:from makefile -i examples/makefile-conversion/example.Makefile \
                        -o examples/makefile-conversion/converted.drun
```

## What Gets Converted

The converter handles:

- ✅ **Targets** → drun tasks
- ✅ **Dependencies** → `depends on` declarations
- ✅ **Variables** → drun variables with interpolation
- ✅ **Comments** → task descriptions
- ✅ **Shell commands** → `run`, `echo`, `create dir`, etc.
- ✅ **.PHONY targets** → properly marked
- ✅ **@ prefix** (silent) → preserved as appropriate drun actions
- ✅ **- prefix** (ignore errors) → wrapped in `try/ignore` blocks

## Example Conversion

### Input Makefile

```makefile
# Build the application
PROJECT_NAME = myapp
VERSION = 1.0.0

.PHONY: build

build: install
	@echo "Building $(PROJECT_NAME) version $(VERSION)..."
	mkdir -p build
	go build -o build/$(PROJECT_NAME)
```

### Output drun

```drun
version: 2.0

# Variables from Makefile (will be set in tasks):
# - $project_name = "myapp"
# - $version = "1.0.0"

task "build" means "Build the application":
	depends on "install"

	# Set variables from Makefile (included in tasks that use them)
	set $project_name to "myapp"
	set $version to "1.0.0"

	info "Running build"

	echo "Building {$project_name} version {$version}..."
	create dir "build"
	run "go build -o build/{$project_name}"

	success "build completed successfully!"
```

## Files in This Directory

- `example.Makefile` - Example Makefile demonstrating common patterns
- `example-converted.drun` - The converted drun equivalent
- `README.md` - This file

## Limitations

Some Makefile features cannot be directly converted:

- Complex shell scripting (use `run` command with full shell script)
- Automatic variables like `$@`, `$<` (need manual conversion)
- Pattern rules and wildcards (convert to explicit tasks)
- Conditional directives (`ifdef`, `ifndef`) (use drun conditionals)
- Functions like `$(shell ...)` (use drun's `capture from shell`)

For complex Makefiles, the converter provides a good starting point that you can then refine manually.

## Tips

1. **Review the output** - Always check the converted file and adjust as needed
2. **Test incrementally** - Test each converted task to ensure it works correctly
3. **Use drun features** - Take advantage of drun's semantic actions for cleaner code
4. **Preserve comments** - Add comments above targets in your Makefile for better descriptions
5. **Simplify variables** - Consider using drun's built-in functions instead of shell substitutions

## Contributing

If you find conversion patterns that could be improved, please contribute to the converter in:
- `internal/make2drun/parser.go` - Makefile parsing logic
- `internal/make2drun/generator.go` - drun generation logic
- `cmd/drun/app/convert.go` - CLI command implementation

