# Dashboard de Atestados Médicos

Aplicativo desktop/web para visualização e análise de atestados médicos a partir de planilhas Excel. Roda como um servidor HTTP local que abre automaticamente no navegador — nenhuma instalação adicional necessária.

## O que faz

- Lê todos os arquivos `.xlsx` da pasta `Atestados/` e exibe os dados em um dashboard interativo
- Mostra KPIs (total de atestados, dias afastados, funcionário destaque, setor líder, CID mais frequente)
- Gráficos de barras e rosca por setor, CID e evolução mensal
- Filtros combinados por ano, mês, setor, CID, funcionário e busca livre
- Detecção automática de atestados duplicados ou sobrepostos (mesmo funcionário, datas conflitantes)
- Exportação dos dados filtrados para CSV
- Cria pasta `Atestados/` e uma planilha modelo automaticamente se não existirem
- Botão para gerar nova planilha modelo para o próximo ano
- Botão para recarregar os dados sem reiniciar o servidor
- Tema escuro/claro com preferência salva no navegador

## Como usar

1. Coloque o executável na mesma pasta onde ficará a pasta `Atestados/`
2. Execute o binário — o navegador abre automaticamente em `http://localhost:8787`
3. Preencha as planilhas em `Atestados/` com os dados de atestados
4. Clique em **Atualizar Dados** para recarregar sem reiniciar

Se a pasta `Atestados/` não existir, ela é criada junto com uma planilha modelo `{anoAtual}.xlsx` pronta para preenchimento.

## Formato das planilhas

Cada arquivo `.xlsx` em `Atestados/` pode ter múltiplas abas (uma por mês, por exemplo). Cada aba deve seguir esta estrutura de colunas:

| A | B | C | D | E | F |
|---|---|---|---|---|---|
| Nome | Cargo | Setor | Data | CID | Dias Afastamento |

- A linha 1 é sempre o cabeçalho (ignorada na leitura)
- Datas aceitas: `DD/MM/AAAA`, `AAAA-MM-DD`, `MM/DD/AAAA`, `AAAA/MM/DD`, `DD-MM-AAAA`, ou número serial do Excel
- Linhas com `Nome` vazio ou igual a `"Nome"` são ignoradas
- Arquivos temporários do Excel (`~$...`) são ignorados automaticamente

## API

O servidor expõe uma API JSON local:

| Método | Rota | Descrição |
|---|---|---|
| GET | `/api/resumo` | KPIs, dados dos gráficos, opções de filtro |
| GET | `/api/dados` | Registros filtrados; `?format=csv` baixa como CSV |
| GET | `/api/overlaps` | Atestados duplicados ou com períodos sobrepostos |
| POST | `/api/reload` | Relê a pasta `Atestados/` sem reiniciar |
| POST | `/api/criar-template` | Cria nova planilha modelo em `Atestados/` |

Filtros disponíveis em `/api/dados`: `?ano=`, `?mes=`, `?setor=`, `?cid=`, `?nome=`, `?q=` (busca livre).

## Detecção de sobreposições

Agrupa registros pelo nome do funcionário e detecta três tipos:

- **duplicado_exato** — mesma data de início e mesmo CID
- **mesmo_dia_cid_diferente** — mesma data de início com CIDs diferentes
- **periodo_sobreposto** — intervalos `[Data, Data + Dias − 1]` se sobrepõem

## Tecnologias

- **Go 1.21** — servidor HTTP, leitura de xlsx, detecção de sobreposições
- **[excelize v2.8](https://github.com/xuri/excelize)** — leitura e escrita de arquivos `.xlsx`
- **`//go:embed static`** — frontend embutido no binário, sem arquivos externos em runtime
- **Vanilla JS + SVG puro** — gráficos e interface sem nenhuma dependência de CDN
- **CSS variables** — sistema de temas claro/escuro com `data-theme` no `<html>`

## Build

Requer Go 1.21+. Para build multiplataforma:

```bash
bash build.sh
```

Gera em `dist/`:

| Arquivo | Plataforma |
|---|---|
| `dashboard-atestados-darwin-arm64` | macOS Apple Silicon |
| `dashboard-atestados-darwin-amd64` | macOS Intel |
| `dashboard-atestados-windows-amd64.exe` | Windows (com ícone embutido) |
| `Dashboard Atestados-arm64.app` | Bundle macOS Apple Silicon |
| `Dashboard Atestados-amd64.app` | Bundle macOS Intel |

Para embutir o ícone no `.exe` do Windows (necessário uma vez, ou quando o ícone mudar):

```bash
# Instalar go-winres se necessário
go install github.com/tc-hib/go-winres@latest
go-winres make --arch amd64
```

Para desenvolvimento local:

```bash
go run .
```

---

Criado por [Glauco Garcia Cetara](mailto:neocetara@hotmail.com) — para Juliane Roberta Imamura
