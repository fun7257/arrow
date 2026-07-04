package arrow

import (
	"net/http"
	"sync"
)

// Linear builds middleware with explicit pre and post phases.
func Linear(pre, post func(*Context)) HandlerFunc {
	return func(c *Context) {
		if pre != nil {
			pre(c)
			if c.aborted {
				return
			}
		}
		if post != nil {
			c.After(post)
		}
	}
}

// Adapt converts a classic net/http middleware into Arrow middleware.
//
// Pre-work (before next.ServeHTTP) runs during the Pre phase. Post-work
// (after next returns) is deferred to the After phase and runs in forward
// registration order alongside other After callbacks.
func Adapt(mw func(http.Handler) http.Handler) HandlerFunc {
	return func(c *Context) {
		if c.aborted {
			return
		}

		gate := make(chan struct{})
		release := make(chan struct{})
		penetrated := false
		var gateOnce sync.Once
		openGate := func() {
			gateOnce.Do(func() { close(gate) })
		}

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if penetrated {
				return
			}
			penetrated = true
			openGate()
			<-release
		})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer func() {
				openGate()
				wg.Done()
			}()
			mw(inner).ServeHTTP(c.Writer, c.Request)
		}()

		<-gate

		if !penetrated {
			wg.Wait()
			c.aborted = true
			if !c.written {
				c.Abort(http.StatusForbidden)
			}
			return
		}

		c.After(func(c *Context) {
			close(release)
			wg.Wait()
		})
	}
}