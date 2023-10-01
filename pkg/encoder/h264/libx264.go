// Package h264 implements cgo bindings for [x264](https://www.videolan.org/developers/x264.html) library.
package h264

/*
#cgo !st pkg-config: x264
#cgo st LDFLAGS: -l:libx264.a

#include "stdint.h"
#include "x264.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

const Build = C.X264_BUILD

// T is opaque handler for encoder
type T struct{}

// Nal is The data within the payload is already NAL-encapsulated; the ref_idc and type
// are merely in the struct for easy access by the calling application.
// All data returned in x264_nal_t, including the data in p_payload, is no longer
// valid after the next call to x264_encoder_encode. Thus, it must be used or copied
// before calling x264_encoder_encode or x264_encoder_headers again.
type Nal struct {
	IRefIdc        int32 /* nal_priority_e */
	IType          int32 /* nal_unit_type_e */
	BLongStartcode int32
	IFirstMb       int32 /* If this NAL is a slice, the index of the first MB in the slice. */
	ILastMb        int32 /* If this NAL is a slice, the index of the last MB in the slice. */

	/* Size of payload (including any padding) in bytes. */
	IPayload int32
	/* If param->b_annexb is set, Annex-B bytestream with startcode.
	 * Otherwise, startcode is replaced with a 4-byte size.
	 * This size is the size used in mp4/similar muxing; it is equal to i_payload-4 */
	/* C.uint8_t */
	PPayload unsafe.Pointer

	/* Size of padding in bytes. */
	IPadding int32
}

const RcCrf = 1

const (
	CspI420  = 0x0002 // yuv 4:2:0 planar
	CspVflip = 0x1000 /* the csp is vertically flipped */

	// CspMask      = 0x00ff /* */
	// CspNone      = 0x0000 /* Invalid mode     */
	// CspI400      = 0x0001 /* monochrome 4:0:0 */

	//CspYv12      = 0x0003 /* yvu 4:2:0 planar */
	//CspNv12      = 0x0004 /* yuv 4:2:0, with one y plane and one packed u+v */
	//CspNv21      = 0x0005 /* yuv 4:2:0, with one y plane and one packed v+u */
	//CspI422      = 0x0006 /* yuv 4:2:2 planar */
	//CspYv16      = 0x0007 /* yvu 4:2:2 planar */
	//CspNv16      = 0x0008 /* yuv 4:2:2, with one y plane and one packed u+v */
	//CspYuyv      = 0x0009 /* yuyv 4:2:2 packed */
	//CspUyvy      = 0x000a /* uyvy 4:2:2 packed */
	//CspV210      = 0x000b /* 10-bit yuv 4:2:2 packed in 32 */
	//CspI444      = 0x000c /* yuv 4:4:4 planar */
	//CspYv24      = 0x000d /* yvu 4:4:4 planar */
	//CspBgr       = 0x000e /* packed bgr 24bits */
	//CspBgra      = 0x000f /* packed bgr 32bits */
	//CspRgb       = 0x0010 /* packed rgb 24bits */
	//CspMax       = 0x0011 /* end of list */
	//CspHighDepth = 0x2000 /* the csp has a depth of 16 bits per pixel component */
)

type Zone struct {
	IStart, IEnd   int32 /* range of frame numbers */
	BForceQp       int32 /* whether to use qp vs bitrate factor */
	IQp            int32
	FBitrateFactor float32
	Param          *Param
}

