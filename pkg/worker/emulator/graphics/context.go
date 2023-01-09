package graphics

import "math"

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
