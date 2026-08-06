# Install xdrun

`xdrun` is the CLI that finds and executes drun specs. Choose the instructions for your operating system, then verify the installation with `xdrun --version`.

## Linux

Run the installer from a terminal:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

The installer detects AMD64 or ARM64, installs `xdrun` to `~/.local/bin`, and attempts to add that directory to your `PATH`.

## macOS

Run the installer from Terminal:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

The installer supports both Apple silicon (ARM64) and Intel (AMD64) Macs. It installs `xdrun` to `~/.local/bin` and removes the macOS quarantine attribute from the downloaded binary.

## Windows

Run the installer from Git Bash:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

The installer supports Windows on AMD64 and ARM64 and installs `xdrun.exe` to `~/bin` by default. Make sure that directory is on your Windows `PATH`.

## Choose an install directory

Pass a directory to install the latest release somewhere other than the platform default:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s -- /usr/local/bin
```

The argument takes precedence over the `INSTALL_DIR` environment variable. To install a specific release into that directory, pass the release tag first:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s -- v2.10.0 /usr/local/bin
```

## Install with Go

If Go is already installed, the same command works on Linux, macOS, and Windows:

```bash
go install github.com/phillarmonic/drun/v2/cmd/xdrun@latest
```

Make sure the Go binary directory—usually `~/go/bin` or `%USERPROFILE%\go\bin`—is on your `PATH`.

## Install a specific version

Pass a release tag to the installer:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s -- v2.10.0
```

Or pin the Go installation:

```bash
go install github.com/phillarmonic/drun/v2/cmd/xdrun@v2.17.0
```

## Verify the installation

```bash
xdrun --version
```

Next, [enable shell autocomplete](autocomplete.md) or continue to [initialize your first spec](initialize.md).
