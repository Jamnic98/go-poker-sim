package api

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-poker-sim/internal/sim"
)

func setupServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	h, err := NewHandler(dir)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, dir
}

func noRedirectClient() *http.Client {
	return &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func seedRun(t *testing.T, dir string, cfg sim.Config) string {
	t.Helper()
	cfg.OutDir = dir
	stats, err := sim.Run(cfg)
	if err != nil {
		t.Fatalf("seed run: %v", err)
	}
	rel, err := filepath.Rel(dir, stats.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(rel)
}

func runURL(srv *httptest.Server, runID, sub string, query url.Values) string {
	u := srv.URL + "/api/runs/" + url.PathEscape(runID)
	if sub != "" {
		u += "/" + sub
	}
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

func request(t *testing.T, client *http.Client, method, url, body string) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

func getJSON(t *testing.T, srv *httptest.Server, url string, out any) int {
	t.Helper()
	status, body := request(t, srv.Client(), http.MethodGet, url, "")
	if status != http.StatusOK {
		t.Fatalf("GET %s: status %d, body %s", url, status, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatalf("GET %s: decode: %v (body %s)", url, err, body)
	}
	return status
}

func TestHealth(t *testing.T) {
	srv, _ := setupServer(t)
	var out map[string]string
	getJSON(t, srv, srv.URL+"/api/health", &out)
	if out["status"] != "ok" {
		t.Errorf("status = %q, want ok", out["status"])
	}
}

func TestListRunsEmpty(t *testing.T) {
	srv, _ := setupServer(t)
	var out struct {
		Runs []RunInfo `json:"runs"`
	}
	getJSON(t, srv, srv.URL+"/api/runs", &out)
	if len(out.Runs) != 0 {
		t.Errorf("got %d runs, want 0", len(out.Runs))
	}
}

func TestListAndDetailRun(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 3, Runs: 40, Workers: 2, ChunkSize: 20, Format: sim.FormatCSV, Seed: 123})

	var list struct {
		Runs []RunInfo `json:"runs"`
	}
	getJSON(t, srv, srv.URL+"/api/runs", &list)
	if len(list.Runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(list.Runs))
	}
	r := list.Runs[0]
	if r.ID != runID {
		t.Errorf("run ID = %q, want %q", r.ID, runID)
	}
	if r.Status != sim.StatusDone {
		t.Errorf("status = %q, want %q", r.Status, sim.StatusDone)
	}
	if r.Players != 3 || r.Runs != 40 || r.Format != "csv" {
		t.Errorf("players/runs/format = %d/%d/%q, want 3/40/csv", r.Players, r.Runs, r.Format)
	}
	if len(r.Chunks) != 2 {
		t.Errorf("chunks = %d, want 2", len(r.Chunks))
	}
	if !r.HasSummary {
		t.Error("HasSummary = false, want true")
	}
	if r.Manifest == nil {
		t.Fatal("manifest is nil")
	}
	if r.Manifest.Seed != 123 || r.Manifest.Hands != 120 || r.Manifest.Chunks != 2 {
		t.Errorf("manifest = %+v, want seed=123 hands=120 chunks=2", r.Manifest)
	}

	var detail RunDetail
	getJSON(t, srv, runURL(srv, runID, "", nil), &detail)
	names := map[string]bool{}
	for _, f := range detail.Files {
		names[f.Name] = true
	}
	for _, want := range []string{"manifest.json", "summary.csv", "categories.csv", "pre_flop_sim_chunk_0.csv", "pre_flop_sim_chunk_1.csv"} {
		if !names[want] {
			t.Errorf("detail files missing %s (have %v)", want, names)
		}
	}

	var chunks struct {
		Chunks []ChunkInfo `json:"chunks"`
	}
	getJSON(t, srv, runURL(srv, runID, "chunks", nil), &chunks)
	if len(chunks.Chunks) != 2 || chunks.Chunks[0].Index != 0 || chunks.Chunks[1].Index != 1 {
		t.Errorf("chunks = %+v, want indexes [0 1]", chunks.Chunks)
	}
}

func TestListRunsWithoutManifest(t *testing.T) {
	srv, dir := setupServer(t)
	legacy := filepath.Join(dir, "2026-01-01", "[10:00:00]-4p-500r")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "pre_flop_sim_chunk_0.csv"), []byte("run_id,seat,pocket,board,hand_type,won\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var list struct {
		Runs []RunInfo `json:"runs"`
	}
	getJSON(t, srv, srv.URL+"/api/runs", &list)
	if len(list.Runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(list.Runs))
	}
	r := list.Runs[0]
	if r.Status != "unknown" {
		t.Errorf("status = %q, want unknown for legacy dir", r.Status)
	}
	if r.Players != 4 || r.Runs != 500 {
		t.Errorf("players/runs parsed from dir name = %d/%d, want 4/500", r.Players, r.Runs)
	}
	if len(r.Chunks) != 1 {
		t.Errorf("chunks = %d, want 1", len(r.Chunks))
	}
}

