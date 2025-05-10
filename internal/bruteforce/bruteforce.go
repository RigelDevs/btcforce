// internal/bruteforce/bruteforce.go
package bruteforce

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"btcforce/internal/gpu"
	"btcforce/internal/hoptracker"
	"btcforce/internal/notify"
	"btcforce/internal/tracker"
	"btcforce/internal/wallet"
	"btcforce/pkg/config"
)

const (
	// Batch size for checking keys
	keyBatchSize = 1000
	// Update interval for worker stats
	statsUpdateInterval = time.Second
	// Detailed log interval
	detailedLogInterval = 100000
)

type WorkerPool struct {
	cfg          *config.Config
	tracker      *tracker.Tracker
	hopTracker   *hoptracker.HopTracker
	workers      int
	gpuWorkers   []*gpu.GPUWorker
	jobChan      chan Job
	resultChan   chan Result
	wg           sync.WaitGroup
	useGPU       bool
	shutdownOnce sync.Once
}

type Job struct {
	ID     int
	Start  *big.Int
	End    *big.Int
	UseGPU bool
}

type Result struct {
	Found       bool
	Address     string
	WIF         string
	PrivateKey  string
	Balance     string
	WorkerID    int
	KeysChecked uint64
}

func NewWorkerPool(cfg *config.Config, tracker *tracker.Tracker, hopTracker *hoptracker.HopTracker) *WorkerPool {
	// Adjust workers based on CPU cores if not specified
	workers := cfg.NumWorkers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	wp := &WorkerPool{
		cfg:        cfg,
		tracker:    tracker,
		hopTracker: hopTracker,
		workers:    workers,
		jobChan:    make(chan Job, workers*2),
		resultChan: make(chan Result, 100),
		useGPU:     cfg.UseGPU,
	}

	// Initialize GPU workers if enabled
	if cfg.UseGPU && gpu.IsAvailable() {
		gpuWorkers, err := gpu.Init()
		if err != nil {
			log.Printf("❌ Failed to initialize GPU: %v, falling back to CPU", err)
			wp.useGPU = false
		} else {
			wp.gpuWorkers = gpuWorkers
			log.Printf("🚀 GPU initialized with %d devices", len(gpuWorkers))

			// Display GPU info
			if info, err := gpu.GetDeviceInfo(); err == nil {
				for _, device := range info {
					// Handle type assertion safely
					var memoryMB uint64
					switch v := device["memory"].(type) {
					case uint64:
						memoryMB = v / (1024 * 1024)
					case int:
						memoryMB = uint64(v) / (1024 * 1024)
					case int64:
						memoryMB = uint64(v) / (1024 * 1024)
					default:
						memoryMB = 0
					}

					log.Printf("📱 GPU %d: %s (Compute %s, Memory: %d MB)",
						device["id"],
						device["name"],
						device["compute"],
						memoryMB)
				}
			}
		}
	}

	return wp
}

func (wp *WorkerPool) Start(ctx context.Context) {
	log.Printf("🚀 Starting worker pool with %d CPU workers", wp.workers)
	if wp.useGPU && len(wp.gpuWorkers) > 0 {
		log.Printf("🚀 Plus %d GPU workers", len(wp.gpuWorkers))
	}

	// Set GOMAXPROCS to use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Start result processor first (to handle any results that come in)
	wp.wg.Add(1)
	go wp.processResults(ctx)

	// Start CPU workers
	for i := 1; i <= wp.workers; i++ {
		wp.wg.Add(1)
		go wp.cpuWorker(ctx, i)
	}

	// Start GPU workers if available
	if wp.useGPU && len(wp.gpuWorkers) > 0 {
		for i, gpuWorker := range wp.gpuWorkers {
			wp.wg.Add(1)
			go wp.gpuWorkerRoutine(ctx, i+wp.workers+1, gpuWorker)
		}
	}

	// Start job generator last
	wp.wg.Add(1)
	go wp.generateJobs(ctx)

	// Wait for all workers to complete
	wp.wg.Wait()

	// Close result channel after all workers are done
	close(wp.resultChan)

	// Cleanup GPU resources
	if wp.useGPU {
		for _, gpuWorker := range wp.gpuWorkers {
			gpuWorker.Cleanup()
		}
	}

	log.Println("Worker pool stopped")
}

