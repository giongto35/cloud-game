package graphics

import (
	"log"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
)

type offscreenSetup struct {
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

var opt = offscreenSetup{}

// OpenGL pixel format
type PixelFormat int

const (
	UnsignedShort5551 PixelFormat = iota
	UnsignedShort565
	UnsignedInt8888Rev
)

func initContext(getProcAddr func(name string) unsafe.Pointer) {
	if err := gl.InitWithProcAddrFunc(getProcAddr); err != nil {
		panic(err)
	}
}

func initFramebuffer(w int, h int, hasDepth bool, hasStencil bool) {
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

	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if status != gl.FRAMEBUFFER_COMPLETE {
		if e := gl.GetError(); e != gl.NO_ERROR {
			log.Printf("[OpenGL] GL error: 0x%X, Frame status: 0x%X", e, status)
			panic("OpenGL error")
		}
		log.Printf("[OpenGL] frame status: 0x%X", status)
		panic("OpenGL framebuffer is invalid")
	}
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

func getFbo() uint32 {
	return opt.fbo
}

func SetPixelFormat(format PixelFormat) {
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
		log.Fatalf("[opengl] Error! Unknown pixel type %v", format)
	}
}

// PrintDriverInfo prints OpenGL information.
func PrintDriverInfo() {
	// OpenGL info
	log.Printf("[OpenGL] Version: %v", get(gl.VERSION))
	log.Printf("[OpenGL] Vendor: %v", get(gl.VENDOR))
	// This string is often the name of the GPU.
	// In the case of Mesa3d, it would be i.e "Gallium 0.4 on NVA8".
	// It might even say "Direct3D" if the Windows Direct3D wrapper is being used.
	log.Printf("[OpenGL] Renderer: %v", get(gl.RENDERER))
	log.Printf("[OpenGL] GLSL Version: %v", get(gl.SHADING_LANGUAGE_VERSION))
}

func getDriverError() uint32 {
	return gl.GetError()
}

func get(name uint32) string {
	return gl.GoStr(gl.GetString(name))
}
