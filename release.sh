#!/bin/bash
set -e
cd "$(dirname "$0")"

# ── Lê versão do build.sh ────────────────────────────────────────────────────
VERSION=$(grep '^VERSION=' build.sh | cut -d'"' -f2)
if [ -z "$VERSION" ]; then
  echo "Erro: não foi possível ler VERSION de build.sh"
  exit 1
fi
TAG="v${VERSION}"

echo "Criando release $TAG..."

# ── Verifica que a tag existe ────────────────────────────────────────────────
if ! git tag | grep -q "^${TAG}$"; then
  echo "Erro: tag $TAG não existe localmente. Crie com: git tag $TAG && git push --tags"
  exit 1
fi

# ── Gera release notes a partir dos commits ──────────────────────────────────
PREV_TAG=$(git tag --sort=-version:refname | grep -v "^${TAG}$" | head -1)

if [ -z "$PREV_TAG" ]; then
  LOG=$(git log --pretty=format:"- %s" | grep -v "Co-Authored-By" | grep -v "^$")
else
  LOG=$(git log "${PREV_TAG}..HEAD" --pretty=format:"- %s" | grep -v "Co-Authored-By" | grep -v "^$")
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

# ── Cria o release e faz upload dos binários ─────────────────────────────────
gh release create "$TAG" \
  --title "$TAG — Dashboard de Atestados Médicos" \
  --notes "$NOTES" \
  dist/dashboard-atestados-darwin-arm64 \
  dist/dashboard-atestados-darwin-amd64 \
  dist/dashboard-atestados-windows-amd64.exe

echo ""
echo "Release $TAG publicado com sucesso!"
echo "https://github.com/c3t4r4/DashAtestados/releases/tag/$TAG"
