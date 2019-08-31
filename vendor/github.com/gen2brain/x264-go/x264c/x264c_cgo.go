// +build !extlib

package x264c

/*
#include "external/x264/common/mc.c"
#include "external/x264/common/predict.c"
#include "external/x264/common/pixel.c"
#include "external/x264/common/macroblock.c"
#include "external/x264/common/frame.c"
#include "external/x264/common/dct.c"
#include "external/x264/common/cpu.c"
#include "external/x264/common/cabac.c"
#include "external/x264/common/common.c"
#include "external/x264/common/osdep.c"
#include "external/x264/common/rectangle.c"
#include "external/x264/common/set.c"
#include "external/x264/common/quant.c"
#include "external/x264/common/deblock.c"
#include "external/x264/common/vlc.c"
#include "external/x264/common/mvpred.c"
#include "external/x264/common/bitstream.c"

//#include "external/x264/encoder/analyse.c"
#include "external/x264/encoder/me.c"
#include "external/x264/encoder/ratecontrol.c"
#include "external/x264/encoder/set.c"
#include "external/x264/encoder/macroblock.c"
#include "external/x264/encoder/cabac.c"
#include "external/x264/encoder/cavlc.c"
#include "external/x264/encoder/encoder.c"
#include "external/x264/encoder/lookahead.c"

#include "external/x264/common/threadpool.c"

#ifdef HAVE_WIN32THREAD
#include "external/x264/common/win32thread.c"
#endif

#cgo android LDFLAGS: -lm
#cgo windows LDFLAGS:
#cgo linux,!android LDFLAGS: -lpthread -lm
#cgo darwin LDFLAGS: -lpthread -lm

#cgo linux,386 CFLAGS: -DSYS_LINUX=1 -DARCH_X86=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=64 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=1 -DTHP=1
#cgo linux,amd64 CFLAGS: -DSYS_LINUX=1 -DARCH_X86_64=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=64 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=1 -DTHP=1
#cgo linux,!android,arm CFLAGS: -DSYS_LINUX=1 -DARCH_ARM=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=4 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=1 -DTHP=1
#cgo linux,!android,arm64 CFLAGS: -DSYS_LINUX=1 -DARCH_AARCH64=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=16 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=1 -DTHP=1
#cgo windows,386 CFLAGS: -DSYS_WINDOWS=1 -DARCH_X86=1 -DHAVE_WIN32THREAD=1 -DSTACK_ALIGNMENT=64 -DHAVE_MALLOC_H=0 -DHAVE_CPU_COUNT=0 -DTHP=0
#cgo windows,amd64 CFLAGS: -DSYS_WINDOWS=1 -DARCH_X86_64=1 -DHAVE_WIN32THREAD=1 -DSTACK_ALIGNMENT=16 -DHAVE_MALLOC_H=0 -DHAVE_CPU_COUNT=0 -DTHP=0
#cgo darwin,amd64 CFLAGS: -DSYS_MACOSX=1 -DARCH_X86_64=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=16 -DHAVE_MALLOC_H=0 -DHAVE_CPU_COUNT=0 -DTHP=1
#cgo android,arm CFLAGS: -DSYS_LINUX=1 -DARCH_ARM=1 -DHAVE_THREAD=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=4 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=0 -DTHP=1
#cgo android,arm64 CFLAGS: -DSYS_LINUX=1 -DARCH_AARCH64=1 -DHAVE_THREAD=1 -DHAVE_POSIXTHREAD=1 -DSTACK_ALIGNMENT=16 -DHAVE_MALLOC_H=1 -DHAVE_CPU_COUNT=1 -DTHP=1

#cgo CFLAGS: -std=gnu99 -Iexternal/x264 -D_GNU_SOURCE -fomit-frame-pointer -Wshadow -O3
#cgo CFLAGS: -DHAVE_THREAD=1 -DHAVE_LOG2F=1 -DHAVE_STRTOK_R=1 -DHAVE_MMAP=1 -DHAVE_GPL=1 -DHAVE_INTERLACED=1
#cgo CFLAGS: -DHAVE_SWSCALE=0 -DHAVE_LAVF=0 -DHAVE_AVS=0 -DUSE_AVXSYNTH=0 -DHAVE_VECTOREXT=1 -DHAVE_BITDEPTH8=1 -DHIGH_BIT_DEPTH=0
#cgo CFLAGS: -DHAVE_OPENCL=0 -DHAVE_ALTIVEC=0 -DHAVE_ALTIVEC_H=0 -DHAVE_FFMS=0 -DHAVE_GPAC=0 -DHAVE_LSMASH=0
#cgo CFLAGS: -DX264_BIT_DEPTH=8 -DX264_GPL=11 -DX264_INTERLACED=1 -DX264_CHROMA_FORMAT=0
*/
import "C"
