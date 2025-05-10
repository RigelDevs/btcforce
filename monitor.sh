#!/bin/bash

# Colors for better readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PORT=${1:-8177}
HOST="localhost"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}jq is required but not installed. Please install it:${NC}"
    echo -e "${YELLOW}  Ubuntu/Debian: sudo apt-get install jq${NC}"
    echo -e "${YELLOW}  CentOS/RHEL: sudo yum install jq${NC}"
    echo -e "${YELLOW}  macOS: brew install jq${NC}"
    exit 1
fi

while true; do
    clear
    echo -e "${BLUE}=== BTC Force Monitor ===${NC}"
    echo -e "Time: $(date)"
    echo ""
    
    # Get runtime stats
    runtime=$(curl -s "http://$HOST:$PORT/runtime")
    if [ $? -eq 0 ] && [ ! -z "$runtime" ]; then
        # Check if response is valid JSON
        if echo "$runtime" | jq . >/dev/null 2>&1; then
            goroutines=$(echo "$runtime" | jq -r '.goroutines // "N/A"')
            mem_alloc=$(echo "$runtime" | jq -r '.memory.alloc // "N/A"')
            heap_alloc=$(echo "$runtime" | jq -r '.memory.heap_alloc // "N/A"')
            num_gc=$(echo "$runtime" | jq -r '.gc.num_gc // "N/A"')
            
            echo -e "${GREEN}Runtime Stats:${NC}"
            echo -e "  Goroutines: ${YELLOW}$goroutines${NC}"
            echo -e "  Memory Allocated: ${YELLOW}${mem_alloc}MB${NC}"
            echo -e "  Heap Allocated: ${YELLOW}${heap_alloc}MB${NC}"
            echo -e "  GC Runs: ${YELLOW}$num_gc${NC}"
        else
            echo -e "${RED}Invalid JSON response from /runtime${NC}"
            echo -e "Response: $runtime"
        fi
    else
        echo -e "${RED}Failed to connect to btcforce at http://$HOST:$PORT${NC}"
        echo -e "Make sure btcforce is running and the port is correct."
    fi
    
    echo ""
    
    # Get main stats
    stats=$(curl -s "http://$HOST:$PORT/stats")
    if [ $? -eq 0 ] && [ ! -z "$stats" ]; then
        if echo "$stats" | jq . >/dev/null 2>&1; then
            total_visited=$(echo "$stats" | jq -r '.total_visited // 0')
            current_speed=$(echo "$stats" | jq -r '.current_speed // 0')
            found_wallets=$(echo "$stats" | jq -r '.found_wallets // 0')
            progress=$(echo "$stats" | jq -r '.progress_percent // "0"')
            duplicates=$(echo "$stats" | jq -r '.duplicate_attempts // 0')
            
            echo -e "${GREEN}Progress Stats:${NC}"
            echo -e "  Keys Checked: ${YELLOW}$(printf "%'d" $total_visited)${NC}"
            echo -e "  Current Speed: ${YELLOW}$(printf "%'d" $current_speed) keys/sec${NC}"
            echo -e "  Progress: ${YELLOW}$progress%${NC}"
            echo -e "  Duplicates: ${YELLOW}$(printf "%'d" $duplicates)${NC}"
            echo -e "  Found Wallets: ${YELLOW}$found_wallets${NC}"
        else
            echo -e "${RED}Invalid JSON response from /stats${NC}"
            echo -e "Response: $stats"
        fi
    else
        echo -e "${RED}Failed to get progress stats${NC}"
    fi
    
    echo ""
    
    # Get worker details
    workers=$(curl -s "http://$HOST:$PORT/workers")
    if [ $? -eq 0 ] && [ ! -z "$workers" ]; then
        if echo "$workers" | jq . >/dev/null 2>&1; then
            if [ "$workers" != "{}" ] && [ "$workers" != "null" ]; then
                echo -e "${GREEN}Worker Status:${NC}"
                echo "$workers" | jq -r 'to_entries[] | "  Worker \(.key): \(.value.Rate // 0 | floor) keys/sec - \(.value.KeysChecked // 0) total"'
            else
                echo -e "${YELLOW}No worker data available yet${NC}"
            fi
        else
            echo -e "${RED}Invalid JSON response from /workers${NC}"
        fi
    else
        echo -e "${RED}Failed to get worker data${NC}"
    fi
    
    echo ""
    echo -e "${BLUE}Press Ctrl+C to exit${NC}"
    echo -e "Refreshing in 5 seconds..."
    
    sleep 5
done