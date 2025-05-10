// internal/gpu/gpu_cuda.go
//go:build windows && cgo
// +build windows,cgo

package gpu

/*
#cgo CFLAGS: -I${SRCDIR}/cuda_headers
#cgo windows LDFLAGS: -L${SRCDIR}/cuda_libs -lcudart64_120 -lcuda

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// Forward declarations for CUDA functions
typedef struct {
    int device;
    char name[256];
    size_t totalMem;
    size_t freeMem;
    int major;
    int minor;
    int multiProcessorCount;
} CudaDeviceInfo;

// External CUDA functions implemented in gpu_cuda_wrapper.c
extern int cuda_get_device_count();
extern int cuda_get_device_info(int device, CudaDeviceInfo* info);
extern int cuda_set_device(int device);
extern void* cuda_malloc(size_t size);
extern void cuda_free(void* ptr);
extern int cuda_memcpy_htod(void* dst, const void* src, size_t size);
extern int cuda_memcpy_dtoh(void* dst, const void* src, size_t size);
extern int cuda_launch_key_generation(void* d_keys, void* d_addresses, uint64_t start, uint64_t count, const char* target);
extern int cuda_device_synchronize();
extern const char* cuda_get_last_error();
*/
import "C"

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"unsafe"
)

const (
	maxBatchSize = 16 * 1024 * 1024 // 16M keys max per batch
	minBatchSize = 1024             // 1K keys minimum
)

// GPUWorker represents a CUDA GPU device
type GPUWorker struct {
	DeviceID          int
	BatchSize         int
	Name              string
	TotalMemory       uint64
	FreeMemory        uint64
	ComputeCapability string
	MultiProcessors   int
	mu                sync.Mutex
}

var (
	initOnce    sync.Once
	initialized bool
	initError   error
	cudaDevices []*GPUWorker
)

// Init initializes CUDA devices
func Init() ([]*GPUWorker, error) {
	initOnce.Do(func() {
		// Check if this is Windows
		if runtime.GOOS != "windows" {
			initError = errors.New("CUDA support is only available on Windows")
			return
		}

		// Get device count
		deviceCount := int(C.cuda_get_device_count())
		if deviceCount <= 0 {
			initError = errors.New("no CUDA devices found")
			return
		}

		cudaDevices = make([]*GPUWorker, 0, deviceCount)

		// Initialize each device
		for i := 0; i < deviceCount; i++ {
			var info C.CudaDeviceInfo
			if C.cuda_get_device_info(C.int(i), &info) == 0 {
				continue
			}

			deviceName := C.GoString(&info.name[0])
			computeCapability := fmt.Sprintf("%d.%d", int(info.major), int(info.minor))

			// Calculate optimal batch size based on memory
			batchSize := calculateOptimalBatchSize(uint64(info.totalMem))

			worker := &GPUWorker{
				DeviceID:          i,
				BatchSize:         batchSize,
				Name:              deviceName,
				TotalMemory:       uint64(info.totalMem),
				FreeMemory:        uint64(info.freeMem),
				ComputeCapability: computeCapability,
				MultiProcessors:   int(info.multiProcessorCount),
			}

			cudaDevices = append(cudaDevices, worker)

			fmt.Printf("Initialized CUDA device %d: %s (CC %s, %d SMs, %.1f GB)\n",
				i, deviceName, computeCapability, worker.MultiProcessors,
				float64(worker.TotalMemory)/(1024*1024*1024))
		}

		if len(cudaDevices) == 0 {
			initError = errors.New("no usable CUDA devices found")
			return
		}

		initialized = true
	})

	return cudaDevices, initError
}

