package arrow

import (
	"net/http/httptest"
	"testing"
)

func TestRunNoMiddlewareAfterOrder(t *testing.T) {
	var order []string

	rec := httptest.NewRecorder()
	ctx := newContext(rec, httptest.NewRequest("GET", "/", nil))

	runNoMiddleware(ctx, func(c *Context) {
		c.After(func(c *Context) { order = append(order, "after") })
		order = append(order, "handler")
	})

	want := []string{"handler", "after"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}