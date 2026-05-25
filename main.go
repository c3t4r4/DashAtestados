package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

//go:embed static
var staticFiles embed.FS

var version = "dev" // overridden at build time via -ldflags "-X main.version=x.y.z"

// ─── Global mutable state ──────────────────────────────────────────────────────

var (
	mu            sync.RWMutex
	dadosGlobal   []Atestado
	overlapGlobal []Overlap
	dirGlobal     string
)

// ─── Data types ────────────────────────────────────────────────────────────────

type Atestado struct {
	ID              int    `json:"id"`
	Nome            string `json:"nome"`
	Cargo           string `json:"cargo"`
	Setor           string `json:"setor"`
	Data            string `json:"data"`    // "2006-01-02"
	DataFim         string `json:"dataFim"` // "2006-01-02"
	CID             string `json:"cid"`
	DiasAfastamento int    `json:"diasAfastamento"`
	Arquivo         string `json:"arquivo"`
	Aba             string `json:"aba"`
}

type OverlapType string

const (
	OverlapDuplicadoExato       OverlapType = "duplicado_exato"
	OverlapMesmoDiaCIDDiferente OverlapType = "mesmo_dia_cid_diferente"
	OverlapPeriodoSobreposto    OverlapType = "periodo_sobreposto"
)

type Overlap struct {
	Tipo      OverlapType `json:"tipo"`
	Registros []Atestado  `json:"registros"`
	Descricao string      `json:"descricao"`
}

type KV struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

type Resumo struct {
	TotalAtestados     int                       `json:"totalAtestados"`
	TotalDias          int                       `json:"totalDias"`
	PorSetor           []KV                      `json:"porSetor"`
	PorCID             []KV                      `json:"porCID"`
	PorFuncionario     []KV                      `json:"porFuncionario"`
	PorAnoMes          map[string]map[string]int `json:"porAnoMes"`
	SetorLider         string                    `json:"setorLider"`
	CIDFrequente       string                    `json:"cidFrequente"`
	FuncDestaque       string                    `json:"funcDestaque"`
	Anos               []string                  `json:"anos"`
	Setores            []string                  `json:"setores"`
	CIDs               []string                  `json:"cids"`
	Nomes              []string                  `json:"nomes"`
	DiretorioAtestados string                    `json:"diretorioAtestados"`
	ArquivosXlsx       int                       `json:"arquivosXlsx"`
	Version            string                    `json:"version"`
}

// ─── Date parsing ──────────────────────────────────────────────────────────────

