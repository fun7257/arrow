package arrow

import (
	"log"
	"net/http"
	"runtime/debug"
	"slices"
)

type pipeline struct {
	middlewares []HandlerFunc
}

func newPipeline() *pipeline {
	return &pipeline{}
}

func (p *pipeline) Use(mw HandlerFunc) {
	p.middlewares = append(p.middlewares, mw)
}

func (p *pipeline) clone() *pipeline {
	return &pipeline{
		middlewares: slices.Clone(p.middlewares),
	}
}

// dispatch runs the route handler (unless aborted) then registered After callbacks.
func dispatch(ctx *Context, handler HandlerFunc) {
	if !ctx.aborted {
		handler(ctx)
	}
	for _, after := range ctx.afters {
		after(ctx)
	}
}

// serveRequest is the zero-middleware hot path: panic recovery, handler, After FIFO.
func serveRequest(ctx *Context, handler HandlerFunc) {
	defer recoverAndRelease(ctx)
	dispatch(ctx, handler)
}

// Run executes the linear penetration pipeline:
// Pre (forward) -> Handler -> After (forward, FIFO).
func (p *pipeline) Run(ctx *Context, handler HandlerFunc) {
	defer recoverAndRelease(ctx)

	for _, mw := range p.middlewares {
		mw(ctx)
		if ctx.aborted {
			break
		}
	}

	dispatch(ctx, handler)
}

func recoverAndRelease(ctx *Context) {
	if r := recover(); r != nil {
		log.Printf("arrow: panic recovered: %v\n%s", r, debug.Stack())
		if !ctx.aborted {
			ctx.Abort(http.StatusInternalServerError)
		}
	}
	releaseContext(ctx)
}