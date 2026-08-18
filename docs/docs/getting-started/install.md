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

Run the PowerShell installer from Windows PowerShell or PowerShell 7:

```powershell
irm https://raw.githubusercontent.com/phillarmonic/drun/master/install.ps1 | iex
```

The installer detects AMD64 or ARM64, installs `xdrun.exe` to `%LOCALAPPDATA%\Programs\xdrun` (creating the directory if needed), and adds that directory to your user `PATH` so you can call `xdrun` from any terminal. Restart your terminal after installing so the updated `PATH` takes effect.

To install a specific release or choose a different directory, pass the `-Version` and `-InstallDir` parameters:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/phillarmonic/drun/master/install.ps1))) -Version v2.10.0 -InstallDir C:\Tools\xdrun
```

If you prefer a Unix-style shell, the Bash installer also works from Git Bash:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

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

## Update xdrun

On Linux and macOS, xdrun can replace itself:

```bash
xdrun --self-update
```

On Windows a running executable cannot overwrite itself, so `xdrun --self-update` automatically launches the PowerShell updater, which waits for xdrun to exit and then replaces the binary:

```powershell
xdrun --self-update
```

You can also run the PowerShell updater directly (for example, to update from a script or pin a version). It locates your existing `xdrun.exe`, compares it against the release, and replaces it in place:

```powershell
irm https://raw.githubusercontent.com/phillarmonic/drun/master/update.ps1 | iex
```

To update to a specific release, pass `-Version`:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/phillarmonic/drun/master/update.ps1))) -Version v2.10.0
```

Next, [enable shell autocomplete](autocomplete.md) or continue to [initialize your first spec](initialize.md).
