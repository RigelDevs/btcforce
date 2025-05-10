#!/bin/bash

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${BLUE}=== BTC Force Log Viewer ===${NC}"
echo "Showing colored logs from btcforce process..."
echo ""

# Function to colorize log output
colorize_logs() {
    while IFS= read -r line; do
        case "$line" in
            *"🚀"*) echo -e "${GREEN}$line${NC}" ;;
            *"⚡"*) echo -e "${YELLOW}$line${NC}" ;;
            *"🔧"*) echo -e "${BLUE}$line${NC}" ;;
            *"📊"*) echo -e "${PURPLE}$line${NC}" ;;
            *"✅"*) echo -e "${GREEN}$line${NC}" ;;
            *"❌"*) echo -e "${RED}$line${NC}" ;;
            *"🛑"*) echo -e "${RED}$line${NC}" ;;
            *"FOUND"*) echo -e "${GREEN}${line}${NC}" ;;
            *"Worker"*) echo -e "${BLUE}$line${NC}" ;;
            *"Goroutines"*) echo -e "${PURPLE}$line${NC}" ;;
            *"Error"*|*"error"*) echo -e "${RED}$line${NC}" ;;
            *) echo "$line" ;;
        esac
    done
}

# Check if btcforce is running
if pgrep -x "btcforce" > /dev/null; then
    echo -e "${GREEN}btcforce is running. Showing logs...${NC}"
    echo ""
    # Tail the process output
    ./btcforce 2>&1 | colorize_logs
else
    echo -e "${YELLOW}btcforce is not running. Starting it now...${NC}"
    echo ""
    # Start btcforce and pipe through colorizer
    ./btcforce 2>&1 | colorize_logs
fi