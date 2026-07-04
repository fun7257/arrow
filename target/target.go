package target

import (
	"github.com/fun7257/arrow"
)

// Target writes an HTTP response through an arrow Context.
type Target interface {
	Respond(c *arrow.Context) error
}

// StatusCarrier is implemented by targets that carry an explicit status code.
type StatusCarrier interface {
	StatusCode() int
}

// Write applies optional hooks and writes t without aborting penetration.
func Write(c *arrow.Context, t Target) error {
	if c.Written() {
		return nil
	}
	if Default.BeforeWrite != nil {
		var err error
		t, err = Default.BeforeWrite(c, t)
		if err != nil {
			return err
		}
	}
	return t.Respond(c)
}

// Abort writes t then calls c.Abort with the response status.
// If the response is already written, only c.Abort is called.
func Abort(c *arrow.Context, t Target) error {
	if c.Written() {
		status := statusOf(t)
		if status == 0 {
			status = c.Status()
		}
		c.Abort(status)
		return nil
	}
	if Default.BeforeWrite != nil {
		var err error
		t, err = Default.BeforeWrite(c, t)
		if err != nil {
			return err
		}
	}
	status := statusOf(t)
	err := t.Respond(c)
	if status == 0 {
		status = c.Status()
	}
	c.Abort(status)
	return err
}

func statusOf(t Target) int {
	if sc, ok := t.(StatusCarrier); ok {
		return sc.StatusCode()
	}
	return 0
}

// Func adapts a response function as a Target.
func Func(fn func(c *arrow.Context) error) Target {
	return funcTarget{fn: fn}
}

type funcTarget struct {
	fn func(c *arrow.Context) error
}

func (f funcTarget) Respond(c *arrow.Context) error {
	return f.fn(c)
}