var dateFmts = []string{
	"02/01/2006",          // Brazilian DD/MM/YYYY
	"2006-01-02",          // ISO 8601
	"01/02/2006",          // US MM/DD/YYYY
	"2006/01/02",          // YYYY/MM/DD
	"02-01-2006",          // DD-MM-YYYY
	"02/01/06",            // Brazilian DD/MM/YY (Excel built-in format 14)
	"01/02/06",            // US MM/DD/YY
	"2/1/2006",            // no-leading-zero D/M/YYYY
	"1/2/2006",            // no-leading-zero M/D/YYYY
	"02.01.2006",          // dot-separated DD.MM.YYYY
	"2006-01-02 15:04:05", // ISO with time component
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 1 {
		t, err := excelize.ExcelDateToTime(f, false)
		if err == nil {
			return t, nil
		}
	}
	for _, fmt := range dateFmts {
		if t, err := time.Parse(fmt, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

// ─── Load data ─────────────────────────────────────────────────────────────────

func loadAtestados(dir string) ([]Atestado, error) {
	var all []Atestado
	nextID := 1

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
			return nil
		}
		if strings.HasPrefix(name, "~$") {
			return nil
		}

		f, err := excelize.OpenFile(path, excelize.Options{RawCellValue: true})
		if err != nil {
			log.Printf("warn: cannot open %s: %v", path, err)
			return nil
		}
		defer f.Close()

		sheets := f.GetSheetList()
		for _, sheet := range sheets {
			rows, err := f.GetRows(sheet)
			if err != nil {
				continue
			}
			for rowIdx, row := range rows {
				if rowIdx == 0 {
					continue
				}
				if len(row) < 1 {
					continue
				}
				nomeVal := ""
				if len(row) > 0 {
					nomeVal = strings.TrimSpace(row[0])
				}
				if nomeVal == "" || strings.EqualFold(nomeVal, "Nome") {
					continue
				}

				cargo := ""
				if len(row) > 1 {
					cargo = strings.TrimSpace(row[1])
				}
				setor := ""
				if len(row) > 2 {
					setor = strings.TrimSpace(row[2])
				}
				dataStr := ""
				if len(row) > 3 {
					dataStr = strings.TrimSpace(row[3])
				}
				cid := ""
				if len(row) > 4 {
					cid = strings.TrimSpace(row[4])
				}
				diasStr := ""
				if len(row) > 5 {
					diasStr = strings.TrimSpace(row[5])
				}

				t, err := parseDate(dataStr)
				if err != nil {
					log.Printf("warn: row %d in %s/%s: %v", rowIdx+1, name, sheet, err)
					continue
				}

				dias := 0
				if diasStr != "" {
					if d, err := strconv.Atoi(strings.TrimSpace(diasStr)); err == nil {
						dias = d
					} else if df, err := strconv.ParseFloat(diasStr, 64); err == nil {
						dias = int(df)
					}
				}
				if dias < 0 {
					dias = 0
				}

				dataFim := t
				if dias > 1 {
					dataFim = t.AddDate(0, 0, dias-1)
				}

				all = append(all, Atestado{
					ID:              nextID,
					Nome:            nomeVal,
					Cargo:           cargo,
					Setor:           setor,
					Data:            t.Format("2006-01-02"),
					DataFim:         dataFim.Format("2006-01-02"),
					CID:             cid,
					DiasAfastamento: dias,
					Arquivo:         name,
					Aba:             sheet,
				})
				nextID++
			}
		}
		return nil
	})

	return all, err
}

// ─── Overlap detection ─────────────────────────────────────────────────────────

func detectOverlaps(dados []Atestado) []Overlap {
	byNome := map[string][]Atestado{}
	for _, a := range dados {
		byNome[a.Nome] = append(byNome[a.Nome], a)
	}

	var overlaps []Overlap

	for _, list := range byNome {
		n := len(list)
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				a, b := list[i], list[j]

				aStart, err1 := time.Parse("2006-01-02", a.Data)
				aEnd, err2 := time.Parse("2006-01-02", a.DataFim)
				bStart, err3 := time.Parse("2006-01-02", b.Data)
				bEnd, err4 := time.Parse("2006-01-02", b.DataFim)

				if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
					continue
				}

				if a.Data == b.Data {
					if strings.EqualFold(a.CID, b.CID) {
						overlaps = append(overlaps, Overlap{
							Tipo:      OverlapDuplicadoExato,
							Registros: []Atestado{a, b},
							Descricao: fmt.Sprintf("%s: data %s, CID %s — duplicado exato", a.Nome, a.Data, a.CID),
						})
					} else {
						overlaps = append(overlaps, Overlap{
							Tipo:      OverlapMesmoDiaCIDDiferente,
							Registros: []Atestado{a, b},
							Descricao: fmt.Sprintf("%s: data %s — mesmo dia com CIDs diferentes (%s vs %s)", a.Nome, a.Data, a.CID, b.CID),
						})
					}
					continue
				}

				if aStart.Before(bEnd.AddDate(0, 0, 1)) && bStart.Before(aEnd.AddDate(0, 0, 1)) {
					overlaps = append(overlaps, Overlap{
						Tipo:      OverlapPeriodoSobreposto,
						Registros: []Atestado{a, b},
						Descricao: fmt.Sprintf("%s: período %s–%s sobrepõe com %s–%s", a.Nome, a.Data, a.DataFim, b.Data, b.DataFim),
					})
				}
			}
		}
	}

	return overlaps
}

// ─── Build resumo ──────────────────────────────────────────────────────────────

