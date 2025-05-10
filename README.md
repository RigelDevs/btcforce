# BTC Force - Go Implementation

Bitcoin private key bruteforce tool with optimized goroutine concurrency.

## Structure

```
btcforce/
├── cmd/
│   └── btcforce/
│       └── main.go           # Entry point
├── internal/
│   ├── bruteforce/
│   │   ├── bruteforce.go     # Bruteforce logic
│   │   └── apiclient.go      # API client
│   ├── wallet/
│   │   └── wallet.go         # Bitcoin wallet operations
│   ├── tracker/
│   │   └── tracker.go        # Progress tracking
│   ├── hoptracker/
│   │   └── hoptracker.go     # Range management
│   ├── notify/
│   │   └── notify.go         # Notifications
│   └── api/
│       └── server.go         # HTTP API server
├── pkg/
│   └── config/
│       └── config.go         # Configuration
├── go.mod
├── go.sum
└── .env
```

## Usage

```bash
go mod download
go build -o btcforce cmd/btcforce/main.go
./btcforce
```
