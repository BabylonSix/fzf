# FZF Live Theming Experiments Log

## Problem
fzf has no mechanism for live color switching while running. Issue #3861 is open upstream with no resolution through v0.68.0. Need SIGUSR1-based live color reload to integrate with BRAVO themer ecosystem.

## Root Cause Hypothesis
fzf reads `FZF_DEFAULT_OPTS` at startup and never re-evaluates colors. Global `Col*` color variables are set once by `initPalette()` during `NewTerminal()`. No signal handler or action exists to reload them.

---

## Attempts

### exp-(change-color-action)-1
**Branch:** `fzf-ftr-(fzf-live-theming)-exp-(change-color-action)-1` (deleted)
**Commits:** 6d093834, 9772d819, 2f2f420d
**Files changed:** `src/terminal.go`, `src/terminal_unix.go`, `src/terminal_windows.go`
**Files created outside repo:** `~/.config/fzf/colors` (color spec file read by handler)
**Approach:**
- Added SIGUSR1 signal handler (channel + goroutine, same pattern as SIGWINCH)
- Handler reads color specs from `~/.config/fzf/colors` file
- Parses specs via existing `parseTheme()`, rebuilds theme
- Copies theme in-place (`*t.theme = *theme`) so renderer pointer stays valid
- Calls `InitTheme()` to repopulate global `Col*` variables
- Triggers `reqFullRedraw` + `reqPreviewRefresh`

**Result:** Partial — fzf chrome works, bat preview broken. `reqPreviewRefresh` only re-renders cached ANSI, doesn't re-execute bat.

**Status:** superseded by exp-2

### exp-(change-color-action)-2
**Branch:** `fzf-ftr-(fzf-live-theming)-exp-(change-color-action)-2` (deleted)
**Commits:** b4f6f58a
**Files changed:** `src/terminal.go`, `src/actiontype_string.go`
**Approach:**
- Added `reqPreviewRerun` request type
- SIGUSR1 handler triggers `reqPreviewRerun` instead of `reqPreviewRefresh`
- Main event loop handles `reqPreviewRerun` by calling `refreshPreview()` — actually re-executes the preview command (bat)
- bat picks up new `--theme` from `~/.config/bat/config` (sed'd by themer)

**Result:** Full success — fzf chrome AND bat preview both live-switch on SIGUSR1.

**Status:** promoted to `stbl-(sigusr1-live-colors)-1`

**Key learnings:**
1. `os.Getenv()` reads process env copy from launch — useless for live updates. Use file-based config.
2. `t.theme = theme` (pointer reassign) breaks renderer — must copy in-place with `*t.theme = *theme`
3. `reqPreviewRefresh` → `printPreview()` only re-renders cached lines. Need `reqPreviewRerun` → `refreshPreview()` to re-execute the command.

---

## Stable Branches

### stbl-(sigusr1-live-colors)-1 (merged to master)
**What's guaranteed:** SIGUSR1 triggers full live color reload (fzf chrome + bat preview re-execution)
**Integration:** Themer writes `~/.config/fzf/colors` + seds `~/.config/bat/config` + `pkill -USR1 fzf`

---

# FZF Mouse Selection Feature

## Problem
fzf has basic mouse support (click moves cursor, shift+click toggles one item) but no Ctrl+click toggle or Shift+click range selection. Need Finder-style mouse interaction matching the yazi model.

## Target Behavior
- Plain click: move cursor, set anchor
- Ctrl+click: toggle item, move cursor, set anchor
- Shift+click: select range from anchor to click, move cursor, set anchor
- Right-click: toggle (existing)

## Attempts
