# repo-pick

English | [简体中文](README.md)

`repo-pick` is a TUI-only tool for downloading files and directories from remote Git repositories. It shallow-clones repositories into a local cache, lets you browse the repository tree in the terminal, and copies the selected file, directory, or entire repository into a local directory.

## Demo

[![repo-pick demo](https://img.youtube.com/vi/ClDb8ylknc0/hqdefault.jpg)](https://youtu.be/ClDb8ylknc0)

## Features

- Two-pane TUI: manage registries on the left and browse the repository tree on the right.
- Register multiple branches from the same remote repository; add/edit dialogs can fetch, search, and select remote branches.
- Reuse the local cache by default; manual updates delete the old cache and run a fresh shallow clone.
- Download files, directories, or the repository root, with confirmation before overwriting an existing target.
- Open cached files with `EDITOR`.
- Show progress for open, update, and download operations.

## Requirements

- `git`
- Go 1.25+ for development or source builds

## Installation

Homebrew:

```bash
brew tap fingergohappy/tap
brew install repo-pick
```

From source:

```bash
go install github.com/finger/repo-pick/cmd/repo-pick@latest
```

## Quick Start

Start:

```bash
repo-pick
```

First run:

```text
a       Add registry
Tab     Switch registry / repository tree focus; opens the current registry when no repository is open
l/Enter Open current registry, or expand a directory
j/k     Move cursor
i       Download current item to the startup directory
?       Show keybinding help
```

## Core Workflow

1. Press `a` in the left `Registry` pane to add a remote Git repository.
2. Enter name and URL, then move to Branch. You can use the remote default branch or search and select a specific branch.
3. Press `l`, `Enter`, or `Tab` to open the current registry. The first open shallow-clones it into the local cache; later opens reuse the cache.
4. Browse directories, expand directories, or search paths in the right `Repository Tree` pane.
5. Select a file, directory, or the root `/`, then press `i` to download it to the startup directory, or `I` to enter a target directory.
6. Select a file and press `e` to open the cached file with `EDITOR`.

In Repository Tree, `/` is the current root. Pressing `i` or `I` on the repository root `/` downloads the entire repository, using the registry name as the target directory name.

Pressing `e` runs the command in the `EDITOR` environment variable, for example `EDITOR=vim` or `EDITOR="code -w"`. If `EDITOR` is not set, repo-pick only shows a status message and does not start an external program.

repo-pick asks for confirmation before risky actions such as deleting a registry or overwriting an existing target. The bottom status bar shows context-aware keybindings based on the focused pane.

## Keybindings

Global:

```text
Tab / Shift+Tab  Switch registry / repository tree focus; opens the current registry when no repository is open
ctrl-w h         Switch to registry
ctrl-w l         Switch to repository tree; opens the current registry when no repository is open
/                Search paths in the current repository
Esc              Close search, confirmation, or error
?                Show/hide help
q / ctrl-c       Quit
```

Registry:

```text
j/k or ↑/↓       Move
l/Enter or →     Open current repository
a                Add registry; enter name/url and optionally choose a remote branch
e                Edit current registry; update name/url/branch
r                Reload registry list; only rereads config and does not update repository contents
d                Delete registry and its cache
u                Update current repository cache; delete old cache and download repository contents again
```

Add/Edit Registry dialog:

```text
Tab / Shift+Tab  Move focus between Name, URL, and Branch
↑/↓              Move in the branch list
Text input       Search branches
ctrl-u           Clear branch search
Enter            Confirm the current field or submit the registry
Esc              Cancel
```

Deleting a registry opens a confirmation dialog. Press `y` to confirm, or `n`/`Esc` to cancel.

Repository Tree:

```text
H                Switch back to registry
h or ←           Return to parent root
j/k or ↑/↓       Move
l/Enter or →     Expand or collapse selected directory
C                Collapse all expanded directories
o                Enter directory and make it the new root; files are located in their parent directory
e                Open current file with EDITOR
i                Download current item to the startup directory
I                Enter a target directory and download current item there
```

## Configuration

User configuration file:

```text
~/.config/repo-pick/config.yaml
```

Example:

```yaml
repositories:
  - name: official
    url: https://github.com/org/tools
  - name: personal
    url: git@github.com:finger/my-tools.git
    branch: main
```

Fields:

- `repositories[].name`: local registry name; must be unique.
- `repositories[].url`: Git repository URL; duplicates are allowed.
- `repositories[].branch`: optional Git branch; branches cannot be duplicated under the same URL. If empty or omitted, the remote default branch is used.
- `repositories[].last_updated_at`: last time the local cache was successfully created or refreshed; maintained automatically by the program.

## Cache Behavior

Repository cache path:

```text
~/.cache/repo-pick/repos/<url-or-url+branch-hash>/
```

`Ensure` behavior:

- Cache exists: read the local working tree directly without network access.
- Cache does not exist: run `git clone --progress --depth 1 --single-branch`; if `branch` is configured, also pass `--branch <branch>`.
- On first successful cache creation, update `last_updated_at` in the configuration.

`Update` behavior:

- Delete the old cache.
- Run a fresh shallow clone.
- On success, update `last_updated_at` in the configuration.
- If the new download fails, the old cache is not restored; the repository cannot be browsed in that run. Run update again or reopen the repository to retry.

When editing a registry, changing URL or branch deletes the cache for the old source. Renaming a registry does not delete the cache.

## Development

```bash
go mod download
go test ./...
```

Main directories:

```text
cmd/repo-pick/             # Program entrypoint; starts the TUI directly
internal/repopick/app/     # Application use case orchestration
internal/repopick/cache/   # Git repository cache lifecycle
internal/repopick/config/  # User configuration reading and writing
internal/repopick/install/ # File and directory copy
internal/repopick/registry/# Repository bookmark management
internal/repopick/tree/    # Cache working tree reading and search
internal/repopick/tui/     # Bubble Tea terminal interface
```
