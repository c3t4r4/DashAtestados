#!/bin/bash
set -e
cd "$(dirname "$0")"

BINARY="dashboard-atestados"
DIST="dist"
ASSETS="assets"
VERSION="1.0.0"
LDFLAGS="-s -w -X main.version=${VERSION}"

mkdir -p "$DIST"

echo "Running go mod tidy..."
go mod tidy

# ── Windows resource (.syso embeds icon into .exe) ─────────────────────────────
if command -v go-winres &>/dev/null; then
  echo "Generating Windows resource (icon)..."
  go-winres make --arch amd64 2>/dev/null || true
else
  echo "Warning: go-winres not found — Windows exe will have no icon."
  echo "  Install: go install github.com/tc-hib/go-winres@latest"
fi

# ── Compile ───────────────────────────────────────────────────────────────────
echo "Building darwin/arm64..."
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o "$DIST/$BINARY-darwin-arm64" .

echo "Building darwin/amd64..."
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o "$DIST/$BINARY-darwin-amd64" .

echo "Building windows/amd64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o "$DIST/$BINARY-windows-amd64.exe" .

# ── macOS .app bundles ────────────────────────────────────────────────────────
make_app() {
  local arch="$1"
  local appdir="$DIST/Dashboard Atestados-${arch}.app"
  rm -rf "$appdir"
  mkdir -p "$appdir/Contents/MacOS"
  mkdir -p "$appdir/Contents/Resources"

  cp "$DIST/$BINARY-darwin-${arch}" "$appdir/Contents/MacOS/$BINARY"
  chmod +x "$appdir/Contents/MacOS/$BINARY"

  # Launcher sets cwd to the .app's parent folder so Atestados/ is found next to it
  cat > "$appdir/Contents/MacOS/launcher" <<'LAUNCHER'
#!/bin/bash
cd "$(dirname "$0")/../../.."
exec "$(dirname "$0")/dashboard-atestados"
LAUNCHER
  chmod +x "$appdir/Contents/MacOS/launcher"

  [ -f "$ASSETS/icon.icns" ] && cp "$ASSETS/icon.icns" "$appdir/Contents/Resources/icon.icns"

  cat > "$appdir/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>    <string>launcher</string>
  <key>CFBundleIconFile</key>      <string>icon</string>
  <key>CFBundleIdentifier</key>    <string>com.glauco.dashboard-atestados</string>
  <key>CFBundleName</key>          <string>Dashboard Atestados</string>
  <key>CFBundleDisplayName</key>   <string>Dashboard Atestados</string>
  <key>CFBundlePackageType</key>   <string>APPL</string>
  <key>CFBundleShortVersionString</key><string>1.0</string>
  <key>CFBundleVersion</key>       <string>1</string>
  <key>LSMinimumSystemVersion</key><string>10.13</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

  echo "  Created: $appdir"
}

echo "Creating macOS .app bundles..."
make_app "arm64"
make_app "amd64"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "Build complete. Output:"
ls -lh "$DIST"
