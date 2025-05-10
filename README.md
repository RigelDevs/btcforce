# BTC Force - Windows GPU Edition

Bitcoin private key bruteforce tool with CUDA GPU acceleration for Windows.

## Features

- **GPU Acceleration**: Utilizes NVIDIA CUDA for high-performance key generation
- **Multi-threading**: Optimized CPU + GPU concurrent processing
- **Windows Native**: Built specifically for Windows systems
- **Real-time Monitoring**: HTTP API for performance monitoring
- **Progress Tracking**: Automatic checkpoint and recovery
- **Multiple Search Strategies**: Random, weighted, early focus, and multi-zone searching

## Requirements

- Windows 10/11 (64-bit)
- Go 1.21 or higher
- NVIDIA GPU with CUDA Compute Capability 3.5+
- CUDA Toolkit 12.0 or higher
- Visual Studio 2019/2022 (for CUDA compilation)
- At least 8GB RAM (16GB recommended)

## GPU Setup

1. Install NVIDIA GPU drivers (latest version)
2. Install CUDA Toolkit from: https://developer.nvidia.com/cuda-toolkit
3. Install Visual Studio with C++ development tools
4. Verify CUDA installation:
   ```
   nvcc --version
   ```

## Quick Start

1. Clone the repository
2. Build the project:
   ```
   scripts\build.cmd
   ```
3. Configure settings in `.env` file
4. Run the program:
   ```
   btcforce.exe
   ```

## Project Structure

```
btcforce/
├── cmd/
│   └── btcforce/
│       └── main.go           # Entry point
├── internal/
│   ├── bruteforce/
│   │   ├── bruteforce.go     # Bruteforce logic with GPU support
│   │   └── apiclient.go      # API client
│   ├── gpu/
│   │   └── gpu.go           # CUDA GPU implementation
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
├── scripts/
│   ├── setup.cmd            # Initial setup script
│   ├── build.cmd            # Build script
│   ├── run.cmd              # Run script
│   ├── monitor.cmd          # Performance monitor
│   ├── debug-api.cmd        # API debugging
│   └── view-log.cmd         # Log viewer
├── go.mod
├── go.sum
└── .env
```

## Configuration

Edit `.env` file:

```env
# GPU Settings
USE_GPU=true
GPU_BATCH_SIZE=1048576
CUDA_PATH=C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0

# General Settings
PORT=8177
NUM_WORKERS=10

# Search Range
MIN_HEX=0
MAX_HEX=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
HOP_SIZE=100000

# Search Strategy
SEARCH_STRATEGY=multi_zone
SEARCH_ZONES=20.0:35.0:75,80.0:95.0:25

# Target Mode
CHECK_MODE=TARGET
TARGET_ADDRESS=1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU
```

## Usage

### Build
```
scripts\build.cmd
```

### Run
```
btcforce.exe
```

### Monitor Performance
```
scripts\monitor.cmd
```

### Debug API
```
scripts\debug-api.cmd
```

## API Endpoints

- `http://localhost:8177/health` - Health check
- `http://localhost:8177/stats` - Progress statistics
- `http://localhost:8177/runtime` - Runtime information
- `http://localhost:8177/workers` - Worker details

## Performance

With GPU acceleration, expected performance:
- RTX 3090: ~5-10 billion keys/sec
- RTX 3080: ~3-6 billion keys/sec
- RTX 3070: ~2-4 billion keys/sec
- RTX 3060: ~1-2 billion keys/sec

CPU-only performance:
- AMD Ryzen 9: ~10-20 million keys/sec
- Intel i9: ~8-15 million keys/sec
- Intel i7: ~5-10 million keys/sec
- Intel i5: ~3-6 million keys/sec

## GPU CUDA Kernels

The GPU implementation includes optimized CUDA kernels for:
- Batch key generation
- SHA-256 hashing
- RIPEMD-160 hashing  
- Base58 encoding
- Address generation

## Troubleshooting

### CUDA not found
```
Error: CUDA not found. GPU support will be disabled.
```
Solution:
1. Install CUDA Toolkit 12.0+
2. Add CUDA to PATH: `C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\bin`
3. Restart terminal/command prompt

### Build errors with GPU
```
undefined reference to `cudaGetDeviceCount'
```
Solution:
1. Ensure Visual Studio is installed with C++ tools
2. Run from Visual Studio Developer Command Prompt
3. Set CGO_ENABLED=1

### GPU memory errors
```
CUDA error: out of memory
```
Solution:
1. Reduce GPU_BATCH_SIZE in .env
2. Close other GPU applications
3. Monitor GPU memory usage with nvidia-smi

## Safety & Legal

⚠️ **WARNING**: This tool is for educational purposes only. Attempting to crack Bitcoin wallets without permission is illegal and unethical.

- Never use this tool on addresses you don't own
- Respect others' property and privacy
- Use responsibly for research/educational purposes only

## License

MIT License - See LICENSE file for details

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## Credits

- CUDA GPU implementation
- Optimized for Windows performance
- Based on btcd Bitcoin library

## Disclaimer

This software is provided "as is" without warranty of any kind. The authors are not responsible for any damages or losses resulting from the use of this software.

---

For support or questions, please open an issue on the repository.