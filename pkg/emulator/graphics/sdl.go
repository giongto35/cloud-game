package graphics

import (
	"log"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/veandco/go-sdl2/sdl"
)

type data struct {
	w      *sdl.Window
	glWCtx sdl.GLContext
}

// singleton state for SDL
var state = data{}

type Config struct {
	Ctx Context
	W   int
	H   int
	Gl  GlConfig
}
type GlConfig struct {
	AutoContext  bool
	VersionMajor uint
	VersionMinor uint
	HasDepth     bool
	HasStencil   bool
}

// Init initializes SDL/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func Init(cfg Config) {
	log.Printf("[SDL] [OpenGL] initialization...")
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Printf("[SDL] error: %v", err)
		panic("SDL initialization failed")
	}

	if cfg.Gl.AutoContext {
		log.Printf("[OpenGL] CONTEXT_AUTO (type: %v v%v.%v)", cfg.Ctx, cfg.Gl.VersionMajor, cfg.Gl.VersionMinor)
	} else {
		switch cfg.Ctx {
		case CtxOpenGlCore:
			setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
			log.Printf("[OpenGL] CONTEXT_PROFILE_CORE")
		case CtxOpenGlEs2:
			setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
			setAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
			setAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 0)
			log.Printf("[OpenGL] CONTEXT_PROFILE_ES 3.0")
		case CtxOpenGl:
			if cfg.Gl.VersionMajor >= 3 {
				setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_COMPATIBILITY)
			}
			log.Printf("[OpenGL] CONTEXT_PROFILE_COMPATIBILITY")
		default:
			log.Printf("Unsupported hw context: %v", cfg.Ctx)
		}
	}

	// In OSX 10.14+ window creation and context creation must happen in the main thread
	thread.MainMaybe(createWindow)

	BindContext()

	initContext(sdl.GLGetProcAddress)
	PrintDriverInfo()
	initFramebuffer(cfg.W, cfg.H, cfg.Gl.HasDepth, cfg.Gl.HasStencil)
}

// Deinit destroys SDL/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func Deinit() {
	log.Printf("[SDL] [OpenGL] deinitialization...")
	destroyFramebuffer()
	// In OSX 10.14+ window deletion must happen in the main thread
	thread.MainMaybe(destroyWindow)
	sdl.Quit()
	log.Printf("[SDL] [OpenGL] deinitialized (%v, %v)", sdl.GetError(), getDriverError())
}

// createWindow creates fake SDL window for OpenGL initialization purposes.
func createWindow() {
	var winTitle = "CloudRetro dummy window"
	var winWidth, winHeight int32 = 1, 1

	var err error
	if state.w, err = sdl.CreateWindow(
		winTitle,
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		winWidth, winHeight,
		sdl.WINDOW_OPENGL|sdl.WINDOW_HIDDEN,
	); err != nil {
		panic(err)
	}
	if state.glWCtx, err = state.w.GLCreateContext(); err != nil {
		panic(err)
	}
}

// destroyWindow destroys previously created SDL window.
func destroyWindow() {
	BindContext()
	sdl.GLDeleteContext(state.glWCtx)
	if err := state.w.Destroy(); err != nil {
		log.Printf("[SDL] couldn't destroy the window, error: %v", err)
	}
}

// BindContext explicitly binds context to current thread.
func BindContext() {
	if err := state.w.GLMakeCurrent(state.glWCtx); err != nil {
		log.Printf("[SDL] error: %v", err)
	}
}

func GetGlFbo() uint32 {
	return getFbo()
}

func GetGlProcAddress(proc string) unsafe.Pointer {
	return sdl.GLGetProcAddress(proc)
}

func setAttribute(attr sdl.GLattr, value int) {
	if err := sdl.GLSetAttribute(attr, value); err != nil {
		log.Printf("[SDL] attribute error: %v", err)
	}
}
