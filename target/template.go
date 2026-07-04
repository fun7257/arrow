package target

import (
	"bytes"
	"html/template"

	"github.com/fun7257/arrow"
)

// WriteTemplate executes an HTML template.
func WriteTemplate(c *arrow.Context, status int, tmpl *template.Template, data any) error {
	return Write(c, templateTarget{
		status: status,
		tmpl:   tmpl,
		data:   data,
	})
}

type templateTarget struct {
	status int
	tmpl   *template.Template
	data   any
}

func (t templateTarget) StatusCode() int {
	return t.status
}

func (t templateTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, t.data); err != nil {
		if Default.OnEncodeError != nil {
			Default.OnEncodeError(c, err)
		}
		return err
	}
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.WriteHeader(t.status)
	_, err := c.Write(buf.Bytes())
	return err
}