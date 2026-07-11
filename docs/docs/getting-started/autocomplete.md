# Shell autocomplete

`xdrun` can generate autocomplete scripts for Bash, Zsh, Fish, and PowerShell. Completion includes tasks from the current drun spec, built-in `cmd:` commands, and CLI flags.

## Bash

Enable completion for the current session:

```bash
source <(xdrun cmd:completion bash)
```

For persistent completion, save the generated script and source it from your Bash profile:

```bash
xdrun cmd:completion bash > ~/.xdrun-completion.bash
echo 'source ~/.xdrun-completion.bash' >> ~/.bashrc
```

Start a new shell or run `source ~/.bashrc`. On macOS installations that use `.bash_profile`, add the source line there instead.

## Zsh

Create a user completion directory and generate the completion file:

```bash
mkdir -p ~/.zsh/completions
xdrun cmd:completion zsh > ~/.zsh/completions/_xdrun
```

Add the directory to `fpath` and enable completion in `~/.zshrc`:

```bash
fpath=(~/.zsh/completions $fpath)
autoload -U compinit
compinit
```

Start a new shell or run `source ~/.zshrc`.

## Fish

Enable completion for the current session:

```bash
xdrun cmd:completion fish | source
```

Install it for future sessions:

```bash
mkdir -p ~/.config/fish/completions
xdrun cmd:completion fish > ~/.config/fish/completions/xdrun.fish
```

## PowerShell

Enable completion for the current session:

```bash
xdrun cmd:completion powershell | Out-String | Invoke-Expression
```

To enable it whenever PowerShell starts, create the profile if necessary and add the command to it:

```bash
if (!(Test-Path $PROFILE)) { New-Item -ItemType File -Path $PROFILE -Force }
Add-Content -Path $PROFILE -Value 'xdrun cmd:completion powershell | Out-String | Invoke-Expression'
```

Start a new PowerShell session or run `. $PROFILE`.

## Try it

From a directory containing a drun spec, type `xdrun ` and press <kbd>Tab</kbd> to see available tasks. Type `xdrun cmd:` and press <kbd>Tab</kbd> to see all built-in commands.

Next, [initialize your first spec](initialize.md).
