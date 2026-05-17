# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally (dev)
go run .

# Full cross-platform build (darwin arm64/amd64 + windows amd64 + macOS .app bundles)
bash build.sh

# Compile check only
go build ./...

# Regenerate Windows .exe icon resource (must run before build.sh if icon changed)
go-winres make --arch amd64
# Install go-winres if missing: go install github.com/tc-hib/go-winres@latest

# Regenerate macOS icon assets from master SVG (requires rsvg-convert + magick + iconutil)
cd assets
for size in 16 32 48 64 128 256 512 1024; do rsvg-convert -w $size -h $size icon.svg -o icon_${size}.png; done
magick icon_16.png icon_32.png icon_48.png icon_256.png icon.ico
# (then rebuild iconset and run iconutil as in build.sh)
```

## Architecture

Single-binary Go HTTP server (`main.go`) that embeds the entire frontend via `//go:embed static`. No external runtime dependencies — the final binary is fully self-contained.

### Data flow

1. **Startup**: `ensureAtestadosDir()` locates or creates `Atestados/` (checks exe dir → .app bundle parent → cwd, in that order). If no `.xlsx` files exist, `createTemplate()` auto-generates `{nextYear}.xlsx` with 12 month sheets.
2. **Load**: `loadAtestados()` walks the directory, parses every `.xlsx` sheet, skipping headers and template-only rows. Dates are parsed from multiple formats including Excel serial numbers.
3. **In-memory state**: All records live in `dadosGlobal []Atestado` protected by `sync.RWMutex`. `POST /api/reload` re-reads the directory without restarting.
4. **Frontend**: Single-file SPA at `static/index.html` — vanilla JS, pure SVG charts (no CDN), CSS variables for dark/light theming.

### API endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/resumo` | KPIs, chart data, filter options, `arquivosXlsx` count |
| GET | `/api/dados` | Filtered records; `?format=csv` triggers download |
| GET | `/api/overlaps` | Duplicate/overlapping certificate detection |
| POST | `/api/reload` | Re-reads `Atestados/` into memory |
| POST | `/api/criar-template` | Creates `{maxExistingYear+1}.xlsx` in `Atestados/` |

### xlsx data contract

Each `.xlsx` in `Atestados/` must have sheets with rows in this column order:
`Nome | Cargo | Setor | Data | CID | Dias Afastamento`

Row 0 is always skipped as header. Rows where `Nome` is empty or literally `"Nome"` are skipped.

### Overlap detection

Three types detected in `detectOverlaps()`, grouped by employee name:
- `duplicado_exato` — same start date + same CID
- `mesmo_dia_cid_diferente` — same start date + different CID
- `periodo_sobreposto` — date ranges overlap (uses DataFim = Data + Dias − 1)

### Build outputs (`dist/`)

| File | Platform |
|---|---|
| `dashboard-atestados-darwin-arm64` | macOS Apple Silicon binary |
| `dashboard-atestados-darwin-amd64` | macOS Intel binary |
| `dashboard-atestados-windows-amd64.exe` | Windows (icon embedded via `rsrc_windows_amd64.syso`) |
| `Dashboard Atestados-arm64.app` | macOS app bundle (arm64, includes icon.icns) |
| `Dashboard Atestados-amd64.app` | macOS app bundle (amd64, includes icon.icns) |

The `.app` bundles use a `launcher` shell script as `CFBundleExecutable` to set the working directory to the bundle's parent folder before starting the Go binary, ensuring `Atestados/` is found/created in the right place when double-clicked.

### Icon assets (`assets/`)

`icon.svg` is the master. From it: `icon.icns` (macOS), `icon.ico` (Windows, embedded via `winres/winres.json`), `favicon-32.png` (served from `static/`, referenced in HTML as fallback). The SVG favicon is also inlined as a base64 `data:` URI in `<head>` for zero-request loading.

### Frontend theming

Dark theme is default (`data-theme="dark"` on `<html>`). `localStorage` persists the user's choice. SVG chart colors re-render on theme toggle via `renderCharts()`. Use `tc(lightVal, darkVal)` helper for theme-aware SVG text/element colors.
