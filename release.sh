#!/bin/bash
set -e
cd "$(dirname "$0")"

# ── Build (já lê VERSION e incrementa o patch) ───────────────────────────────
echo "Compilando..."
bash build.sh

# ── Lê versão pós-build ───────────────────────────────────────────────────────
VERSION=$(cat VERSION | tr -d '[:space:]')
TAG="v${VERSION}"

echo "Criando release ${TAG}..."

# ── Commit + tag ─────────────────────────────────────────────────────────────
git add VERSION
git commit -m "chore: release ${TAG}"
git tag "$TAG"
git push && git push --tags

# ── Release notes ────────────────────────────────────────────────────────────
PREV_TAG=$(git tag --sort=-version:refname | grep -v "^${TAG}$" | head -1)
if [ -z "$PREV_TAG" ]; then
  LOG=$(git log --pretty=format:"- %s" | grep -v "Co-Authored-By" | grep -v "^$")
else
  LOG=$(git log "${PREV_TAG}..${TAG}" --pretty=format:"- %s" | grep -v "Co-Authored-By" | grep -v "^$")
fi

NOTES="## O que mudou

${LOG}

---
**Binários disponíveis:**

| Arquivo | Plataforma |
|---|---|
| \`dashboard-atestados-darwin-arm64\` | macOS Apple Silicon (M1/M2/M3) |
| \`dashboard-atestados-darwin-amd64\` | macOS Intel |
| \`dashboard-atestados-windows-amd64.exe\` | Windows 64-bit |

> Coloque o executável na mesma pasta onde ficará \`Atestados/\` e execute.
> O navegador abre automaticamente em http://localhost:8787"

# ── Publica release ──────────────────────────────────────────────────────────
gh release create "$TAG" \
  --title "${TAG} — Dashboard de Atestados Médicos" \
  --notes "$NOTES" \
  dist/dashboard-atestados-darwin-arm64 \
  dist/dashboard-atestados-darwin-amd64 \
  dist/dashboard-atestados-windows-amd64.exe

echo ""
echo "Release ${TAG} publicado!"
echo "https://github.com/c3t4r4/DashAtestados/releases/tag/${TAG}"
