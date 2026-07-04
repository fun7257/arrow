package target

import "github.com/fun7257/arrow"

// Options configures target response behavior.
type Options struct {
	JSONEscapeHTML bool
	// OnEncodeError is called when encoding fails before headers are sent.
	// Buffered encoders avoid committing partial responses on encode errors.
	OnEncodeError func(c *arrow.Context, err error)
	BeforeWrite   func(c *arrow.Context, t Target) (Target, error)
	// SelectFormat chooses a negotiated format name from the Accept header.
	// Built-in names are "json" and "xml"; registered content types from
	// RegisterEncoder are also valid return values.
	SelectFormat func(c *arrow.Context, accept string) string
}

// Default holds package-level response options.
var Default Options