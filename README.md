# go-poker-sim

A Go CLI that simulates all-in pre-flop poker hands: it deals hole cards to N
players, runs out a full 5-card board, evaluates every 7-card hand and records
the winner(s), including split pots. Hands are simulated concurrently across a
goroutine worker pool, and results are streamed to chunked CSV (or JSONL) files
plus a per-starting-pocket win-rate summary.

A React dashboard (`web/`) visualises the results and can start new
simulations through a small Go HTTP API (`sim/cmd/api`).

## Layout

- `sim/` — the Go module: CLI in `sim/cmd/go-poker-sim`, HTTP API in
  `sim/cmd/api`, library packages in `sim/internal/...`
- `web/` — React + Vite + Recharts dashboard (Vitest tests)
- `.github/workflows/ci.yml` — CI for both

## Quickstart

```sh
# 1. simulate from the CLI (optional — the dashboard can do this too)
cd sim
go run ./cmd/go-poker-sim --players 6 --runs 100000

# 2. start the API (loopback only) pointing at the same results dir
go run ./cmd/api -addr 127.0.0.1:8080 -results results

# 3. start the dashboard (proxies /api to 127.0.0.1:8080)
cd ../web
npm install
npm run dev    # open http://localhost:5173
```

The dashboard lists past runs, lets you start a new simulation (players, runs,
workers, chunk, format, seed, top-starting-hands), and opens any run in
detail: hand-category and top-pocket charts, the full starting-hand win-rate
table (search/sort/pagination) and a hand-by-hand explorer with filters
(run ID, seat, result, hand type) that renders pockets and boards as cards.

## Usage (CLI)

```sh
cd sim
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
| `-tsh` / `-top-starting-hands` | `0` (all) | Limit `summary.csv` to the top N starting hands by win% (max 1326) |

## Results format

Results land in `results/<date>/[<time>]-<players>p-<runs>r/` (with a `-2`,
`-3`, … suffix on collision):

- `manifest.json` — machine-readable run metadata: config (players, runs,
  workers, chunk size, format, seed, top-starting-hands), status
  (`running` → `done`/`failed`), start/finish timestamps, duration, hand and
  chunk counts. Written at the start of the run and updated at the end, so
  tooling never has to parse the directory name or stderr logs.
- `pre_flop_sim_chunk_<n>.csv` — one row per player per run:
  `run_id,seat,pocket,board,hand_type,won` (pocket/board are space-separated
  card tokens, e.g. `AH KD`). With `-format jsonl` the same records are written
  as newline-delimited JSON with keys `run_id,seat,pocket,board,hand_type,won`.
- `summary.csv` — aggregated per starting pocket (canonicalised, rank-desc):
  `pocket,hands,wins,win_pct`, sorted by win percentage descending. Tied hands
  count as a win for every player involved (split pot).
- `categories.csv` — hand-type frequency breakdown: `hand_type,hands,pct`.

At the end of a run the top 10 starting hands by win% are also logged to stderr.

Chunks are buffered in memory per chunk and written sorted by `(run_id, seat)`,
so output is byte-for-byte reproducible for a given `-seed`, independent of the
worker count (each run uses its own RNG seeded from `(seed, runID)`).

## HTTP API

`sim/cmd/api` exposes the results directory and the simulator over loopback
HTTP. Run IDs are `<date>/<run-dir>`, URL-encoded as a single path segment.

| Endpoint | Description |
|----------|-------------|
| `GET /api/health` | liveness probe |
| `GET /api/runs` | list all runs (manifest + chunk info), newest first |
| `GET /api/runs/{id}` | run detail: manifest + file listing |
| `GET /api/runs/{id}/summary` | starting-pocket rows: `pocket,hands,wins,win_pct` |
| `GET /api/runs/{id}/categories` | hand-type rows: `hand_type,hands,pct` |
| `GET /api/runs/{id}/chunks` | chunk files with index/format/size |
| `GET /api/runs/{id}/hands?run_id&seat&won&hand_type&offset&limit` | paginated per-hand rows (limit ≤ 1000, default 50) |
| `POST /api/runs` | start a simulation (JSON body like the CLI flags); returns `202` + job |
| `GET /api/jobs` / `GET /api/jobs/{id}` | job status; `run_id` is set when done |
| `DELETE /api/runs/{id}` | delete a run directory |

Example:

```sh
curl -X POST http://127.0.0.1:8080/api/runs \
  -H 'Content-Type: application/json' \
  -d '{"players":6,"runs":100000,"seed":42}'
curl 'http://127.0.0.1:8080/api/runs/2026-01-01%2F%5B10%3A00%3A00%5D-6p-100000r/hands?run_id=42'
```

CORS is restricted to localhost origins; the Vite dev server additionally
proxies `/api` to `http://127.0.0.1:8080`.

## How it works

- `sim/internal/card` — `Card{Rank, Suit}` with token parsing (`"10H"`).
- `sim/internal/deck` — 52-card deck, Fisher-Yates shuffle with an injected RNG,
  `Pop` for dealing.
- `sim/internal/board` — community cards with strict stage rules (flop = 3 on an
  empty board, then turn, then river); violations return errors.
- `sim/internal/player` — a seat holding up to 2 hole cards.
- `sim/internal/dealer` — round-robin hole-card dealing, flop/turn/river.
- `sim/internal/hand` — evaluates 5-7 cards into a comparable
  `Rank{Category, Tiebreak [5]int8}`. The wheel straight is stored 5-high with
  the ace low, flushes compare only their best 5 cards, and kickers are part of
  the tiebreak — so ties (split pots) are exactly `Rank` equality.
- `sim/internal/evaluator` — ranks all players' hands and returns the winning
  indexes (more than one = split pot).
- `sim/internal/sim` — worker pool: run IDs go through a jobs channel, each worker
  plays a hand with a per-run seeded `rand.PCG`, results flow to a single
  writer goroutine that aggregates stats and writes chunk files + summary +
  manifest.
- `sim/internal/api` — stdlib-only HTTP server over the results directory
  (discovery, detail, paginated hand queries, async run jobs).

## Testing

```sh
cd sim && go test ./...        # Go: unit + API tests (httptest)
cd sim && go test -race ./...  # with the race detector
cd web && npm test             # dashboard: vitest (api client, components, flows)
cd web && npm run build        # typecheck + production build
```

Table-driven unit tests cover every Go package: card parsing/validation, deck
integrity and shuffle properties, board stage rules, player limits, dealer
round-robin order and global card uniqueness, hand evaluation for all 10
categories plus edge cases (wheel vs 6-high straight, 6-card flush top-5,
two-pair kickers, double-trips full house), winner/split-pot logic, the sim
(determinism across worker counts, output file structure, manifest contents,
per-run winner correctness) and the HTTP API (run listing/detail, summary and
categories parsing, hands pagination/filters verified against the on-disk
files, the full create-run → poll-job → query-results flow, validation errors,
path-traversal rejection, CORS).
