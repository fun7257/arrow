package arrow

import (
	"sync"
)

var contextPool sync.Pool

func acquireContext() *Context {
	v := contextPool.Get()
	if v == nil {
		return &Context{}
	}
	return v.(*Context)
}

func releaseContext(c *Context) {
	if c == nil {
		return
	}
	releaseWrap(c)
	c.Writer = nil
	c.Request = nil
	c.sw = statusWriter{}
	c.aborted = false
	c.code = 0
	c.written = false
	c.afters = c.afters[:0]
	c.keys = nil
	contextPool.Put(c)
}