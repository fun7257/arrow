package target

import (
	"net/http"

	"github.com/fun7257/arrow"
)

// WriteRedirect issues an HTTP redirect.
func WriteRedirect(c *arrow.Context, code int, url string) error {
	return Write(c, redirectTarget{code: code, url: url})
}

// MovedPermanently redirects with 301.
func MovedPermanently(c *arrow.Context, url string) error {
	return WriteRedirect(c, http.StatusMovedPermanently, url)
}

// Found redirects with 302.
func Found(c *arrow.Context, url string) error {
	return WriteRedirect(c, http.StatusFound, url)
}

// SeeOther redirects with 303.
func SeeOther(c *arrow.Context, url string) error {
	return WriteRedirect(c, http.StatusSeeOther, url)
}

// TemporaryRedirect redirects with 307.
func TemporaryRedirect(c *arrow.Context, url string) error {
	return WriteRedirect(c, http.StatusTemporaryRedirect, url)
}

// PermanentRedirect redirects with 308.
func PermanentRedirect(c *arrow.Context, url string) error {
	return WriteRedirect(c, http.StatusPermanentRedirect, url)
}

type redirectTarget struct {
	code int
	url  string
}

func (r redirectTarget) StatusCode() int {
	return r.code
}

func (r redirectTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	http.Redirect(c.Writer, c.Request, r.url, r.code)
	return nil
}