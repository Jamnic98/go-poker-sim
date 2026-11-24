package sim

import (
	"encoding/csv"
	"encoding/json"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go-poker-sim/internal/card"
	"go-poker-sim/internal/evaluator"
	"go-poker-sim/internal/hand"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{Players: 2, Runs: 10, Workers: 2, ChunkSize: 5, OutDir: "out", Format: FormatCSV}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid config returned error: %v", err)
	}
	cases := []struct {
		name string
		mut  func(c *Config)
	}{
		{"players too few", func(c *Config) { c.Players = 1 }},
		{"players too many", func(c *Config) { c.Players = 11 }},
		{"zero runs", func(c *Config) { c.Runs = 0 }},
		{"negative runs", func(c *Config) { c.Runs = -5 }},
		{"zero workers", func(c *Config) { c.Workers = 0 }},
		{"zero chunk size", func(c *Config) { c.ChunkSize = 0 }},
		{"bad format", func(c *Config) { c.Format = "xml" }},
		{"empty out dir", func(c *Config) { c.OutDir = "" }},
		{"negative top starting hands", func(c *Config) { c.TopStartingHands = -1 }},
		{"too many top starting hands", func(c *Config) { c.TopStartingHands = MaxTopStartingHands + 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.mut(&c)
			if err := c.Validate(); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestPlayRunDeterministicForSameSeed(t *testing.T) {
	a, err := playRun(9, 4, rand.New(rand.NewPCG(5, 9)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := playRun(9, 4, rand.New(rand.NewPCG(5, 9)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("same seed+runID should produce identical results")
	}
	c, err := playRun(9, 4, rand.New(rand.NewPCG(6, 9)))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a, c) {
		t.Error("different seed should produce different results")
	}
}

func TestPlayRunStructureAndWinners(t *testing.T) {
	res, err := playRun(0, 6, rand.New(rand.NewPCG(42, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 6 {
		t.Fatalf("got %d results, want 6 (one per player)", len(res))
	}
	seen := make(map[string]bool)
	winners := 0
	var board []string
	for _, r := range res {
		if r.RunID != 0 {
			t.Errorf("RunID = %d, want 0", r.RunID)
		}
		if len(r.Pocket) != 2 {
			t.Errorf("seat %d pocket len = %d, want 2", r.Seat, len(r.Pocket))
		}
		if len(r.Board) != 5 {
			t.Errorf("seat %d board len = %d, want 5", r.Seat, len(r.Board))
		}
		if r.HandType == "" {
			t.Errorf("seat %d has empty hand type", r.Seat)
		}
		for _, tok := range r.Pocket {
			if seen[tok] {
				t.Errorf("duplicate card %s within a single run", tok)
			}
			seen[tok] = true
		}
		if board == nil {
			board = r.Board
			for _, tok := range r.Board {
				if seen[tok] {
					t.Errorf("board card %s duplicates a hole card", tok)
				}
				seen[tok] = true
			}
		} else if strings.Join(r.Board, " ") != strings.Join(board, " ") {
			t.Errorf("seat %d board = %v, want %v (shared board)", r.Seat, r.Board, board)
		}
		if r.Won {
			winners++
		}
	}
	if len(seen) != 6*2+5 {
		t.Errorf("run dealt %d unique cards, want %d", len(seen), 6*2+5)
	}
	if winners < 1 {
		t.Error("expected at least one winner per run")
	}

	// Cross-check the Won flags against an independent evaluation.
	ranks := make([]hand.Rank, len(res))
	for i, r := range res {
		cards := make([]card.Card, 0, 7)
		for _, tok := range append(append([]string{}, r.Pocket...), r.Board...) {
			c, err := card.Parse(tok)
			if err != nil {
				t.Fatal(err)
			}
			cards = append(cards, c)
		}
		rk, err := hand.Evaluate(cards)
		if err != nil {
			t.Fatal(err)
		}
		ranks[i] = rk
	}
	winIdx := evaluator.Winners(ranks)
	winSet := make(map[int]bool)
	for _, w := range winIdx {
		winSet[w] = true
	}
	for i, r := range res {
		if r.Won != winSet[i] {
			t.Errorf("seat %d Won = %v, evaluator says %v", i, r.Won, winSet[i])
		}
	}
}

func readCSVFile(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return rows
}

func TestRunCSVOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Players: 3, Runs: 250, Workers: 4, ChunkSize: 100, OutDir: dir, Format: FormatCSV, Seed: 123}
	stats, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if stats.Runs != 250 || stats.Hands != 750 {
		t.Errorf("stats = %d runs / %d hands, want 250 / 750", stats.Runs, stats.Hands)
	}

	entries, err := os.ReadDir(stats.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	var chunkFiles, summaryFiles []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "pre_flop_sim_chunk_") {
			chunkFiles = append(chunkFiles, e.Name())
		}
		if e.Name() == "summary.csv" {
			summaryFiles = append(summaryFiles, e.Name())
		}
	}
	if len(chunkFiles) != 3 {
		t.Errorf("got %d chunk files, want 3 (250 runs / chunk 100)", len(chunkFiles))
	}
	if len(summaryFiles) != 1 {
		t.Errorf("got %d summary files, want 1", len(summaryFiles))
	}

	type row struct {
		runID, seat int
		pocket      []string
		board       []string
		handType    string
		won         bool
	}
	var rows []row
	perRun := make(map[int][]row)
	for _, name := range chunkFiles {
		data := readCSVFile(t, filepath.Join(stats.OutDir, name))
		if len(data) == 0 {
			t.Fatalf("chunk %s is empty", name)
		}
		header := data[0]
		wantHeader := []string{"run_id", "seat", "pocket", "board", "hand_type", "won"}
		if !reflect.DeepEqual(header, wantHeader) {
			t.Fatalf("chunk %s header = %v, want %v", name, header, wantHeader)
		}
		for _, rec := range data[1:] {
			runID, err := strconv.Atoi(rec[0])
			if err != nil {
				t.Fatalf("bad run_id %q: %v", rec[0], err)
			}
			seat, err := strconv.Atoi(rec[1])
			if err != nil {
				t.Fatalf("bad seat %q: %v", rec[1], err)
			}
			won, err := strconv.ParseBool(rec[5])
			if err != nil {
				t.Fatalf("bad won %q: %v", rec[5], err)
			}
			r := row{runID: runID, seat: seat, pocket: strings.Fields(rec[2]), board: strings.Fields(rec[3]), handType: rec[4], won: won}
			rows = append(rows, r)
			perRun[runID] = append(perRun[runID], r)
		}
	}
	if len(rows) != 750 {
		t.Errorf("total result rows = %d, want 750", len(rows))
	}
	if len(perRun) != 250 {
		t.Errorf("distinct run IDs = %d, want 250", len(perRun))
	}
	for runID, rs := range perRun {
		if len(rs) != 3 {
			t.Errorf("run %d has %d rows, want 3", runID, len(rs))
		}
		seen := make(map[string]bool)
		winners := 0
		board := rs[0].board
		for _, r := range rs {
			for _, tok := range r.pocket {
				if seen[tok] {
					t.Errorf("run %d: duplicate card %s", runID, tok)
				}
				seen[tok] = true
			}
			if strings.Join(r.board, " ") != strings.Join(board, " ") {
				t.Errorf("run %d: seat %d board differs from seat 0", runID, r.seat)
			}
			if r.won {
				winners++
			}
		}
		for _, tok := range board {
			if seen[tok] {
				t.Errorf("run %d: board card %s duplicates a hole card", runID, tok)
			}
			seen[tok] = true
		}
		if len(seen) != 3*2+5 {
			t.Errorf("run %d has %d unique cards, want 11", runID, len(seen))
		}
		if winners < 1 {
			t.Errorf("run %d has no winner", runID)
		}
	}

	summary := readCSVFile(t, filepath.Join(stats.OutDir, "summary.csv"))
	if len(summary) < 2 {
		t.Fatal("summary.csv has no data rows")
	}
	if !reflect.DeepEqual(summary[0], []string{"pocket", "hands", "wins", "win_pct"}) {
		t.Fatalf("summary header = %v", summary[0])
	}
	totalHands := 0
	for _, rec := range summary[1:] {
		pocket := strings.Fields(rec[0])
		if len(pocket) != 2 {
			t.Errorf("summary pocket %q is not 2 cards", rec[0])
		}
		hands, err := strconv.Atoi(rec[1])
		if err != nil {
			t.Fatal(err)
		}
		wins, err := strconv.Atoi(rec[2])
		if err != nil {
			t.Fatal(err)
		}
		if wins > hands {
			t.Errorf("pocket %s: wins %d > hands %d", rec[0], wins, hands)
		}
		totalHands += hands
	}
	if totalHands != 750 {
		t.Errorf("summary total hands = %d, want 750", totalHands)
	}
}

func TestRunJSONLOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Players: 2, Runs: 30, Workers: 3, ChunkSize: 1000, OutDir: dir, Format: FormatJSONL, Seed: 9}
	stats, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	chunkPath := filepath.Join(stats.OutDir, "pre_flop_sim_chunk_0.jsonl")
	f, err := os.Open(chunkPath)
	if err != nil {
		t.Fatalf("open chunk: %v", err)
	}
	defer f.Close()
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 60 {
		t.Fatalf("jsonl has %d lines, want 60", len(lines))
	}
	for i, line := range lines {
		var r HandResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line %d invalid JSON: %v", i, err)
		}
		if len(r.Pocket) != 2 || len(r.Board) != 5 || r.HandType == "" {
			t.Errorf("line %d malformed result: %+v", i, r)
		}
	}
}

func TestRunDeterministicAcrossWorkerCounts(t *testing.T) {
	run := func(workers int) map[string]*PocketStat {
		cfg := Config{Players: 5, Runs: 120, Workers: workers, ChunkSize: 50, OutDir: t.TempDir(), Format: FormatCSV, Seed: 777}
		stats, err := Run(cfg)
		if err != nil {
			t.Fatalf("Run(workers=%d): %v", workers, err)
		}
		return stats.Pockets
	}
	s1 := run(1)
	s8 := run(8)
	if !reflect.DeepEqual(s1, s8) {
		t.Error("same seed should give identical pocket stats regardless of worker count")
	}
}

func TestRunInvalidConfig(t *testing.T) {
	if _, err := Run(Config{Players: 1, Runs: 10, Workers: 1, ChunkSize: 5, OutDir: t.TempDir(), Format: FormatCSV}); err == nil {
		t.Error("Run with 1 player expected error, got nil")
	}
	if _, err := Run(Config{Players: 2, Runs: 0, Workers: 1, ChunkSize: 5, OutDir: t.TempDir(), Format: FormatCSV}); err == nil {
		t.Error("Run with 0 runs expected error, got nil")
	}
	if _, err := Run(Config{Players: 2, Runs: 10, Workers: 1, ChunkSize: 5, OutDir: t.TempDir(), Format: "xml"}); err == nil {
		t.Error("Run with bad format expected error, got nil")
	}
}

func TestRunTopStartingHandsAndCategories(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Players: 2, Runs: 200, Workers: 3, ChunkSize: 100, OutDir: dir, Format: FormatCSV, Seed: 31, TopStartingHands: 5}
	stats, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	summary := readCSVFile(t, filepath.Join(stats.OutDir, "summary.csv"))
	if len(summary) != 1+5 {
		t.Errorf("summary.csv has %d rows, want header + 5 (tsh cap)", len(summary))
	}
	// win_pct must be non-increasing down the file
	for i := 2; i < len(summary); i++ {
		prev, err := strconv.ParseFloat(summary[i-1][3], 64)
		if err != nil {
			t.Fatal(err)
		}
		cur, err := strconv.ParseFloat(summary[i][3], 64)
		if err != nil {
			t.Fatal(err)
		}
		if cur > prev {
			t.Errorf("summary not sorted by win_pct desc: row %d %.4f > row %d %.4f", i, cur, i-1, prev)
		}
	}

	categories := readCSVFile(t, filepath.Join(stats.OutDir, "categories.csv"))
	if len(categories) < 2 {
		t.Fatal("categories.csv has no data rows")
	}
	if !reflect.DeepEqual(categories[0], []string{"hand_type", "hands", "pct"}) {
		t.Fatalf("categories header = %v", categories[0])
	}
	total := 0
	statsTotal := 0
	for _, rec := range categories[1:] {
		n, err := strconv.Atoi(rec[1])
		if err != nil {
			t.Fatal(err)
		}
		total += n
	}
	for _, n := range stats.Categories {
		statsTotal += n
	}
	if total != 400 || statsTotal != 400 {
		t.Errorf("categories totals: file=%d stats=%d, want 400 (200 runs x 2 players)", total, statsTotal)
	}
}

func TestRunWritesManifest(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Players: 3, Runs: 40, Workers: 2, ChunkSize: 20, OutDir: dir, Format: FormatCSV, Seed: 123, TopStartingHands: 10}
	stats, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(stats.OutDir, ManifestFileName))
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if m.Version != ManifestVersion {
		t.Errorf("manifest version = %d, want %d", m.Version, ManifestVersion)
	}
	if m.Status != StatusDone {
		t.Errorf("manifest status = %q, want %q", m.Status, StatusDone)
	}
	if m.Players != 3 || m.Runs != 40 || m.Workers != 2 || m.ChunkSize != 20 {
		t.Errorf("manifest config = %+v, want players=3 runs=40 workers=2 chunk=20", m)
	}
	if m.Format != FormatCSV || m.Seed != 123 || m.TopStartingHands != 10 {
		t.Errorf("manifest format/seed/tsh = %q/%d/%d, want csv/123/10", m.Format, m.Seed, m.TopStartingHands)
	}
	if m.Hands != 120 {
		t.Errorf("manifest hands = %d, want 120", m.Hands)
	}
	if m.Chunks != 2 {
		t.Errorf("manifest chunks = %d, want 2 (40 runs / chunk 20)", m.Chunks)
	}
	if m.Error != "" {
		t.Errorf("manifest error = %q, want empty", m.Error)
	}
	if m.StartedAt.IsZero() || m.FinishedAt.Before(m.StartedAt) {
		t.Errorf("manifest times: started=%v finished=%v", m.StartedAt, m.FinishedAt)
	}
}

func TestRunManifestRecordsEffectiveSeed(t *testing.T) {
	cfg := Config{Players: 2, Runs: 10, Workers: 1, ChunkSize: 100, OutDir: t.TempDir(), Format: FormatCSV, Seed: 0}
	stats, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(stats.OutDir, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Seed == 0 {
		t.Error("manifest seed = 0, want the effective (time-based) seed to be recorded")
	}
	if stats.Seed != m.Seed {
		t.Errorf("stats seed %d != manifest seed %d", stats.Seed, m.Seed)
	}
}

func TestPocketKeyCanonicalisesOrder(t *testing.T) {
	if got := pocketKey([]string{"KD", "AH"}); got != "AH KD" {
		t.Errorf("pocketKey([KD AH]) = %q, want %q", got, "AH KD")
	}
	if got := pocketKey([]string{"AH", "KD"}); got != "AH KD" {
		t.Errorf("pocketKey([AH KD]) = %q, want %q", got, "AH KD")
	}
	if got := pocketKey([]string{"2H", "2C"}); got != "2C 2H" {
		t.Errorf("pocketKey([2H 2C]) = %q, want %q (suit tiebreak)", got, "2C 2H")
	}
}