func (wp *WorkerPool) cpuWorker(ctx context.Context, id int) {
	defer wp.wg.Done()

	checker := NewChecker(wp.cfg)
	log.Printf("🔧 CPU Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 CPU Worker %d stopping due to context cancellation", id)
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				log.Printf("🛑 CPU Worker %d: job channel closed", id)
				return
			}

			if job.UseGPU && wp.useGPU {
				// This job is for GPU, put it back (but check if channel is still open)
				select {
				case <-ctx.Done():
					return
				case wp.jobChan <- job:
					time.Sleep(100 * time.Millisecond)
					continue
				default:
					// Channel might be full or closed, skip this job
					continue
				}
			}

			jobSize := new(big.Int).Sub(job.End, job.Start)
			log.Printf("⚡ CPU Worker %d received job %d: %x to %x (size: %s)",
				id, job.ID, job.Start, job.End, jobSize.String())

			wp.processCPUJob(ctx, id, job, checker)
		}
	}
}

func (wp *WorkerPool) gpuWorkerRoutine(ctx context.Context, id int, gpuWorker *gpu.GPUWorker) {
	defer wp.wg.Done()

	checker := NewChecker(wp.cfg)
	log.Printf("🔧 GPU Worker %d started (Device %d)", id, gpuWorker.DeviceID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 GPU Worker %d stopping due to context cancellation", id)
			return
		case job, ok := <-wp.jobChan:
			if !ok {
				log.Printf("🛑 GPU Worker %d: job channel closed", id)
				return
			}

			if !job.UseGPU {
				// This job is for CPU, put it back (with context check)
				select {
				case <-ctx.Done():
					return
				case wp.jobChan <- job:
					time.Sleep(100 * time.Millisecond)
					continue
				default:
					// Channel might be full or closed, skip this job
					continue
				}
			}

			jobSize := new(big.Int).Sub(job.End, job.Start)
			log.Printf("⚡ GPU Worker %d received job %d: %x to %x (size: %s)",
				id, job.ID, job.Start, job.End, jobSize.String())

			wp.processGPUJob(ctx, id, job, gpuWorker, checker)
		}
	}
}

func (wp *WorkerPool) processGPUJob(ctx context.Context, workerID int, job Job, gpuWorker *gpu.GPUWorker, checker *Checker) {
	start := time.Now()
	keysChecked := uint64(0)

	// Process range using GPU
	keys, addresses, err := gpuWorker.ProcessRange(job.Start, job.End)
	if err != nil {
		log.Printf("❌ GPU Worker %d error: %v", workerID, err)
		return
	}

	// Check the generated addresses
	for i := range addresses {
		select {
		case <-ctx.Done():
			log.Printf("GPU Worker %d interrupted during processing", workerID)
			return
		default:
		}

		// Convert to proper address format and check
		privateKey := keys[i]
		walletInfo := wallet.FromPrivateKeyHex(privateKey)
		if walletInfo != nil {
			found, balance := checker.Check(walletInfo)
			if found {
				log.Printf("🎯 GPU Worker %d FOUND TARGET!", workerID)
				// Send result with context check
				select {
				case <-ctx.Done():
					return
				case wp.resultChan <- Result{
					Found:       true,
					Address:     walletInfo.Address,
					WIF:         walletInfo.WIF,
					PrivateKey:  privateKey,
					Balance:     balance,
					WorkerID:    workerID,
					KeysChecked: keysChecked,
				}:
					// Result sent successfully
				default:
					// Result channel might be full or closed
					log.Printf("Warning: GPU Worker %d could not send found wallet to result channel", workerID)
				}
			}
		}

		keysChecked++
		atomic.AddUint64(&wp.tracker.TotalVisited, 1)
	}

	// Update stats
	elapsed := time.Since(start).Seconds()
	if elapsed == 0 {
		elapsed = 0.001
	}
	rate := float64(keysChecked) / elapsed
	wp.tracker.UpdateWorkerStats(workerID, keysChecked, rate)

	// Mark range as completed
	wp.hopTracker.MarkRangeCompleted(job.Start, job.End)

	log.Printf("✅ GPU Worker %d completed job %d: %d keys in %.2f seconds (%.0f keys/sec)",
		workerID, job.ID, keysChecked, elapsed, rate)
}

