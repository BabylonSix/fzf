# FZF Color Internals Research

## Key Finding: This is very doable

Colors are stored as **global variables** (`ColPrompt`, `ColNormal`, `ColBorder`, etc.) read on every render cycle — not cached. Swapping them via `initPalette()` and triggering a full redraw will work.

---

## 1. Color Storage

**`ColorTheme` struct** — `src/tui/tui.go:474-523`
- ~50 named `ColorAttr` fields (Fg, Bg, Match, Prompt, Border, PreviewFg, etc.)
- Each `ColorAttr` = `Color` (int32) + `Attr` (bitmask: bold, dim, italic, etc.)

**Global `Col*` variables** — `src/tui/tui.go:858-900`
- `ColPrompt`, `ColNormal`, `ColInput`, `ColMatch`, `ColBorder`, `ColPreview`, etc.
- Set by `initPalette(theme)` at startup
- **Read directly during rendering** — not cached per frame

**`Terminal.theme`** — `src/terminal.go:426`
- Stores pointer to `ColorTheme`, set at construction

## 2. Color Parsing

**`parseTheme`** — `src/options.go:1367-1582`
- Accepts comma-separated specs: `dark,prompt:red,bg:#1e1e2e`
- Base scheme names: `dark`, `light`, `base16`/`16`, `bw`/`no`
- Component-color pairs: 50 component names × (named colors, 0-255, #rrggbb, attributes)
- **Multiple `--color` flags merge incrementally** — partial specs overlay existing theme

**`InitTheme`** — `src/tui/tui.go:1161-1303`
- Merges user overrides with base theme
- Derives dependent colors (ListFg from Fg, SelectedBg from ListBg, etc.)
- Calls `initPalette(theme)` to populate global `Col*` variables

## 3. `--listen` Server & Action Dispatch

**Server** — `src/server.go` (278 lines)
- Custom HTTP server (no net/http, keeps binary small)
- Started at `terminal.go:1290`

**Dispatch flow:**
1. HTTP POST → `handleHttpRequest` (server.go:154)
2. Body parsed via `parseSingleActionList` (options.go:1692) — same parser as `--bind`
3. Actions sent to `server.actionChannel`
4. Main event loop picks them up (`terminal.go:5948`)
5. Processed by `doAction` switch — same as keyboard actions

## 4. Existing `change-*` Action Pattern

All follow the same structure via `capture` closure (terminal.go:6070-6085):

```go
case actChangePrompt:
    t.promptString = a.a
    t.prompt, t.promptLen = t.parsePrompt(a.a)
    req(reqPrompt)
```

Pattern: **update state → request redraw**

Each `change-*` has three variants: `change-X`, `transform-X`, `bg-transform-X`

## 5. Signal Handling

**Current signals:**
- `SIGWINCH` → resize + full redraw (terminal.go:5432-5444)
- `SIGINT/SIGTERM/SIGHUP` → quit (terminal.go:5418-5430)
- **SIGUSR1/SIGUSR2 — completely free**

SIGWINCH pattern (channel + goroutine) is exactly what we'd replicate for SIGUSR1.

## 6. Implementation Plan

### Files to modify:

1. **`src/terminal.go`** — Add `actChangeColor` to enum (~line 557). Add handler in `doAction` switch (~line 6087). Add `baseTheme` field to Terminal struct.

2. **`src/options.go`** — Add `"change-color"` to action name parser (~line 2063).

3. **`src/terminal_unix.go`** — Add SIGUSR1 listener (like SIGWINCH).

4. **Run `go generate`** — updates `actiontype_string.go`

### `change-color` handler logic:

```go
case actChangeColor, actTransformColor, actBgTransformColor:
    capture(false, func(colorSpec string) {
        baseTheme, newTheme, err := parseTheme(t.theme, colorSpec)
        if err == nil {
            t.theme = newTheme
            if baseTheme != nil {
                t.baseTheme = baseTheme
            }
            tui.InitTheme(t.theme, t.baseTheme, t.bold, t.black, t.inputBorder != nil, t.headerBorder != nil)
            req(reqFullRedraw)
        }
    })
```

### SIGUSR1 handler:

```go
usr1Chan := make(chan os.Signal, 1)
signal.Notify(usr1Chan, syscall.SIGUSR1)
go func() {
    for {
        select {
        case <-ctx.Done():
            return
        case <-usr1Chan:
            // Read color spec from env/file, post change-color action
            t.reqBox.Set(reqFullRedraw, nil)
        }
    }
}()
```

### Risks:
- **Global variable mutation** — safe because render loop holds mutex locks
- **Base theme tracking** — need to store `baseTheme` in Terminal struct (currently discarded after init)
- **Thread safety** — signal handler posts to channel (existing pattern), no direct state mutation
- **Partial specs** — `parseTheme` already handles this correctly

### Integration with themer:

**Option A — `--listen` (cleanest):**
```bash
curl -XPOST localhost:$FZF_PORT -d 'change-color(dark,bg:#1e1e2e,...)'
```

**Option B — SIGUSR1 (consistent with ecosystem):**
```bash
pkill -USR1 fzf  # reads spec from FZF_COLOR env or ~/.config/fzf/colors
```

**Option C — Both** (change-color action + SIGUSR1 as thin wrapper)