func buildResumo(dados []Atestado) Resumo {
	totalDias := 0
	setorCount := map[string]int{}
	cidCount := map[string]int{}
	funcCount := map[string]int{}
	porAnoMes := map[string]map[string]int{}
	anosSet := map[string]bool{}
	setoresSet := map[string]bool{}
	cidsSet := map[string]bool{}
	nomesSet := map[string]bool{}

	for _, a := range dados {
		totalDias += a.DiasAfastamento
		if a.Setor != "" {
			setorCount[a.Setor]++
			setoresSet[a.Setor] = true
		}
		if a.CID != "" {
			cidCount[a.CID]++
			cidsSet[a.CID] = true
		}
		if a.Nome != "" {
			funcCount[a.Nome]++
			nomesSet[a.Nome] = true
		}

		t, err := time.Parse("2006-01-02", a.Data)
		if err == nil {
			year := t.Format("2006")
			month := t.Format("01")
			anosSet[year] = true
			if porAnoMes[year] == nil {
				porAnoMes[year] = map[string]int{}
			}
			porAnoMes[year][month]++
		}
	}

	toKVSlice := func(m map[string]int, top int) []KV {
		out := make([]KV, 0, len(m))
		for k, v := range m {
			out = append(out, KV{Key: k, Value: v})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Value != out[j].Value {
				return out[i].Value > out[j].Value
			}
			return out[i].Key < out[j].Key
		})
		if top > 0 && len(out) > top {
			out = out[:top]
		}
		return out
	}

	toSortedKeys := func(m map[string]bool) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}

	porSetor := toKVSlice(setorCount, 12)
	porCID := toKVSlice(cidCount, 12)
	porFunc := toKVSlice(funcCount, 12)

	setorLider := ""
	if len(porSetor) > 0 {
		setorLider = porSetor[0].Key
	}
	cidFreq := ""
	if len(porCID) > 0 {
		cidFreq = porCID[0].Key
	}
	funcDest := ""
	if len(porFunc) > 0 {
		funcDest = porFunc[0].Key
	}

	return Resumo{
		TotalAtestados: len(dados),
		TotalDias:      totalDias,
		PorSetor:       porSetor,
		PorCID:         porCID,
		PorFuncionario: porFunc,
		PorAnoMes:      porAnoMes,
		SetorLider:     setorLider,
		CIDFrequente:   cidFreq,
		FuncDestaque:   funcDest,
		Anos:           toSortedKeys(anosSet),
		Setores:        toSortedKeys(setoresSet),
		CIDs:           toSortedKeys(cidsSet),
		Nomes:          toSortedKeys(nomesSet),
	}
}

// ─── Count xlsx files ──────────────────────────────────────────────────────────

func countXlsxFiles(dir string) int {
	n := 0
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(strings.ToLower(name), ".xlsx") && !strings.HasPrefix(name, "~$") {
			n++
		}
		return nil
	})
	return n
}

// ─── Ensure Atestados dir ──────────────────────────────────────────────────────

func ensureAtestadosDir() (string, error) {
	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		// Direct exe dir
		candidates = append(candidates, filepath.Join(exeDir, "Atestados"))
		// Inside macOS .app bundle: exe is at Foo.app/Contents/MacOS/binary
		// Atestados/ lives next to the .app bundle
		if strings.HasSuffix(filepath.ToSlash(filepath.Dir(exeDir)), "Contents/MacOS") {
			bundleParent := filepath.Dir(filepath.Dir(exeDir))
			candidates = append(candidates, filepath.Join(bundleParent, "Atestados"))
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "Atestados"))
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}

	// Create alongside executable (or cwd as fallback)
	var baseDir string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		if strings.HasSuffix(filepath.ToSlash(filepath.Dir(exeDir)), "Contents/MacOS") {
			baseDir = filepath.Dir(filepath.Dir(exeDir))
		} else {
			baseDir = exeDir
		}
	} else if cwd, err := os.Getwd(); err == nil {
		baseDir = cwd
	} else {
		return "", fmt.Errorf("não foi possível determinar diretório base")
	}

	dir := filepath.Join(baseDir, "Atestados")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar Atestados/: %w", err)
	}
	log.Printf("Diretório Atestados/ criado em: %s", dir)
	return dir, nil
}

