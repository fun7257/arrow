package target

import (
	"strconv"
	"strings"

	"github.com/fun7257/arrow"
)

// defaultSelectFormat picks json or xml from Accept using a simplified q-value comparison.
// It does not implement full RFC 7231 content negotiation; use Options.SelectFormat to override.
//
// Contract:
//   - */* is treated as json.
//   - When multiple candidates share the highest q-value, the first listed in Accept wins.
//   - Registered content types from RegisterEncoder are candidates when listed explicitly.
func defaultSelectFormat(_ *arrow.Context, accept string) string {
	best := "json"
	bestQ := 1.0
	hasCandidate := false

	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		media, q := parseAcceptPart(part)
		format := formatForMedia(media)
		if format == "" {
			continue
		}
		if !hasCandidate || q > bestQ {
			best = format
			bestQ = q
			hasCandidate = true
		}
	}
	return best
}

func parseAcceptPart(part string) (media string, q float64) {
	media = part
	q = 1.0
	if i := strings.Index(part, ";"); i >= 0 {
		media = strings.TrimSpace(part[:i])
		for _, param := range strings.Split(part[i+1:], ";") {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				if v, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64); err == nil {
					q = v
				}
			}
		}
	}
	return strings.ToLower(media), q
}

func formatForMedia(media string) string {
	switch media {
	case "*/*":
		return "json"
	case "application/json", "text/json":
		return "json"
	case "application/xml", "text/xml":
		return "xml"
	default:
		bytesEncoderMu.RLock()
		_, ok := bytesEncoders[media]
		bytesEncoderMu.RUnlock()
		if ok {
			return media
		}
		return ""
	}
}

// WriteNegotiated writes JSON or XML based on Accept.
//
// Built-in formats are "json" and "xml". When SelectFormat returns a content type
// registered via RegisterEncoder, the body is JSON-encoded first and the resulting
// bytes are passed to the registered encoder. Custom encoders therefore receive
// JSON bytes and may transform or wrap them; they do not receive the original Go value.
func WriteNegotiated[T any](c *arrow.Context, status int, body T) error {
	sel := Default.SelectFormat
	if sel == nil {
		sel = defaultSelectFormat
	}
	format := sel(c, c.Request.Header.Get("Accept"))
	switch format {
	case "xml":
		return WriteXML(c, status, body)
	case "json":
		return WriteJSON(c, status, body)
	default:
		bytesEncoderMu.RLock()
		_, ok := bytesEncoders[format]
		bytesEncoderMu.RUnlock()
		if ok {
			return writeNegotiatedRegistered(c, status, format, body)
		}
		return WriteJSON(c, status, body)
	}
}

// writeNegotiatedRegistered JSON-encodes body, then passes bytes to the registered encoder.
func writeNegotiatedRegistered[T any](c *arrow.Context, status int, contentType string, body T) error {
	var buf strings.Builder
	enc := JSONEncoder[T]{}
	if err := enc.Encode(&buf, body); err != nil {
		if Default.OnEncodeError != nil {
			Default.OnEncodeError(c, err)
		}
		return err
	}
	return WriteBytes(c, status, contentType, []byte(buf.String()))
}