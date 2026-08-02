package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	maxRequestBody = 10 << 20 // 10 MB
	maxTests       = 100
	maxRatePerMin  = 30
)

// Server handles HTTP requests from competitive-companion.
type Server struct {
	httpServer *http.Server
	samplesDir string
	autoRunBin string
	secret     string
	onProblem  func(count int)
	mu         sync.Mutex
	lastReq    time.Time
	reqCount   int
}

// NewServer creates a new server with the given configuration.
func NewServer(samplesDir, autoRunBin, secret string) *Server {
	return &Server{
		samplesDir: samplesDir,
		autoRunBin: autoRunBin,
		secret:     secret,
	}
}

// SetOnProblem registers a callback invoked after a problem's samples are
// saved successfully. The callback must not block (use a buffered channel).
// Used by `cpt test` wait mode to wake up once a problem arrives.
func (s *Server) SetOnProblem(cb func(count int)) {
	s.onProblem = cb
}

// Start begins listening on the given host:port.
func (s *Server) Start(host string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRequest)

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", host, port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Only respond to exact "/" path
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content-Type check (CSRF protection)
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Shared secret authentication
	if s.secret != "" {
		if r.Header.Get("X-CPT-Secret") != s.secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Rate limiting (per-minute window)
	if !s.checkRateLimit() {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request body too large or unreadable", http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Sanitize task metadata for terminal printing
	taskName := sanitize(task.Name)
	taskGroup := task.Group
	if taskGroup == "" {
		taskGroup = task.URL
	}
	taskGroup = sanitize(taskGroup)

	fmt.Printf("📥 Received: %s — %s\n", taskName, taskGroup)

	// Serialize save + auto-run to prevent races
	s.mu.Lock()
	defer s.mu.Unlock()

	// Save samples with capped test count
	count, err := SaveSamples(task, s.samplesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save samples error: %v\n", err)
		http.Error(w, "Failed to save samples", http.StatusInternalServerError)
		return
	}

	fmt.Printf("   Saved %d samples to %s/\n", count, s.samplesDir)

	// Notify a waiting client (e.g. `cpt test` wait mode)
	if s.onProblem != nil {
		s.onProblem(count)
	}

	// Auto-run asynchronously if configured
	if s.autoRunBin != "" {
		fmt.Println()
		go func() {
			if err := RunAll(s.autoRunBin, s.samplesDir, 2, 0); err != nil {
				fmt.Printf("   Auto-run error: %v\n", err)
			}
		}()
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"samples": count,
	})
}

// checkRateLimit implements a simple per-minute rate limiter.
func (s *Server) checkRateLimit() bool {
	now := time.Now()
	if now.Sub(s.lastReq) > time.Minute {
		s.reqCount = 0
		s.lastReq = now
	}
	s.reqCount++
	return s.reqCount <= maxRatePerMin
}

// sanitize strips non-printable characters and ANSI escape sequences.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) && r != '\x1b' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