func (wp *WorkerPool) processCPUJob(ctx context.Context, workerID int, job Job, checker *Checker) {
	start := time.Now()
	keysChecked := uint64(0)
	current := new(big.Int).Set(job.Start)
	one := big.NewInt(1)

	// Pre-allocate for better performance
	jobSize := new(big.Int).Sub(job.End, job.Start)
	estimatedKeys := jobSize.Uint64()

	log.Printf("CPU Worker %d processing job %d: %x to %x (estimated %d keys)",
		workerID, job.ID, job.Start, job.End, estimatedKeys)

	// Initialize worker stats
	wp.tracker.UpdateWorkerStats(workerID, 0, 0)

	lastUpdate := time.Now()
	lastDetailedLog := time.Now()
	localKeysChecked := uint64(0)

	for current.Cmp(job.End) < 0 {
		select {
		case <-ctx.Done():
			log.Printf("CPU Worker %d interrupted, saving progress", workerID)
			return
		default:
		}

		// Process keys in batches for better performance
		batchEnd := new(big.Int).Add(current, big.NewInt(keyBatchSize))
		if batchEnd.Cmp(job.End) > 0 {
			batchEnd.Set(job.End)
		}

		for current.Cmp(batchEnd) < 0 {
			// Generate wallet info
			walletInfo := wallet.FromPrivateKey(current)
			if walletInfo != nil {
				// Check if this is what we're looking for
				found, balance := checker.Check(walletInfo)
				if found {
					log.Printf("🎯 CPU Worker %d FOUND TARGET!", workerID)
					wp.resultChan <- Result{
						Found:       true,
						Address:     walletInfo.Address,
						WIF:         walletInfo.WIF,
						PrivateKey:  fmt.Sprintf("%064x", current),
						Balance:     balance,
						WorkerID:    workerID,
						KeysChecked: keysChecked,
					}
				}
			}

			// Mark as visited
			wp.tracker.MarkVisited(current)
			atomic.AddUint64(&wp.tracker.TotalVisited, 1)

			current.Add(current, one)
			keysChecked++
			localKeysChecked++
		}

		// Update stats periodically
		now := time.Now()
		if now.Sub(lastUpdate) >= statsUpdateInterval {
			elapsed := now.Sub(start).Seconds()
			rate := float64(keysChecked) / elapsed
			wp.tracker.UpdateWorkerStats(workerID, keysChecked, rate)
			lastUpdate = now
		}

		// Detailed logging at intervals
		if now.Sub(lastDetailedLog) >= 10*time.Second || localKeysChecked >= detailedLogInterval {
			elapsed := now.Sub(start).Seconds()
			rate := float64(keysChecked) / elapsed
			progress := float64(keysChecked) / float64(estimatedKeys) * 100

			log.Printf("CPU Worker %d: %d/%d keys (%.1f%%), rate: %.0f keys/sec, current: %x",
				workerID, keysChecked, estimatedKeys, progress, rate, current)

			lastDetailedLog = now
			localKeysChecked = 0
		}
	}

	// Final update
	elapsed := time.Since(start).Seconds()
	if elapsed == 0 {
		elapsed = 0.001 // Prevent division by zero
	}
	rate := float64(keysChecked) / elapsed
	wp.tracker.UpdateWorkerStats(workerID, keysChecked, rate)

	// Mark range as completed
	wp.hopTracker.MarkRangeCompleted(job.Start, job.End)

	log.Printf("✅ CPU Worker %d completed job %d: %d keys in %.2f seconds (%.0f keys/sec)",
		workerID, job.ID, keysChecked, elapsed, rate)
}

