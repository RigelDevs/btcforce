// internal/gpu/cuda_headers/cuda_wrapper.h
#ifndef CUDA_WRAPPER_H
#define CUDA_WRAPPER_H

#include <stdint.h>
#include <stddef.h>

typedef struct {
    int device;
    char name[256];
    size_t totalMem;
    size_t freeMem;
    int major;
    int minor;
    int multiProcessorCount;
} CudaDeviceInfo;

#ifdef __cplusplus
extern "C" {
#endif

// CUDA device management
int cuda_get_device_count();
int cuda_get_device_info(int device, CudaDeviceInfo* info);
int cuda_set_device(int device);

// Memory management
void* cuda_malloc(size_t size);
void cuda_free(void* ptr);
int cuda_memcpy_htod(void* dst, const void* src, size_t size);
int cuda_memcpy_dtoh(void* dst, const void* src, size_t size);

// Kernel execution
int cuda_launch_key_generation(void* d_keys, void* d_addresses, uint64_t start, uint64_t count, const char* target);

// Synchronization and error handling
int cuda_device_synchronize();
const char* cuda_get_last_error();

#ifdef __cplusplus
}
#endif

#endif // CUDA_WRAPPER_H