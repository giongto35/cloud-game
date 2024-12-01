package network

import "time"

const retry = 10 * time.Second

type Retry struct {
	t    time.Duration
	fail bool
}

func NewRetry() Retry {
	return Retry{t: retry}
}

func (r *Retry) Fail() *Retry        { r.fail = true; time.Sleep(r.t); return r }
func (r *Retry) Multiply(x int)      { r.t *= time.Duration(x) }
func (r *Retry) Success()            { r.t = retry; r.fail = false }
func (r *Retry) Time() time.Duration { return r.t }
