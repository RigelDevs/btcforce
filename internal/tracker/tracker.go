// internal/tracker/tracker.go
package tracker

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"btcforce/pkg/config"
)

type Tracker struct {
	TotalVisited   uint64
	workerStats    map[int]*WorkerStat // Changed to pointer for easier updates
	statsMutex     sync.RWMutex
	visitedRing    []string
	visitedSet     map[string]bool
	ringMutex      sync.Mutex
	duplicateCount uint64
}

type WorkerStat struct {
	WorkerID    int       `json:"worker_id"`
	KeysChecked uint64    `json:"keys_checked"`
	Rate        float64   `json:"rate"`
	LastUpdate  time.Time `json:"last_update"`
	Status      string    `json:"status"`
}

type Stats struct {
	TotalVisited           uint64  `json:"total_visited"`
	CurrentSpeed           uint64  `json:"current_speed"`
	FoundWallets           int     `json:"found_wallets"`
	ProgressPercentRaw     float64 `json:"-"`
	ProgressPercentDisplay string  `json:"progress_percent"`
	DuplicateAttempts      uint64  `json:"duplicate_attempts"`
}

const MaxVisited = 100000

func New() *Tracker {
	return &Tracker{
		workerStats: make(map[int]*WorkerStat),
		visitedRing: make([]string, 0, MaxVisited),
		visitedSet:  make(map[string]bool),
	}
}

func (t *Tracker) MarkVisited(key *big.Int) {
	hex := key.Text(16)

	t.ringMutex.Lock()
	defer t.ringMutex.Unlock()

	if t.visitedSet[hex] {
		return
	}

	// Ring buffer implementation for memory efficiency
	if len(t.visitedRing) >= MaxVisited {
		// Remove oldest
		oldest := t.visitedRing[0]
		t.visitedRing = t.visitedRing[1:]
		delete(t.visitedSet, oldest)
	}

	t.visitedRing = append(t.visitedRing, hex)
	t.visitedSet[hex] = true
}

func (t *Tracker) UpdateWorkerStats(workerID int, keysChecked uint64, rate float64) {
	t.statsMutex.Lock()
	defer t.statsMutex.Unlock()

	// Create or update worker stat
	if stat, exists := t.workerStats[workerID]; exists {
		stat.KeysChecked = keysChecked
		stat.Rate = rate
		stat.LastUpdate = time.Now()
		stat.Status = "active"
	} else {
		t.workerStats[workerID] = &WorkerStat{
			WorkerID:    workerID,
			KeysChecked: keysChecked,
			Rate:        rate,
			LastUpdate:  time.Now(),
			Status:      "active",
		}
	}
}

func (t *Tracker) GetWorkerDetails() []WorkerStat {
	t.statsMutex.RLock()
	defer t.statsMutex.RUnlock()

	// Create a slice of workers for JSON serialization
	workers := make([]WorkerStat, 0, len(t.workerStats))

	for _, stat := range t.workerStats {
		// Update status based on last update time
		workerCopy := *stat // Copy the stat
		if time.Since(stat.LastUpdate) > 30*time.Second {
			workerCopy.Status = "idle"
		} else if time.Since(stat.LastUpdate) > 10*time.Second {
			workerCopy.Status = "slow"
		}
		workers = append(workers, workerCopy)
	}

	// Sort workers by ID for consistent output
	// Simple bubble sort since we typically have few workers
	for i := 0; i < len(workers); i++ {
		for j := i + 1; j < len(workers); j++ {
			if workers[i].WorkerID > workers[j].WorkerID {
				workers[i], workers[j] = workers[j], workers[i]
			}
		}
	}

	return workers
}

func (t *Tracker) GetStats() *Stats {
	t.statsMutex.RLock()
	defer t.statsMutex.RUnlock()

	var totalSpeed float64
	activeWorkers := 0

	for _, stat := range t.workerStats {
		// Only count active workers in speed calculation
		if time.Since(stat.LastUpdate) <= 30*time.Second {
			totalSpeed += stat.Rate
			activeWorkers++
		}
	}

	// Count found wallets
	foundWallets := 0
	if data, err := os.ReadFile("wallets_found.log"); err == nil {
		foundWallets = countOccurrences(string(data), "FOUND BY WORKER")
	}

	// Calculate progress
	cfg, _ := config.Load()
	minHex := cfg.MinHex
	maxHex := cfg.MaxHex
	visited := atomic.LoadUint64(&t.TotalVisited)

	var progressRaw float64
	var progressDisplay string

	if maxHex.Cmp(minHex) > 0 {
		rangeSize := new(big.Int).Sub(maxHex, minHex)
		visitedBig := new(big.Int).SetUint64(visited)

		// Calculate percentage with high precision
		scale := new(big.Int).SetUint64(1e18)
		percentBig := new(big.Int).Mul(visitedBig, scale)
		percentBig.Div(percentBig, rangeSize)

		progressRaw, _ = new(big.Float).SetInt(percentBig).Float64()
		progressRaw /= 1e18

		if percentBig.Cmp(big.NewInt(1e18)) > 0 {
			progressDisplay = fmt.Sprintf("%.6e", progressRaw)
		} else {
			progressDisplay = fmt.Sprintf("%.18f", progressRaw)
		}
	}

	return &Stats{
		TotalVisited:           visited,
		CurrentSpeed:           uint64(totalSpeed),
		FoundWallets:           foundWallets,
		ProgressPercentRaw:     progressRaw,
		ProgressPercentDisplay: progressDisplay,
		DuplicateAttempts:      atomic.LoadUint64(&t.duplicateCount),
	}
}

func (t *Tracker) SaveProgress() error {
	visited := atomic.LoadUint64(&t.TotalVisited)
	data := map[string]interface{}{
		"total_visited": visited,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile("progress.json", jsonData, 0644)
}

func (t *Tracker) LoadProgress() error {
	data, err := os.ReadFile("progress.json")
	if err != nil {
		return err
	}

	var progress map[string]interface{}
	if err := json.Unmarshal(data, &progress); err != nil {
		// Try parsing as plain number for backward compatibility
		var visited uint64
		if _, err := fmt.Sscanf(string(data), "%d", &visited); err == nil {
			atomic.StoreUint64(&t.TotalVisited, visited)
			return nil
		}
		return err
	}

	if visited, ok := progress["total_visited"].(float64); ok {
		atomic.StoreUint64(&t.TotalVisited, uint64(visited))
	}

	return nil
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i < len(s); {
		if idx := strings.Index(s[i:], substr); idx >= 0 {
			count++
			i += idx + len(substr)
		} else {
			break
		}
	}
	return count
}
