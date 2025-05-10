// internal/gpu/gpu_cuda_wrapper.c
// C wrapper for CUDA functionality

#include <cuda_runtime.h>
#include <string.h>
#include <stdio.h>

typedef struct {
    int device;
    char name[256];
    size_t totalMem;
    size_t freeMem;
    int major;
    int minor;
    int multiProcessorCount;
} CudaDeviceInfo;

// Get number of CUDA devices
int cuda_get_device_count() {
    int count = 0;
    cudaError_t err = cudaGetDeviceCount(&count);
    if (err != cudaSuccess) {
        return 0;
    }
    return count;
}

// Get device information
int cuda_get_device_info(int device, CudaDeviceInfo* info) {
    cudaDeviceProp prop;
    cudaError_t err = cudaGetDeviceProperties(&prop, device);
    if (err != cudaSuccess) {
        return 0;
    }
    
    info->device = device;
    strncpy(info->name, prop.name, 255);
    info->name[255] = '\0';
    info->totalMem = prop.totalGlobalMem;
    info->major = prop.major;
    info->minor = prop.minor;
    info->multiProcessorCount = prop.multiProcessorCount;
    
    // Get free memory
    size_t free, total;
    cudaMemGetInfo(&free, &total);
    info->freeMem = free;
    
    return 1;
}

// Set active CUDA device
int cuda_set_device(int device) {
    return cudaSetDevice(device) == cudaSuccess ? 1 : 0;
}

// Allocate GPU memory
void* cuda_malloc(size_t size) {
    void* ptr = NULL;
    cudaError_t err = cudaMalloc(&ptr, size);
    if (err != cudaSuccess) {
        return NULL;
    }
    return ptr;
}

// Free GPU memory
void cuda_free(void* ptr) {
    cudaFree(ptr);
}

// Copy from host to device
int cuda_memcpy_htod(void* dst, const void* src, size_t size) {
    return cudaMemcpy(dst, src, size, cudaMemcpyHostToDevice) == cudaSuccess ? 1 : 0;
}

// Copy from device to host
int cuda_memcpy_dtoh(void* dst, const void* src, size_t size) {
    return cudaMemcpy(dst, src, size, cudaMemcpyDeviceToHost) == cudaSuccess ? 1 : 0;
}

// Synchronize device
int cuda_device_synchronize() {
    return cudaDeviceSynchronize() == cudaSuccess ? 1 : 0;
}

// Get last error as string
const char* cuda_get_last_error() {
    cudaError_t err = cudaGetLastError();
    return cudaGetErrorString(err);
}

// External CUDA kernel functions (implemented in .cu file)
extern "C" {
    int cuda_launch_key_generation(void* d_keys, void* d_addresses, uint64_t start, uint64_t count, const char* target);
}

// Stub implementation for the kernel launcher
// This would be implemented in a .cu file with actual CUDA kernels
int cuda_launch_key_generation(void* d_keys, void* d_addresses, uint64_t start, uint64_t count, const char* target) {
    // This is a placeholder - actual implementation would be in a .cu file
    // For now, just fill with sequential values for testing
    
    uint64_t* keys = (uint64_t*)malloc(count * sizeof(uint64_t));
    if (!keys) return 0;
    
    for (uint64_t i = 0; i < count; i++) {
        keys[i] = start + i;
    }
    
    // Copy to device
    cudaError_t err = cudaMemcpy(d_keys, keys, count * sizeof(uint64_t), cudaMemcpyHostToDevice);
    free(keys);
    
    if (err != cudaSuccess) {
        return 0;
    }
    
    // For addresses, we'd generate them on GPU normally
    // This is just a placeholder
    if (d_addresses) {
        char* addresses = (char*)malloc(count * 64);
        if (addresses) {
            memset(addresses, 0, count * 64);
            for (uint64_t i = 0; i < count; i++) {
                snprintf(&addresses[i * 64], 64, "1Address%llu", start + i);
            }
            cudaMemcpy(d_addresses, addresses, count * 64, cudaMemcpyHostToDevice);
            free(addresses);
        }
    }
    
    return 1;
}