type Param struct {
	/* CPU flags */
	Cpu               uint32
	IThreads          int32 /* encode multiple frames in parallel */
	ILookaheadThreads int32 /* multiple threads for lookahead analysis */
	BSlicedThreads    int32 /* Whether to use slice-based threading. */
	BDeterministic    int32 /* whether to allow non-deterministic optimizations when threaded */
	BCpuIndependent   int32 /* force canonical behavior rather than cpu-dependent optimal algorithms */
	ISyncLookahead    int32 /* threaded lookahead buffer */

	/* Video Properties */
	IWidth      int32
	IHeight     int32
	ICsp        int32 /* CSP of encoded bitstream */
	IBitdepth   int32
	ILevelIdc   int32
	IFrameTotal int32 /* number of frames to encode if known, else 0 */

	/* NAL HRD
	 * Uses Buffering and Picture Timing SEIs to signal HRD
	 * The HRD in H.264 was not designed with VFR in mind.
	 * It is therefore not recommended to use NAL HRD with VFR.
	 * Furthermore, reconfiguring the VBV (via x264_encoder_reconfig)
	 * will currently generate invalid HRD. */
	INalHrd int32

	Vui struct {
		/* they will be reduced to be 0 < x <= 65535 and prime */
		ISarHeight int32
		ISarWidth  int32

		IOverscan int32 /* 0=undef, 1=no overscan, 2=overscan */

		/* see h264 annex E for the values of the following */
		IVidformat int32
		BFullrange int32
		IColorprim int32
		ITransfer  int32
		IColmatrix int32
		IChromaLoc int32 /* both top & bottom */
	}

	/* Bitstream parameters */
	IFrameReference int32 /* Maximum number of reference frames */
	IDpbSize        int32 /* Force a DPB size larger than that implied by B-frames and reference frames.
	 * Useful in combination with interactive error resilience. */
	IKeyintMax         int32 /* Force an IDR keyframe at this interval */
	IKeyintMin         int32 /* Scenecuts closer together than this are coded as I, not IDR. */
	IScenecutThreshold int32 /* how aggressively to insert extra I frames */
	BIntraRefresh      int32 /* Whether or not to use periodic intra refresh instead of IDR frames. */

	IBframe         int32 /* how many b-frame between 2 references pictures */
	IBframeAdaptive int32
	IBframeBias     int32
	IBframePyramid  int32 /* Keep some B-frames as references: 0=off, 1=strict hierarchical, 2=normal */
	BOpenGop        int32
	BBlurayCompat   int32
	IAvcintraClass  int32
	IAvcintraFlavor int32

	BDeblockingFilter        int32
	IDeblockingFilterAlphac0 int32 /* [-6, 6] -6 light filter, 6 strong */
	IDeblockingFilterBeta    int32 /* [-6, 6]  idem */

	BCabac        int32
	ICabacInitIdc int32

	BInterlaced       int32
	BConstrainedIntra int32

	ICqmPreset int32
	PszCqmFile *int8    /* filename (in UTF-8) of CQM file, JM format */
	Cqm4iy     [16]byte /* used only if i_cqm_preset == X264_CQM_CUSTOM */
	Cqm4py     [16]byte
	Cqm4ic     [16]byte
	Cqm4pc     [16]byte
	Cqm8iy     [64]byte
	Cqm8py     [64]byte
	Cqm8ic     [64]byte
	Cqm8pc     [64]byte

	/* Log */
	PfLog       *[0]byte
	PLogPrivate unsafe.Pointer
	ILogLevel   int32
	BFullRecon  int32 /* fully reconstruct frames, even when not necessary for encoding.  Implied by psz_dump_yuv */
	PszDumpYuv  *int8 /* filename (in UTF-8) for reconstructed frames */

	/* Encoder analyser parameters */
	Analyse struct {
		Intra uint32 /* intra partitions */
		Inter uint32 /* inter partitions */

		BTransform8x8   int32
		IWeightedPred   int32 /* weighting for P-frames */
		BWeightedBipred int32 /* implicit weighting for B-frames */
		IDirectMvPred   int32 /* spatial vs temporal mv prediction */
		IChromaQpOffset int32

		IMeMethod        int32   /* motion estimation algorithm to use (X264_ME_*) */
		IMeRange         int32   /* integer pixel motion estimation search range (from predicted mv) */
		IMvRange         int32   /* maximum length of a mv (in pixels). -1 = auto, based on level */
		IMvRangeThread   int32   /* minimum space between threads. -1 = auto, based on number of threads. */
		ISubpelRefine    int32   /* subpixel motion estimation quality */
		BChromaMe        int32   /* chroma ME for subpel and mode decision in P-frames */
		BMixedReferences int32   /* allow each mb partition to have its own reference number */
		ITrellis         int32   /* trellis RD quantization */
		BFastPskip       int32   /* early SKIP detection on P-frames */
		BDctDecimate     int32   /* transform coefficient thresholding on P-frames */
		INoiseReduction  int32   /* adaptive pseudo-deadzone */
		FPsyRd           float32 /* Psy RD strength */
		FPsyTrellis      float32 /* Psy trellis strength */
		BPsy             int32   /* Toggle all psy optimizations */

		BMbInfo       int32 /* Use input mb_info data in x264_picture_t */
		BMbInfoUpdate int32 /* Update the values in mb_info according to the results of encoding. */

		/* the deadzone size that will be used in luma quantization */
		ILumaDeadzone [2]int32

		BPsnr int32 /* compute and print PSNR stats */
		BSsim int32 /* compute and print SSIM stats */
	}

	/* Rate control parameters */
	Rc struct {
		IRcMethod int32 /* X264_RC_* */

		IQpConstant int32 /* 0=lossless */
		IQpMin      int32 /* min allowed QP value */
		IQpMax      int32 /* max allowed QP value */
		IQpStep     int32 /* max QP step between frames */

		IBitrate       int32
		FRfConstant    float32 /* 1pass VBR, nominal QP */
		FRfConstantMax float32 /* In CRF mode, maximum CRF as caused by VBV */
		FRateTolerance float32
		IVbvMaxBitrate int32
		IVbvBufferSize int32
		FVbvBufferInit float32 /* <=1: fraction of buffer_size. >1: kbit */
		FIpFactor      float32
		FPbFactor      float32

		/* VBV filler: force CBR VBV and use filler bytes to ensure hard-CBR.
		 * Implied by NAL-HRD CBR. */
		BFiller int32

		IAqMode     int32 /* psy adaptive QP. (X264_AQ_*) */
		FAqStrength float32
		BMbTree     int32 /* Macroblock-tree ratecontrol. */
		ILookahead  int32

		/* 2pass */
		BStatWrite int32 /* Enable stat writing in psz_stat_out */
		PszStatOut *int8 /* output filename (in UTF-8) of the 2pass stats file */
		BStatRead  int32 /* Read stat from psz_stat_in and use it */
		PszStatIn  *int8 /* input filename (in UTF-8) of the 2pass stats file */

		/* 2pass params (same as ffmpeg ones) */
		FQcompress      float32 /* 0.0 => cbr, 1.0 => constant qp */
		FQblur          float32 /* temporally blur quants */
		FComplexityBlur float32 /* temporally blur complexity */
		Zones           *Zone   /* ratecontrol overrides */
		IZones          int32   /* number of zone_t's */
		PszZones        *int8   /* alternate method of specifying zones */
	}

	/* Cropping Rectangle parameters: added to those implicitly defined by
	   non-mod16 video resolutions. */
	CropRect struct {
		ILeft   int32
		ITop    int32
		IRight  int32
		IBottom int32
	}

	/* frame packing arrangement flag */
	IFramePacking int32

	/* alternative transfer SEI */
	IAlternativeTransfer int32

	/* Muxing parameters */
	BAud           int32 /* generate access unit delimiters */
	BRepeatHeaders int32 /* put SPS/PPS before each keyframe */
	BAnnexb        int32 /* if set, place start codes (4 bytes) before NAL units,
	 * otherwise place size (4 bytes) before NAL units. */
	ISpsId    int32 /* SPS and PPS id number */
	BVfrInput int32 /* VFR input.  If 1, use timebase and timestamps for ratecontrol purposes.
	 * If 0, use fps only. */
	BPulldown    int32 /* use explicity set timebase for CFR */
	IFpsNum      uint32
	IFpsDen      uint32
	ITimebaseNum uint32 /* Timebase numerator */
	ITimebaseDen uint32 /* Timebase denominator */

	BTff int32

	/* Pulldown:
	 * The correct pic_struct must be passed with each input frame.
	 * The input timebase should be the timebase corresponding to the output framerate. This should be constant.
	 * e.g. for 3:2 pulldown timebase should be 1001/30000
	 * The PTS passed with each frame must be the PTS of the frame after pulldown is applied.
	 * Frame doubling and tripling require b_vfr_input set to zero (see H.264 Table D-1)
	 *
	 * Pulldown changes are not clearly defined in H.264. Therefore, it is the calling app's responsibility to manage this.
	 */

	BPicStruct int32

	/* Fake Interlaced.
	 *
	 * Used only when b_interlaced=0. Setting this flag makes it possible to flag the stream as PAFF interlaced yet
	 * encode all frames progressively. It is useful for encoding 25p and 30p Blu-Ray streams.
	 */
	BFakeInterlaced int32

	/* Don't optimize header parameters based on video content, e.g. ensure that splitting an input video, compressing
	 * each part, and stitching them back together will result in identical SPS/PPS. This is necessary for stitching
	 * with container formats that don't allow multiple SPS/PPS. */
	BStitchable int32

	BOpencl        int32          /* use OpenCL when available */
	IOpenclDevice  int32          /* specify count of GPU devices to skip, for CLI users */
	OpenclDeviceId unsafe.Pointer /* pass explicit cl_device_id as void*, for API users */
	PszClbinFile   *int8          /* filename (in UTF-8) of the compiled OpenCL kernel cache file */

	/* Slicing parameters */
	iSliceMaxSize  int32 /* Max size per slice in bytes; includes estimated NAL overhead. */
	iSliceMaxMbs   int32 /* Max number of MBs per slice; overrides iSliceCount. */
	iSliceMinMbs   int32 /* Min number of MBs per slice */
	iSliceCount    int32 /* Number of slices per frame: forces rectangular slices. */
	iSliceCountMax int32 /* Absolute cap on slices per frame; stops applying slice-max-size
	 * and slice-max-mbs if this is reached. */

	ParamFree   *func(arg unsafe.Pointer)
	NaluProcess *func(H []T, Nal []Nal, Opaque unsafe.Pointer)

	Opaque unsafe.Pointer
}