func readSummaryCSV(t *testing.T, dir, runID string) [][]string {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, filepath.FromSlash(runID), "summary.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestSummaryEndpoint(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 3, Runs: 40, Workers: 2, ChunkSize: 20, Format: sim.FormatCSV, Seed: 123})

	var out struct {
		Rows []PocketRow `json:"rows"`
	}
	getJSON(t, srv, runURL(srv, runID, "summary", nil), &out)

	disk := readSummaryCSV(t, dir, runID)
	if len(out.Rows) != len(disk)-1 {
		t.Fatalf("endpoint rows = %d, disk rows = %d", len(out.Rows), len(disk)-1)
	}
	total := 0
	for i, row := range out.Rows {
		rec := disk[i+1]
		if row.Pocket != rec[0] {
			t.Errorf("row %d pocket = %q, want %q", i, row.Pocket, rec[0])
		}
		wantHands, _ := strconv.Atoi(rec[1])
		wantWins, _ := strconv.Atoi(rec[2])
		if row.Hands != wantHands || row.Wins != wantWins {
			t.Errorf("row %d hands/wins = %d/%d, want %d/%d", i, row.Hands, row.Wins, wantHands, wantWins)
		}
		if i > 0 && row.WinPct > out.Rows[i-1].WinPct {
			t.Errorf("win_pct not descending at row %d", i)
		}
		total += row.Hands
	}
	if total != 120 {
		t.Errorf("total hands = %d, want 120", total)
	}
}

func TestCategoriesEndpoint(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 2, Runs: 50, Workers: 2, ChunkSize: 100, Format: sim.FormatCSV, Seed: 7})

	var out struct {
		Rows []CategoryRow `json:"rows"`
	}
	getJSON(t, srv, runURL(srv, runID, "categories", nil), &out)
	if len(out.Rows) == 0 {
		t.Fatal("no category rows")
	}
	total := 0
	for _, row := range out.Rows {
		if row.HandType == "" {
			t.Error("empty hand_type")
		}
		total += row.Hands
	}
	if total != 100 {
		t.Errorf("categories total = %d, want 100", total)
	}
}

func countWonInChunks(t *testing.T, dir, runID string) int {
	t.Helper()
	runDir := filepath.Join(dir, filepath.FromSlash(runID))
	entries, err := os.ReadDir(runDir)
	if err != nil {
		t.Fatal(err)
	}
	won := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "pre_flop_sim_chunk_") || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		f, err := os.Open(filepath.Join(runDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		r := csv.NewReader(f)
		records, err := r.ReadAll()
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		for i, rec := range records {
			if i == 0 {
				continue
			}
			if b, _ := strconv.ParseBool(rec[5]); b {
				won++
			}
		}
	}
	return won
}

