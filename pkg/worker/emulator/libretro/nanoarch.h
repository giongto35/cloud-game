#ifndef FRONTEND_H__
#define FRONTEND_H__

void bridge_retro_init(void *f);

void bridge_retro_deinit(void *f);

unsigned bridge_retro_api_version(void *f);

void bridge_retro_get_system_info(void *f, struct retro_system_info *si);

void bridge_retro_get_system_av_info(void *f, struct retro_system_av_info *si);

bool bridge_retro_set_environment(void *f, void *callback);

void bridge_retro_set_video_refresh(void *f, void *callback);

void bridge_retro_set_input_poll(void *f, void *callback);

void bridge_retro_set_input_state(void *f, void *callback);

void bridge_retro_set_audio_sample(void *f, void *callback);

void bridge_retro_set_audio_sample_batch(void *f, void *callback);

bool bridge_retro_load_game(void *f, struct retro_game_info *gi);

void bridge_retro_unload_game(void *f);

void bridge_retro_run(void *f);

void bridge_retro_set_controller_port_device(void *f, unsigned port, unsigned device);

size_t bridge_retro_get_memory_size(void *f, unsigned id);

void *bridge_retro_get_memory_data(void *f, unsigned id);

bool bridge_retro_serialize(void *f, void *data, size_t size);

bool bridge_retro_unserialize(void *f, void *data, size_t size);

size_t bridge_retro_serialize_size(void *f);

bool coreEnvironment_cgo(unsigned cmd, void *data);

void coreVideoRefresh_cgo(void *data, unsigned width, unsigned height, size_t pitch);

void coreInputPoll_cgo();

void coreAudioSample_cgo(int16_t left, int16_t right);

size_t coreAudioSampleBatch_cgo(const int16_t *data, size_t frames);

int16_t coreInputState_cgo(unsigned port, unsigned device, unsigned index, unsigned id);

void coreLog_cgo(int level, const char *msg);

uintptr_t coreGetCurrentFramebuffer_cgo();

retro_proc_address_t coreGetProcAddress_cgo(const char *sym);

void bridge_context_reset(retro_hw_context_reset_t f);

void initVideo_cgo();

void deinitVideo_cgo();

void bridge_execute(void *f);

#endif
