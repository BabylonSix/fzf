# FZF Live Theming Experiments Log

## Problem
fzf has no mechanism for live color switching while running. Issue #3861 is open upstream with no resolution through v0.68.0. Need SIGUSR1-based live color reload to integrate with BRAVO themer ecosystem.

## Root Cause Hypothesis
fzf reads `FZF_DEFAULT_OPTS` at startup and never re-evaluates colors. Global `Col*` color variables are set once by `initPalette()` during `NewTerminal()`. No signal handler or action exists to reload them.

---

## Attempts

### exp-(change-color-action)-1
**Branch:** `fzf-ftr-(fzf-live-theming)-exp-(change-color-action)-1`
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

**Result:**
- fzf chrome (bg, borders, text, prompt, info, etc.) switches correctly
- bat preview does NOT update colors on live switch
- `reqPreviewRefresh` only re-renders cached ANSI output — does NOT re-execute the preview command
- bat reads `--theme` from `~/.config/bat/config` (sed'd by themer), but since bat is never re-invoked, the new theme is never picked up

**Status:** partial success — fzf chrome works, bat preview broken

**Key learnings:**
1. `os.Getenv()` reads process env copy from launch — useless for live updates. Use file-based config.
2. `t.theme = theme` (pointer reassign) breaks renderer — must copy in-place with `*t.theme = *theme`
3. `reqPreviewRefresh` → `printPreview()` only re-renders cached lines. `reqPreviewEnqueue` → `refreshPreview()` re-executes the command.

---

## Next Ideas
- Use `refreshPreview()` instead of `reqPreviewRefresh` in SIGUSR1 handler to force bat re-execution
- Or trigger `reqPreviewEnqueue` directly with fresh environment