// ProcessRange processes a range of Bitcoin private keys on GPU
func (w *GPUWorker) ProcessRange(start, end *big.Int) ([]string, []string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !initialized {
		return nil, nil, errors.New("CUDA not initialized")
	}

	// Set device
	if C.cuda_set_device(C.int(w.DeviceID)) == 0 {
		return nil, nil, fmt.Errorf("failed to set CUDA device %d", w.DeviceID)
	}

	// Calculate range size
	rangeSize := new(big.Int).Sub(end, start)
	count := rangeSize.Uint64()

	// Limit to batch size
	if count > uint64(w.BatchSize) {
		count = uint64(w.BatchSize)
	}

	// Allocate device memory for keys and addresses
	keysSize := count * 8       // 8 bytes per uint64 key
	addressesSize := count * 64 // 64 chars per address (overallocation for safety)

	dKeys := C.cuda_malloc(C.size_t(keysSize))
	if dKeys == nil {
		return nil, nil, fmt.Errorf("failed to allocate GPU memory for keys: %s", C.GoString(C.cuda_get_last_error()))
	}
	defer C.cuda_free(dKeys)

	dAddresses := C.cuda_malloc(C.size_t(addressesSize))
	if dAddresses == nil {
		return nil, nil, fmt.Errorf("failed to allocate GPU memory for addresses: %s", C.GoString(C.cuda_get_last_error()))
	}
	defer C.cuda_free(dAddresses)

	// Prepare start value
	startValue := start.Uint64()

	// Launch CUDA kernel
	if C.cuda_launch_key_generation(dKeys, dAddresses, C.uint64_t(startValue), C.uint64_t(count), nil) == 0 {
		return nil, nil, fmt.Errorf("failed to launch CUDA kernel: %s", C.GoString(C.cuda_get_last_error()))
	}

	// Wait for kernel completion
	if C.cuda_device_synchronize() == 0 {
		return nil, nil, fmt.Errorf("CUDA synchronization failed: %s", C.GoString(C.cuda_get_last_error()))
	}

	// Allocate host memory for results
	hostKeys := make([]uint64, count)
	hostAddresses := make([]byte, addressesSize)

	// Copy results back to host
	if C.cuda_memcpy_dtoh(unsafe.Pointer(&hostKeys[0]), dKeys, C.size_t(keysSize)) == 0 {
		return nil, nil, fmt.Errorf("failed to copy keys from GPU: %s", C.GoString(C.cuda_get_last_error()))
	}

	if C.cuda_memcpy_dtoh(unsafe.Pointer(&hostAddresses[0]), dAddresses, C.size_t(addressesSize)) == 0 {
		return nil, nil, fmt.Errorf("failed to copy addresses from GPU: %s", C.GoString(C.cuda_get_last_error()))
	}

	// Convert results to string format
	keys := make([]string, count)
	addresses := make([]string, count)

	for i := uint64(0); i < count; i++ {
		// Convert key to hex string
		key := new(big.Int).SetUint64(hostKeys[i])
		keys[i] = fmt.Sprintf("%064x", key)

		// Extract address (null-terminated string)
		addressStart := i * 64
		addressEnd := addressStart + 64
		for j := addressStart; j < addressEnd; j++ {
			if hostAddresses[j] == 0 {
				addresses[i] = string(hostAddresses[addressStart:j])
				break
			}
		}

		// Fallback if no null terminator found
		if addresses[i] == "" {
			addresses[i] = string(hostAddresses[addressStart:addressEnd])
		}
	}

	return keys, addresses, nil
}

// CheckAddress checks if a specific Bitcoin address exists in a batch
func (w *GPUWorker) CheckAddress(start, end *big.Int, targetAddress string) (bool, *big.Int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !initialized {
		return false, nil, errors.New("CUDA not initialized")
	}

	// Set device
	if C.cuda_set_device(C.int(w.DeviceID)) == 0 {
		return false, nil, fmt.Errorf("failed to set CUDA device %d", w.DeviceID)
	}

	// Calculate range size
	rangeSize := new(big.Int).Sub(end, start)
	count := rangeSize.Uint64()

	// Limit to batch size
	if count > uint64(w.BatchSize) {
		count = uint64(w.BatchSize)
	}

	// Allocate device memory
	keysSize := count * 8
	dKeys := C.cuda_malloc(C.size_t(keysSize))
	if dKeys == nil {
		return false, nil, fmt.Errorf("failed to allocate GPU memory: %s", C.GoString(C.cuda_get_last_error()))
	}
	defer C.cuda_free(dKeys)

	// Allocate for found flag and key
	dFoundFlag := C.cuda_malloc(C.size_t(4)) // int32
	dFoundKey := C.cuda_malloc(C.size_t(8))  // uint64
	if dFoundFlag == nil || dFoundKey == nil {
		return false, nil, fmt.Errorf("failed to allocate GPU memory for results")
	}
	defer C.cuda_free(dFoundFlag)
	defer C.cuda_free(dFoundKey)

	// Initialize found flag to 0
	zeroFlag := int32(0)
	C.cuda_memcpy_htod(dFoundFlag, unsafe.Pointer(&zeroFlag), 4)

	// Convert target address to C string
	cTarget := C.CString(targetAddress)
	defer C.free(unsafe.Pointer(cTarget))

	// Launch kernel with target address
	startValue := start.Uint64()
	if C.cuda_launch_key_generation(dKeys, nil, C.uint64_t(startValue), C.uint64_t(count), cTarget) == 0 {
		return false, nil, fmt.Errorf("failed to launch CUDA kernel: %s", C.GoString(C.cuda_get_last_error()))
	}

	// Wait for completion
	if C.cuda_device_synchronize() == 0 {
		return false, nil, fmt.Errorf("CUDA synchronization failed: %s", C.GoString(C.cuda_get_last_error()))
	}

	// Check if found
	var foundFlag int32
	var foundKey uint64

	C.cuda_memcpy_dtoh(unsafe.Pointer(&foundFlag), dFoundFlag, 4)
	C.cuda_memcpy_dtoh(unsafe.Pointer(&foundKey), dFoundKey, 8)

	if foundFlag != 0 {
		return true, new(big.Int).SetUint64(foundKey), nil
	}

	return false, nil, nil
}

