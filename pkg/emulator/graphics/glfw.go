package graphics

import (
	"log"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type data struct {
	w *glfw.Window
}

// singleton state for GLFW
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

// Init initializes GLFW/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func Init(cfg Config) {
	log.Printf("[GLFW] [OpenGL] initialization...")
	if err := glfw.Init(); err != nil {
		log.Printf("[GLFW] error: %v", err)
		panic("GLFW initialization failed")
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)

	if cfg.Gl.AutoContext {
		log.Printf("[OpenGL] CONTEXT_AUTO (type: %v v%v.%v)", cfg.Ctx, cfg.Gl.VersionMajor, cfg.Gl.VersionMinor)
	} else {
		switch cfg.Ctx {
		case CtxOpenGlCore:
			glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
			log.Printf("[OpenGL] CONTEXT_PROFILE_CORE")
			break
		case CtxOpenGlEs2:
			glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLESAPI)
			glfw.WindowHint(glfw.ContextVersionMajor, 3)
			glfw.WindowHint(glfw.ContextVersionMinor, 0)
			log.Printf("[OpenGL] CONTEXT_PROFILE_ES 3.0")
			break
		case CtxOpenGl:
			if cfg.Gl.VersionMajor >= 3 {
				glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
			}
			log.Printf("[OpenGL] CONTEXT_PROFILE_COMPATIBILITY")
			break
		default:
			log.Printf("Unsupported hw context: %v", cfg.Ctx)
		}
	}

	// In OSX 10.14+ window creation and context creation must happen in the main thread
	thread.MainMaybe(createWindow)
	state.w.MakeContextCurrent()
	initContext(glfw.GetProcAddress)

	PrintDriverInfo()
	initFramebuffer(cfg.W, cfg.H, cfg.Gl.HasDepth, cfg.Gl.HasStencil)
}

// Deinit destroys GLFW/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func Deinit() {
	log.Printf("[GLFW] [OpenGL] deinitialization...")
	destroyFramebuffer()
	// In OSX 10.14+ window deletion must happen in the main thread
	thread.MainMaybe(func() {
		destroyWindow()
		glfw.Terminate()
	})
	log.Printf("[GLFW] [OpenGL] deinitialized")
}

// createWindow creates fake GLFW window for OpenGL initialization purposes.
func createWindow() {
	var err error
	if state.w, err = glfw.CreateWindow(1, 1, "CloudRetro dummy window", nil, nil);
		err != nil {
		panic(err)
	}
}

// destroyWindow destroys previously created GLFW window.
func destroyWindow() {
	state.w.MakeContextCurrent()
	state.w.Destroy()
}

// BindContext explicitly binds context to current thread.
func BindContext() { state.w.MakeContextCurrent() }

func GetGlFbo() uint32 { return getFbo() }

func GetGlProcAddress(proc string) unsafe.Pointer { return glfw.GetProcAddress(proc) }
