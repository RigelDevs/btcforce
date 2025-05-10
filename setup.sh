#!/bin/bash

# Create Go BTC Force project structure

echo "Creating project structure for btcforce-go..."

# Create main directories
mkdir -p cmd/btcforce
mkdir -p internal/bruteforce
mkdir -p internal/wallet
mkdir -p internal/tracker
mkdir -p internal/hoptracker
mkdir -p internal/notify
mkdir -p internal/api
mkdir -p pkg/config

# Create empty Go files
touch cmd/btcforce/main.go
touch internal/bruteforce/bruteforce.go
touch internal/bruteforce/apiclient.go
touch internal/wallet/wallet.go
touch internal/tracker/tracker.go
touch internal/hoptracker/hoptracker.go
touch internal/notify/notify.go
touch internal/api/server.go
touch pkg/config/config.go

# Create configuration files
touch go.mod
touch go.sum
touch .env

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.dll
*.so
*.dylib
btcforce

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# Environment variables
.env

# Database
visited_db/

# Logs
*.log

# Progress files
progress.json
checkpoint.json
wallets_found.log

# IDE
.idea/
.vscode/
*.swp
*.swo
EOF

# Create README
cat > README.md << 'EOF'
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
EOF

# Make the main script executable
chmod +x cmd/btcforce/main.go

echo "Project structure created successfully!"
echo "Navigate to: cd btcforce-go"
echo "Start coding!"