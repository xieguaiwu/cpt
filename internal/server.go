package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Server handles HTTP requests from competitive-companion.
type Server struct {
	httpServer *http.Server
	samplesDir string
	autoRunBin string
}

// NewServer creates a new server with the given configuration.
func NewServer(samplesDir, autoRunBin string) *Server {
	return &Server{
		samplesDir: samplesDir,
		autoRunBin: autoRunBin,
	}
}

// Start begins listening on the given address.
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
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
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Print received summary
	group := task.Group
	if group == "" {
		group = task.URL
	}
	fmt.Printf("📥 Received: %s — %s\n", task.Name, group)

	// Save samples
	count, err := SaveSamples(task, s.samplesDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save samples: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("   Saved %d samples to %s/\n", count, s.samplesDir)

	// Auto-run if configured
	if s.autoRunBin != "" {
		fmt.Println()
		if err := RunAll(s.autoRunBin, s.samplesDir, 2, 0); err != nil {
			fmt.Printf("   Auto-run error: %v\n", err)
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"samples": count,
	})
}


