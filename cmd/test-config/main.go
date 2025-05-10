// cmd/test-config/main.go
package main

import (
	"fmt"
	"log"
	"math/big"

	"btcforce/internal/hoptracker"
	"btcforce/pkg/config"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("=== Configuration Test ===")
	fmt.Printf("MIN_HEX: %x\n", cfg.MinHex)
	fmt.Printf("MAX_HEX: %x\n", cfg.MaxHex)
	fmt.Printf("HOP_SIZE: %s\n", cfg.HopSize.String())
	fmt.Printf("Strategy: %s\n", cfg.SearchStrategy)
	fmt.Printf("Workers: %d\n", cfg.NumWorkers)

	// Calculate range size
	rangeSize := new(big.Int).Sub(cfg.MaxHex, cfg.MinHex)
	fmt.Printf("Range size: %s\n", rangeSize.String())

	// Test hop tracker
	fmt.Println("\n=== Testing Hop Tracker ===")
	hopTracker, err := hoptracker.New(42, 1000, cfg.SearchStrategy)
	if err != nil {
		log.Fatalf("Failed to create hop tracker: %v", err)
	}
	defer hopTracker.Close()

	// Generate some test hops
	for i := 0; i < 5; i++ {
		start, end := hopTracker.NextHop()
		if start == nil || end == nil {
			fmt.Printf("Hop %d: NIL range\n", i+1)
		} else {
			hopSize := new(big.Int).Sub(end, start)
			fmt.Printf("Hop %d: %x-%x (size: %s)\n", i+1, start, end, hopSize.String())
		}
	}
}
