#include "libretro.h"

#include <pthread.h>
#include <stdbool.h>
#include <stdarg.h>
#include <stdatomic.h>
#include <stdio.h>
#include <string.h>

#define RETRO_ENVIRONMENT_GET_CLEAR_ALL_THREAD_WAITS_CB (3 | 0x800000)

// ============================================================================
// Call types for same_thread operations
// ============================================================================

enum call_type {
    CALL_VOID = 0,
    CALL_SERIALIZE = 1,
    CALL_UNSERIALIZE = 2,
};

// ============================================================================
// Lock-free call structure
// ============================================================================

typedef struct {
    atomic_int state;      // 0=idle, 1=pending, 2=done
    int type;
    void *fn;
    void *arg1;
    size_t arg2;
    bool result;
} lf_call_t;

static lf_call_t lf_call = {0};
static atomic_int thread_running = 0;
static pthread_t worker_thread;

// ============================================================================
// Logging
// ============================================================================

void core_log_cgo(enum retro_log_level level, const char *fmt, ...) {
    char msg[2048] = {0};
    va_list va;
    va_start(va, fmt);
    vsnprintf(msg, sizeof(msg), fmt, va);
    va_end(va);
    void coreLog(enum retro_log_level level, const char *msg);
    coreLog(level, msg);
}

// ============================================================================
// Bridge functions for calling libretro core
// ============================================================================

void bridge_call(void *f) {
    ((void (*)(void)) f)();
}

void bridge_set_callback(void *f, void *callback) {
    ((void (*)(void *))f)(callback);
}

unsigned bridge_retro_api_version(void *f) {
    return ((unsigned (*)(void)) f)();
}

void bridge_retro_get_system_info(void *f, struct retro_system_info *si) {
    ((void (*)(struct retro_system_info *)) f)(si);
}

void bridge_retro_get_system_av_info(void *f, struct retro_system_av_info *si) {
    ((void (*)(struct retro_system_av_info *)) f)(si);
}

bool bridge_retro_set_environment(void *f, void *callback) {
    return ((bool (*)(retro_environment_t)) f)((retro_environment_t) callback);
}

void bridge_retro_set_input_state(void *f, void *callback) {
    ((int16_t (*)(retro_input_state_t)) f)((retro_input_state_t) callback);
}

bool bridge_retro_load_game(void *f, struct retro_game_info *gi) {
    return ((bool (*)(struct retro_game_info *)) f)(gi);
}

size_t bridge_retro_get_memory_size(void *f, unsigned id) {
    return ((size_t (*)(unsigned)) f)(id);
}

void *bridge_retro_get_memory_data(void *f, unsigned id) {
    return ((void *(*)(unsigned)) f)(id);
}

size_t bridge_retro_serialize_size(void *f) {
    return ((size_t (*)(void)) f)();
}

bool bridge_retro_serialize(void *f, void *data, size_t size) {
    return ((bool (*)(void *, size_t)) f)(data, size);
}

bool bridge_retro_unserialize(void *f, void *data, size_t size) {
    return ((bool (*)(void *, size_t)) f)(data, size);
}

void bridge_retro_set_controller_port_device(void *f, unsigned port, unsigned device) {
    ((void (*)(unsigned, unsigned)) f)(port, device);
}

void bridge_retro_keyboard_callback(void *cb, bool down, unsigned keycode, uint32_t character, uint16_t keyModifiers) {
    (*(retro_keyboard_event_t *) cb)(down, keycode, character, keyModifiers);
}

void bridge_context_reset(retro_hw_context_reset_t f) {
    f();
}

// ============================================================================
// Environment callback
// ============================================================================

static bool clear_all_thread_waits_cb(unsigned v, void *data) {
    core_log_cgo(RETRO_LOG_DEBUG, "CLEAR_ALL_THREAD_WAITS_CB (%d)\n", v);
    return true;
}

bool core_environment_cgo(unsigned cmd, void *data) {
    bool coreEnvironment(unsigned, void *);

    switch (cmd)
    {
        case RETRO_ENVIRONMENT_GET_VARIABLE_UPDATE:
            return false;
        case RETRO_ENVIRONMENT_GET_AUDIO_VIDEO_ENABLE:
            return false;
        case RETRO_ENVIRONMENT_GET_CLEAR_ALL_THREAD_WAITS_CB:
            *(retro_environment_t *)data = clear_all_thread_waits_cb;
            return true;
        case RETRO_ENVIRONMENT_GET_INPUT_MAX_USERS:
            *(unsigned *)data = 4;
            core_log_cgo(RETRO_LOG_DEBUG, "Set max users: %d\n", 4);
            return true;
        case RETRO_ENVIRONMENT_GET_INPUT_BITMASKS:
            return false;
        case RETRO_ENVIRONMENT_SHUTDOWN:
            return false;
        case RETRO_ENVIRONMENT_GET_SAVESTATE_CONTEXT:
            if (data != NULL) *(int *)data = RETRO_SAVESTATE_CONTEXT_NORMAL;
            return true;
    }

    return coreEnvironment(cmd, data);
}

// ============================================================================
// Core callbacks
// ============================================================================

void core_video_refresh_cgo(void *data, unsigned width, unsigned height, size_t pitch) {
    void coreVideoRefresh(void *, unsigned, unsigned, size_t);
    coreVideoRefresh(data, width, height, pitch);
}

void core_input_poll_cgo() {
}

int16_t core_input_state_cgo(unsigned port, unsigned device, unsigned index, unsigned id) {
    int16_t coreInputState(unsigned, unsigned, unsigned, unsigned);
    return coreInputState(port, device, index, id);
}

