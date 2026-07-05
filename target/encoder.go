package target

import (
	"encoding/json"
	"encoding/xml"
	"io"
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