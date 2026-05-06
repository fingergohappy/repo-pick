# repo-pick

English | [简体中文](README.md)

`repo-pick` is a TUI-only tool for downloading files and directories from remote Git repositories.

After startup, it opens a terminal interface: the left pane manages repository bookmarks, and the right pane browses the repository tree. Repositories are shallow-cloned into a local cache. Selected files or directories can then be downloaded to the startup directory or a custom target directory.

## Features

- Manage remote Git repository bookmarks from a terminal interface.
- Save different branches for the same repository URL and switch between them.
- Browse repository contents from a local shallow-clone cache to avoid repeated downloads.
- Browse directory trees, search paths, and manually refresh repository contents.
- Show Git clone progress when opening or refreshing a repository.
- Download a single file or an entire directory with local copy progress.
- Prompt to overwrite or cancel when the target path already exists.

## Installation

Install with Homebrew:

```bash
brew tap fingergohappy/tap
brew install repo-pick
```

To publish a new version, push a `vX.Y.Z` tag. GitHub Actions will build release binaries and update the formula in `fingergohappy/homebrew-tap`.

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

## Cache

Repository cache path:

```text
~/.cache/repo-pick/repos/<url-or-url+branch-hash>/
```

`Ensure` behavior:

- Cache exists: read the local working tree directly without network access.
- Cache does not exist: run `git clone --depth 1 --single-branch`; if `branch` is configured, also pass `--branch <branch>`.

`Update` behavior:

- Delete the old cache.
- Run a fresh shallow clone.
- If the new download fails, the old cache is not restored; the repository cannot be browsed in that run.

## Keybindings

Global:

```text
ctrl-w h Switch to registry
ctrl-w l Switch to repository tree; opens the current registry when no repository is open
/       Search paths in the current repository
Esc     Close search, confirmation, or error
?       Show/hide help
q       Quit
```

Registry:

```text
j/k     Move
l       Open current repository
a       Add registry; enter name/url in the dialog and optionally choose a remote branch
e       Edit current registry; update name/url/branch in the dialog
r       Reload registry list; only rereads config and does not update repository contents
d       Delete registry and its cache
u       Update current repository cache; delete old cache and download repository contents again
```

Repository Tree:

```text
h       Return to parent root
j/k     Move
l       Expand or collapse selected directory
o       Enter directory and make it the new root; files are located in their parent directory
i       Download current item to the startup directory
I       Enter a target directory and download current item there
```

## Project Structure

```text
cmd/repo-pick/             # Program entrypoint; starts the TUI directly
internal/repopick/app/     # Application use case orchestration
internal/repopick/cache/   # Git repository cache lifecycle
internal/repopick/config/  # User configuration reading and writing
internal/repopick/install/ # File and directory copy
internal/repopick/registry/# Repository bookmark management
internal/repopick/tree/    # Cache working tree reading and search
internal/repopick/tui/     # Bubble Tea terminal interface
configs/                   # Configuration examples
docs/                      # Design and task documents
test/testdata/             # Test data
```

## Development

```bash
go mod download
go test ./...
```
