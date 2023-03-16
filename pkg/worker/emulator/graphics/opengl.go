package graphics

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/graphics/gl"
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

var (
	opt = offscreenSetup{}
	buf []byte
)

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
	gl.BindTexture(gl.Texture2d, opt.tex)

	gl.TexParameteri(gl.Texture2d, gl.TextureMinFilter, gl.NEAREST)
	gl.TexParameteri(gl.Texture2d, gl.TextureMagFilter, gl.NEAREST)

	gl.TexImage2D(gl.Texture2d, 0, gl.RGBA8, opt.width, opt.height, 0, opt.pixType, opt.pixFormat, nil)
	gl.BindTexture(gl.Texture2d, 0)

	// framebuffer init
	gl.GenFramebuffers(1, &opt.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, opt.fbo)

	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.ColorAttachment0, gl.Texture2d, opt.tex, 0)

	// more buffers init
	if opt.hasDepth {
		gl.GenRenderbuffers(1, &opt.rbo)
		gl.BindRenderbuffer(gl.RENDERBUFFER, opt.rbo)
		if opt.hasStencil {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.Depth24Stencil8, opt.width, opt.height)
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DepthStencilAttachment, gl.RENDERBUFFER, opt.rbo)
		} else {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DepthComponent24, opt.width, opt.height)
			gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DepthAttachment, gl.RENDERBUFFER, opt.rbo)
		}
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	}

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FramebufferComplete {
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

func ReadFramebuffer(bytes, w, h uint) []byte {
	data := buf[:bytes]
	gl.BindFramebuffer(gl.FRAMEBUFFER, opt.fbo)
	gl.ReadPixels(0, 0, int32(w), int32(h), opt.pixType, opt.pixFormat, unsafe.Pointer(&data[0]))
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	return data
}

func getFbo() uint32 { return opt.fbo }

func SetBuffer(size int) { buf = make([]byte, size) }

func SetPixelFormat(format PixelFormat) error {
	switch format {
	case UnsignedShort5551:
		opt.pixFormat = gl.UnsignedShort5551
		opt.pixType = gl.BGRA
	case UnsignedShort565:
		opt.pixFormat = gl.UnsignedShort565
		opt.pixType = gl.RGB
	case UnsignedInt8888Rev:
		opt.pixFormat = gl.UnsignedInt8888Rev
		opt.pixType = gl.BGRA
	default:
		return errors.New("unknown pixel format")
	}
	return nil
}

func GetGLVersionInfo() string  { return get(gl.VERSION) }
func GetGLVendorInfo() string   { return get(gl.VENDOR) }
func GetGLRendererInfo() string { return get(gl.RENDERER) }
func GetGLSLInfo() string       { return get(gl.ShadingLanguageVersion) }
func GetGLError() uint32        { return gl.GetError() }

func get(name uint32) string { return gl.GoStr(gl.GetString(name)) }
