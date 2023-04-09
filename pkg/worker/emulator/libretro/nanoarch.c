#include "libretro.h"
#include <pthread.h>
#include <stdbool.h>
#include <stdarg.h>
#include <stdio.h>

int initialized = 0;

void coreLog(enum retro_log_level level, const char *msg);

void bridge_retro_init(void *f) {
    coreLog(RETRO_LOG_INFO, "Initialization...\n");
    ((void (*)(void)) f)();
}

void bridge_retro_deinit(void *f) {
    coreLog(RETRO_LOG_INFO, "Deinitialiazation...\n");
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
    coreLog(RETRO_LOG_INFO, "Loading the game...\n");
    return ((bool (*)(struct retro_game_info *)) f)(gi);
}

void bridge_retro_unload_game(void *f) {
    coreLog(RETRO_LOG_INFO, "Unloading the game...\n");
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
    ((size_t (*)(void)) f)();
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
    if (v > 0) {
        coreLog(RETRO_LOG_DEBUG, "CLEAR_ALL_THREAD_WAITS_CB (1)\n");
    } else {
        coreLog(RETRO_LOG_DEBUG, "CLEAR_ALL_THREAD_WAITS_CB (0)\n");
    }
    return true;
}

void bridge_clear_all_thread_waits_cb(void *data) {
    *(retro_environment_t *)data = clear_all_thread_waits_cb;
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

void core_log_cgo(enum retro_log_level level, const char *fmt, ...) {
    char msg[4096] = {0};
    va_list va;
    va_start(va, fmt);
    vsnprintf(msg, sizeof(msg), fmt, va);
    va_end(va);
    coreLog(level, msg);
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

void *function;
pthread_t thread;
pthread_mutex_t run_mutex;
pthread_cond_t run_cv;
pthread_mutex_t done_mutex;
pthread_cond_t done_cv;

void *run_loop(void *unused) {
    coreLog(RETRO_LOG_DEBUG, "UnLIBCo run loop start\n");
    pthread_mutex_lock(&done_mutex);
    pthread_mutex_lock(&run_mutex);
    pthread_cond_signal(&done_cv);
    pthread_mutex_unlock(&done_mutex);
    while (initialized) {
        pthread_cond_wait(&run_cv, &run_mutex);
        ((void (*)(void)) function)();
        pthread_mutex_lock(&done_mutex);
        pthread_cond_signal(&done_cv);
        pthread_mutex_unlock(&done_mutex);
    }
    pthread_mutex_unlock(&run_mutex);
    coreLog(RETRO_LOG_DEBUG, "UnLIBCo run loop stop\n");
}

void stop() { initialized = 0; }

void same_thread(void *f) {
    if (!initialized) {
        initialized = 1;
        pthread_mutex_init(&run_mutex, NULL);
        pthread_cond_init(&run_cv, NULL);
        pthread_mutex_init(&done_mutex, NULL);
        pthread_cond_init(&done_cv, NULL);
        pthread_mutex_lock(&done_mutex);
        pthread_create(&thread, NULL, run_loop, NULL);
        pthread_cond_wait(&done_cv, &done_mutex);
        pthread_mutex_unlock(&done_mutex);
    }
    pthread_mutex_lock(&run_mutex);
    pthread_mutex_lock(&done_mutex);
    function = f;
    pthread_cond_signal(&run_cv);
    pthread_mutex_unlock(&run_mutex);
    pthread_cond_wait(&done_cv, &done_mutex);
    pthread_mutex_unlock(&done_mutex);
}
