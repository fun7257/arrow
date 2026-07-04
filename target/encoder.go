package target

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"sync"
)

// Encoder serializes a typed body and reports its Content-Type.
type Encoder[T any] interface {
	ContentType() string
	Encode(w io.Writer, v T) error
}

// JSONEncoder encodes values as JSON.
// Encode appends a trailing newline, matching encoding/json.Encoder behavior.
type JSONEncoder[T any] struct{}

func (JSONEncoder[T]) ContentType() string {
	return "application/json; charset=utf-8"
}

func (JSONEncoder[T]) Encode(w io.Writer, v T) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(Default.JSONEscapeHTML)
	return enc.Encode(v)
}

// IndentJSONEncoder encodes values as indented JSON.
type IndentJSONEncoder[T any] struct {
	Prefix string
	Indent string
}

func (IndentJSONEncoder[T]) ContentType() string {
	return "application/json; charset=utf-8"
}

func (e IndentJSONEncoder[T]) Encode(w io.Writer, v T) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(Default.JSONEscapeHTML)
	enc.SetIndent(e.Prefix, e.Indent)
	return enc.Encode(v)
}

// XMLEncoder encodes values as XML.
type XMLEncoder[T any] struct{}

func (XMLEncoder[T]) ContentType() string {
	return "application/xml; charset=utf-8"
}

func (XMLEncoder[T]) Encode(w io.Writer, v T) error {
	enc := xml.NewEncoder(w)
	return enc.Encode(v)
}

// PlainEncoder encodes plain text bodies.
type PlainEncoder struct{}

func (PlainEncoder) ContentType() string {
	return "text/plain; charset=utf-8"
}

func (PlainEncoder) Encode(w io.Writer, v string) error {
	_, err := io.WriteString(w, v)
	return err
}

// HTMLEncoder encodes HTML bodies.
type HTMLEncoder struct{}

func (HTMLEncoder) ContentType() string {
	return "text/html; charset=utf-8"
}

func (HTMLEncoder) Encode(w io.Writer, v string) error {
	_, err := io.WriteString(w, v)
	return err
}

// BytesEncoder encodes raw bytes with a fixed Content-Type.
type BytesEncoder struct {
	ContentType string
}

func (e BytesEncoder) Encode(w io.Writer, v []byte) error {
	_, err := w.Write(v)
	return err
}

var (
	bytesEncoderMu sync.RWMutex
	bytesEncoders  = map[string]func(io.Writer, []byte) error{}
)

// RegisterEncoder registers a custom []byte encoder for contentType.
// Content types registered here can be selected by WriteNegotiated when listed in Accept
// or returned from Options.SelectFormat.
//
// In the WriteNegotiated registered-encoder path, the body is JSON-encoded first and
// the resulting bytes are passed to fn. Encoders receive JSON bytes, not the original value.
func RegisterEncoder(contentType string, fn func(w io.Writer, v []byte) error) {
	bytesEncoderMu.Lock()
	defer bytesEncoderMu.Unlock()
	if fn == nil {
		delete(bytesEncoders, contentType)
		return
	}
	bytesEncoders[contentType] = fn
}

func encodeBytes(contentType string, w io.Writer, v []byte) error {
	bytesEncoderMu.RLock()
	fn, ok := bytesEncoders[contentType]
	bytesEncoderMu.RUnlock()
	if ok {
		return fn(w, v)
	}
	_, err := w.Write(v)
	return err
}