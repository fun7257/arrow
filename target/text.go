package target

import "github.com/fun7257/arrow"

// WritePlain writes plain text.
func WritePlain(c *arrow.Context, status int, body string) error {
	return Write(c, Encoded[string]{
		Status:  status,
		Encoder: PlainEncoder{},
		Body:    body,
	})
}

// WriteHTML writes HTML.
func WriteHTML(c *arrow.Context, status int, body string) error {
	return Write(c, Encoded[string]{
		Status:  status,
		Encoder: HTMLEncoder{},
		Body:    body,
	})
}

// WriteBytes writes raw bytes with contentType.
func WriteBytes(c *arrow.Context, status int, contentType string, body []byte) error {
	return Write(c, bytesTarget{
		status:      status,
		contentType: contentType,
		body:        body,
	})
}

type bytesTarget struct {
	status      int
	contentType string
	body        []byte
}

func (b bytesTarget) StatusCode() int {
	return b.status
}

func (b bytesTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	if b.contentType != "" {
		c.Writer.Header().Set("Content-Type", b.contentType)
	}
	c.WriteHeader(b.status)
	if len(b.body) == 0 {
		return nil
	}
	return encodeBytes(b.contentType, c.Writer, b.body)
}