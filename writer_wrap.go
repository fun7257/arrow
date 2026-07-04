package arrow

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// wrapResponseWriter returns a ResponseWriter that tracks status and exposes
// optional interfaces only when the underlying writer supports them.
// Separate concrete types are used so nil embedded interfaces cannot satisfy
// type assertions (see Go promotion rules for embedded interface fields).
func wrapResponseWriter(sw *statusWriter) http.ResponseWriter {
	w := sw.ResponseWriter

	f, hasF := w.(http.Flusher)
	h, hasH := w.(http.Hijacker)
	p, hasP := w.(http.Pusher)
	rf, hasR := w.(io.ReaderFrom)

	mask := 0
	if hasF {
		mask |= 1
	}
	if hasH {
		mask |= 2
	}
	if hasP {
		mask |= 4
	}
	if hasR {
		mask |= 8
	}

	switch mask {
	case 0:
		return sw
	case 1:
		return &wrapF{sw, f}
	case 2:
		return &wrapH{sw, h}
	case 3:
		return &wrapFH{sw, f, h}
	case 4:
		return &wrapP{sw, p}
	case 5:
		return &wrapFP{sw, f, p}
	case 6:
		return &wrapHP{sw, h, p}
	case 7:
		return &wrapFHP{sw, f, h, p}
	case 8:
		return &wrapR{sw, &readerFromDelegator{inner: sw, delegate: rf}}
	case 9:
		return &wrapFR{sw, f, &readerFromDelegator{inner: sw, delegate: rf}}
	case 10:
		return &wrapHR{sw, h, &readerFromDelegator{inner: sw, delegate: rf}}
	case 11:
		return &wrapFHR{sw, f, h, &readerFromDelegator{inner: sw, delegate: rf}}
	case 12:
		return &wrapPR{sw, p, &readerFromDelegator{inner: sw, delegate: rf}}
	case 13:
		return &wrapFPR{sw, f, p, &readerFromDelegator{inner: sw, delegate: rf}}
	case 14:
		return &wrapHPR{sw, h, p, &readerFromDelegator{inner: sw, delegate: rf}}
	case 15:
		return &wrapFHPR{sw, f, h, p, &readerFromDelegator{inner: sw, delegate: rf}}
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

func (w *wrapF) Flush()               { w.f.Flush(); markFlushed(w.statusWriter) }
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
func (w *wrapP) Unwrap() http.ResponseWriter                       { return w.statusWriter }

type wrapFP struct {
	*statusWriter
	f http.Flusher
	p http.Pusher
}

func (w *wrapFP) Flush()                                          { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFP) Unwrap() http.ResponseWriter                     { return w.statusWriter }

type wrapHP struct {
	*statusWriter
	h http.Hijacker
	p http.Pusher
}

func (w *wrapHP) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return w.h.Hijack() }
func (w *wrapHP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapHP) Unwrap() http.ResponseWriter                     { return w.statusWriter }

type wrapFHP struct {
	*statusWriter
	f http.Flusher
	h http.Hijacker
	p http.Pusher
}

func (w *wrapFHP) Flush()                                          { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHP) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return w.h.Hijack() }
func (w *wrapFHP) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFHP) Unwrap() http.ResponseWriter                     { return w.statusWriter }

type wrapR struct {
	*statusWriter
	rf io.ReaderFrom
}

func (w *wrapR) ReadFrom(r io.Reader) (int64, error) { return w.rf.ReadFrom(r) }
func (w *wrapR) Unwrap() http.ResponseWriter         { return w.statusWriter }

type wrapFR struct {
	*statusWriter
	f  http.Flusher
	rf io.ReaderFrom
}

func (w *wrapFR) Flush()                            { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFR) ReadFrom(r io.Reader) (int64, error) { return w.rf.ReadFrom(r) }
func (w *wrapFR) Unwrap() http.ResponseWriter       { return w.statusWriter }

type wrapHR struct {
	*statusWriter
	h  http.Hijacker
	rf io.ReaderFrom
}

func (w *wrapHR) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapHR) ReadFrom(r io.Reader) (int64, error)          { return w.rf.ReadFrom(r) }
func (w *wrapHR) Unwrap() http.ResponseWriter                    { return w.statusWriter }

type wrapFHR struct {
	*statusWriter
	f  http.Flusher
	h  http.Hijacker
	rf io.ReaderFrom
}

func (w *wrapFHR) Flush()                                       { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHR) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.h.Hijack() }
func (w *wrapFHR) ReadFrom(r io.Reader) (int64, error)          { return w.rf.ReadFrom(r) }
func (w *wrapFHR) Unwrap() http.ResponseWriter                  { return w.statusWriter }

type wrapPR struct {
	*statusWriter
	p  http.Pusher
	rf io.ReaderFrom
}

func (w *wrapPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapPR) ReadFrom(r io.Reader) (int64, error)             { return w.rf.ReadFrom(r) }
func (w *wrapPR) Unwrap() http.ResponseWriter                       { return w.statusWriter }

type wrapFPR struct {
	*statusWriter
	f  http.Flusher
	p  http.Pusher
	rf io.ReaderFrom
}

func (w *wrapFPR) Flush()                                          { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFPR) ReadFrom(r io.Reader) (int64, error)           { return w.rf.ReadFrom(r) }
func (w *wrapFPR) Unwrap() http.ResponseWriter                     { return w.statusWriter }

type wrapHPR struct {
	*statusWriter
	h  http.Hijacker
	p  http.Pusher
	rf io.ReaderFrom
}

func (w *wrapHPR) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return w.h.Hijack() }
func (w *wrapHPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapHPR) ReadFrom(r io.Reader) (int64, error)           { return w.rf.ReadFrom(r) }
func (w *wrapHPR) Unwrap() http.ResponseWriter                     { return w.statusWriter }

type wrapFHPR struct {
	*statusWriter
	f  http.Flusher
	h  http.Hijacker
	p  http.Pusher
	rf io.ReaderFrom
}

func (w *wrapFHPR) Flush()                                          { w.f.Flush(); markFlushed(w.statusWriter) }
func (w *wrapFHPR) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return w.h.Hijack() }
func (w *wrapFHPR) Push(target string, opts *http.PushOptions) error { return w.p.Push(target, opts) }
func (w *wrapFHPR) ReadFrom(r io.Reader) (int64, error)           { return w.rf.ReadFrom(r) }
func (w *wrapFHPR) Unwrap() http.ResponseWriter                     { return w.statusWriter }