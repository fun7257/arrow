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

// runNoMiddleware is used only by pipeline.Run when len(middlewares)==0.
// Bench hot paths use the router inline closure instead (see router.register).
func runNoMiddleware(ctx *Context, handler HandlerFunc) {
	defer recoverAndRelease(ctx)
	handler(ctx)
	for _, after := range ctx.afters {
		after(ctx)
	}
}

// Run executes the linear penetration pipeline:
// Pre (forward) -> Handler -> After (forward, FIFO).
func (p *pipeline) Run(ctx *Context, handler HandlerFunc) {
	if len(p.middlewares) == 0 {
		runNoMiddleware(ctx, handler)
		return
	}

	defer recoverAndRelease(ctx)

	for _, mw := range p.middlewares {
		mw(ctx)
		if ctx.aborted {
			break
		}
	}

	if !ctx.aborted {
		handler(ctx)
	}

	for _, after := range ctx.afters {
		after(ctx)
	}
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