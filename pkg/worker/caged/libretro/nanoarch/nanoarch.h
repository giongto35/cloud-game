#ifndef FRONTEND_H__
#define FRONTEND_H__

bool bridge_retro_load_game(void *f, struct retro_game_info *gi);
void bridge_retro_unload_game(void *f);
bool bridge_retro_serialize(void *f, void *data, size_t size);
size_t bridge_retro_serialize_size(void *f);
bool bridge_retro_unserialize(void *f, void *data, size_t size);
bool bridge_retro_set_environment(void *f, void *callback);
unsigned bridge_retro_api_version(void *f);
size_t bridge_retro_get_memory_size(void *f, unsigned id);
void *bridge_retro_get_memory_data(void *f, unsigned id);
void bridge_context_reset(retro_hw_context_reset_t f);
void bridge_retro_deinit(void *f);
void bridge_retro_get_system_av_info(void *f, struct retro_system_av_info *si);
void bridge_retro_get_system_info(void *f, struct retro_system_info *si);
void bridge_retro_init(void *f);
void bridge_retro_run(void *f);
void bridge_retro_set_audio_sample(void *f, void *callback);
void bridge_retro_set_audio_sample_batch(void *f, void *callback);
void bridge_retro_set_controller_port_device(void *f, unsigned port, unsigned device);
void bridge_retro_set_input_poll(void *f, void *callback);
void bridge_retro_set_input_state(void *f, void *callback);
void bridge_retro_set_video_refresh(void *f, void *callback);
void bridge_clear_all_thread_waits_cb(void *f);
void bridge_retro_keyboard_callback(void *f, bool down, unsigned keycode, uint32_t character, uint16_t keyModifiers);

bool core_environment_cgo(unsigned cmd, void *data);
int16_t core_input_state_cgo(unsigned port, unsigned device, unsigned index, unsigned id);
retro_proc_address_t core_get_proc_address_cgo(const char *sym);
size_t core_audio_sample_batch_cgo(const int16_t *data, size_t frames);
uintptr_t core_get_current_framebuffer_cgo();
void core_audio_sample_cgo(int16_t left, int16_t right);
void core_input_poll_cgo();
void core_log_cgo(int level, const char *msg);
void core_video_refresh_cgo(void *data, unsigned width, unsigned height, size_t pitch);
void init_video_cgo();
void deinit_video_cgo();

void same_thread(void *f);
void *same_thread_with_args2(void *f, int type, void *arg1, void *arg2);
void same_thread_stop();

#endif
