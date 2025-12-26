package graphics

import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/graphics/gl"
)

type Context int

const (
	CtxNone Context = iota
	CtxOpenGl
	CtxOpenGlEs2
	CtxOpenGlCore
	CtxOpenGlEs3
	CtxOpenGlEsVersion
	CtxVulkan
	CtxUnknown = math.MaxInt32 - 1
	CtxDummy   = math.MaxInt32
)

type PixelFormat int

const (
	UnsignedShort5551 PixelFormat = iota
	UnsignedShort565
	UnsignedInt8888Rev
)

var (
	fbo, tex, rbo      uint32
	hasDepth           bool
	pixType, pixFormat uint32
	buf                []byte
	bufPtr             unsafe.Pointer
)

func initContext(getProcAddr func(name string) unsafe.Pointer) {
	if err := gl.InitWithProcAddrFunc(getProcAddr); err != nil {
		panic(err)
	}
	gl.PixelStorei(gl.PackAlignment, 1)
}

func initFramebuffer(width, height int, depth, stencil bool) error {
	w, h := int32(width), int32(height)
	hasDepth = depth

	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.Texture2d, tex)
	gl.TexParameteri(gl.Texture2d, gl.TextureMinFilter, gl.NEAREST)
	gl.TexParameteri(gl.Texture2d, gl.TextureMagFilter, gl.NEAREST)
	gl.TexImage2D(gl.Texture2d, 0, gl.RGBA8, w, h, 0, pixType, pixFormat, nil)
	gl.BindTexture(gl.Texture2d, 0)

	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.ColorAttachment0, gl.Texture2d, tex, 0)

	if depth {
		gl.GenRenderbuffers(1, &rbo)
		gl.BindRenderbuffer(gl.RENDERBUFFER, rbo)
		format, attachment := uint32(gl.DepthComponent24), uint32(gl.DepthAttachment)
		if stencil {
			format, attachment = gl.Depth24Stencil8, gl.DepthStencilAttachment
		}
		gl.RenderbufferStorage(gl.RENDERBUFFER, format, w, h)
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, attachment, gl.RENDERBUFFER, rbo)
		gl.BindRenderbuffer(gl.RENDERBUFFER, 0)
	}

	if status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FramebufferComplete {
		return fmt.Errorf("framebuffer incomplete: 0x%X", status)
	}
	return nil
}

func destroyFramebuffer() {
	if hasDepth {
		gl.DeleteRenderbuffers(1, &rbo)
	}
	gl.DeleteFramebuffers(1, &fbo)
	gl.DeleteTextures(1, &tex)
}

func ReadFramebuffer(size, w, h uint) []byte {
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
	gl.ReadPixels(0, 0, int32(w), int32(h), pixType, pixFormat, bufPtr)
	return buf[:size]
}

func SetBuffer(size int) {
	buf = make([]byte, size)
	bufPtr = unsafe.Pointer(&buf[0])
}

func SetPixelFormat(format PixelFormat) error {
	switch format {
	case UnsignedShort5551:
		pixFormat, pixType = gl.UnsignedShort5551, gl.BGRA
	case UnsignedShort565:
		pixFormat, pixType = gl.UnsignedShort565, gl.RGB
	case UnsignedInt8888Rev:
		pixFormat, pixType = gl.UnsignedInt8888Rev, gl.BGRA
	default:
		return errors.New("unknown pixel format")
	}
	return nil
}

func GLInfo() (version, vendor, renderer, glsl string) {
	return gl.GoStr(gl.GetString(gl.VERSION)),
		gl.GoStr(gl.GetString(gl.VENDOR)),
		gl.GoStr(gl.GetString(gl.RENDERER)),
		gl.GoStr(gl.GetString(gl.ShadingLanguageVersion))
}

func GlFbo() uint32 { return fbo }
