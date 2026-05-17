# Graph Report - .  (2026-05-17)

## Corpus Check
- Corpus is ~11,843 words - fits in a single context window. You may not need a graph.

## Summary
- 97 nodes · 159 edges · 12 communities (11 shown, 1 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 14 edges (avg confidence: 0.9)
- Token cost: 74,920 input · 4,150 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Frontend API Client|Frontend API Client]]
- [[_COMMUNITY_Icon & Visual Assets|Icon & Visual Assets]]
- [[_COMMUNITY_Charts & Theming|Charts & Theming]]
- [[_COMMUNITY_Core Data Model|Core Data Model]]
- [[_COMMUNITY_KPI Summary Pipeline|KPI Summary Pipeline]]
- [[_COMMUNITY_Data Ingestion & Reload|Data Ingestion & Reload]]
- [[_COMMUNITY_Server Bootstrap|Server Bootstrap]]
- [[_COMMUNITY_Overlap Detection|Overlap Detection]]
- [[_COMMUNITY_Template Management|Template Management]]
- [[_COMMUNITY_Embedded Frontend Serving|Embedded Frontend Serving]]
- [[_COMMUNITY_Resume API (AST)|Resume API (AST)]]
- [[_COMMUNITY_Build Configuration|Build Configuration]]

## God Nodes (most connected - your core abstractions)
1. `Application Icon (SVG source)` - 17 edges
2. `main` - 16 edges
3. `main()` - 9 edges
4. `init (JS)` - 9 edges
5. `apiResumo` - 7 edges
6. `apiDados` - 7 edges
7. `dadosGlobal` - 7 edges
8. `corsJSON()` - 6 edges
9. `detectOverlaps` - 6 edges
10. `reloadData` - 6 edges

## Surprising Connections (you probably didn't know these)
- `Overlap detection strategy` --describes--> `detectOverlaps`  [EXTRACTED]
  CLAUDE.md → main.go
- `loadResumo (JS)` --calls_http--> `apiResumo`  [EXTRACTED]
  static/index.html → main.go
- `loadDados (JS)` --calls_http--> `apiDados`  [EXTRACTED]
  static/index.html → main.go
- `exportCSV (JS)` --calls_http--> `apiDados`  [EXTRACTED]
  static/index.html → main.go
- `loadOverlaps (JS)` --calls_http--> `apiOverlapsHandler`  [EXTRACTED]
  static/index.html → main.go

## Hyperedges (group relationships)
- **Global mutable state shared across HTTP handlers** — maingo_dadosglobal, maingo_overlapglobal, maingo_dirglobal, maingo_syncrwmutex, maingo_apiresumo, maingo_apidados, maingo_apioverlapshandler, maingo_apireload, maingo_reloaddata [EXTRACTED 0.95]
- **Data ingestion pipeline: xlsx → Atestado structs → overlaps + resumo** — maingo_loadatestados, maingo_parsedate, maingo_atestado, maingo_detectoverlaps, maingo_overlap, maingo_buildresumo, maingo_resumo [EXTRACTED 0.95]
- **Frontend render cycle triggered on load and filter** — indexhtml_init, indexhtml_applyfilters, indexhtml_renderkpis, indexhtml_rendercharts, indexhtml_renderoverlaps, indexhtml_rendertable [EXTRACTED 0.95]
- **Complete Icon Asset Family for Dashboard Application** — assets_icon_svg, assets_icon_32, assets_icon_256, assets_icon_512, assets_favicon_32, static_favicon_32 [INFERRED 0.95]

## Communities (12 total, 1 thin omitted)

### Community 0 - "Frontend API Client"
Cohesion: 0.15
Nodes (23): Data flow: startup → load → in-memory → frontend, criarTemplate (JS), exportCSV (JS), loadOverlaps (JS), refresh (JS), apiCriarTemplate, apiDados, apiOverlapsHandler (+15 more)

### Community 1 - "Icon & Visual Assets"
Cohesion: 0.13
Nodes (20): Favicon 32px PNG, Application Icon 256px PNG, Application Icon 32px PNG, Application Icon 512px PNG, Application Icon (SVG source), Color: Bar Chart Amber (#F39C12), Color: Bar Chart Blue (#2E75B6), Color: Bar Chart Green (#27AE60) (+12 more)

### Community 2 - "Charts & Theming"
Cohesion: 0.23
Nodes (13): Frontend theming design, applyFilters (JS), drawDonut (JS), drawGroupedBar (JS), drawHBar (JS), init (JS), loadDados (JS), renderCharts (JS) (+5 more)

### Community 3 - "Core Data Model"
Cohesion: 0.29
Nodes (7): Atestado, KV, apiDados(), filterDados(), Overlap, OverlapType, Resumo

### Community 4 - "KPI Summary Pipeline"
Cohesion: 0.47
Nodes (6): loadResumo (JS), apiResumo, buildResumo, countXlsxFiles, KV, Resumo

### Community 5 - "Data Ingestion & Reload"
Cohesion: 0.4
Nodes (5): apiReload(), detectOverlaps(), loadAtestados(), parseDate(), reloadData()

### Community 6 - "Server Bootstrap"
Cohesion: 0.4
Nodes (5): ensureAtestadosDir(), findFreePort(), main(), openBrowser(), spaHandler()

### Community 7 - "Overlap Detection"
Cohesion: 0.6
Nodes (5): Overlap detection strategy, Atestado, detectOverlaps, Overlap, OverlapType

### Community 8 - "Template Management"
Cohesion: 0.5
Nodes (4): apiCriarTemplate(), apiOverlapsHandler(), corsJSON(), createTemplate()

### Community 9 - "Embedded Frontend Serving"
Cohesion: 0.5
Nodes (4): Single-binary Go HTTP server architecture, index.html SPA, spaHandler, staticFiles (embed.FS)

### Community 10 - "Resume API (AST)"
Cohesion: 0.67
Nodes (3): apiResumo(), buildResumo(), countXlsxFiles()

## Knowledge Gaps
- **25 isolated node(s):** `Atestado`, `OverlapType`, `Overlap`, `KV`, `Resumo` (+20 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `main` connect `Frontend API Client` to `Embedded Frontend Serving`, `KPI Summary Pipeline`, `Overlap Detection`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `init (JS)` connect `Charts & Theming` to `Frontend API Client`, `Embedded Frontend Serving`, `KPI Summary Pipeline`?**
  _High betweenness centrality (0.087) - this node is a cross-community bridge._
- **Are the 4 inferred relationships involving `Application Icon (SVG source)` (e.g. with `Application Icon 32px PNG` and `Application Icon 256px PNG`) actually correct?**
  _`Application Icon (SVG source)` has 4 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Atestado`, `OverlapType`, `Overlap` to the rest of the system?**
  _25 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Icon & Visual Assets` be split into smaller, more focused modules?**
  _Cohesion score 0.13 - nodes in this community are weakly interconnected._