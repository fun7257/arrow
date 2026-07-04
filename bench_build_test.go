package arrow_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/fun7257/arrow"
)

func stdlibPattern(method, pattern string) string {
	if method == "" {
		return pattern
	}
	return method + " " + pattern
}

func stdlibHandler(resp string) http.HandlerFunc {
	body := []byte(resp)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}
}

func registerStdlibRoutes(tb testing.TB, mux *http.ServeMux, routes []BenchRoute) {
	tb.Helper()
	for _, rt := range routes {
		mux.Handle(stdlibPattern(rt.Method, rt.Pattern), stdlibHandler(rt.Response))
	}
}

func registerArrowRoutes(app *arrow.Router, routes []BenchRoute) {
	for _, rt := range routes {
		resp := rt.Response
		h := func(c *arrow.Context) {
			io.WriteString(c.Writer, resp)
		}
		switch rt.Method {
		case http.MethodGet:
			app.GET(rt.Pattern, h)
		case http.MethodPost:
			app.POST(rt.Pattern, h)
		case http.MethodPut:
			app.PUT(rt.Pattern, h)
		case http.MethodDelete:
			app.DELETE(rt.Pattern, h)
		case http.MethodPatch:
			app.PATCH(rt.Pattern, h)
		default:
			app.Handle(rt.Method, rt.Pattern, h)
		}
	}
}

func buildStdlibMux(tb testing.TB, s BenchScenario) *http.ServeMux {
	tb.Helper()
	mux := http.NewServeMux()
	registerStdlibRoutes(tb, mux, s.Routes)
	return mux
}

func buildArrowApp(s BenchScenario, middleware ...arrow.HandlerFunc) http.Handler {
	app := arrow.New()
	for _, mw := range middleware {
		app.Use(mw)
	}
	registerArrowRoutes(app, s.Routes)
	return app.Handler()
}

func benchRequest(req BenchRequest) *http.Request {
	r, err := http.NewRequest(req.Method, req.Path, nil)
	if err != nil {
		panic(err)
	}
	if req.Host != "" {
		r.Host = req.Host
	}
	return r
}

// noopMiddleware is a lightweight counter middleware for pipeline overhead measurement.
func noopMiddleware() arrow.HandlerFunc {
	return func(c *arrow.Context) {
		c.After(func(c *arrow.Context) {})
	}
}

func middlewareStack(depth int) []arrow.HandlerFunc {
	mws := make([]arrow.HandlerFunc, depth)
	for i := range mws {
		mws[i] = noopMiddleware()
	}
	return mws
}