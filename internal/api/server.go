// internal/api/server.go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"btcforce/internal/hoptracker"
	"btcforce/internal/tracker"
)

type Server struct {
	port       int
	tracker    *tracker.Tracker
	hopTracker *hoptracker.HopTracker
	server     *http.Server
}

func NewServer(port int, tracker *tracker.Tracker, hopTracker *hoptracker.HopTracker) *Server {
	return &Server{
		port:       port,
		tracker:    tracker,
		hopTracker: hopTracker,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/runtime", s.handleRuntime)
	mux.HandleFunc("/workers", s.handleWorkers)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.tracker.GetStats()
	stats.DuplicateAttempts = s.hopTracker.GetDuplicateStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"cpu_count":  runtime.NumCPU(),
		"memory": map[string]uint64{
			"alloc":       m.Alloc / 1024 / 1024,      // MB
			"total_alloc": m.TotalAlloc / 1024 / 1024, // MB
			"sys":         m.Sys / 1024 / 1024,        // MB
			"heap_alloc":  m.HeapAlloc / 1024 / 1024,  // MB
			"heap_sys":    m.HeapSys / 1024 / 1024,    // MB
		},
		"gc": map[string]interface{}{
			"num_gc":  m.NumGC,
			"last_gc": time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleWorkers(w http.ResponseWriter, r *http.Request) {
	stats := s.tracker.GetWorkerDetails()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
