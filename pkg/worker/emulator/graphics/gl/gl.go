package gl

// Custom OpenGL bindings
// Based on https://github.com/go-gl/gl/tree/master/v2.1/gl

/*
#cgo egl,windows LDFLAGS: -lEGL
#cgo egl,darwin  LDFLAGS: -lEGL
#cgo !gles2,darwin        LDFLAGS: -framework OpenGL
#cgo gles2,darwin         LDFLAGS: -lGLESv2
#cgo !gles2,windows       LDFLAGS: -lopengl32
#cgo gles2,windows        LDFLAGS: -lGLESv2
#cgo !egl,linux !egl,freebsd !egl,openbsd pkg-config: gl
#cgo egl,linux egl,freebsd egl,openbsd    pkg-config: egl

#if defined(_WIN32) && !defined(APIENTRY) && !defined(__CYGWIN__) && !defined(__SCITECH_SNAP__)
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN 1
#endif

#include <windows.h>

#endif
#ifndef APIENTRY
#define APIENTRY
#endif
#ifndef APIENTRYP
#define APIENTRYP APIENTRY*
#endif
#ifndef GLAPI
#define GLAPI extern
#endif

#include <KHR/khrplatform.h>

typedef unsigned int GLenum;
typedef unsigned char GLboolean;
typedef unsigned int GLbitfield;
typedef khronos_int8_t GLbyte;
typedef khronos_uint8_t GLubyte;
typedef khronos_int16_t GLshort;
typedef khronos_uint16_t GLushort;
typedef int GLint;
typedef unsigned int GLuint;
typedef khronos_int32_t GLclampx;
typedef int GLsizei;
typedef khronos_float_t GLfloat;
typedef khronos_float_t GLclampf;
typedef double GLdouble;
typedef double GLclampd;
typedef void *GLeglClientBufferEXT;
typedef void *GLeglImageOES;
typedef char GLchar;
typedef char GLcharARB;


#ifdef __APPLE__
typedef void *GLhandleARB;
#else
typedef unsigned int GLhandleARB;
#endif

typedef const GLubyte *(APIENTRYP GPGETSTRING)(GLenum name);
typedef GLenum (APIENTRYP GPCHECKFRAMEBUFFERSTATUS)(GLenum target);
typedef GLenum (APIENTRYP GPGETERROR)();
typedef void (APIENTRYP GPBINDFRAMEBUFFER)(GLenum target, GLuint framebuffer);
typedef void (APIENTRYP GPBINDRENDERBUFFER)(GLenum target, GLuint renderbuffer);
typedef void (APIENTRYP GPBINDTEXTURE)(GLenum target, GLuint texture);
typedef void (APIENTRYP GPDELETEFRAMEBUFFERS)(GLsizei n, const GLuint *framebuffers);
typedef void (APIENTRYP GPDELETERENDERBUFFERS)(GLsizei n, const GLuint *renderbuffers);
typedef void (APIENTRYP GPDELETETEXTURES)(GLsizei n, const GLuint* textures);
typedef void (APIENTRYP GPFRAMEBUFFERRENDERBUFFER)(GLenum target, GLenum attachment, GLenum renderbuffertarget, GLuint renderbuffer);
typedef void (APIENTRYP GPFRAMEBUFFERTEXTURE2D)(GLenum target, GLenum attachment, GLenum textarget, GLuint texture, GLint level);
typedef void (APIENTRYP GPGENFRAMEBUFFERS)(GLsizei n, GLuint *framebuffers);
typedef void (APIENTRYP GPGENRENDERBUFFERS)(GLsizei n, GLuint *renderbuffers);
typedef void (APIENTRYP GPGENTEXTURES)(GLsizei n, GLuint *textures);
typedef void (APIENTRYP GPREADPIXELS)(GLint x, GLint y, GLsizei width, GLsizei height, GLenum format, GLenum type, void *pixels);
typedef void (APIENTRYP GPRENDERBUFFERSTORAGE)(GLenum target, GLenum internalformat, GLsizei width, GLsizei height);
typedef void (APIENTRYP GPTEXIMAGE2D)(GLenum target, GLint level, GLint internalformat, GLsizei width, GLsizei height, GLint border, GLenum format, GLenum type, const void *pixels);
typedef void (APIENTRYP GPTEXPARAMETERI)(GLenum target, GLenum pname, GLint param);

static const GLubyte *getString(GPGETSTRING ptr, GLenum name) { return (*ptr)(name); }
static GLenum getError(GPGETERROR ptr) { return (*ptr)(); }
static void bindTexture(GPBINDTEXTURE ptr, GLenum target, GLuint texture) { (*ptr)(target, texture); }
static void bindFramebuffer(GPBINDFRAMEBUFFER ptr, GLenum target, GLuint framebuffer) { (*ptr)(target, framebuffer); }
static void bindRenderbuffer(GPBINDRENDERBUFFER ptr, GLenum target, GLuint buf) { (*ptr)(target, buf); }
static void texParameteri(GPTEXPARAMETERI ptr, GLenum target, GLenum pname, GLint param) {
  (*ptr)(target, pname, param);
}
static void texImage2D(GPTEXIMAGE2D ptr, GLenum target, GLint level, GLint internalformat, GLsizei width, GLsizei height, GLint border, GLenum format, GLenum type, const void *pixels) {
  (*ptr)(target, level, internalformat, width, height, border, format, type, pixels);
}
static void genFramebuffers(GPGENFRAMEBUFFERS ptr, GLsizei n, GLuint *framebuffers) { (*ptr)(n, framebuffers); }
static void genTextures(GPGENTEXTURES ptr, GLsizei n, GLuint *textures) { (*ptr)(n, textures); }
static void framebufferTexture2D(GPFRAMEBUFFERTEXTURE2D ptr, GLenum target, GLenum attachment, GLenum textarget, GLuint texture, GLint level) {
  (*ptr)(target, attachment, textarget, texture, level);
}
static void genRenderbuffers(GPGENRENDERBUFFERS ptr, GLsizei n, GLuint *renderbuffers) { (*ptr)(n, renderbuffers); }
static void renderbufferStorage(GPRENDERBUFFERSTORAGE ptr, GLenum target, GLenum internalformat, GLsizei width, GLsizei height) {
  (*ptr)(target, internalformat, width, height);
}
static void framebufferRenderbuffer(GPFRAMEBUFFERRENDERBUFFER ptr, GLenum target, GLenum attachment, GLenum renderbuffertarget, GLuint renderbuffer) {
  (*ptr)(target, attachment, renderbuffertarget, renderbuffer);
}
static GLenum checkFramebufferStatus(GPCHECKFRAMEBUFFERSTATUS ptr, GLenum target) { return (*ptr)(target); }
static void deleteRenderbuffers(GPDELETERENDERBUFFERS ptr, GLsizei n, const GLuint *renderbuffers) {
  (*ptr)(n, renderbuffers);
}
static void deleteFramebuffers(GPDELETEFRAMEBUFFERS ptr, GLsizei n, const GLuint *framebuffers) {
  (*ptr)(n, framebuffers);
}
static void deleteTextures(GPDELETETEXTURES ptr, GLsizei n, const GLuint *textures) { (*ptr)(n, textures); }
static void readPixels(GPREADPIXELS ptr, GLint x, GLint y, GLsizei width, GLsizei height, GLenum format, GLenum type, void *pixels) {
  (*ptr)(x, y, width, height, format, type, pixels);
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

const (
	VENDOR                 = 0x1F00
	VERSION                = 0x1F02
	RENDERER               = 0x1F01
	ShadingLanguageVersion = 0x8B8C
	Texture2d              = 0x0DE1
	RENDERBUFFER           = 0x8D41
	FRAMEBUFFER            = 0x8D40
	TextureMinFilter       = 0x2801
	TextureMagFilter       = 0x2800
	NEAREST                = 0x2600
	RGBA8                  = 0x8058
	BGRA                   = 0x80E1
	RGB                    = 0x1907
	ColorAttachment0       = 0x8CE0
	Depth24Stencil8        = 0x88F0
	DepthStencilAttachment = 0x821A
	DepthComponent24       = 0x81A6
	DepthAttachment        = 0x8D00
	FramebufferComplete    = 0x8CD5

	UnsignedShort5551  = 0x8034
	UnsignedShort565   = 0x8363
	UnsignedInt8888Rev = 0x8367
)

var (
	gpGetString               C.GPGETSTRING
	gpGenTextures             C.GPGENTEXTURES
	gpGetError                C.GPGETERROR
	gpBindTexture             C.GPBINDTEXTURE
	gpBindFramebuffer         C.GPBINDFRAMEBUFFER
	gpTexParameteri           C.GPTEXPARAMETERI
	gpTexImage2D              C.GPTEXIMAGE2D
	gpGenFramebuffers         C.GPGENFRAMEBUFFERS
	gpFramebufferTexture2D    C.GPFRAMEBUFFERTEXTURE2D
	gpGenRenderbuffers        C.GPGENRENDERBUFFERS
	gpBindRenderbuffer        C.GPBINDRENDERBUFFER
	gpRenderbufferStorage     C.GPRENDERBUFFERSTORAGE
	gpFramebufferRenderbuffer C.GPFRAMEBUFFERRENDERBUFFER
	gpCheckFramebufferStatus  C.GPCHECKFRAMEBUFFERSTATUS
	gpDeleteRenderbuffers     C.GPDELETERENDERBUFFERS
	gpDeleteFramebuffers      C.GPDELETEFRAMEBUFFERS
	gpDeleteTextures          C.GPDELETETEXTURES
	gpReadPixels              C.GPREADPIXELS
)

func InitWithProcAddrFunc(getProcAddr func(name string) unsafe.Pointer) error {
	if gpGetString = (C.GPGETSTRING)(getProcAddr("glGetString")); gpGetString == nil {
		return errors.New("glGetString")
	}
	if gpGenTextures = (C.GPGENTEXTURES)(getProcAddr("glGenTextures")); gpGenTextures == nil {
		return errors.New("glGenTextures")
	}
	if gpGetError = (C.GPGETERROR)(getProcAddr("glGetError")); gpGetError == nil {
		return errors.New("glGetError")
	}
	if gpBindTexture = (C.GPBINDTEXTURE)(getProcAddr("glBindTexture")); gpBindTexture == nil {
		return errors.New("glBindTexture")
	}
	if gpBindFramebuffer = (C.GPBINDFRAMEBUFFER)(getProcAddr("glBindFramebuffer")); gpBindFramebuffer == nil {
		return errors.New("glBindFramebuffer")
	}
	if gpTexParameteri = (C.GPTEXPARAMETERI)(getProcAddr("glTexParameteri")); gpTexParameteri == nil {
		return errors.New("glTexParameteri")
	}
	if gpTexImage2D = (C.GPTEXIMAGE2D)(getProcAddr("glTexImage2D")); gpTexImage2D == nil {
		return errors.New("glTexImage2D")
	}
	gpGenFramebuffers = (C.GPGENFRAMEBUFFERS)(getProcAddr("glGenFramebuffers"))
	gpFramebufferTexture2D = (C.GPFRAMEBUFFERTEXTURE2D)(getProcAddr("glFramebufferTexture2D"))
	gpGenRenderbuffers = (C.GPGENRENDERBUFFERS)(getProcAddr("glGenRenderbuffers"))
	gpBindRenderbuffer = (C.GPBINDRENDERBUFFER)(getProcAddr("glBindRenderbuffer"))
	gpRenderbufferStorage = (C.GPRENDERBUFFERSTORAGE)(getProcAddr("glRenderbufferStorage"))
	gpFramebufferRenderbuffer = (C.GPFRAMEBUFFERRENDERBUFFER)(getProcAddr("glFramebufferRenderbuffer"))
	gpCheckFramebufferStatus = (C.GPCHECKFRAMEBUFFERSTATUS)(getProcAddr("glCheckFramebufferStatus"))
	gpDeleteRenderbuffers = (C.GPDELETERENDERBUFFERS)(getProcAddr("glDeleteRenderbuffers"))
	gpDeleteFramebuffers = (C.GPDELETEFRAMEBUFFERS)(getProcAddr("glDeleteFramebuffers"))
	if gpDeleteTextures = (C.GPDELETETEXTURES)(getProcAddr("glDeleteTextures")); gpDeleteTextures == nil {
		return errors.New("glDeleteTextures")
	}
	gpReadPixels = (C.GPREADPIXELS)(getProcAddr("glReadPixels"))
	if gpReadPixels == nil {
		return errors.New("glReadPixels")
	}
	return nil
}

func GetString(name uint32) *uint8 { return (*uint8)(C.getString(gpGetString, (C.GLenum)(name))) }
func GenTextures(n int32, textures *uint32) {
	C.genTextures(gpGenTextures, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(textures)))
}
func BindTexture(target uint32, texture uint32) {
	C.bindTexture(gpBindTexture, (C.GLenum)(target), (C.GLuint)(texture))
}
func BindFramebuffer(target uint32, framebuffer uint32) {
	C.bindFramebuffer(gpBindFramebuffer, (C.GLenum)(target), (C.GLuint)(framebuffer))
}
func TexParameteri(target uint32, name uint32, param int32) {
	C.texParameteri(gpTexParameteri, (C.GLenum)(target), (C.GLenum)(name), (C.GLint)(param))
}
func TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	C.texImage2D(gpTexImage2D, (C.GLenum)(target), (C.GLint)(level), (C.GLint)(internalformat), (C.GLsizei)(width), (C.GLsizei)(height), (C.GLint)(border), (C.GLenum)(format), (C.GLenum)(xtype), pixels)
}
func GenFramebuffers(n int32, framebuffers *uint32) {
	C.genFramebuffers(gpGenFramebuffers, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(framebuffers)))
}
func FramebufferTexture2D(target uint32, attachment uint32, texTarget uint32, texture uint32, level int32) {
	C.framebufferTexture2D(gpFramebufferTexture2D, (C.GLenum)(target), (C.GLenum)(attachment), (C.GLenum)(texTarget), (C.GLuint)(texture), (C.GLint)(level))
}
func GenRenderbuffers(n int32, renderbuffers *uint32) {
	C.genRenderbuffers(gpGenRenderbuffers, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(renderbuffers)))
}
func BindRenderbuffer(target uint32, renderbuffer uint32) {
	C.bindRenderbuffer(gpBindRenderbuffer, (C.GLenum)(target), (C.GLuint)(renderbuffer))
}
func RenderbufferStorage(target uint32, internalformat uint32, width int32, height int32) {
	C.renderbufferStorage(gpRenderbufferStorage, (C.GLenum)(target), (C.GLenum)(internalformat), (C.GLsizei)(width), (C.GLsizei)(height))
}
func FramebufferRenderbuffer(target uint32, attachment uint32, renderbufferTarget uint32, renderbuffer uint32) {
	C.framebufferRenderbuffer(gpFramebufferRenderbuffer, (C.GLenum)(target), (C.GLenum)(attachment), (C.GLenum)(renderbufferTarget), (C.GLuint)(renderbuffer))
}
func CheckFramebufferStatus(target uint32) uint32 {
	return (uint32)(C.checkFramebufferStatus(gpCheckFramebufferStatus, (C.GLenum)(target)))
}
func DeleteRenderbuffers(n int32, renderbuffers *uint32) {
	C.deleteRenderbuffers(gpDeleteRenderbuffers, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(renderbuffers)))
}
func DeleteFramebuffers(n int32, framebuffers *uint32) {
	C.deleteFramebuffers(gpDeleteFramebuffers, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(framebuffers)))
}
func DeleteTextures(n int32, textures *uint32) {
	C.deleteTextures(gpDeleteTextures, (C.GLsizei)(n), (*C.GLuint)(unsafe.Pointer(textures)))
}
func ReadPixels(x int32, y int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	C.readPixels(gpReadPixels, (C.GLint)(x), (C.GLint)(y), (C.GLsizei)(width), (C.GLsizei)(height), (C.GLenum)(format), (C.GLenum)(xtype), pixels)
}

func GetError() uint32 { return (uint32)(C.getError(gpGetError)) }

func GoStr(str *uint8) string { return C.GoString((*C.char)(unsafe.Pointer(str))) }