// ─── Create template xlsx ──────────────────────────────────────────────────────

func createTemplate(dir string) (string, error) {
	// Find the highest year among existing xlsx files (e.g. 2025.xlsx, 2026.xlsx → use 2027)
	maxYear := time.Now().Year()
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if !strings.HasSuffix(strings.ToLower(n), ".xlsx") || strings.HasPrefix(n, "~$") {
				continue
			}
			base := strings.TrimSuffix(strings.ToLower(n), ".xlsx")
			if y, err := strconv.Atoi(base); err == nil && y >= 2000 && y < 3000 {
				if y >= maxYear {
					maxYear = y + 1
				}
			}
		}
	}

	year := strconv.Itoa(maxYear)
	name := year + ".xlsx"
	path := filepath.Join(dir, name)
	// Avoid overwriting if that year file already exists for another reason
	if _, err := os.Stat(path); err == nil {
		for i := 2; ; i++ {
			c := filepath.Join(dir, fmt.Sprintf("%s_%d.xlsx", year, i))
			if _, err2 := os.Stat(c); os.IsNotExist(err2) {
				path = c
				name = filepath.Base(c)
				break
			}
		}
	}

	months := []string{
		"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
	}
	headers := []interface{}{"Nome", "Cargo", "Setor", "Data", "CID", "Dias Afastamento"}

	f := excelize.NewFile()
	defer f.Close()

	for i, month := range months {
		var sheet string
		if i == 0 {
			f.SetSheetName("Sheet1", month)
			sheet = month
		} else {
			if _, err := f.NewSheet(month); err != nil {
				return "", fmt.Errorf("erro ao criar aba %s: %w", month, err)
			}
			sheet = month
		}
		if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
			return "", fmt.Errorf("erro ao definir cabeçalho na aba %s: %w", sheet, err)
		}
	}

	if err := f.SaveAs(path); err != nil {
		return "", fmt.Errorf("erro ao salvar planilha modelo: %w", err)
	}
	return name, nil
}

// ─── Reload data from disk ─────────────────────────────────────────────────────

func reloadData() error {
	mu.RLock()
	dir := dirGlobal
	mu.RUnlock()

	newDados, err := loadAtestados(dir)
	if err != nil {
		return err
	}
	newOverlaps := detectOverlaps(newDados)

	mu.Lock()
	dadosGlobal = newDados
	overlapGlobal = newOverlaps
	mu.Unlock()

	log.Printf("Recarregados %d registros", len(newDados))
	return nil
}

// ─── HTTP handlers ─────────────────────────────────────────────────────────────

func corsJSON(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func apiResumo(w http.ResponseWriter, r *http.Request) {
	corsJSON(w)
	mu.RLock()
	dados := dadosGlobal
	dir := dirGlobal
	mu.RUnlock()
	resumo := buildResumo(dados)
	resumo.DiretorioAtestados = dir
	resumo.ArquivosXlsx = countXlsxFiles(dir)
	resumo.Version = version
	json.NewEncoder(w).Encode(resumo)
}

func filterDados(dados []Atestado, r *http.Request) []Atestado {
	q := r.URL.Query()
	ano := strings.TrimSpace(q.Get("ano"))
	mes := strings.TrimSpace(q.Get("mes"))
	setor := strings.TrimSpace(q.Get("setor"))
	cid := strings.TrimSpace(q.Get("cid"))
	nome := strings.TrimSpace(q.Get("nome"))
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))

	var out []Atestado
	for _, a := range dados {
		if ano != "" && !strings.HasPrefix(a.Data, ano) {
			continue
		}
		if mes != "" {
			if len(a.Data) >= 7 && a.Data[5:7] != mes {
				continue
			}
		}
		if setor != "" && !strings.EqualFold(a.Setor, setor) {
			continue
		}
		if cid != "" && !strings.EqualFold(a.CID, cid) {
			continue
		}
		if nome != "" && !strings.Contains(strings.ToLower(a.Nome), strings.ToLower(nome)) {
			continue
		}
		if search != "" {
			hay := strings.ToLower(a.Nome + " " + a.Cargo + " " + a.Setor + " " + a.CID + " " + a.Data)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		out = append(out, a)
	}
	if out == nil {
		out = []Atestado{}
	}
	return out
}

