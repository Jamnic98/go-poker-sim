package api

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go-poker-sim/internal/sim"
)

type ChunkInfo struct {
	Name   string `json:"name"`
	Index  int    `json:"index"`
	Format string `json:"format"`
	Size   int64  `json:"size_bytes"`
}

type RunInfo struct {
	ID         string        `json:"id"`
	Date       string        `json:"date"`
	Dir        string        `json:"dir"`
	Status     string        `json:"status"`
	Players    int           `json:"players"`
	Runs       int           `json:"runs"`
	Format     string        `json:"format,omitempty"`
	Chunks     []ChunkInfo   `json:"chunks"`
	HasSummary bool          `json:"has_summary"`
	Manifest   *sim.Manifest `json:"manifest,omitempty"`
}

type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size_bytes"`
}

type RunDetail struct {
	RunInfo
	Files []FileInfo `json:"files"`
}

type PocketRow struct {
	Pocket string  `json:"pocket"`
	Hands  int     `json:"hands"`
	Wins   int     `json:"wins"`
	WinPct float64 `json:"win_pct"`
}

type CategoryRow struct {
	HandType string  `json:"hand_type"`
	Hands    int     `json:"hands"`
	Pct      float64 `json:"pct"`
}

type HandsResponse struct {
	Total  int              `json:"total"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
	Rows   []sim.HandResult `json:"rows"`
}

var runDirPattern = regexp.MustCompile(`^\[[0-9:]{8}\]-([0-9]+)p-([0-9]+)r(-[0-9]+)?$`)

// listRunInfos scans the results root for run directories, newest first.
func (s *Server) listRunInfos() ([]RunInfo, error) {
	dateEntries, err := os.ReadDir(s.resultsDir)
	if err != nil {
		return nil, err
	}
	runs := make([]RunInfo, 0)
	for _, de := range dateEntries {
		if !de.IsDir() {
			continue
		}
		dateDir := filepath.Join(s.resultsDir, de.Name())
		runEntries, err := os.ReadDir(dateDir)
		if err != nil {
			return nil, err
		}
		for _, re := range runEntries {
			if !re.IsDir() {
				continue
			}
			info, err := s.runInfo(de.Name(), re.Name(), filepath.Join(dateDir, re.Name()))
			if err != nil {
				return nil, err
			}
			runs = append(runs, info)
		}
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID > runs[j].ID })
	return runs, nil
}

func (s *Server) runInfo(date, name, dir string) (RunInfo, error) {
	info := RunInfo{
		ID:     date + "/" + name,
		Date:   date,
		Dir:    name,
		Status: "unknown",
	}
	if m, err := readManifest(filepath.Join(dir, sim.ManifestFileName)); err == nil {
		info.Manifest = m
		info.Status = m.Status
		info.Players = m.Players
		info.Runs = m.Runs
		info.Format = m.Format
	} else if match := runDirPattern.FindStringSubmatch(name); match != nil {
		info.Players, _ = strconv.Atoi(match[1])
		info.Runs, _ = strconv.Atoi(match[2])
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return info, err
	}
	chunks := make([]ChunkInfo, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if idx, ok := chunkIndex(e.Name()); ok {
			st, err := e.Info()
			if err != nil {
				return info, err
			}
			chunks = append(chunks, ChunkInfo{
				Name:   e.Name(),
				Index:  idx,
				Format: strings.TrimPrefix(filepath.Ext(e.Name()), "."),
				Size:   st.Size(),
			})
		}
		if e.Name() == "summary.csv" {
			info.HasSummary = true
		}
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].Index < chunks[j].Index })
	info.Chunks = chunks
	return info, nil
}

func readManifest(path string) (*sim.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m sim.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func chunkIndex(name string) (int, bool) {
	const prefix = "pre_flop_sim_chunk_"
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	base := strings.TrimPrefix(name, prefix)
	n, err := strconv.Atoi(strings.TrimSuffix(base, filepath.Ext(base)))
	if err != nil {
		return 0, false
	}
	return n, true
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request, id, dir string) {
	date, name := splitID(id)
	info, err := s.runInfo(date, name, dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	files := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		st, err := e.Info()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		files = append(files, FileInfo{Name: e.Name(), Size: st.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	writeJSON(w, http.StatusOK, RunDetail{RunInfo: info, Files: files})
}

func splitID(id string) (date, name string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", id
}

func (s *Server) handleChunks(w http.ResponseWriter, r *http.Request, id, dir string) {
	date, name := splitID(id)
	info, err := s.runInfo(date, name, dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chunks": info.Chunks})
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request, dir string) {
	path := filepath.Join(dir, "summary.csv")
	records, err := readCSVRecords(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reading summary.csv: "+err.Error())
		return
	}
	rows := make([]PocketRow, 0, len(records))
	for i, rec := range records {
		if i == 0 && rec[0] == "pocket" {
			continue
		}
		if len(rec) != 4 {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("summary.csv row %d has %d fields, want 4", i, len(rec)))
			return
		}
		hands, err1 := strconv.Atoi(rec[1])
		wins, err2 := strconv.Atoi(rec[2])
		pct, err3 := strconv.ParseFloat(rec[3], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("summary.csv row %d malformed: %v", i, rec))
			return
		}
		rows = append(rows, PocketRow{Pocket: rec[0], Hands: hands, Wins: wins, WinPct: pct})
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request, dir string) {
	path := filepath.Join(dir, "categories.csv")
	records, err := readCSVRecords(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reading categories.csv: "+err.Error())
		return
	}
	rows := make([]CategoryRow, 0, len(records))
	for i, rec := range records {
		if i == 0 && rec[0] == "hand_type" {
			continue
		}
		if len(rec) != 3 {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("categories.csv row %d has %d fields, want 3", i, len(rec)))
			return
		}
		hands, err1 := strconv.Atoi(rec[1])
		pct, err2 := strconv.ParseFloat(rec[2], 64)
		if err1 != nil || err2 != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("categories.csv row %d malformed: %v", i, rec))
			return
		}
		rows = append(rows, CategoryRow{HandType: rec[0], Hands: hands, Pct: pct})
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

type handFilter struct {
	hasRunID bool
	runID    int
	hasSeat  bool
	seat     int
	hasWon   bool
	won      bool
	handType string
}

func (f handFilter) match(r sim.HandResult) bool {
	if f.hasRunID && r.RunID != f.runID {
		return false
	}
	if f.hasSeat && r.Seat != f.seat {
		return false
	}
	if f.hasWon && r.Won != f.won {
		return false
	}
	if f.handType != "" && r.HandType != f.handType {
		return false
	}
	return true
}

func (s *Server) handleHands(w http.ResponseWriter, r *http.Request, dir string) {
	q := r.URL.Query()
	f := handFilter{}
	var err error
	if v := q.Get("run_id"); v != "" {
		if f.runID, err = strconv.Atoi(v); err != nil || f.runID < 0 {
			writeError(w, http.StatusBadRequest, "run_id must be a non-negative integer")
			return
		}
		f.hasRunID = true
	}
	if v := q.Get("seat"); v != "" {
		if f.seat, err = strconv.Atoi(v); err != nil || f.seat < 0 {
			writeError(w, http.StatusBadRequest, "seat must be a non-negative integer")
			return
		}
		f.hasSeat = true
	}
	if v := q.Get("won"); v != "" {
		if f.won, err = strconv.ParseBool(v); err != nil {
			writeError(w, http.StatusBadRequest, "won must be true or false")
			return
		}
		f.hasWon = true
	}
	f.handType = q.Get("hand_type")

	offset := 0
	if v := q.Get("offset"); v != "" {
		if offset, err = strconv.Atoi(v); err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "offset must be a non-negative integer")
			return
		}
	}
	limit := defaultHandsLimit
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if limit > maxHandsLimit {
			limit = maxHandsLimit
		}
	}

	chunks, err := listChunkFiles(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := HandsResponse{Offset: offset, Limit: limit, Rows: []sim.HandResult{}}
	for _, path := range chunks {
		err := streamChunkRows(path, func(row sim.HandResult) {
			if !f.match(row) {
				return
			}
			resp.Total++
			if resp.Total > offset && len(resp.Rows) < limit {
				resp.Rows = append(resp.Rows, row)
			}
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "reading "+filepath.Base(path)+": "+err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// listChunkFiles returns chunk file paths ordered by chunk index.
func listChunkFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type indexed struct {
		idx  int
		path string
	}
	var chunks []indexed
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if idx, ok := chunkIndex(e.Name()); ok {
			chunks = append(chunks, indexed{idx, filepath.Join(dir, e.Name())})
		}
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].idx < chunks[j].idx })
	paths := make([]string, len(chunks))
	for i, c := range chunks {
		paths[i] = c.path
	}
	return paths, nil
}

// streamChunkRows parses a chunk file (CSV or JSONL, chosen by extension) and
// invokes fn for every row in file order (which is (run_id, seat) sorted).
func streamChunkRows(path string, fn func(sim.HandResult)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.HasSuffix(path, ".jsonl") {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		line := 0
		for sc.Scan() {
			line++
			b := bytes.TrimSpace(sc.Bytes())
			if len(b) == 0 {
				continue
			}
			var row sim.HandResult
			if err := json.Unmarshal(b, &row); err != nil {
				return fmt.Errorf("line %d: %w", line, err)
			}
			fn(row)
		}
		return sc.Err()
	}

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	line := 0
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		line++
		if line == 1 && len(rec) > 0 && rec[0] == "run_id" {
			continue
		}
		if len(rec) != 6 {
			return fmt.Errorf("line %d: got %d fields, want 6", line, len(rec))
		}
		runID, err := strconv.Atoi(rec[0])
		if err != nil {
			return fmt.Errorf("line %d: bad run_id %q", line, rec[0])
		}
		seat, err := strconv.Atoi(rec[1])
		if err != nil {
			return fmt.Errorf("line %d: bad seat %q", line, rec[1])
		}
		won, err := strconv.ParseBool(rec[5])
		if err != nil {
			return fmt.Errorf("line %d: bad won %q", line, rec[5])
		}
		fn(sim.HandResult{
			RunID:    runID,
			Seat:     seat,
			Pocket:   strings.Fields(rec[2]),
			Board:    strings.Fields(rec[3]),
			HandType: rec[4],
			Won:      won,
		})
	}
}

func readCSVRecords(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	return r.ReadAll()
}
