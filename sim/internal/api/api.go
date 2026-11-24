// Package api exposes the go-poker-sim results directory and simulator over
// HTTP so the web dashboard can list past runs, inspect them in detail and
// start new simulations. It only uses the standard library and binds to
// loopback by default (see cmd/api).
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"go-poker-sim/internal/sim"
)

const (
	jobRunning = "running"
	jobDone    = "done"
	jobFailed  = "failed"

	defaultHandsLimit = 50
	maxHandsLimit     = 1000
)

// RunRequest is the JSON body of POST /api/runs. Zero-valued optional fields
// fall back to the same defaults as the CLI.
type RunRequest struct {
	Players          int    `json:"players"`
	Runs             int    `json:"runs"`
	Workers          int    `json:"workers,omitempty"`
	Chunk            int    `json:"chunk,omitempty"`
	Format           string `json:"format,omitempty"`
	Seed             uint64 `json:"seed,omitempty"`
	TopStartingHands int    `json:"top_starting_hands,omitempty"`
}

// Job tracks an asynchronous simulation started via POST /api/runs.
type Job struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	Request    RunRequest `json:"request"`
	RunID      string     `json:"run_id,omitempty"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt time.Time  `json:"finished_at,omitempty"`
}

type Server struct {
	resultsDir string

	mu   sync.Mutex
	jobs map[string]*Job
}

// NewHandler builds the API handler rooted at resultsDir (created if missing).
func NewHandler(resultsDir string) (http.Handler, error) {
	abs, err := filepath.Abs(resultsDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("creating results dir: %w", err)
	}
	s := &Server{resultsDir: abs, jobs: make(map[string]*Job)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("POST /api/runs", s.handleCreateRun)
	mux.HandleFunc("GET /api/jobs", s.handleListJobs)
	mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
	mux.HandleFunc("GET /api/runs/", s.handleRunSubtree)
	mux.HandleFunc("DELETE /api/runs/", s.handleRunSubtree)
	return s.cors(mux), nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.listRunInfos()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	cfg := sim.Config{
		Players:          req.Players,
		Runs:             req.Runs,
		Workers:          req.Workers,
		ChunkSize:        req.Chunk,
		OutDir:           s.resultsDir,
		Format:           req.Format,
		Seed:             req.Seed,
		TopStartingHands: req.TopStartingHands,
	}
	if cfg.Workers == 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = sim.DefaultChunkSize
	}
	if cfg.Format == "" {
		cfg.Format = sim.FormatCSV
	}
	if err := cfg.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid configuration: "+err.Error())
		return
	}

	job := &Job{
		ID:        newJobID(),
		Status:    jobRunning,
		Request:   req,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	go s.execute(job, cfg)
	log.Printf("api: started job %s (%d runs, %d players, format=%s, seed=%d)", job.ID, cfg.Runs, cfg.Players, cfg.Format, cfg.Seed)
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) execute(job *Job, cfg sim.Config) {
	stats, err := sim.Run(cfg)

	s.mu.Lock()
	defer s.mu.Unlock()
	job.FinishedAt = time.Now()
	if err != nil {
		job.Status = jobFailed
		job.Error = err.Error()
		log.Printf("api: job %s failed: %v", job.ID, err)
		return
	}
	job.Status = jobDone
	if rel, relErr := filepath.Rel(s.resultsDir, stats.OutDir); relErr == nil {
		job.RunID = filepath.ToSlash(rel)
	}
	log.Printf("api: job %s done -> %s", job.ID, job.RunID)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	s.mu.Unlock()
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.After(jobs[j].CreatedAt) })
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.Lock()
	job, ok := s.jobs[id]
	s.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "unknown job id "+id)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// handleRunSubtree serves GET/DELETE /api/runs/<date>/<dir>[/summary|/categories|/chunks|/hands].
// The run id is the run directory relative to the results root, URL-encoded as
// a single path segment by clients (e.g. "2026-01-01%2F%5B12%3A00%3A00%5D-6p-100r").
func (s *Server) handleRunSubtree(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.EscapedPath(), "/api/runs/")
	id, sub, err := splitRunPath(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dir, err := s.resolveRunDir(id)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "run not found: "+id)
		} else {
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		switch sub {
		case "":
			s.handleRunDetail(w, r, id, dir)
		case "summary":
			s.handleSummary(w, r, dir)
		case "categories":
			s.handleCategories(w, r, dir)
		case "chunks":
			s.handleChunks(w, r, id, dir)
		case "hands":
			s.handleHands(w, r, dir)
		default:
			writeError(w, http.StatusNotFound, "unknown sub-resource "+sub)
		}
	case http.MethodDelete:
		if sub != "" {
			writeError(w, http.StatusNotFound, "unknown sub-resource "+sub)
			return
		}
		if err := os.RemoveAll(dir); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("api: deleted run %s", id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// splitRunPath decodes the escaped remainder of /api/runs/ into a run id
// ("<date>/<dir>") and an optional sub-resource suffix.
func splitRunPath(raw string) (id, sub string, err error) {
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid escaped path: %v", err)
	}
	parts := strings.Split(strings.Trim(decoded, "/"), "/")
	if len(parts) > 0 {
		switch parts[len(parts)-1] {
		case "summary", "categories", "chunks", "hands":
			sub = parts[len(parts)-1]
			parts = parts[:len(parts)-1]
		}
	}
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("run id must have the form <date>/<run-dir>, got %q", decoded)
	}
	return parts[0] + "/" + parts[1], sub, nil
}

// resolveRunDir maps a run id to an absolute directory inside the results
// root, rejecting anything that escapes it.
func (s *Server) resolveRunDir(id string) (string, error) {
	for _, p := range strings.Split(id, "/") {
		if p == "" || p == "." || p == ".." || strings.ContainsAny(p, `\`) {
			return "", fmt.Errorf("invalid run id %q", id)
		}
	}
	dir := filepath.Join(s.resultsDir, filepath.FromSlash(id))
	if dir != s.resultsDir && !strings.HasPrefix(dir, s.resultsDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("run id %q escapes the results directory", id)
	}
	st, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%q is not a run directory", id)
	}
	return dir, nil
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func newJobID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: encoding response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
