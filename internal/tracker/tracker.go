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
	workerStats    map[int]WorkerStat
	statsMutex     sync.RWMutex
	visitedRing    []string
	visitedSet     map[string]bool
	ringMutex      sync.Mutex
	duplicateCount uint64
}

type WorkerStat struct {
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
		workerStats: make(map[int]WorkerStat),
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

	t.workerStats[workerID] = WorkerStat{
		KeysChecked: keysChecked,
		Rate:        rate,
		LastUpdate:  time.Now(),
		Status:      "active",
	}
}

func (t *Tracker) GetWorkerDetails() map[int]WorkerStat {
	t.statsMutex.RLock()
	defer t.statsMutex.RUnlock()

	// Create a copy to avoid race conditions
	details := make(map[int]WorkerStat)
	for id, stat := range t.workerStats {
		// Update status based on last update time
		if time.Since(stat.LastUpdate) > 30*time.Second {
			stat.Status = "idle"
		}
		details[id] = stat
	}

	return details
}

func (t *Tracker) GetStats() *Stats {
	t.statsMutex.RLock()
	defer t.statsMutex.RUnlock()

	var totalSpeed float64
	for _, stat := range t.workerStats {
		totalSpeed += stat.Rate
	}

	// Count found wallets
	foundWallets := 0
	if data, err := os.ReadFile("wallets_found.log"); err == nil {
		foundWallets = countOccurrences(string(data), "FOUND\nAddress:")
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
	return os.WriteFile("progress.json", []byte(fmt.Sprintf("%d", visited)), 0644)
}

func (t *Tracker) LoadProgress() error {
	data, err := os.ReadFile("progress.json")
	if err != nil {
		return err
	}

	var visited uint64
	if err := json.Unmarshal(data, &visited); err == nil {
		atomic.StoreUint64(&t.TotalVisited, visited)
	} else {
		// Try parsing as plain number
		fmt.Sscanf(string(data), "%d", &visited)
		atomic.StoreUint64(&t.TotalVisited, visited)
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