func TestHandsPaginationAndFilters(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 3, Runs: 40, Workers: 2, ChunkSize: 20, Format: sim.FormatCSV, Seed: 99})
	base := runURL(srv, runID, "hands", nil)

	var page struct {
		Total  int              `json:"total"`
		Offset int              `json:"offset"`
		Limit  int              `json:"limit"`
		Rows   []sim.HandResult `json:"rows"`
	}
	getJSON(t, srv, base+"?limit=25", &page)
	if page.Total != 120 || len(page.Rows) != 25 || page.Limit != 25 {
		t.Fatalf("page = total %d rows %d limit %d, want 120/25/25", page.Total, len(page.Rows), page.Limit)
	}
	for i := 1; i < len(page.Rows); i++ {
		a, b := page.Rows[i-1], page.Rows[i]
		if a.RunID > b.RunID || (a.RunID == b.RunID && a.Seat > b.Seat) {
			t.Fatalf("rows not sorted by (run_id, seat) at %d: %+v then %+v", i, a, b)
		}
	}

	getJSON(t, srv, base+"?limit=25&offset=110", &page)
	if page.Total != 120 || len(page.Rows) != 10 {
		t.Fatalf("last page = total %d rows %d, want 120/10", page.Total, len(page.Rows))
	}

	getJSON(t, srv, base+"?limit=5000", &page)
	if page.Limit != maxHandsLimit || len(page.Rows) != 120 {
		t.Fatalf("limit cap = %d rows %d, want limit %d rows 120", page.Limit, len(page.Rows), maxHandsLimit)
	}

	getJSON(t, srv, base+"?run_id=7", &page)
	if page.Total != 3 || len(page.Rows) != 3 {
		t.Fatalf("run_id=7: total %d rows %d, want 3/3", page.Total, len(page.Rows))
	}
	winners := 0
	for _, r := range page.Rows {
		if r.RunID != 7 {
			t.Errorf("row run_id = %d, want 7", r.RunID)
		}
		if len(r.Pocket) != 2 || len(r.Board) != 5 || r.HandType == "" {
			t.Errorf("malformed row %+v", r)
		}
		if r.Won {
			winners++
		}
	}
	if winners < 1 {
		t.Error("run 7 has no winner")
	}

	getJSON(t, srv, base+"?seat=1", &page)
	if page.Total != 40 {
		t.Fatalf("seat=1 total = %d, want 40", page.Total)
	}
	for _, r := range page.Rows {
		if r.Seat != 1 {
			t.Fatalf("seat filter broken: %+v", r)
		}
	}

	wantWon := countWonInChunks(t, dir, runID)
	getJSON(t, srv, base+"?won=true", &page)
	if page.Total != wantWon {
		t.Fatalf("won=true total = %d, want %d", page.Total, wantWon)
	}

	var cats struct {
		Rows []CategoryRow `json:"rows"`
	}
	getJSON(t, srv, runURL(srv, runID, "categories", nil), &cats)
	q := url.Values{"hand_type": {cats.Rows[0].HandType}}
	getJSON(t, srv, runURL(srv, runID, "hands", q), &page)
	if page.Total != cats.Rows[0].Hands {
		t.Fatalf("hand_type=%q total = %d, want %d", cats.Rows[0].HandType, page.Total, cats.Rows[0].Hands)
	}

	for _, bad := range []string{"?run_id=abc", "?limit=0", "?offset=-1", "?won=maybe", "?seat=-2"} {
		status, body := request(t, srv.Client(), http.MethodGet, base+bad, "")
		if status != http.StatusBadRequest {
			t.Errorf("GET hands%s: status %d, want 400 (body %s)", bad, status, body)
		}
	}
}

