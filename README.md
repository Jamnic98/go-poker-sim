# go-poker-sim

A Go CLI that simulates all-in pre-flop poker hands: it deals hole cards to N
players, runs out a full 5-card board, evaluates every 7-card hand and records
the winner(s), including split pots. Hands are simulated concurrently across a
goroutine worker pool, and results are streamed to chunked CSV (or JSONL) files
plus a per-starting-pocket win-rate summary.

## Usage

```sh
go run ./cmd/go-poker-sim --players 6 --runs 100000
# or build a binary
go build -o go-poker-sim ./cmd/go-poker-sim
./go-poker-sim -players 6 -runs 100000 -workers 8 -format csv -seed 42
```

### Flags

| Flag        | Default          | Description                                          |
|-------------|------------------|------------------------------------------------------|
| `-players`  | `2`              | Number of players at the table (2-10)                |
| `-runs`     | required         | Number of hands to simulate (> 0)                    |
| `-workers`  | `runtime.NumCPU()` | Number of concurrent worker goroutines              |
| `-chunk`    | `10000`          | Number of runs per output chunk file                 |
| `-out`      | `results`        | Output directory                                     |
| `-format`   | `csv`            | Output format: `csv` or `jsonl`                      |
| `-seed`     | `0` (random)     | RNG seed; set it for reproducible runs               |

## Output

Results land in `results/<date>/[<time>]-<players>p-<runs>r/`:

- `pre_flop_sim_chunk_<n>.csv` — one row per player per run:
  `run_id,seat,pocket,board,hand_type,won` (pocket/board are space-separated
  card tokens, e.g. `AH KD`). With `-format jsonl` the same records are written
  as newline-delimited JSON.
- `summary.csv` — aggregated per starting pocket (canonicalised, rank-desc):
  `pocket,hands,wins,win_pct`, sorted by win percentage descending. Tied hands
  count as a win for every player involved (split pot).

Chunks are buffered in memory per chunk and written sorted by `(run_id, seat)`,
so output is byte-for-byte reproducible for a given `-seed`, independent of the
worker count (each run uses its own RNG seeded from `(seed, runID)`).

## How it works

- `internal/card` — `Card{Rank, Suit}` with token parsing (`"10H"`).
- `internal/deck` — 52-card deck, Fisher-Yates shuffle with an injected RNG,
  `Pop` for dealing.
- `internal/board` — community cards with strict stage rules (flop = 3 on an
  empty board, then turn, then river); violations return errors.
- `internal/player` — a seat holding up to 2 hole cards.
- `internal/dealer` — round-robin hole-card dealing, flop/turn/river.
- `internal/hand` — evaluates 5-7 cards into a comparable
  `Rank{Category, Tiebreak [5]int8}`. The wheel straight is stored 5-high with
  the ace low, flushes compare only their best 5 cards, and kickers are part of
  the tiebreak — so ties (split pots) are exactly `Rank` equality.
- `internal/evaluator` — ranks all players' hands and returns the winning
  indexes (more than one = split pot).
- `internal/sim` — worker pool: run IDs go through a jobs channel, each worker
  plays a hand with a per-run seeded `rand.PCG`, results flow to a single
  writer goroutine that aggregates stats and writes chunk files + summary.

## Testing

```sh
go test ./...
```

Table-driven unit tests cover every package: card parsing/validation, deck
integrity and shuffle properties, board stage rules, player limits, dealer
round-robin order and global card uniqueness, hand evaluation for all 10
categories plus edge cases (wheel vs 6-high straight, 6-card flush top-5,
two-pair kickers, double-trips full house), winner/split-pot logic, and the sim
(determinism across worker counts, output file structure, per-run winner
correctness).