/****************************************************************************
 * H.264 level restriction information
 ****************************************************************************/

type Level struct {
	LevelIdc  byte
	Mbps      int32  /* max macroblock processing rate (macroblocks/sec) */
	FrameSize int32  /* max frame size (macroblocks) */
	Dpb       int32  /* max decoded picture buffer (mbs) */
	Bitrate   int32  /* max bitrate (kbit/sec) */
	Cpb       int32  /* max vbv buffer (kbit) */
	MvRange   uint16 /* max vertical mv component range (pixels) */
	MvsPer2mb byte   /* max mvs per 2 consecutive mbs. */
	SliceRate byte   /* ?? */
	Mincr     byte   /* min compression ratio */
	Bipred8x8 byte   /* limit bipred to >=8x8 */
	Direct8x8 byte   /* limit b_direct to >=8x8 */
	FrameOnly byte   /* forbid interlacing */
}

type PicStruct int32

type Hrd struct {
	CpbInitialArrivalTime float64
	CpbFinalArrivalTime   float64
	CpbRemovalTime        float64

	DpbOutputTime float64
}

type SeiPayload struct {
	PayloadSize int32
	PayloadType int32
	Payload     *byte
}

type Sei struct {
	NumPayloads int32
	Payloads    *SeiPayload
	/* In: optional callback to free each payload AND x264_sei_payload_t when used. */
	SeiFree *func(arg0 unsafe.Pointer)
}