func (wp *WorkerPool) generateJobs(ctx context.Context) {
	defer wp.wg.Done()
	defer close(wp.jobChan) // Close channel when generator exits

	jobID := 0
	consecutiveFailures := 0
	maxConsecutiveFailures := 10
	gpuJobCounter := 0

	log.Println("🏭 Job generator started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Job generator stopping due to context cancellation")
			return
		default:
			// Get next hop from tracker
			start, end := wp.hopTracker.NextHop()

			// Validate the range
			if start == nil || end == nil {
				log.Printf("❌ Nil range from hop tracker")
				consecutiveFailures++
				if consecutiveFailures >= maxConsecutiveFailures {
					log.Printf("❌ Too many consecutive failures (%d), stopping job generator", consecutiveFailures)
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if start.Cmp(end) >= 0 {
				log.Printf("❌ Invalid range: start=%x >= end=%x", start, end)
				consecutiveFailures++
				if consecutiveFailures >= maxConsecutiveFailures {
					log.Printf("❌ Too many consecutive failures (%d), stopping job generator", consecutiveFailures)
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Reset failure counter on success
			consecutiveFailures = 0

			jobID++

			// Decide if this job should use GPU
			useGPU := false
			if wp.useGPU && len(wp.gpuWorkers) > 0 {
				// Distribute jobs between CPU and GPU
				gpuJobCounter++
				useGPU = (gpuJobCounter % 3) == 0 // Every 3rd job goes to GPU
			}

			job := Job{
				ID:     jobID,
				Start:  new(big.Int).Set(start),
				End:    new(big.Int).Set(end),
				UseGPU: useGPU,
			}

			jobSize := new(big.Int).Sub(end, start)
			workerType := "CPU"
			if useGPU {
				workerType = "GPU"
			}
			log.Printf("📦 Generated %s job %d: %x to %x (size: %s keys)",
				workerType, job.ID, start, end, jobSize.String())

			// Send job with context check
			select {
			case <-ctx.Done():
				return
			case wp.jobChan <- job:
				// Job successfully sent
			}
		}
	}
}

func (wp *WorkerPool) processResults(ctx context.Context) {
	defer wp.wg.Done()

	log.Println("📊 Result processor started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Result processor stopping due to context cancellation")
			// Drain any remaining results
			for {
				select {
				case result, ok := <-wp.resultChan:
					if !ok {
						return
					}
					if result.Found {
						wp.handleFoundWallet(result)
					}
				default:
					return
				}
			}
		case result, ok := <-wp.resultChan:
			if !ok {
				log.Println("Result processor: channel closed")
				return
			}

			if result.Found {
				log.Printf("🎉 WALLET FOUND BY WORKER %d!", result.WorkerID)
				wp.handleFoundWallet(result)
			}
		}
	}
}

func (wp *WorkerPool) handleFoundWallet(result Result) {
	msg := fmt.Sprintf("[%s] FOUND BY WORKER %d\nAddress: %s\nWIF: %s\nHEX: %s\nBalance: %s\nKeys Checked: %d\n\n",
		time.Now().Format(time.RFC3339),
		result.WorkerID,
		result.Address,
		result.WIF,
		result.PrivateKey,
		result.Balance,
		result.KeysChecked,
	)

	log.Printf("🎉 %s", msg)

	// Log to file
	if err := wallet.LogFound(msg); err != nil {
		log.Printf("❌ Failed to log wallet: %v", err)
	}

	// Send notification
	if wp.cfg.EnableNotifications {
		go func() {
			if err := notify.SendWhatsApp(msg, wp.cfg); err != nil {
				log.Printf("❌ Failed to send WhatsApp notification: %v", err)
			}
		}()
	}
}

// Checker handles the actual checking logic
type Checker struct {
	cfg    *config.Config
	client *APIClient
}

func NewChecker(cfg *config.Config) *Checker {
	c := &Checker{cfg: cfg}
	if cfg.CheckMode == config.APIMode {
		c.client = NewAPIClient(cfg)
	}
	return c
}

func (c *Checker) Check(wallet *wallet.WalletInfo) (bool, string) {
	switch c.cfg.CheckMode {
	case config.APIMode:
		if c.client != nil {
			return c.client.CheckAddress(wallet)
		}
		return false, "API client not initialized"
	case config.TargetMode:
		if wallet.Address == c.cfg.TargetAddress {
			return true, "Target found"
		}
		return false, ""
	default:
		return false, "Unknown check mode"
	}
}
