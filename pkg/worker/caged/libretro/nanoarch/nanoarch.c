#include "libretro.h"
#include <pthread.h>
#include <stdbool.h>
#include <stdarg.h>
#include <stdio.h>

int initialized = 0;

typedef struct {
	int   type;
	void* fn;
	void* arg1;
	void* arg2;
	void* result;
} call_def_t;

call_def_t call;

enum call_type {
    CALL_VOID = -1,
    CALL_SERIALIZE = 1,
    CALL_UNSERIALIZE = 2,
};

void *same_thread_with_args(void *f, int type, ...);

void core_log_cgo(enum retro_log_level level, const char *fmt, ...) {
    char msg[2048] = {0};
    va_list va;
    va_start(va, fmt);
    vsnprintf(msg, sizeof(msg), fmt, va);
    va_end(va);
    void coreLog(enum retro_log_level level, const char *msg);
    coreLog(level, msg);
}

void bridge_retro_init(void *f) {
    core_log_cgo(RETRO_LOG_DEBUG, "Initialization...\n");
    ((void (*)(void)) f)();
}

void bridge_retro_deinit(void *f) {
    core_log_cgo(RETRO_LOG_DEBUG, "Deinitialiazation...\n");
    ((void (*)(void)) f)();
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

void bridge_retro_set_video_refresh(void *f, void *callback) {
    ((bool (*)(retro_video_refresh_t)) f)((retro_video_refresh_t) callback);
}

void bridge_retro_set_input_poll(void *f, void *callback) {
    ((bool (*)(retro_input_poll_t)) f)((retro_input_poll_t) callback);
}

void bridge_retro_set_input_state(void *f, void *callback) {
    ((bool (*)(retro_input_state_t)) f)((retro_input_state_t) callback);
}

void bridge_retro_set_audio_sample(void *f, void *callback) {
    ((bool (*)(retro_audio_sample_t)) f)((retro_audio_sample_t) callback);
}

void bridge_retro_set_audio_sample_batch(void *f, void *callback) {
    ((bool (*)(retro_audio_sample_batch_t)) f)((retro_audio_sample_batch_t) callback);
}

bool bridge_retro_load_game(void *f, struct retro_game_info *gi) {
    core_log_cgo(RETRO_LOG_DEBUG, "Loading the game...\n");
    return ((bool (*)(struct retro_game_info *)) f)(gi);
}

void bridge_retro_unload_game(void *f) {
    core_log_cgo(RETRO_LOG_DEBUG, "Unloading the game...\n");
    ((void (*)(void)) f)();
}

void bridge_retro_run(void *f) {
    ((void (*)(void)) f)();
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

static bool clear_all_thread_waits_cb(unsigned v, void *data) {
    core_log_cgo(RETRO_LOG_DEBUG, "CLEAR_ALL_THREAD_WAITS_CB (%d)\n", v);
    return true;
}

void bridge_clear_all_thread_waits_cb(void *data) {
    *(retro_environment_t *)data = clear_all_thread_waits_cb;
}

void bridge_retro_keyboard_callback(void *cb, bool down, unsigned keycode, uint32_t character, uint16_t keyModifiers) {
    (*(retro_keyboard_event_t *) cb)(down, keycode, character, keyModifiers);
}

bool core_environment_cgo(unsigned cmd, void *data) {
    bool coreEnvironment(unsigned, void *);
    return coreEnvironment(cmd, data);
}

void core_video_refresh_cgo(void *data, unsigned width, unsigned height, size_t pitch) {
    void coreVideoRefresh(void *, unsigned, unsigned, size_t);
    coreVideoRefresh(data, width, height, pitch);
}

void core_input_poll_cgo() {
    void coreInputPoll();
    coreInputPoll();
}

int16_t core_input_state_cgo(unsigned port, unsigned device, unsigned index, unsigned id) {
    int16_t coreInputState(unsigned, unsigned, unsigned, unsigned);
    return coreInputState(port, device, index, id);
}

void core_audio_sample_cgo(int16_t left, int16_t right) {
    void coreAudioSample(int16_t, int16_t);
    coreAudioSample(left, right);
}

size_t core_audio_sample_batch_cgo(const int16_t *data, size_t frames) {
    size_t coreAudioSampleBatch(const int16_t *, size_t);
    return coreAudioSampleBatch(data, frames);
}

uintptr_t core_get_current_framebuffer_cgo() {
    uintptr_t coreGetCurrentFramebuffer();
    return coreGetCurrentFramebuffer();
}

retro_proc_address_t core_get_proc_address_cgo(const char *sym) {
    retro_proc_address_t coreGetProcAddress(const char *sym);
    return coreGetProcAddress(sym);
}

void bridge_context_reset(retro_hw_context_reset_t f) {
    f();
}

void init_video_cgo() {
    void initVideo();
    initVideo();
}

void deinit_video_cgo() {
    void deinitVideo();
    deinitVideo();
}

typedef struct {
   pthread_mutex_t m;
   pthread_cond_t cond;
} mutex_t;

void mutex_init(mutex_t *m) {
    pthread_mutex_init(&m->m, NULL);
    pthread_cond_init(&m->cond, NULL);
}

void mutex_destroy(mutex_t *m) {
    pthread_mutex_trylock(&m->m);
    pthread_mutex_unlock(&m->m);
    pthread_mutex_destroy(&m->m);
    pthread_cond_signal(&m->cond);
    pthread_cond_destroy(&m->cond);
}

void mutex_lock(mutex_t *m)   { pthread_mutex_lock(&m->m); }
void mutex_wait(mutex_t *m)   { pthread_cond_wait(&m->cond, &m->m); }
void mutex_unlock(mutex_t *m) { pthread_mutex_unlock(&m->m); }
void mutex_signal(mutex_t *m) { pthread_cond_signal(&m->cond); }

static pthread_t thread;
mutex_t run_mutex, done_mutex;

void *run_loop(void *unused) {
    core_log_cgo(RETRO_LOG_DEBUG, "UnLibCo run loop start\n");
    mutex_lock(&done_mutex);
    mutex_lock(&run_mutex);
    mutex_signal(&done_mutex);
    mutex_unlock(&done_mutex);
    while (initialized) {
        mutex_wait(&run_mutex);
        switch (call.type) {
            case CALL_SERIALIZE:
            case CALL_UNSERIALIZE:
              *(bool*)call.result = ((bool (*)(void*, size_t))call.fn)(call.arg1, *(size_t*)call.arg2);
              break;
            default:
                ((void (*)(void)) call.fn)();
        }
        mutex_lock(&done_mutex);
        mutex_signal(&done_mutex);
        mutex_unlock(&done_mutex);
    }
    mutex_destroy(&run_mutex);
    mutex_destroy(&done_mutex);
    pthread_detach(thread);
    core_log_cgo(RETRO_LOG_DEBUG, "UnLibCo run loop stop\n");
}

void same_thread_stop() {
    initialized = 0;
}

void *same_thread_with_args(void *f, int type, ...) {
    if (!initialized) {
        initialized = 1;
        mutex_init(&run_mutex);
        mutex_init(&done_mutex);
        mutex_lock(&done_mutex);
        pthread_create(&thread, NULL, run_loop, NULL);
        mutex_wait(&done_mutex);
        mutex_unlock(&done_mutex);
    }
    mutex_lock(&run_mutex);
    mutex_lock(&done_mutex);

    call.type = type;
    call.fn = f;

    if (type != CALL_VOID) {
        va_list args;
        va_start(args, type);
        switch (type) {
            case CALL_SERIALIZE:
            case CALL_UNSERIALIZE:
                call.arg1 = va_arg(args, void*);
                size_t size;
                size = va_arg(args, size_t);
                call.arg2 = &size;
                bool result;
                call.result = &result;
              break;
        }
        va_end(args);
    }
    mutex_signal(&run_mutex);
    mutex_unlock(&run_mutex);
    mutex_wait(&done_mutex);
    mutex_unlock(&done_mutex);
    return call.result;
}

void *same_thread_with_args2(void *f, int type, void *arg1, void *arg2) {
    return same_thread_with_args(f, type, arg1, arg2);
}

void same_thread(void *f) {
    same_thread_with_args(f, CALL_VOID);
}
