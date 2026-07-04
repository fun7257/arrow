package target

import "github.com/fun7257/arrow"

// WriteXML writes body as XML with status.
func WriteXML[T any](c *arrow.Context, status int, body T) error {
	return Write(c, XML[T](status, body))
}