// pkg/config/config.go
package config

import (
	"math/big"
	"os"
	"strconv"
	"strings"
)

type SearchStrategy string

const (
	FullRandom     SearchStrategy = "full_random"
	WeightedRandom SearchStrategy = "weighted_random"
	EarlyFocus     SearchStrategy = "early_focus"
	MultiZone      SearchStrategy = "multi_zone"
)

type CheckMode string

const (
	APIMode    CheckMode = "API"
	TargetMode CheckMode = "TARGET"
)

type SearchZone struct {
	StartPct float64
	EndPct   float64
	Weight   float64
}

type Config struct {
	// General
	Port       int
	NumWorkers int
	Seed       int64
	MaxAreas   int

	// Search range
	MinHex  *big.Int
	MaxHex  *big.Int
	HopSize *big.Int

	// Search strategy
	SearchStrategy SearchStrategy
	SearchZones    []SearchZone
	EarlyFocusPct  float64

	// Check mode
	CheckMode     CheckMode
	TargetAddress string
	APIURL        string
	MaxRetries    int
	APITimeout    int

	// Notifications
	EnableNotifications bool
	NotifyPhone         string
	NotifyURL           string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:       getEnvInt("PORT", 8177),
		NumWorkers: getEnvInt("NUM_WORKERS", 10),
		Seed:       42,
		MaxAreas:   1000,
		HopSize:    new(big.Int),
	}

	// Parse HopSize
	hopSize := getEnv("HOP_SIZE", "100000")
	cfg.HopSize.SetString(hopSize, 10)

	// Parse range
	minHex := strings.TrimPrefix(getEnv("MIN_HEX", "0"), "0x")
	maxHex := strings.TrimPrefix(getEnv("MAX_HEX", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), "0x")

	cfg.MinHex = new(big.Int)
	cfg.MinHex.SetString(minHex, 16)

	cfg.MaxHex = new(big.Int)
	cfg.MaxHex.SetString(maxHex, 16)

	// Search strategy
	strategy := getEnv("SEARCH_STRATEGY", "multi_zone")
	switch strings.ToLower(strategy) {
	case "full_random":
		cfg.SearchStrategy = FullRandom
	case "weighted_random":
		cfg.SearchStrategy = WeightedRandom
	case "early_focus":
		cfg.SearchStrategy = EarlyFocus
	default:
		cfg.SearchStrategy = MultiZone
	}

	// Parse search zones
	cfg.SearchZones = parseSearchZones(getEnv("SEARCH_ZONES", "20.0:35.0:75,80.0:95.0:25"))
	cfg.EarlyFocusPct = getEnvFloat("EARLY_FOCUS_PERCENT", 49.01)

	// Check mode
	checkMode := getEnv("CHECK_MODE", "TARGET")
	if strings.ToUpper(checkMode) == "API" {
		cfg.CheckMode = APIMode
	} else {
		cfg.CheckMode = TargetMode
	}

	cfg.TargetAddress = getEnv("TARGET_ADDRESS", "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU")
	cfg.APIURL = getEnv("API_URL", "http://localhost:4444/check")
	cfg.MaxRetries = getEnvInt("MAX_RETRIES", 3)
	cfg.APITimeout = getEnvInt("API_TIMEOUT", 5000)

	// Notifications
	cfg.EnableNotifications = getEnvBool("ENABLE_NOTIFICATIONS", true)
	cfg.NotifyPhone = getEnv("NOTIFY_PHONE", "081355554144")
	cfg.NotifyURL = getEnv("NOTIFY_URL", "http://wanotif.banksultra.id/api/v1/whatsapp/send")

	return cfg, nil
}

func parseSearchZones(zoneStr string) []SearchZone {
	var zones []SearchZone
	parts := strings.Split(zoneStr, ",")

	for _, part := range parts {
		fields := strings.Split(part, ":")
		if len(fields) == 3 {
			start, _ := strconv.ParseFloat(fields[0], 64)
			end, _ := strconv.ParseFloat(fields[1], 64)
			weight, _ := strconv.ParseFloat(fields[2], 64)

			zones = append(zones, SearchZone{
				StartPct: start / 100.0,
				EndPct:   end / 100.0,
				Weight:   weight,
			})
		}
	}

	return zones
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}
