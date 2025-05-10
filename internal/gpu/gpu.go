// internal/gpu/gpu.go
package gpu

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
)

// GPUWorker simulates GPU operations using parallel goroutines
type GPUWorker struct {
	DeviceID  int
	BatchSize int
	Name      string
}

// Init initializes GPU workers (simulated using CPU parallel processing)
func Init() ([]*GPUWorker, error) {
	// Create simulated GPU workers based on CPU cores
	numCores := runtime.NumCPU()
	workers := make([]*GPUWorker, 1)

	workers[0] = &GPUWorker{
		DeviceID:  0,
		BatchSize: 1000000, // 1M keys per batch
		Name:      fmt.Sprintf("CPU Parallel Processor (%d cores)", numCores),
	}

	return workers, nil
}

// ProcessRange processes a range of keys using parallel computation
func (w *GPUWorker) ProcessRange(start, end *big.Int) ([]string, []string, error) {
	rangeSize := new(big.Int).Sub(end, start)
	count := rangeSize.Uint64()

	if count > uint64(w.BatchSize) {
		count = uint64(w.BatchSize)
	}

	keys := make([]string, count)
	addresses := make([]string, count)

	// Use all CPU cores for parallel processing
	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	// Split work among goroutines
	chunkSize := count / uint64(numWorkers)
	if chunkSize == 0 {
		chunkSize = 1
	}

	for i := 0; i < numWorkers; i++ {
		startIdx := uint64(i) * chunkSize
		endIdx := startIdx + chunkSize
		if i == numWorkers-1 {
			endIdx = count
		}

		wg.Add(1)
		go func(start, end uint64, baseNum *big.Int) {
			defer wg.Done()

			current := new(big.Int).Set(baseNum)
			current.Add(current, big.NewInt(int64(start)))

			for j := start; j < end; j++ {
				// Generate key
				keys[j] = fmt.Sprintf("%064x", current)

				// Generate simplified address (first 40 chars of hex)
				addrHex := fmt.Sprintf("%x", current)
				if len(addrHex) > 40 {
					addresses[j] = "1" + addrHex[:40]
				} else {
					addresses[j] = "1" + fmt.Sprintf("%040s", addrHex)
				}

				current.Add(current, big.NewInt(1))
			}
		}(startIdx, endIdx, start)
	}

	wg.Wait()

	return keys, addresses, nil
}

// Cleanup releases resources (no-op for CPU simulation)
func (w *GPUWorker) Cleanup() {
	// Nothing to cleanup for CPU simulation
}

// IsAvailable checks if GPU acceleration is available (always true for simulation)
func IsAvailable() bool {
	return runtime.GOOS == "windows"
}

// GetDeviceInfo returns information about available GPU devices
func GetDeviceInfo() ([]map[string]interface{}, error) {
	numCores := runtime.NumCPU()

	devices := []map[string]interface{}{
		{
			"id":          0,
			"name":        fmt.Sprintf("CPU Parallel Processor (%d cores)", numCores),
			"compute":     "Simulated",
			"memory":      uint64(numCores) * 1024 * 1024 * 1024, // Ensure uint64 type
			"max_threads": numCores,
			"cores":       numCores,
		},
	}

	return devices, nil
}

// GetGPUCount returns the number of available GPUs (simulated)
func GetGPUCount() int {
	return 1
}

// GetMemoryInfo returns memory usage for the device
func (w *GPUWorker) GetMemoryInfo() (used, total uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.Sys
}

// SetBatchSize updates the batch size for processing
func (w *GPUWorker) SetBatchSize(size int) {
	w.BatchSize = size
}

// GetBatchSize returns current batch size
func (w *GPUWorker) GetBatchSize() int {
	return w.BatchSize
}
