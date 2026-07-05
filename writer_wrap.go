package arrow

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
)

var wrapPools [16]sync.Pool

func init() {
	wrapPools[1].New = func() any { return &wrapF{} }
	wrapPools[2].New = func() any { return &wrapH{} }
	wrapPools[3].New = func() any { return &wrapFH{} }
	wrapPools[4].New = func() any { return &wrapP{} }
	wrapPools[5].New = func() any { return &wrapFP{} }
	wrapPools[6].New = func() any { return &wrapHP{} }
	wrapPools[7].New = func() any { return &wrapFHP{} }
	wrapPools[8].New = func() any { return &wrapR{} }
	wrapPools[9].New = func() any { return &wrapFR{} }
	wrapPools[10].New = func() any { return &wrapHR{} }
	wrapPools[11].New = func() any { return &wrapFHR{} }
	wrapPools[12].New = func() any { return &wrapPR{} }
	wrapPools[13].New = func() any { return &wrapFPR{} }
	wrapPools[14].New = func() any { return &wrapHPR{} }
	wrapPools[15].New = func() any { return &wrapFHPR{} }
}

func releaseWrap(c *Context) {
	if c.wrapPtr == nil {
		return
	}
	if c.wrapPtr == &c.inlineF {
		c.inlineF = wrapF{}
	} else {
		wrapPools[c.wrapMask].Put(c.wrapPtr)
	}
	c.wrapPtr = nil
	c.wrapMask = 0
}

var writerMaskCache sync.Map // map[reflect.Type]uint8

func writerIfaceMask(w http.ResponseWriter) (mask uint8, f http.Flusher, h http.Hijacker, p http.Pusher, rf io.ReaderFrom) {
	t := reflect.TypeOf(w)
	if v, ok := writerMaskCache.Load(t); ok {
		mask = v.(uint8)
	} else {
		if _, ok := w.(http.Flusher); ok {
			mask |= 1
		}
		if _, ok := w.(http.Hijacker); ok {
			mask |= 2
		}
		if _, ok := w.(http.Pusher); ok {
			mask |= 4
		}
		if _, ok := w.(io.ReaderFrom); ok {
			mask |= 8
		}
		writerMaskCache.Store(t, mask)
	}
	if mask&1 != 0 {
		f, _ = w.(http.Flusher)
	}
	if mask&2 != 0 {
		h, _ = w.(http.Hijacker)
	}
	if mask&4 != 0 {
		p, _ = w.(http.Pusher)
	}
	if mask&8 != 0 {
		rf, _ = w.(io.ReaderFrom)
	}
	return mask, f, h, p, rf
}

// wrapResponseWriter returns a ResponseWriter that tracks status and exposes
// optional interfaces only when the underlying writer supports them.
// Separate concrete types are used so nil embedded interfaces cannot satisfy
// type assertions (see Go promotion rules for embedded interface fields).
func wrapResponseWriter(sw *statusWriter, c *Context) http.ResponseWriter {
	w := sw.ResponseWriter
	mask, f, h, p, rf := writerIfaceMask(w)

	switch mask {
	case 0:
		return sw
	case 1:
		c.inlineF.statusWriter = sw
		c.inlineF.f = f
		c.wrapMask = 1
		c.wrapPtr = &c.inlineF
		return &c.inlineF
	case 2:
		wr := wrapPools[2].Get().(*wrapH)
		wr.statusWriter = sw
		wr.h = h
		c.wrapMask = 2
		c.wrapPtr = wr
		return wr
	case 3:
		wr := wrapPools[3].Get().(*wrapFH)
		wr.statusWriter = sw
		wr.f = f
		wr.h = h
		c.wrapMask = 3
		c.wrapPtr = wr
		return wr
	case 4:
		wr := wrapPools[4].Get().(*wrapP)
		wr.statusWriter = sw
		wr.p = p
		c.wrapMask = 4
		c.wrapPtr = wr
		return wr
	case 5:
		wr := wrapPools[5].Get().(*wrapFP)
		wr.statusWriter = sw
		wr.f = f
		wr.p = p
		c.wrapMask = 5
		c.wrapPtr = wr
		return wr
	case 6:
		wr := wrapPools[6].Get().(*wrapHP)
		wr.statusWriter = sw
		wr.h = h
		wr.p = p
		c.wrapMask = 6
		c.wrapPtr = wr
		return wr
	case 7:
		wr := wrapPools[7].Get().(*wrapFHP)
		wr.statusWriter = sw
		wr.f = f
		wr.h = h
		wr.p = p
		c.wrapMask = 7
		c.wrapPtr = wr
		return wr
	case 8:
		wr := wrapPools[8].Get().(*wrapR)
		wr.statusWriter = sw
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 8
		c.wrapPtr = wr
		return wr
	case 9:
		wr := wrapPools[9].Get().(*wrapFR)
		wr.statusWriter = sw
		wr.f = f
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 9
		c.wrapPtr = wr
		return wr
	case 10:
		wr := wrapPools[10].Get().(*wrapHR)
		wr.statusWriter = sw
		wr.h = h
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 10
		c.wrapPtr = wr
		return wr
	case 11:
		wr := wrapPools[11].Get().(*wrapFHR)
		wr.statusWriter = sw
		wr.f = f
		wr.h = h
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 11
		c.wrapPtr = wr
		return wr
	case 12:
		wr := wrapPools[12].Get().(*wrapPR)
		wr.statusWriter = sw
		wr.p = p
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 12
		c.wrapPtr = wr
		return wr
	case 13:
		wr := wrapPools[13].Get().(*wrapFPR)
		wr.statusWriter = sw
		wr.f = f
		wr.p = p
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 13
		c.wrapPtr = wr
		return wr
	case 14:
		wr := wrapPools[14].Get().(*wrapHPR)
		wr.statusWriter = sw
		wr.h = h
		wr.p = p
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 14
		c.wrapPtr = wr
		return wr
	case 15:
		wr := wrapPools[15].Get().(*wrapFHPR)
		wr.statusWriter = sw
		wr.f = f
		wr.h = h
		wr.p = p
		wr.d = readerFromDelegator{inner: sw, delegate: rf}
		c.wrapMask = 15
		c.wrapPtr = wr
		return wr
	default:
		return sw
	}
}

