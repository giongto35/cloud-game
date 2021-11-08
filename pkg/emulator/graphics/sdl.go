package graphics

import (
	"fmt"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/veandco/go-sdl2/sdl"
)

type SDL struct {
	glWCtx sdl.GLContext
	w      *sdl.Window
	log    *logger.Logger
}

type Config struct {
	Ctx            Context
	W              int
	H              int
	GLAutoContext  bool
	GLVersionMajor uint
	GLVersionMinor uint
	GLHasDepth     bool
	GLHasStencil   bool
}

// NewSDLContext initializes SDL/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func NewSDLContext(cfg Config, log *logger.Logger) (*SDL, error) {
	log.Debug().Msg("[SDL/OpenGL] initialization...")

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return nil, fmt.Errorf("SDL initialization fail: %w", err)
	}

	display := SDL{log: log}

	if cfg.GLAutoContext {
		log.Debug().Msgf("[OpenGL] CONTEXT_AUTO (type: %v v%v.%v)", cfg.Ctx, cfg.GLVersionMajor, cfg.GLVersionMinor)
	} else {
		switch cfg.Ctx {
		case CtxOpenGlCore:
			display.setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
			log.Debug().Msgf("[OpenGL] CONTEXT_PROFILE_CORE")
		case CtxOpenGlEs2:
			display.setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES)
			display.setAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
			display.setAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 0)
			log.Debug().Msgf("[OpenGL] CONTEXT_PROFILE_ES 3.0")
		case CtxOpenGl:
			if cfg.GLVersionMajor >= 3 {
				display.setAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_COMPATIBILITY)
			}
			log.Debug().Msgf("[OpenGL] CONTEXT_PROFILE_COMPATIBILITY")
		default:
			log.Error().Msgf("[OpenGL] Unsupported hw context: %v", cfg.Ctx)
		}
	}

	var err error
	// In OSX 10.14+ window creation and context creation must happen in the main thread
	thread.MainMaybe(func() { display.w, display.glWCtx, err = createWindow() })
	if err != nil {
		return nil, fmt.Errorf("window fail: %w", err)
	}

	if err := display.BindContext(); err != nil {
		return nil, fmt.Errorf("bind context fail: %w", err)
	}
	initContext(sdl.GLGetProcAddress)
	if err := initFramebuffer(cfg.W, cfg.H, cfg.GLHasDepth, cfg.GLHasStencil); err != nil {
		return nil, fmt.Errorf("OpenGL initialization fail: %w", err)
	}
	return &display, nil
}

// Deinit destroys SDL/OpenGL context.
// Uses main thread lock (see thread/mainthread).
func (s *SDL) Deinit() error {
	s.log.Debug().Msg("[SDL/OpenGL] deinitialization...")
	destroyFramebuffer()
	var err error
	// In OSX 10.14+ window deletion must happen in the main thread
	thread.MainMaybe(func() {
		err = s.destroyWindow()
	})
	if err != nil {
		return fmt.Errorf("[SDL/OpenGL] deinit fail: %w", err)
	}
	sdl.Quit()
	s.log.Debug().Msgf("[SDL/OpenGL] deinitialized codes:(%v, %v)", sdl.GetError(), GetGLError())
	return nil
}

// createWindow creates a fake SDL window just for OpenGL initialization purposes.
func createWindow() (*sdl.Window, sdl.GLContext, error) {
	w, err := sdl.CreateWindow(
		"CloudRetro dummy window",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		1, 1,
		sdl.WINDOW_OPENGL|sdl.WINDOW_HIDDEN,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("window creation fail: %w", err)
	}
	glWCtx, err := w.GLCreateContext()
	if err != nil {
		return nil, nil, fmt.Errorf("window OpenGL context fail: %w", err)
	}
	return w, glWCtx, nil
}

// destroyWindow destroys previously created SDL window.
func (s *SDL) destroyWindow() error {
	if err := s.BindContext(); err != nil {
		return err
	}
	sdl.GLDeleteContext(s.glWCtx)
	if err := s.w.Destroy(); err != nil {
		return fmt.Errorf("window destroy fail: %w", err)
	}
	return nil
}

// BindContext explicitly binds context to current thread.
func (s *SDL) BindContext() error { return s.w.GLMakeCurrent(s.glWCtx) }

// setAttribute tries to set a GL attribute or prints error.
func (s *SDL) setAttribute(attr sdl.GLattr, value int) {
	if err := sdl.GLSetAttribute(attr, value); err != nil {
		s.log.Error().Err(err).Msg("[SDL] attribute")
	}
}

func GetGlFbo() uint32 { return getFbo() }

func GetGlProcAddress(proc string) unsafe.Pointer { return sdl.GLGetProcAddress(proc) }
