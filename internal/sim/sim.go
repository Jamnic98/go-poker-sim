package sim

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-poker-sim/internal/board"
	"go-poker-sim/internal/card"
	"go-poker-sim/internal/dealer"
	"go-poker-sim/internal/deck"
	"go-poker-sim/internal/evaluator"
	"go-poker-sim/internal/hand"
	"go-poker-sim/internal/player"
)

const (
	MinPlayers       = 2
	MaxPlayers       = 10
	DefaultChunkSize = 10_000
	FormatCSV        = "csv"
	FormatJSONL      = "jsonl"
)

type Config struct {
	Players   int
	Runs      int
	Workers   int
	ChunkSize int
	OutDir    string
	Format    string
	Seed      uint64 // 0 means random (time-based)
}

func (c Config) Validate() error {
	if c.Players < MinPlayers || c.Players > MaxPlayers {
		return fmt.Errorf("players must be between %d and %d, got %d", MinPlayers, MaxPlayers, c.Players)
	}
	if c.Runs <= 0 {
		return errors.New("runs must be greater than 0")
	}
	if c.Workers <= 0 {
		return errors.New("workers must be greater than 0")
	}
	if c.ChunkSize <= 0 {
		return errors.New("chunk size must be greater than 0")
	}
	if c.Format != FormatCSV && c.Format != FormatJSONL {
		return fmt.Errorf("format must be %q or %q, got %q", FormatCSV, FormatJSONL, c.Format)
	}
	if c.OutDir == "" {
		return errors.New("out dir must not be empty")
	}
	return nil
}

// HandResult is the per-player outcome of one simulated run.
type HandResult struct {
	RunID    int      `json:"run_id"`
	Seat     int      `json:"seat"`
	Pocket   []string `json:"pocket"`
	Board    []string `json:"board"`
	HandType string   `json:"hand_type"`
	Won      bool     `json:"won"`
}

type PocketStat struct {
	Pocket string
	Hands  int
	Wins   int
}

func (p PocketStat) WinPct() float64 {
	if p.Hands == 0 {
		return 0
	}
	return 100 * float64(p.Wins) / float64(p.Hands)
}

type Stats struct {
	Runs     int
	Hands    int
	Duration time.Duration
	OutDir   string
	Seed     uint64
	Pockets  map[string]*PocketStat
}

type runOutput struct {
	results []HandResult
	err     error
}

// playRun simulates one all-in pre-flop hand: shuffle, deal hole cards to all
// players, deal a full 5-card board, evaluate every 7-card hand and mark the
// winner(s). The RNG is seeded per run so results are reproducible regardless
// of which worker executes the run.
func playRun(runID, players int, rng *rand.Rand) ([]HandResult, error) {
	d := deck.New()
	d.Shuffle(rng)
	dl := dealer.New(d)

	playersSlice := make([]*player.Player, players)
	for i := range playersSlice {
		playersSlice[i] = player.New(i)
	}
	if err := dl.DealHoleCards(playersSlice); err != nil {
		return nil, err
	}
	b := board.New()
	if err := dl.DealFlop(b); err != nil {
		return nil, err
	}
	if err := dl.DealTurn(b); err != nil {
		return nil, err
	}
	if err := dl.DealRiver(b); err != nil {
		return nil, err
	}

	boardCards := b.Cards()
	boardTokens := make([]string, len(boardCards))
	for i, c := range boardCards {
		boardTokens[i] = c.Token()
	}

	ranks := make([]hand.Rank, len(playersSlice))
	for i, p := range playersSlice {
		r, err := hand.Evaluate(append(p.Pocket(), boardCards...))
		if err != nil {
			return nil, fmt.Errorf("evaluating seat %d: %w", i, err)
		}
		ranks[i] = r
	}

	winSet := make(map[int]bool)
	for _, w := range evaluator.Winners(ranks) {
		winSet[w] = true
	}

	results := make([]HandResult, len(playersSlice))
	for i, p := range playersSlice {
		pocket := p.Pocket()
		results[i] = HandResult{
			RunID:    runID,
			Seat:     i,
			Pocket:   []string{pocket[0].Token(), pocket[1].Token()},
			Board:    boardTokens,
			HandType: ranks[i].Category.String(),
			Won:      winSet[i],
		}
	}
	return results, nil
}

