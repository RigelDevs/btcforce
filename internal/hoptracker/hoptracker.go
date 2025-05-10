// internal/hoptracker/hoptracker.go
package hoptracker

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"sync/atomic"

	"btcforce/pkg/config"

	"github.com/cockroachdb/pebble"
)

type HopTracker struct {
	db               *pebble.DB
	hopSize          *big.Int
	minRange         *big.Int
	maxRange         *big.Int
	strategy         config.SearchStrategy
	searchZones      []config.SearchZone
	mu               sync.Mutex
	inProgressMu     sync.RWMutex
	inProgressRanges map[string]bool
	duplicateCount   uint64
}

type Checkpoint struct {
	LastAlignedHex string `json:"last_aligned_hex"`
}

func New(seed int64, maxAreas int, strategy config.SearchStrategy) (*HopTracker, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create database directory if it doesn't exist
	if err := os.MkdirAll("visited_db", 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open Pebble database (faster than RocksDB for our use case)
	opts := &pebble.Options{
		MaxOpenFiles: 1000,
	}

	db, err := pebble.Open("visited_db", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ht := &HopTracker{
		db:               db,
		hopSize:          cfg.HopSize,
		minRange:         cfg.MinHex,
		maxRange:         cfg.MaxHex,
		strategy:         strategy,
		searchZones:      cfg.SearchZones,
		inProgressRanges: make(map[string]bool),
	}

	return ht, nil
}

func (ht *HopTracker) NextHop() (*big.Int, *big.Int) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	switch ht.strategy {
	case config.WeightedRandom:
		return ht.nextWeighted()
	case config.EarlyFocus:
		return ht.nextEarly()
	case config.MultiZone:
		return ht.nextMultiZone()
	default:
		return ht.nextRandom()
	}
}

func (ht *HopTracker) nextRandom() (*big.Int, *big.Int) {
	rangeDiff := new(big.Int).Sub(ht.maxRange, ht.minRange)

	for {
		// Generate random bytes
		bytes := make([]byte, 32)
		rand.Read(bytes)

		raw := new(big.Int).SetBytes(bytes)
		candidate := new(big.Int).Mod(raw, rangeDiff)
		candidate.Add(candidate, ht.minRange)

		// Align to hop size
		aligned := new(big.Int).Div(candidate, ht.hopSize)
		aligned.Mul(aligned, ht.hopSize)

		if !ht.alreadyVisited(aligned) {
			ht.markVisited(aligned)
			end := new(big.Int).Add(aligned, ht.hopSize)

			// Add to in-progress tracking
			rangeKey := fmt.Sprintf("%x-%x", aligned, end)
			ht.inProgressMu.Lock()
			ht.inProgressRanges[rangeKey] = true
			ht.inProgressMu.Unlock()

			return aligned, end
		}
	}
}

func (ht *HopTracker) nextMultiZone() (*big.Int, *big.Int) {
	// Calculate total weight
	totalWeight := 0.0
	for _, zone := range ht.searchZones {
		totalWeight += zone.Weight
	}

	// Select zone based on weight
	r := randFloat() * totalWeight
	var selectedZone config.SearchZone

	for _, zone := range ht.searchZones {
		if r <= zone.Weight {
			selectedZone = zone
			break
		}
		r -= zone.Weight
	}

	// Generate random within selected zone
	rangeDiff := new(big.Int).Sub(ht.maxRange, ht.minRange)
	zoneStart := new(big.Int).Mul(rangeDiff, big.NewInt(int64(selectedZone.StartPct*1e6)))
	zoneStart.Div(zoneStart, big.NewInt(1e6))
	zoneStart.Add(zoneStart, ht.minRange)

	zoneEnd := new(big.Int).Mul(rangeDiff, big.NewInt(int64(selectedZone.EndPct*1e6)))
	zoneEnd.Div(zoneEnd, big.NewInt(1e6))
	zoneEnd.Add(zoneEnd, ht.minRange)

	// Ensure zoneEnd > zoneStart
	if zoneEnd.Cmp(zoneStart) <= 0 {
		zoneEnd = new(big.Int).Add(zoneStart, ht.hopSize)
	}

	zoneRange := new(big.Int).Sub(zoneEnd, zoneStart)

	for {
		bytes := make([]byte, 32)
		rand.Read(bytes)

		raw := new(big.Int).SetBytes(bytes)
		candidate := new(big.Int).Mod(raw, zoneRange)
		candidate.Add(candidate, zoneStart)

		aligned := new(big.Int).Div(candidate, ht.hopSize)
		aligned.Mul(aligned, ht.hopSize)

		if !ht.alreadyVisited(aligned) {
			ht.markVisited(aligned)
			end := new(big.Int).Add(aligned, ht.hopSize)

			rangeKey := fmt.Sprintf("%x-%x", aligned, end)
			ht.inProgressMu.Lock()
			ht.inProgressRanges[rangeKey] = true
			ht.inProgressMu.Unlock()

			return aligned, end
		}
	}
}

func (ht *HopTracker) nextWeighted() (*big.Int, *big.Int) {
	// 70% chance for early range (first 1%)
	if randFloat() < 0.7 {
		return ht.nextEarly()
	}
	return ht.nextRandom()
}

func (ht *HopTracker) nextEarly() (*big.Int, *big.Int) {
	cfg, _ := config.Load()
	earlyPct := cfg.EarlyFocusPct / 100.0

	rangeDiff := new(big.Int).Sub(ht.maxRange, ht.minRange)
	earlyEnd := new(big.Int).Mul(rangeDiff, big.NewInt(int64(earlyPct*1e6)))
	earlyEnd.Div(earlyEnd, big.NewInt(1e6))
	earlyEnd.Add(earlyEnd, ht.minRange)

	// Ensure earlyEnd > minRange
	if earlyEnd.Cmp(ht.minRange) <= 0 {
		earlyEnd = new(big.Int).Add(ht.minRange, ht.hopSize)
	}

	earlyRange := new(big.Int).Sub(earlyEnd, ht.minRange)

	for {
		bytes := make([]byte, 32)
		rand.Read(bytes)

		raw := new(big.Int).SetBytes(bytes)
		candidate := new(big.Int).Mod(raw, earlyRange)
		candidate.Add(candidate, ht.minRange)

		aligned := new(big.Int).Div(candidate, ht.hopSize)
		aligned.Mul(aligned, ht.hopSize)

		if !ht.alreadyVisited(aligned) {
			ht.markVisited(aligned)
			end := new(big.Int).Add(aligned, ht.hopSize)

			rangeKey := fmt.Sprintf("%x-%x", aligned, end)
			ht.inProgressMu.Lock()
			ht.inProgressRanges[rangeKey] = true
			ht.inProgressMu.Unlock()

			return aligned, end
		}
	}
}

func (ht *HopTracker) alreadyVisited(key *big.Int) bool {
	hexKey := hex.EncodeToString(key.Bytes())

	// Check if in progress
	endKey := new(big.Int).Add(key, ht.hopSize)
	rangeKey := fmt.Sprintf("%x-%x", key, endKey)

	ht.inProgressMu.RLock()
	if ht.inProgressRanges[rangeKey] {
		ht.inProgressMu.RUnlock()
		atomic.AddUint64(&ht.duplicateCount, 1)
		return true
	}
	ht.inProgressMu.RUnlock()

	// Check database
	_, closer, err := ht.db.Get([]byte(hexKey))
	if err == nil {
		closer.Close()
		atomic.AddUint64(&ht.duplicateCount, 1)
		return true
	}

	return false
}

func (ht *HopTracker) markVisited(key *big.Int) {
	hexKey := hex.EncodeToString(key.Bytes())
	err := ht.db.Set([]byte(hexKey), []byte("1"), pebble.Sync)
	if err != nil {
		fmt.Printf("Failed to mark visited: %v\n", err)
	}

	// Save checkpoint periodically
	if atomic.LoadUint64(&ht.duplicateCount)%1000 == 0 {
		ht.saveCheckpoint(hexKey)
	}
}

func (ht *HopTracker) saveCheckpoint(hexKey string) {
	checkpoint := Checkpoint{
		LastAlignedHex: hexKey,
	}

	data, err := json.Marshal(checkpoint)
	if err != nil {
		return
	}

	_ = os.WriteFile("checkpoint.json", data, 0644)
}

func (ht *HopTracker) MarkRangeCompleted(start, end *big.Int) {
	rangeKey := fmt.Sprintf("%x-%x", start, end)

	ht.inProgressMu.Lock()
	delete(ht.inProgressRanges, rangeKey)
	ht.inProgressMu.Unlock()
}

func (ht *HopTracker) GetDuplicateStats() uint64 {
	return atomic.LoadUint64(&ht.duplicateCount)
}

func (ht *HopTracker) VisitedCount() uint64 {
	iter, err := ht.db.NewIter(nil)
	if err != nil {
		fmt.Printf("Failed to create iterator: %v\n", err)
		return 0
	}
	defer iter.Close()

	count := uint64(0)
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	// Each entry represents hop_size keys
	hopSize := ht.hopSize.Uint64()
	return count * hopSize
}

func (ht *HopTracker) Close() error {
	// Save final checkpoint
	if ht.db != nil {
		// Get a random key as checkpoint
		iter, err := ht.db.NewIter(nil)
		if err != nil {
			return fmt.Errorf("failed to create iterator: %w", err)
		}
		if iter.Last() && iter.Valid() {
			ht.saveCheckpoint(string(iter.Key()))
		}
		iter.Close()
	}

	return ht.db.Close()
}

// Helper function for random float
func randFloat() float64 {
	b := make([]byte, 8)
	rand.Read(b)
	return float64(binary.LittleEndian.Uint64(b)) / (1 << 64)
}
