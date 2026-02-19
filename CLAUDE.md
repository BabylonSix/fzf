# CLAUDE.md

Fork of junegunn/fzf with SIGUSR1 live color reloading.

## Playbook
- /Users/bravo/.dotfiles/zsh/GIT-FORMAT.md — branch naming format
- /Users/bravo/.dotfiles/zsh/AGENT-WORKFLOW.md — experiment/stable workflow, branch lifecycle, experiments log

## Build & Test
- `go build -o fzf` (build binary in repo root)
- `PATH="$HOME/go/bin:$PATH" go generate ./src/` (regenerate enum stringers after adding request types)
- `fzftest` (shell function — runs `~/ai-projects/fzf/fzf`, defined in test-tools.sh)

## Deploy (custom build → daily driver)
- `go build -o fzf`
- `cp fzf ~/.dotfiles/bin/fzf`
- Custom binary at `~/.dotfiles/bin/fzf` takes priority via PATH ordering (env/path.sh)

## Key Paths
- Config: `~/.config/fzf/` (colors file written by themer)
- Themer integration: `~/.dotfiles/config/themer/themer.sh` (`_themer_set_fzf`, `_themer_set_bat`)
- Shell wrappers: `test-tools.sh` (`fzftest` function)
- Colors file: `~/.config/fzf/colors` (read by SIGUSR1 handler)

## Architecture
- Colors are global `Col*` variables in `src/tui/tui.go`, set by `initPalette()`, read every render cycle
- `Terminal.theme` is a shared `*ColorTheme` pointer — renderer caches from it at window creation
- SIGUSR1 handler in `src/terminal.go` → `reloadColors()` → reads colors file → copies theme in-place → full redraw + preview re-execution

## Project-Specific Gotchas
- Must copy theme in-place (`*t.theme = *theme`), NOT reassign pointer — renderer holds old pointer
- `reqPreviewRefresh` only re-renders cached ANSI — use `reqPreviewRerun` to re-execute preview command
- `os.Getenv()` reads process env from launch time — useless for live updates, use file I/O
- `stringer` must be in PATH for `go generate` — install with `go install golang.org/x/tools/cmd/stringer@latest`
- After adding new `req*` or `act*` constants, must run `go generate ./src/` before building