type Image struct {
	ICsp    int32             /* Colorspace */
	IPlane  int32             /* Number of image planes */
	IStride [4]int32          /* Strides for each plane */
	Plane   [4]unsafe.Pointer /* Pointers to each plane */
}

type ImageProperties struct {
	/* In: an array of quantizer offsets to be applied to this image during encoding.
	 *     These are added on top of the decisions made by x264.
	 *     Offsets can be fractional; they are added before QPs are rounded to integer.
	 *     Adaptive quantization must be enabled to use this feature.  Behavior if quant
	 *     offsets differ between encoding passes is undefined. */
	QuantOffsets *float32
	/* In: optional callback to free quant_offsets when used.
	*     Useful if one wants to use a different quant_offset array for each frame. */
	QuantOffsetsFree *func(arg0 unsafe.Pointer)

	/* In: optional array of flags for each macroblock.
	 *     Allows specifying additional information for the encoder such as which macroblocks
	 *     remain unchanged.  Usable flags are listed below.
	 *     x264_param_t.analyse.b_mb_info must be set to use this, since x264 needs to track
	 *     extra data internally to make full use of this information.
	 *
	 * Out: if b_mb_info_update is set, x264 will update this array as a result of encoding.
	 *
	 *      For "MBINFO_CONSTANT", it will remove this flag on any macroblock whose decoded
	 *      pixels have changed.  This can be useful for e.g. noting which areas of the
	 *      frame need to actually be blitted. Note: this intentionally ignores the effects
	 *      of deblocking for the current frame, which should be fine unless one needs exact
	 *      pixel-perfect accuracy.
	 *
	 *      Results for MBINFO_CONSTANT are currently only set for P-frames, and are not
	 *      guaranteed to enumerate all blocks which haven't changed.  (There may be false
	 *      negatives, but no false positives.)
	 */
	MbInfo *byte
	/* In: optional callback to free mb_info when used. */
	MbInfoFree *func(arg0 unsafe.Pointer)

	/* Out: SSIM of the the frame luma (if x264_param_t.b_ssim is set) */
	FSsim float64
	/* Out: Average PSNR of the frame (if x264_param_t.b_psnr is set) */
	FPsnrAvg float64
	/* Out: PSNR of Y, U, and V (if x264_param_t.b_psnr is set) */
	FPsnr [3]float64

	/* Out: Average effective CRF of the encoded frame */
	FCrfAvg float64
}

