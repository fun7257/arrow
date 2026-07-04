package target

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/fun7257/arrow"
)

// Problem is an RFC 7807 problem details payload.
type Problem struct {
	Type     string            `json:"type,omitempty"`
	Title    string            `json:"title,omitempty"`
	Status   int               `json:"status,omitempty"`
	Detail   string            `json:"detail,omitempty"`
	Instance string            `json:"instance,omitempty"`
	// Extra holds extension members merged into the top-level object.
	// Keys matching standard RFC 7807 fields (type, title, status, detail, instance) are skipped.
	Extra map[string]string `json:"-"`
}

var problemReservedKeys = map[string]struct{}{
	"type": {}, "title": {}, "status": {}, "detail": {}, "instance": {},
}

// MarshalJSON merges Extra members into the top-level problem object.
func (p Problem) MarshalJSON() ([]byte, error) {
	type problemAlias Problem
	alias := problemAlias(p)
	raw, err := json.Marshal(alias)
	if err != nil {
		return nil, err
	}
	if len(p.Extra) == 0 {
		return raw, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	for k, v := range p.Extra {
		if _, reserved := problemReservedKeys[k]; reserved {
			continue
		}
		m[k] = v
	}
	return json.Marshal(m)
}

type problemEncoder struct{}

func (problemEncoder) ContentType() string {
	return "application/problem+json; charset=utf-8"
}

func (problemEncoder) Encode(w io.Writer, v Problem) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(Default.JSONEscapeHTML)
	return enc.Encode(v)
}

// WriteProblem writes an RFC 7807 problem response.
func WriteProblem(c *arrow.Context, p Problem) error {
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}
	return Write(c, Encoded[Problem]{
		Status:  p.Status,
		Encoder: problemEncoder{},
		Body:    p,
	})
}

// AbortProblem aborts with an RFC 7807 problem response.
func AbortProblem(c *arrow.Context, p Problem) error {
	return Abort(c, problemAbortTarget{p: p})
}

type problemAbortTarget struct {
	p Problem
}

func (t problemAbortTarget) StatusCode() int {
	if t.p.Status == 0 {
		return http.StatusInternalServerError
	}
	return t.p.Status
}

func (t problemAbortTarget) Respond(c *arrow.Context) error {
	p := t.p
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}
	return Encoded[Problem]{
		Status:  p.Status,
		Encoder: problemEncoder{},
		Body:    p,
	}.Respond(c)
}