func markFlushed(sw *statusWriter) {
	sw.written = true
}

type wrapF struct {
	*statusWriter
	f http.Flusher
}

func (w *wrapF) Flush()                      { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapF) Unwrap() http.ResponseWriter { return w.statusWriter }

type wrapH struct {
	*statusWriter
	h http.Hijacker
}

func (w *wrapH) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapH) Unwrap() http.ResponseWriter                  { return w.statusWriter }

type wrapFH struct {
	*statusWriter
	f http.Flusher
	h http.Hijacker
}

func (w *wrapFH) Flush()                                       { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFH) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapFH) Unwrap() http.ResponseWriter                  { return w.statusWriter }

type wrapP struct {
	*statusWriter
	p http.Pusher
}

func (w *wrapP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapP) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapFP struct {
	*statusWriter
	f http.Flusher
	p http.Pusher
}

func (w *wrapFP) Flush()                                           { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFP) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapHP struct {
	*statusWriter
	h http.Hijacker
	p http.Pusher
}

func (w *wrapHP) Hijack() (net.Conn, *bufio.ReadWriter, error)     { return w.h.Hijack() }
func (w *wrapHP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapHP) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapFHP struct {
	*statusWriter
	f http.Flusher
	h http.Hijacker
	p http.Pusher
}

func (w *wrapFHP) Flush()                                           { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHP) Hijack() (net.Conn, *bufio.ReadWriter, error)     { return w.h.Hijack() }
func (w *wrapFHP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFHP) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapR struct {
	*statusWriter
	d readerFromDelegator
}

func (w *wrapR) ReadFrom(r io.Reader) (int64, error) { return w.d.ReadFrom(r) }
func (w *wrapR) Unwrap() http.ResponseWriter         { return w.statusWriter }

type wrapFR struct {
	*statusWriter
	f http.Flusher
	d readerFromDelegator
}

func (w *wrapFR) Flush()                              { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFR) ReadFrom(r io.Reader) (int64, error) { return w.d.ReadFrom(r) }
func (w *wrapFR) Unwrap() http.ResponseWriter         { return w.statusWriter }

type wrapHR struct {
	*statusWriter
	h http.Hijacker
	d readerFromDelegator
}

func (w *wrapHR) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapHR) ReadFrom(r io.Reader) (int64, error)          { return w.d.ReadFrom(r) }
func (w *wrapHR) Unwrap() http.ResponseWriter                  { return w.statusWriter }

type wrapFHR struct {
	*statusWriter
	f http.Flusher
	h http.Hijacker
	d readerFromDelegator
}

func (w *wrapFHR) Flush()                                       { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHR) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapFHR) ReadFrom(r io.Reader) (int64, error)          { return w.d.ReadFrom(r) }
func (w *wrapFHR) Unwrap() http.ResponseWriter                  { return w.statusWriter }

type wrapPR struct {
	*statusWriter
	p http.Pusher
	d readerFromDelegator
}

func (w *wrapPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapPR) ReadFrom(r io.Reader) (int64, error)              { return w.d.ReadFrom(r) }
func (w *wrapPR) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapFPR struct {
	*statusWriter
	f http.Flusher
	p http.Pusher
	d readerFromDelegator
}

func (w *wrapFPR) Flush()                                           { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFPR) ReadFrom(r io.Reader) (int64, error)              { return w.d.ReadFrom(r) }
func (w *wrapFPR) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapHPR struct {
	*statusWriter
	h http.Hijacker
	p http.Pusher
	d readerFromDelegator
}

func (w *wrapHPR) Hijack() (net.Conn, *bufio.ReadWriter, error)     { return w.h.Hijack() }
func (w *wrapHPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapHPR) ReadFrom(r io.Reader) (int64, error)              { return w.d.ReadFrom(r) }
func (w *wrapHPR) Unwrap() http.ResponseWriter                      { return w.statusWriter }

type wrapFHPR struct {
	*statusWriter
	f http.Flusher
	h http.Hijacker
	p http.Pusher
	d readerFromDelegator
}

func (w *wrapFHPR) Flush()                                           { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHPR) Hijack() (net.Conn, *bufio.ReadWriter, error)     { return w.h.Hijack() }
func (w *wrapFHPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFHPR) ReadFrom(r io.Reader) (int64, error)              { return w.d.ReadFrom(r) }
func (w *wrapFHPR) Unwrap() http.ResponseWriter                      { return w.statusWriter }