// Run executes cfg.Runs simulated hands across cfg.Workers goroutines and
// writes chunked per-hand results plus a per-starting-pocket summary.
func Run(cfg Config) (*Stats, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	seed := cfg.Seed
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	now := time.Now()
	runDir := filepath.Join(cfg.OutDir, now.Format("2006-01-02"),
		fmt.Sprintf("[%s]-%dp-%dr", now.Format("15:04:05"), cfg.Players, cfg.Runs))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	start := time.Now()
	stats := &Stats{
		Runs:    cfg.Runs,
		Hands:   cfg.Runs * cfg.Players,
		OutDir:  runDir,
		Seed:    seed,
		Pockets: make(map[string]*PocketStat),
	}

	jobs := make(chan int, cfg.Workers)
	results := make(chan runOutput, cfg.Workers)

	var wg sync.WaitGroup
	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for runID := range jobs {
				rng := rand.New(rand.NewPCG(seed, uint64(runID)))
				res, err := playRun(runID, cfg.Players, rng)
				results <- runOutput{results: res, err: err}
			}
		}()
	}
	go func() {
		for runID := 0; runID < cfg.Runs; runID++ {
			jobs <- runID
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	if err := collect(cfg, runDir, results, stats); err != nil {
		return nil, err
	}
	stats.Duration = time.Since(start)
	return stats, nil
}

func collect(cfg Config, runDir string, results <-chan runOutput, stats *Stats) error {
	var firstErr error
	runsDone := 0
	chunkIdx := 0
	var chunk []HandResult

	flushChunk := func() error {
		if len(chunk) == 0 {
			return nil
		}
		sort.Slice(chunk, func(i, j int) bool {
			if chunk[i].RunID != chunk[j].RunID {
				return chunk[i].RunID < chunk[j].RunID
			}
			return chunk[i].Seat < chunk[j].Seat
		})
		path := filepath.Join(runDir, fmt.Sprintf("pre_flop_sim_chunk_%d.%s", chunkIdx, cfg.Format))
		if err := writeChunk(path, cfg.Format, chunk); err != nil {
			return err
		}
		log.Printf("wrote chunk %d (%d runs) -> %s", chunkIdx, len(chunk)/cfg.Players, path)
		chunkIdx++
		chunk = nil
		return nil
	}

	for out := range results {
		if out.err != nil {
			if firstErr == nil {
				firstErr = out.err
			}
			continue
		}
		if runsDone > 0 && runsDone%cfg.ChunkSize == 0 {
			if err := flushChunk(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		chunk = append(chunk, out.results...)
		for _, r := range out.results {
			key := pocketKey(r.Pocket)
			st, ok := stats.Pockets[key]
			if !ok {
				st = &PocketStat{Pocket: key}
				stats.Pockets[key] = st
			}
			st.Hands++
			if r.Won {
				st.Wins++
			}
		}
		runsDone++
	}
	if err := flushChunk(); err != nil && firstErr == nil {
		firstErr = err
	}
	if firstErr != nil {
		return firstErr
	}
	return writeSummary(filepath.Join(runDir, "summary.csv"), stats)
}

func writeChunk(path, format string, rows []HandResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	if format == FormatCSV {
		w := csv.NewWriter(bw)
		if err := w.Write([]string{"run_id", "seat", "pocket", "board", "hand_type", "won"}); err != nil {
			return err
		}
		for _, r := range rows {
			if err := w.Write([]string{
				strconv.Itoa(r.RunID),
				strconv.Itoa(r.Seat),
				strings.Join(r.Pocket, " "),
				strings.Join(r.Board, " "),
				r.HandType,
				strconv.FormatBool(r.Won),
			}); err != nil {
				return err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return err
		}
	} else {
		for _, r := range rows {
			b, err := json.Marshal(r)
			if err != nil {
				return err
			}
			if _, err := bw.Write(append(b, '\n')); err != nil {
				return err
			}
		}
	}
	return bw.Flush()
}

func writeSummary(path string, stats *Stats) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	w := csv.NewWriter(bw)
	if err := w.Write([]string{"pocket", "hands", "wins", "win_pct"}); err != nil {
		return err
	}
	pockets := make([]*PocketStat, 0, len(stats.Pockets))
	for _, p := range stats.Pockets {
		pockets = append(pockets, p)
	}
	sort.Slice(pockets, func(i, j int) bool {
		pi, pj := pockets[i].WinPct(), pockets[j].WinPct()
		if pi != pj {
			return pi > pj
		}
		return pockets[i].Pocket < pockets[j].Pocket
	})
	for _, p := range pockets {
		if err := w.Write([]string{
			p.Pocket,
			strconv.Itoa(p.Hands),
			strconv.Itoa(p.Wins),
			strconv.FormatFloat(p.WinPct(), 'f', 4, 64),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return bw.Flush()
}

// pocketKey canonicalises a pocket (e.g. ["KD","AH"] -> "AH KD") so the same
// starting hand is aggregated together regardless of deal order.
func pocketKey(pocket []string) string {
	cards := make([]card.Card, 0, len(pocket))
	for _, tok := range pocket {
		if c, err := card.Parse(tok); err == nil {
			cards = append(cards, c)
		}
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Rank != cards[j].Rank {
			return cards[i].Rank > cards[j].Rank
		}
		return cards[i].Suit < cards[j].Suit
	})
	tokens := make([]string, len(cards))
	for i, c := range cards {
		tokens[i] = c.Token()
	}
	return strings.Join(tokens, " ")
}