type Picture struct {
	/* In: force picture type (if not auto)
	 *     If x264 encoding parameters are violated in the forcing of picture types,
	 *     x264 will correct the input picture type and log a warning.
	 * Out: type of the picture encoded */
	IType int32
	/* In: force quantizer for != X264_QP_AUTO */
	IQpplus1 int32
	/* In: pic_struct, for pulldown/doubling/etc...used only if b_pic_struct=1.
	 *     use pic_struct_e for pic_struct inputs
	 * Out: pic_struct element associated with frame */
	IPicStruct int32
	/* Out: whether this frame is a keyframe.  Important when using modes that result in
	 * SEI recovery points being used instead of IDR frames. */
	BKeyframe int32
	/* In: user pts, Out: pts of encoded picture (user)*/
	IPts int64
	/* Out: frame dts. When the pts of the first frame is close to zero,
	 *      initial frames may have a negative dts which must be dealt with by any muxer */
	IDts int64
	/* In: custom encoding parameters to be set from this frame forwards
	   (in coded order, not display order). If NULL, continue using
	   parameters from the previous frame.  Some parameters, such as
	   aspect ratio, can only be changed per-GOP due to the limitations
	   of H.264 itself; in this case, the caller must force an IDR frame
	   if it needs the changed parameter to apply immediately. */
	Param *Param
	/* In: raw image data */
	/* Out: reconstructed image data.  x264 may skip part of the reconstruction process,
	   e.g. deblocking, in frames where it isn't necessary.  To force complete
	   reconstruction, at a small speed cost, set b_full_recon. */
	Img Image
	/* In: optional information to modify encoder decisions for this frame
	 * Out: information about the encoded frame */
	Prop ImageProperties
	/* Out: HRD timing information. Output only when i_nal_hrd is set. */
	Hrdiming Hrd
	/* In: arbitrary user SEI (e.g subtitles, AFDs) */
	ExtraSei Sei
	/* private user data. copied from input to output frames. */
	Opaque unsafe.Pointer
}

