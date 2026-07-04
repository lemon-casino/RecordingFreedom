#ifndef LIKELY_VOICE_ENHANCER_H
#define LIKELY_VOICE_ENHANCER_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct LikelyVoiceEnhancer LikelyVoiceEnhancer;

#ifndef LIKELY_VOICE_ENHANCER_EXPORT
# if defined(_WIN32) && defined(LIKELY_VOICE_ENHANCER_BUILD_DLL)
#  define LIKELY_VOICE_ENHANCER_EXPORT __declspec(dllexport)
# elif defined(__GNUC__) && defined(LIKELY_VOICE_ENHANCER_BUILD_DLL)
#  define LIKELY_VOICE_ENHANCER_EXPORT __attribute__ ((visibility ("default")))
# else
#  define LIKELY_VOICE_ENHANCER_EXPORT
# endif
#endif

LIKELY_VOICE_ENHANCER_EXPORT const char* likely_voice_enhancer_algorithm(void);
LIKELY_VOICE_ENHANCER_EXPORT int likely_voice_enhancer_required_sample_rate(void);
LIKELY_VOICE_ENHANCER_EXPORT int likely_voice_enhancer_frame_size(void);
LIKELY_VOICE_ENHANCER_EXPORT int likely_voice_enhancer_is_supported_format(int sample_rate, int channels);

LIKELY_VOICE_ENHANCER_EXPORT LikelyVoiceEnhancer* likely_voice_enhancer_create(int sample_rate, int channels, float output_gain);
LIKELY_VOICE_ENHANCER_EXPORT LikelyVoiceEnhancer* likely_voice_enhancer_create_milli_gain(int sample_rate, int channels, int output_gain_milli);
LIKELY_VOICE_ENHANCER_EXPORT void likely_voice_enhancer_destroy(LikelyVoiceEnhancer* enhancer);
LIKELY_VOICE_ENHANCER_EXPORT void likely_voice_enhancer_reset(LikelyVoiceEnhancer* enhancer);

LIKELY_VOICE_ENHANCER_EXPORT int likely_voice_enhancer_process_interleaved_float(
    LikelyVoiceEnhancer* enhancer,
    float* samples,
    int frame_count,
    int channels);

LIKELY_VOICE_ENHANCER_EXPORT int likely_voice_enhancer_process_interleaved_int16(
    LikelyVoiceEnhancer* enhancer,
    int16_t* samples,
    int frame_count,
    int channels);

#ifdef __cplusplus
}
#endif

#endif
