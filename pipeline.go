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

// finishRequest runs the handler (unless aborted) then After callbacks in FIFO order.
func finishRequest(ctx *Context, handler HandlerFunc) {
	if !ctx.aborted {
		handler(ctx)
	}
	for _, after := range ctx.afters {
		after(ctx)
	}
}

// executeZeroMiddleware is the shared handler+after body for zero-middleware
// requests. Router inlines this logic (no pre-abort on zero-mw registration);
// runNoMiddleware and pipeline.Run call through finishRequest.
func executeZeroMiddleware(ctx *Context, handler HandlerFunc) {
	finishRequest(ctx, handler)
}

// runNoMiddleware is the zero-middleware entry used by pipeline.Run when
// len(middlewares)==0. Router registration uses serveZeroMiddlewareFromHTTP.
func runNoMiddleware(ctx *Context, handler HandlerFunc) {
	zeroMiddlewarePipelineDispatches.Add(1)
	defer recoverAndRelease(ctx)
	executeZeroMiddleware(ctx, handler)
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

	finishRequest(ctx, handler)
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