func TestHandsJSONL(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 2, Runs: 10, Workers: 2, ChunkSize: 1000, Format: sim.FormatJSONL, Seed: 9})

	var page struct {
		Total int              `json:"total"`
		Rows  []sim.HandResult `json:"rows"`
	}
	getJSON(t, srv, runURL(srv, runID, "hands", url.Values{"limit": {"100"}}), &page)
	if page.Total != 20 || len(page.Rows) != 20 {
		t.Fatalf("jsonl hands: total %d rows %d, want 20/20", page.Total, len(page.Rows))
	}
	for _, r := range page.Rows {
		if len(r.Pocket) != 2 || len(r.Board) != 5 || r.HandType == "" {
			t.Errorf("malformed jsonl row %+v", r)
		}
	}
}

func TestCreateRunFlow(t *testing.T) {
	srv, _ := setupServer(t)
	body := `{"players":2,"runs":30,"chunk":10,"format":"csv","seed":5}`
	status, data := request(t, srv.Client(), http.MethodPost, srv.URL+"/api/runs", body)
	if status != http.StatusAccepted {
		t.Fatalf("POST /api/runs: status %d, body %s", status, data)
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		t.Fatal(err)
	}
	if job.ID == "" || job.Status != jobRunning {
		t.Fatalf("job = %+v, want id set and status running", job)
	}

	deadline := time.Now().Add(20 * time.Second)
	for {
		var j Job
		getJSON(t, srv, srv.URL+"/api/jobs/"+job.ID, &j)
		if j.Status == jobDone || j.Status == jobFailed {
			if j.Status != jobDone {
				t.Fatalf("job failed: %s", j.Error)
			}
			if j.RunID == "" {
				t.Fatal("done job has empty run_id")
			}
			job = j
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %s did not finish in time (status %s)", job.ID, j.Status)
		}
		time.Sleep(50 * time.Millisecond)
	}

	var list struct {
		Runs []RunInfo `json:"runs"`
	}
	getJSON(t, srv, srv.URL+"/api/runs", &list)
	found := false
	for _, r := range list.Runs {
		if r.ID == job.RunID {
			found = true
			if r.Status != sim.StatusDone || r.Players != 2 || r.Runs != 30 {
				t.Errorf("listed run = %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("run %s not in /api/runs list: %+v", job.RunID, list.Runs)
	}

	var page struct {
		Total int              `json:"total"`
		Rows  []sim.HandResult `json:"rows"`
	}
	getJSON(t, srv, runURL(srv, job.RunID, "hands", url.Values{"limit": {"1000"}}), &page)
	if page.Total != 60 {
		t.Errorf("new run hands total = %d, want 60", page.Total)
	}

	var jobs struct {
		Jobs []Job `json:"jobs"`
	}
	getJSON(t, srv, srv.URL+"/api/jobs", &jobs)
	if len(jobs.Jobs) != 1 {
		t.Errorf("jobs list = %d, want 1", len(jobs.Jobs))
	}
}

func TestCreateRunValidation(t *testing.T) {
	srv, _ := setupServer(t)
	bodies := []string{
		`{}`,
		`{"players":1,"runs":10}`,
		`{"players":2,"runs":0}`,
		`{"players":2,"runs":10,"format":"xml"}`,
		`{"players":11,"runs":10}`,
		`not json`,
	}
	for _, b := range bodies {
		status, data := request(t, srv.Client(), http.MethodPost, srv.URL+"/api/runs", b)
		if status != http.StatusBadRequest {
			t.Errorf("POST %s: status %d, want 400 (body %s)", b, status, data)
		}
	}
	var jobs struct {
		Jobs []Job `json:"jobs"`
	}
	getJSON(t, srv, srv.URL+"/api/jobs", &jobs)
	if len(jobs.Jobs) != 0 {
		t.Errorf("invalid requests created %d jobs, want 0", len(jobs.Jobs))
	}
}

func TestDeleteRun(t *testing.T) {
	srv, dir := setupServer(t)
	runID := seedRun(t, dir, sim.Config{Players: 2, Runs: 10, Workers: 1, ChunkSize: 100, Format: sim.FormatCSV, Seed: 3})

	status, _ := request(t, srv.Client(), http.MethodDelete, runURL(srv, runID, "", nil), "")
	if status != http.StatusNoContent {
		t.Fatalf("DELETE: status %d, want 204", status)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(runID))); !os.IsNotExist(err) {
		t.Error("run dir still exists after DELETE")
	}
	status, body := request(t, srv.Client(), http.MethodGet, runURL(srv, runID, "", nil), "")
	if status != http.StatusNotFound {
		t.Errorf("GET after DELETE: status %d, want 404 (body %s)", status, body)
	}
}

func TestRunNotFound(t *testing.T) {
	srv, _ := setupServer(t)
	missing := "2020-01-01/[00:00:00]-2p-1r"
	for _, sub := range []string{"", "summary", "hands"} {
		status, _ := request(t, srv.Client(), http.MethodGet, runURL(srv, missing, sub, nil), "")
		if status != http.StatusNotFound {
			t.Errorf("GET missing run sub=%q: status %d, want 404", sub, status)
		}
	}
	status, _ := request(t, srv.Client(), http.MethodDelete, runURL(srv, missing, "", nil), "")
	if status != http.StatusNotFound {
		t.Errorf("DELETE missing run: status %d, want 404", status)
	}
}

func TestTraversalRejected(t *testing.T) {
	srv, dir := setupServer(t)
	client := noRedirectClient()
	paths := []string{
		"/api/runs/..%2F..%2F..%2Fetc%2Fpasswd",
		"/api/runs/..%2F..%2Fetc",
		"/api/runs/foo",
		"/api/runs/",
		"/api/runs/%2E%2E%2Fsecret",
	}
	for _, p := range paths {
		status, body := request(t, client, http.MethodGet, srv.URL+p, "")
		if status == http.StatusOK {
			t.Errorf("GET %s: status 200, body %s", p, body)
		}
		if strings.Contains(string(body), "root:") {
			t.Errorf("GET %s leaked file contents", p)
		}
		status, _ = request(t, client, http.MethodDelete, srv.URL+p, "")
		if status == http.StatusNoContent {
			t.Errorf("DELETE %s: status 204", p)
		}
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("results dir damaged: %v", err)
	}
}

func TestCORS(t *testing.T) {
	srv, _ := setupServer(t)

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/runs", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS: status %d, want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("ACAO = %q, want http://localhost:5173", got)
	}

	status, _ := request(t, srv.Client(), http.MethodGet, srv.URL+"/api/health", "")
	_ = status
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/health", nil)
	req.Header.Set("Origin", "http://127.0.0.1:4173")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4173" {
		t.Errorf("ACAO for 127.0.0.1:4173 = %q", got)
	}

	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/health", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO for external origin = %q, want empty", got)
	}
}

func TestSplitRunPath(t *testing.T) {
	cases := []struct {
		raw     string
		wantID  string
		wantSub string
		wantErr bool
	}{
		{"2026-01-01%2F%5B10%3A00%3A00%5D-6p-100r", "2026-01-01/[10:00:00]-6p-100r", "", false},
		{"2026-01-01%2F%5B10%3A00%3A00%5D-6p-100r%2Fhands", "2026-01-01/[10:00:00]-6p-100r", "hands", false},
		{"2026-01-01/[10:00:00]-6p-100r/summary", "2026-01-01/[10:00:00]-6p-100r", "summary", false},
		{"foo", "", "", true},
		{"", "", "", true},
		{"a/b/c", "", "", true},
	}
	for _, tc := range cases {
		id, sub, err := splitRunPath(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("splitRunPath(%q): expected error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitRunPath(%q): unexpected error %v", tc.raw, err)
			continue
		}
		if id != tc.wantID || sub != tc.wantSub {
			t.Errorf("splitRunPath(%q) = %q, %q; want %q, %q", tc.raw, id, sub, tc.wantID, tc.wantSub)
		}
	}
}