func (p *Picture) freePlanes() {
	for _, ptr := range p.Img.Plane {
		C.free(ptr)
	}
}

func (t *T) cptr() *C.x264_t { return (*C.x264_t)(unsafe.Pointer(t)) }

func (n *Nal) cptr() *C.x264_nal_t { return (*C.x264_nal_t)(unsafe.Pointer(n)) }

func (p *Param) cptr() *C.x264_param_t { return (*C.x264_param_t)(unsafe.Pointer(p)) }

func (p *Picture) cptr() *C.x264_picture_t { return (*C.x264_picture_t)(unsafe.Pointer(p)) }

// ParamDefault - fill Param with default values and do CPU detection.
func ParamDefault(param *Param) { C.x264_param_default(param.cptr()) }

// ParamDefaultPreset - the same as ParamDefault, but also use the passed preset and tune to modify the default settings
// (either can be nil, which implies no preset or no tune, respectively).
//
// Currently available presets are, ordered from fastest to slowest:
// "ultrafast", "superfast", "veryfast", "faster", "fast", "medium", "slow", "slower", "veryslow", "placebo".
//
// Currently available tunings are:
// "film", "animation", "grain", "stillimage", "psnr", "ssim", "fastdecode", "zerolatency".
//
// Returns 0 on success, negative on failure (e.g. invalid preset/tune name).
func ParamDefaultPreset(param *Param, preset string, tune string) int32 {
	cpreset := C.CString(preset)
	defer C.free(unsafe.Pointer(cpreset))
	ctune := C.CString(tune)
	defer C.free(unsafe.Pointer(ctune))
	return (int32)(C.x264_param_default_preset(param.cptr(), cpreset, ctune))
}

// ParamApplyProfile - applies the restrictions of the given profile.
//
// Currently available profiles are, from most to least restrictive:
// "baseline", "main", "high", "high10", "high422", "high444".
// (can be nil, in which case the function will do nothing).
//
// Returns 0 on success, negative on failure (e.g. invalid profile name).
func ParamApplyProfile(param *Param, profile string) int32 {
	cprofile := C.CString(profile)
	defer C.free(unsafe.Pointer(cprofile))
	return (int32)(C.x264_param_apply_profile(param.cptr(), cprofile))
}

// EncoderOpen - create a new encoder handler, all parameters from Param are copied.
func EncoderOpen(param *Param) *T {
	ret := C.x264_encoder_open(param.cptr())
	return *(**T)(unsafe.Pointer(&ret))
}

// EncoderEncode - encode one picture.
// Returns the number of bytes in the returned NALs, negative on error and zero if no NAL units returned.
func EncoderEncode(enc *T, ppNal []*Nal, piNal *int32, picIn *Picture, picOut *Picture) int32 {
	cenc := enc.cptr()

	cppNal := (**C.x264_nal_t)(unsafe.Pointer(&ppNal[0]))
	cpiNal := (*C.int)(unsafe.Pointer(piNal))

	cpicIn := picIn.cptr()
	cpicOut := picOut.cptr()

	return (int32)(C.x264_encoder_encode(cenc, cppNal, cpiNal, cpicIn, cpicOut))
}

// EncoderClose closes an encoder handler.
func EncoderClose(enc *T) { C.x264_encoder_close(enc.cptr()) }

// EncoderIntraRefresh - If an intra refresh is not in progress, begin one with the next P-frame.
// If an intra refresh is in progress, begin one as soon as the current one finishes.
// Requires that BIntraRefresh be set.
//
// Should not be called during an x264_encoder_encode.
//func EncoderIntraRefresh(enc *T) { C.x264_encoder_intra_refresh(enc.cptr()) }
