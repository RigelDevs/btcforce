// cmd/btcforce/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"btcforce/internal/api"
	"btcforce/internal/bruteforce"
	"btcforce/internal/gpu"
	"btcforce/internal/hoptracker"
	"btcforce/internal/tracker"
	"btcforce/pkg/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Display banner
	displayBanner()

	// Display system information
	displaySystemInfo(cfg)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Shutting down gracefully...")
		cancel()
	}()

	// Initialize components
	tracker := tracker.New()
	hopTracker, err := hoptracker.New(cfg.Seed, cfg.MaxAreas, cfg.SearchStrategy)
	if err != nil {
		log.Fatalf("Failed to create hop tracker: %v", err)
	}
	defer hopTracker.Close()

	// Load previous progress
	if err := tracker.LoadProgress(); err != nil {
		log.Printf("Starting fresh (no previous progress found)")
	} else {
		log.Printf("Resumed from checkpoint: %d keys checked", tracker.TotalVisited)
	}

	// Start services
	if err := startServices(ctx, cfg, tracker, hopTracker); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}

	// Save final progress
	if err := tracker.SaveProgress(); err != nil {
		log.Printf("Failed to save progress: %v", err)
	}

	fmt.Println("\nShutdown complete")
}

func displayBanner() {
	fmt.Printf(`
██████╗ ████████╗ ██████╗    ███████╗ ██████╗ ██████╗  ██████╗███████╗
██╔══██╗╚══██╔══╝██╔════╝    ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔════╝
██████╔╝   ██║   ██║         █████╗  ██║   ██║██████╔╝██║     █████╗  
██╔══██╗   ██║   ██║         ██╔══╝  ██║   ██║██╔══██╗██║     ██╔══╝  
██████╔╝   ██║   ╚██████╗    ██║     ╚██████╔╝██║  ██║╚██████╗███████╗
╚═════╝    ╚═╝    ╚═════╝    ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝╚══════╝
                    Bitcoin Private Key Brute Force Tool
`)
}

func displaySystemInfo(cfg *config.Config) {
	fmt.Println("System Information:")
	fmt.Printf("  OS: %s\n", runtime.GOOS)
	fmt.Printf("  Arch: %s\n", runtime.GOARCH)
	fmt.Printf("  CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("  Go Version: %s\n", runtime.Version())
	fmt.Println()

	// Check GPU support
	if cfg.UseGPU {
		if gpu.IsAvailable() {
			fmt.Println("GPU Support: ENABLED")
			devices, err := gpu.GetDeviceInfo()
			if err == nil && len(devices) > 0 {
				for _, device := range devices {
					fmt.Printf("  Device: %s\n", device["name"])
					// Handle type assertion safely for cores
					if cores, ok := device["cores"].(int); ok {
						fmt.Printf("  Cores: %d\n", cores)
					}
				}
			}
		} else {
			fmt.Println("GPU Support: NOT AVAILABLE (falling back to CPU)")
			cfg.UseGPU = false
		}
	} else {
		fmt.Println("GPU Support: DISABLED")
	}
	fmt.Println()

	// Display configuration
	fmt.Println("Configuration:")
	fmt.Printf("  Workers: %d\n", cfg.NumWorkers)
	fmt.Printf("  Search Strategy: %s\n", cfg.SearchStrategy)
	fmt.Printf("  Check Mode: %s\n", cfg.CheckMode)
	if cfg.CheckMode == config.TargetMode {
		fmt.Printf("  Target Address: %s\n", cfg.TargetAddress)
	}
	fmt.Printf("  Search Range: %x...%x\n", cfg.MinHex, cfg.MaxHex)
	fmt.Printf("  Hop Size: %s\n", cfg.HopSize.String())
	fmt.Println()
}

func startServices(ctx context.Context, cfg *config.Config, tracker *tracker.Tracker, hopTracker *hoptracker.HopTracker) error {
	var wg sync.WaitGroup

	// Create worker pool
	pool := bruteforce.NewWorkerPool(cfg, tracker, hopTracker)

	// Start API server
	apiServer := api.NewServer(cfg.Port, tracker, hopTracker)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting API server on port %d", cfg.Port)
		if err := apiServer.Start(ctx); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start worker pool
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting brute force workers...")
		pool.Start(ctx)
	}()

	// Start performance monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		monitorPerformance(ctx, tracker)
	}()

	// Start progress saver
	wg.Add(1)
	go func() {
		defer wg.Done()
		periodicSave(ctx, tracker)
	}()

	wg.Wait()
	return nil
}

func monitorPerformance(ctx context.Context, tracker *tracker.Tracker) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := tracker.GetStats()
			elapsed := time.Since(startTime)

			fmt.Println("\n=== Performance Report ===")
			fmt.Printf("Elapsed Time: %s\n", elapsed.Round(time.Second))
			fmt.Printf("Total Keys Checked: %d\n", stats.TotalVisited)
			fmt.Printf("Current Speed: %d keys/sec\n", stats.CurrentSpeed)
			fmt.Printf("Progress: %s%%\n", stats.ProgressPercentDisplay)
			fmt.Printf("Duplicate Attempts: %d\n", stats.DuplicateAttempts)
			fmt.Printf("Found Wallets: %d\n", stats.FoundWallets)
			fmt.Println("========================")
		}
	}
}

func periodicSave(ctx context.Context, tracker *tracker.Tracker) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := tracker.SaveProgress(); err != nil {
				log.Printf("Failed to save progress: %v", err)
			} else {
				log.Printf("Progress saved: %d keys checked", tracker.TotalVisited)
			}
		}
	}
}
