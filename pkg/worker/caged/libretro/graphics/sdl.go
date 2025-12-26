package graphics

import (
	"fmt"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

type SDL struct {
	w   *sdl.Window
	ctx sdl.GLContext
}

type Config struct {
	Ctx            Context
	W, H           int
	GLAutoContext  bool
	GLVersionMajor uint
	GLVersionMinor uint
	GLHasDepth     bool
	GLHasStencil   bool
}

func NewSDLContext(cfg Config) (*SDL, error) {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return nil, fmt.Errorf("sdl: %w", err)
	}

	if !cfg.GLAutoContext {
		if err := setGLAttrs(cfg.Ctx); err != nil {
			return nil, err
		}
	}

	w, err := sdl.CreateWindow("cloud-retro", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1, 1, sdl.WINDOW_OPENGL|sdl.WINDOW_HIDDEN)
	if err != nil {
		return nil, fmt.Errorf("window: %w", err)
	}

	ctx, err := w.GLCreateContext()
	if err != nil {
		err1 := w.Destroy()
		return nil, fmt.Errorf("gl context: %w, destroy err: %w", err, err1)
	}

	if err = w.GLMakeCurrent(ctx); err != nil {
		return nil, fmt.Errorf("gl bind: %w", err)
	}

	initContext(sdl.GLGetProcAddress)

	if err = initFramebuffer(cfg.W, cfg.H, cfg.GLHasDepth, cfg.GLHasStencil); err != nil {
		return nil, fmt.Errorf("fbo: %w", err)
	}

	return &SDL{w: w, ctx: ctx}, nil
}

func setGLAttrs(ctx Context) error {
	set := sdl.GLSetAttribute
	switch ctx {
	case CtxOpenGlCore:
		return set(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	case CtxOpenGlEs2:
		for _, a := range [][2]int{
			{sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_ES},
			{sdl.GL_CONTEXT_MAJOR_VERSION, 3},
			{sdl.GL_CONTEXT_MINOR_VERSION, 0},
		} {
			if err := set(sdl.GLattr(a[0]), a[1]); err != nil {
				return err
			}
		}
		return nil
	case CtxOpenGl:
		return set(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_COMPATIBILITY)
	default:
		return fmt.Errorf("unsupported gl context: %v", ctx)
	}
}

func (s *SDL) Deinit() error {
	destroyFramebuffer()
	sdl.GLDeleteContext(s.ctx)
	err := s.w.Destroy()
	sdl.Quit()
	return err
}

func (s *SDL) BindContext() error              { return s.w.GLMakeCurrent(s.ctx) }
func GlProcAddress(proc string) unsafe.Pointer { return sdl.GLGetProcAddress(proc) }

func TryInit() error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}
	sdl.Quit()
	return nil
}