// Cleanup releases GPU resources
func (w *GPUWorker) Cleanup() {
	// CUDA cleanup is handled automatically by the driver
	// But we can do explicit cleanup if needed
}

// IsAvailable checks if CUDA is available
func IsAvailable() bool {
	if !initialized {
		Init()
	}
	return initialized && len(cudaDevices) > 0
}

// GetDeviceInfo returns information about available CUDA devices
func GetDeviceInfo() ([]map[string]interface{}, error) {
	if !initialized {
		if _, err := Init(); err != nil {
			return nil, err
		}
	}

	info := make([]map[string]interface{}, len(cudaDevices))
	for i, device := range cudaDevices {
		info[i] = map[string]interface{}{
			"id":              device.DeviceID,
			"name":            device.Name,
			"compute":         device.ComputeCapability,
			"memory":          device.TotalMemory,
			"free_memory":     device.FreeMemory,
			"cores":           device.MultiProcessors * getCoresPerSM(device.ComputeCapability),
			"multiprocessors": device.MultiProcessors,
			"batch_size":      device.BatchSize,
		}
	}

	return info, nil
}

// GetGPUCount returns the number of available GPUs
func GetGPUCount() int {
	if !initialized {
		Init()
	}
	return len(cudaDevices)
}

// Helper functions

func calculateOptimalBatchSize(totalMemory uint64) int {
	// Reserve 20% of memory for system use
	availableMemory := totalMemory * 80 / 100

	// Each key processing needs approximately 1KB of memory
	batchSize := int(availableMemory / 1024)

	// Clamp to reasonable limits
	if batchSize > maxBatchSize {
		batchSize = maxBatchSize
	}
	if batchSize < minBatchSize {
		batchSize = minBatchSize
	}

	// Round to nearest power of 2 for better GPU performance
	return roundToPowerOf2(batchSize)
}

func roundToPowerOf2(n int) int {
	power := 1
	for power < n {
		power *= 2
	}
	return power
}

func getCoresPerSM(computeCapability string) int {
	// Estimate CUDA cores per SM based on compute capability
	switch computeCapability {
	case "3.0", "3.5", "3.7": // Kepler
		return 192
	case "5.0", "5.2": // Maxwell
		return 128
	case "6.0": // Pascal GP100
		return 64
	case "6.1", "6.2": // Pascal
		return 128
	case "7.0": // Volta
		return 64
	case "7.5": // Turing
		return 64
	case "8.0": // Ampere GA100
		return 64
	case "8.6": // Ampere
		return 128
	case "8.9": // Ada Lovelace
		return 128
	default:
		return 128 // Default estimate
	}
}

// GetMemoryInfo returns current GPU memory usage
func (w *GPUWorker) GetMemoryInfo() (used, total uint64) {
	var info C.CudaDeviceInfo
	if C.cuda_get_device_info(C.int(w.DeviceID), &info) == 1 {
		total = uint64(info.totalMem)
		used = total - uint64(info.freeMem)
		return used, total
	}
	return 0, 0
}

// SetBatchSize updates the batch size
func (w *GPUWorker) SetBatchSize(size int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if size < minBatchSize {
		size = minBatchSize
	}
	if size > maxBatchSize {
		size = maxBatchSize
	}

	w.BatchSize = size
}

// GetBatchSize returns current batch size
func (w *GPUWorker) GetBatchSize() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.BatchSize
}
