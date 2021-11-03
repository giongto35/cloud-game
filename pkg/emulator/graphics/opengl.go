package graphics

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
)

type (
	offscreenSetup struct {
		tex uint32
		fbo uint32
		rbo uint32

		width  int32
		height int32

		pixType   uint32
		pixFormat uint32

		hasDepth   bool
		hasStencil bool
	}
	PixelFormat int
)

const (
	UnsignedShort5551 PixelFormat = iota
	UnsignedShort565
	UnsignedInt8888Rev
)

var opt = offscreenSetup{}

func initContext(getProcAddr func(name string) unsafe.Pointer) {
	if err := gl.InitWithProcAddrFunc(getProcAddr); err != nil {
		panic(err)
	}
}

func initFramebuffer(w int, h int, hasDepth bool, hasStencil bool) error {
	opt.width = int32(w)
	opt.height = int32(h)
	opt.hasDepth = hasDepth
	opt.hasStencil = hasStencil

	// texture init
	gl.GenTextures(1, &opt.tex)
	gl.BindTexture(gl.TEXTURE_2D, opt.tex)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, opt.width, opt.height, 0, opt.pixType, opt.pixFormat, nil)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	// framebuffer init
	gl.GenFramebuffers(1, &opt.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, opt.fbo)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, opt.tex, 0)

	// more buffers init
	if opt.hasDepth {
		gl.GenRenderbuffers(1, &opt.rbo)
		gl.BindRenderbuffer(gl.RENDERBUFFER, opt.rbo)
		if opt.hasStencil {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, opt.width, opt.height)
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, opt.rbo)
		} else {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, opt.width, opt.height)
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, opt.rbo)
		}
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	}

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		return fmt.Errorf("invalid framebuffer (0x%X)", status)
	}
	return nil
}

func destroyFramebuffer() {
	if opt.hasDepth {
		gl.DeleteRenderbuffers(1, &opt.rbo)
	}
	gl.DeleteFramebuffers(1, &opt.fbo)
	gl.DeleteTextures(1, &opt.tex)
}

func ReadFramebuffer(bytes int, w int, h int) []byte {
	data := make([]byte, bytes)
	gl.BindFramebuffer(gl.FRAMEBUFFER, opt.fbo)
	gl.ReadPixels(0, 0, int32(w), int32(h), opt.pixType, opt.pixFormat, gl.Ptr(&data[0]))
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return data
}

func getFbo() uint32 { return opt.fbo }

func SetPixelFormat(format PixelFormat) error {
	switch format {
	case UnsignedShort5551:
		opt.pixFormat = gl.UNSIGNED_SHORT_5_5_5_1
		opt.pixType = gl.BGRA
	case UnsignedShort565:
		opt.pixFormat = gl.UNSIGNED_SHORT_5_6_5
		opt.pixType = gl.RGB
	case UnsignedInt8888Rev:
		opt.pixFormat = gl.UNSIGNED_INT_8_8_8_8_REV
		opt.pixType = gl.BGRA
	default:
		return errors.New("unknown pixel format")
	}
	return nil
}

func GetGLVersionInfo() string  { return get(gl.VERSION) }
func GetGLVendorInfo() string   { return get(gl.VENDOR) }
func GetGLRendererInfo() string { return get(gl.RENDERER) }
func GetGLSLInfo() string       { return get(gl.SHADING_LANGUAGE_VERSION) }
func GetGLError() uint32        { return gl.GetError() }

func get(name uint32) string { return gl.GoStr(gl.GetString(name)) }