size_t core_audio_sample_batch_cgo(const int16_t *data, size_t frames) {
    size_t coreAudioSampleBatch(const int16_t *, size_t);
    return coreAudioSampleBatch(data, frames);
}

void core_audio_sample_cgo(int16_t left, int16_t right) {
    int16_t frame[2] = { left, right };
    core_audio_sample_batch_cgo(frame, 1);
}

uintptr_t core_get_current_framebuffer_cgo() {
    uintptr_t coreGetCurrentFramebuffer();
    return coreGetCurrentFramebuffer();
}

retro_proc_address_t core_get_proc_address_cgo(const char *sym) {
    retro_proc_address_t coreGetProcAddress(const char *sym);
    return coreGetProcAddress(sym);
}

// ============================================================================
// Video init/deinit
// ============================================================================

void init_video_cgo() {
    void initVideo();
    initVideo();
}

void deinit_video_cgo() {
    void deinitVideo();
    deinitVideo();
}

// ============================================================================
// CPU pause hints for spin loops
// ============================================================================

static inline void cpu_relax(void) {
#if defined(__x86_64__) || defined(_M_X64) || defined(__i386__) || defined(_M_IX86)
    __asm__ volatile("pause" ::: "memory");
#elif defined(__aarch64__)
    __asm__ volatile("isb" ::: "memory");
#elif defined(__arm__)
    __asm__ volatile("yield" ::: "memory");
#else
    // Generic fallback - compiler barrier
    __asm__ volatile("" ::: "memory");
#endif
}

// ============================================================================
// Lock-free same_thread implementation.
// Needed due to C/Go stack grow issues (libco).
// ============================================================================

static void *run_loop_fast(void *unused) {
    core_log_cgo(RETRO_LOG_DEBUG, "Worker thread started\n");

    while (atomic_load_explicit(&thread_running, memory_order_acquire)) {
        // Check if there's a pending call
        int state = atomic_load_explicit(&lf_call.state, memory_order_acquire);

        if (state == 1) {
            // Execute the call
            switch (lf_call.type) {
                case CALL_SERIALIZE:
                    lf_call.result = ((bool (*)(void*, size_t))lf_call.fn)(
                        lf_call.arg1, lf_call.arg2);
                    break;
                case CALL_UNSERIALIZE:
                    lf_call.result = ((bool (*)(void*, size_t))lf_call.fn)(
                        lf_call.arg1, lf_call.arg2);
                    break;
                case CALL_VOID:
                default:
                    ((void (*)(void))lf_call.fn)();
                    break;
            }

            // Mark as done
            atomic_store_explicit(&lf_call.state, 2, memory_order_release);
        } else {
            // Spin with CPU hint to reduce power consumption
            cpu_relax();
        }
    }

    core_log_cgo(RETRO_LOG_DEBUG, "Worker thread stopped\n");
    return NULL;
}

// Initialize the worker thread if not already running
static void same_thread_ensure_init(void) {
    int expected = 0;
    if (atomic_compare_exchange_strong_explicit(
            &thread_running, &expected, 1,
            memory_order_acq_rel, memory_order_acquire)) {
        // We won the race to initialize
        atomic_store_explicit(&lf_call.state, 0, memory_order_release);
        pthread_create(&worker_thread, NULL, run_loop_fast, NULL);
        core_log_cgo(RETRO_LOG_DEBUG, "Worker thread initialized\n");
    }
}

// Stop the worker thread
void same_thread_stop(void) {
    if (atomic_load_explicit(&thread_running, memory_order_acquire)) {
        atomic_store_explicit(&thread_running, 0, memory_order_release);
        pthread_join(worker_thread, NULL);
        core_log_cgo(RETRO_LOG_DEBUG, "Worker thread stopped\n");
    }
}

// Execute a void function on the worker thread
void same_thread(void *f) {
    same_thread_ensure_init();

    // Wait for any previous call to complete
    while (atomic_load_explicit(&lf_call.state, memory_order_acquire) != 0) {
        cpu_relax();
    }

    // Set up the call
    lf_call.fn = f;
    lf_call.type = CALL_VOID;

    // Signal that a call is pending
    atomic_store_explicit(&lf_call.state, 1, memory_order_release);

    // Wait for completion
    while (atomic_load_explicit(&lf_call.state, memory_order_acquire) != 2) {
        cpu_relax();
    }

    // Reset to idle
    atomic_store_explicit(&lf_call.state, 0, memory_order_release);
}

// Execute a serialize/unserialize function on the worker thread
// Returns pointer to the result (stored in lf_call.result)
bool same_thread_serialize(void *f, int type, void *data, size_t size) {
    same_thread_ensure_init();

    // Wait for any previous call to complete
    while (atomic_load_explicit(&lf_call.state, memory_order_acquire) != 0) {
        cpu_relax();
    }

    // Set up the call - store values directly, not pointers to locals!
    lf_call.fn = f;
    lf_call.type = type;
    lf_call.arg1 = data;
    lf_call.arg2 = size;
    lf_call.result = false;

    // Signal that a call is pending
    atomic_store_explicit(&lf_call.state, 1, memory_order_release);

    // Wait for completion
    while (atomic_load_explicit(&lf_call.state, memory_order_acquire) != 2) {
        cpu_relax();
    }

    // Get result before resetting
    bool result = lf_call.result;

    // Reset to idle
    atomic_store_explicit(&lf_call.state, 0, memory_order_release);

    return result;
}

// Execute functions on the same thread.
void *same_thread_with_args2(void *f, int type, void *arg1, void *arg2) {
    size_t size = *(size_t*)arg2;

    static _Thread_local bool result_storage;
    result_storage = same_thread_serialize(f, type, arg1, size);

    return &result_storage;
}