func apiDados(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	dados := dadosGlobal
	mu.RUnlock()

	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=atestados.csv")
		filtered := filterDados(dados, r)
		cw := csv.NewWriter(w)
		cw.Write([]string{"ID", "Nome", "Cargo", "Setor", "Data", "DataFim", "CID", "DiasAfastamento", "Arquivo", "Aba"})
		for _, a := range filtered {
			cw.Write([]string{
				strconv.Itoa(a.ID), a.Nome, a.Cargo, a.Setor, a.Data, a.DataFim,
				a.CID, strconv.Itoa(a.DiasAfastamento), a.Arquivo, a.Aba,
			})
		}
		cw.Flush()
		return
	}

	corsJSON(w)
	filtered := filterDados(dados, r)
	json.NewEncoder(w).Encode(filtered)
}

func apiOverlapsHandler(w http.ResponseWriter, r *http.Request) {
	corsJSON(w)
	mu.RLock()
	overlaps := overlapGlobal
	mu.RUnlock()
	if overlaps == nil {
		overlaps = []Overlap{}
	}
	json.NewEncoder(w).Encode(overlaps)
}

func apiReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	corsJSON(w)
	if err := reloadData(); err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
		return
	}
	mu.RLock()
	n := len(dadosGlobal)
	mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"total":    n,
		"mensagem": fmt.Sprintf("%d registros carregados", n),
	})
}

func apiCriarTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	corsJSON(w)
	mu.RLock()
	dir := dirGlobal
	mu.RUnlock()
	fname, err := createTemplate(dir)
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"arquivo":  fname,
		"mensagem": fmt.Sprintf("Planilha '%s' criada em Atestados/. Preencha os dados e clique em Atualizar Dados.", fname),
	})
}

// ─── SPA handler ───────────────────────────────────────────────────────────────

func spaHandler(staticFS embed.FS) http.HandlerFunc {
	fsys, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		_, err := fsys.Open(path)
		if err != nil {
			data, _ := staticFS.ReadFile("static/index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

// ─── Port finder ───────────────────────────────────────────────────────────────

func findFreePort(start int) int {
	for port := start; port < start+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return start
}

// ─── Open browser ──────────────────────────────────────────────────────────────

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

// ─── Main ──────────────────────────────────────────────────────────────────────

func main() {
	dir, err := ensureAtestadosDir()
	if err != nil {
		log.Fatalf("Erro: %v", err)
	}
	dirGlobal = dir
	log.Printf("Diretório de atestados: %s", dir)

	dados, err := loadAtestados(dir)
	if err != nil {
		log.Fatalf("Erro ao carregar atestados: %v", err)
	}
	log.Printf("Carregados %d registros", len(dados))

	if countXlsxFiles(dir) == 0 {
		fname, err := createTemplate(dir)
		if err != nil {
			log.Printf("Aviso: não foi possível criar planilha modelo: %v", err)
		} else {
			log.Printf("Pasta vazia — planilha modelo criada: Atestados/%s", fname)
		}
	}

	dadosGlobal = dados
	overlapGlobal = detectOverlaps(dados)
	log.Printf("Detectadas %d sobreposições", len(overlapGlobal))

	port := findFreePort(8787)
	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/resumo", apiResumo)
	mux.HandleFunc("/api/dados", apiDados)
	mux.HandleFunc("/api/overlaps", apiOverlapsHandler)
	mux.HandleFunc("/api/reload", apiReload)
	mux.HandleFunc("/api/criar-template", apiCriarTemplate)
	mux.HandleFunc("/", spaHandler(staticFiles))

	log.Printf("Servidor iniciado em %s", url)
	fmt.Printf("\n  Dashboard disponível em: %s\n\n", url)

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(url)
	}()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
