#ifndef SPEEX_RESAMPLER_H
#define SPEEX_RESAMPLER_H

#define spx_int16_t short
#define spx_int32_t int
#define spx_uint16_t unsigned short
#define spx_uint32_t unsigned int

#define SPEEX_RESAMPLER_QUALITY_MAX 10
#define SPEEX_RESAMPLER_QUALITY_MIN 0
#define SPEEX_RESAMPLER_QUALITY_DEFAULT 4
#define SPEEX_RESAMPLER_QUALITY_VOIP 3
#define SPEEX_RESAMPLER_QUALITY_DESKTOP 5
enum {
   RESAMPLER_ERR_SUCCESS         = 0,
   RESAMPLER_ERR_ALLOC_FAILED    = 1,
   RESAMPLER_ERR_BAD_STATE       = 2,
   RESAMPLER_ERR_INVALID_ARG     = 3,
   RESAMPLER_ERR_PTR_OVERLAP     = 4,

   RESAMPLER_ERR_MAX_ERROR
};
struct SpeexResamplerState_;
typedef struct SpeexResamplerState_ SpeexResamplerState;
/** Create a new resampler with integer input and output rates.
 * @param nb_channels Number of channels to be processed
 * @param in_rate Input sampling rate (integer number of Hz).
 * @param out_rate Output sampling rate (integer number of Hz).
 * @param quality Resampling quality between 0 and 10, where 0 has poor quality
 * and 10 has very high quality.
 * @return Newly created resampler state
 * @retval NULL Error: not enough memory
 */
SpeexResamplerState *speex_resampler_init(spx_uint32_t nb_channels,
                                          spx_uint32_t in_rate,
                                          spx_uint32_t out_rate,
                                          int quality,
                                          int *err);
/** Destroy a resampler state.
 * @param st Resampler state
 */
void speex_resampler_destroy(SpeexResamplerState *st);


/** Make sure that the first samples to go out of the resamplers don't have
 * leading zeros. This is only useful before starting to use a newly created
 * resampler. It is recommended to use that when resampling an audio file, as
 * it will generate a file with the same length. For real-time processing,
 * it is probably easier not to use this call (so that the output duration
 * is the same for the first frame).
 * @param st Resampler state
 */
int speex_resampler_skip_zeros(SpeexResamplerState *st);

/** Resample an interleaved int array. The input and output buffers must *not* overlap.
 * @param st Resampler state
 * @param in Input buffer
 * @param in_len Number of input samples in the input buffer. Returns the number
 * of samples processed. This is all per-channel.
 * @param out Output buffer
 * @param out_len Size of the output buffer. Returns the number of samples written.
 * This is all per-channel.
 */
int speex_resampler_process_interleaved_int(SpeexResamplerState *st,
                                             const spx_int16_t *in,
                                             spx_uint32_t *in_len,
                                             spx_int16_t *out,
                                             spx_uint32_t *out_len);
const char *speex_resampler_strerror(int err);
#endif