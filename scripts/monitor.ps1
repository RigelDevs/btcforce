param (
    [int]$Port = 8177,
    [string]$Host = "localhost"
)

$baseUrl = "http://${Host}:${Port}"

function Format-ByteSize {
    param([long]$bytes)
    if ($bytes -gt 1GB) {
        return "{0:N2} GB" -f ($bytes / 1GB)
    } elseif ($bytes -gt 1MB) {
        return "{0:N2} MB" -f ($bytes / 1MB)
    } elseif ($bytes -gt 1KB) {
        return "{0:N2} KB" -f ($bytes / 1KB)
    } else {
        return "{0} bytes" -f $bytes
    }
}

function Format-Number {
    param([long]$num)
    return $num.ToString("N0")
}

function Format-Rate {
    param([long]$rate)
    if ($rate -gt 1000000) {
        return "{0:N2}M keys/sec" -f ($rate / 1000000)
    } elseif ($rate -gt 1000) {
        return "{0:N2}K keys/sec" -f ($rate / 1000)
    } else {
        return "$rate keys/sec"
    }
}

while ($true) {
    Clear-Host
    
    Write-Host "=== BTC Force Monitor ===" -ForegroundColor Cyan
    Write-Host "Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""
    
    try {
        # Get runtime stats
        $runtime = Invoke-RestMethod -Uri "$baseUrl/runtime" -Method Get -ErrorAction Stop
        
        Write-Host "System Information:" -ForegroundColor Yellow
        Write-Host "  CPU Cores: $($runtime.cpu_count)"
        Write-Host "  Goroutines: $($runtime.goroutines)"
        Write-Host "  Memory Usage: $(Format-ByteSize ($runtime.memory.alloc * 1MB))"
        Write-Host "  Total Allocated: $(Format-ByteSize ($runtime.memory.total_alloc * 1MB))"
        Write-Host "  Last GC: $($runtime.gc.last_gc)"
        Write-Host ""
        
        # Get progress stats
        $stats = Invoke-RestMethod -Uri "$baseUrl/stats" -Method Get -ErrorAction Stop
        
        Write-Host "Progress Statistics:" -ForegroundColor Yellow
        Write-Host "  Total Keys Checked: $(Format-Number $stats.total_visited)"
        Write-Host "  Current Speed: $(Format-Rate $stats.current_speed)"
        Write-Host "  Progress: $($stats.progress_percent)%"
        Write-Host "  Found Wallets: $($stats.found_wallets)"
        Write-Host "  Duplicate Attempts: $(Format-Number $stats.duplicate_attempts)"
        Write-Host ""
        
        # Get worker details
        $workers = Invoke-RestMethod -Uri "$baseUrl/workers" -Method Get -ErrorAction Stop
        
        Write-Host "Worker Status:" -ForegroundColor Yellow
        if ($workers.workers -and $workers.workers.Count -gt 0) {
            Write-Host "  Active Workers: $($workers.summary.active_workers)/$($workers.total)"
            Write-Host "  Total Rate: $(Format-Rate $workers.summary.total_rate)"
            Write-Host ""
            
            Write-Host "Individual Workers:" -ForegroundColor Cyan
            foreach ($worker in $workers.workers) {
                $status = switch ($worker.status) {
                    "active" { "Active" }
                    "idle" { "Idle" }
                    "slow" { "Slow" }
                    default { $worker.status }
                }
                
                $statusColor = switch ($worker.status) {
                    "active" { "Green" }
                    "idle" { "Red" }
                    "slow" { "Yellow" }
                    default { "Gray" }
                }
                
                Write-Host ("  Worker {0,2}: {1,-6} | Rate: {2,15} | Keys: {3,12}" -f 
                    $worker.worker_id, 
                    $status,
                    (Format-Rate $worker.rate),
                    (Format-Number $worker.keys_checked)
                ) -ForegroundColor $statusColor
            }
        } else {
            Write-Host "  No worker data available yet" -ForegroundColor Red
        }
        
    } catch {
        Write-Host "Error connecting to API: $_" -ForegroundColor Red
        Write-Host "Make sure btcforce is running on port $Port" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "Press Ctrl+C to exit" -ForegroundColor Gray
    Write-Host "Refreshing in 5 seconds..." -ForegroundColor Gray
    
    Start-Sleep -Seconds 5
}