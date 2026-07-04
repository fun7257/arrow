package arrow_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/fun7257/arrow"
)

func arrowHandlerPathValue(key, prefix string) arrow.HandlerFunc {
	return func(c *arrow.Context) {
		pv := c.Request.PathValue(key)
		c.Writer.Header().Set(parityPathValueHeader, pv)
		c.Write([]byte(prefix + pv))
	}
}

// --- Routing parity (ServeMux semantics) ---

func TestInheritParityGETRoute(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/hello", stdHandler("ok"))

	app := arrow.New()
	app.GET("/hello", func(c *arrow.Context) { c.Write([]byte("ok")) })

	assertParity(t, "GET", baseline, app.Handler(), newRequest(http.MethodGet, "/hello", ""))
}

func TestInheritParityAnyAllMethods(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, "", "/any", stdHandlerMethod("hit"))

	app := arrow.New()
	app.Any("/any", func(c *arrow.Context) {
		c.Write([]byte("hit:" + c.Request.Method))
	})

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		assertParity(t, "Any/"+method, baseline, app.Handler(), newRequest(method, "/any", ""))
	}
}

func TestInheritParityGETMatchesHEAD(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/", stdHandler("body"))

	app := arrow.New()
	app.GET("/", func(c *arrow.Context) { c.Write([]byte("body")) })

	assertParity(t, "HEAD", baseline, app.Handler(), newRequest(http.MethodHead, "/", ""))
}

func TestInheritParity404NotFound(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/exists", stdHandler("ok"))

	app := arrow.New()
	app.GET("/exists", func(c *arrow.Context) { c.Write([]byte("ok")) })

	assertParity(t, "404", baseline, app.Handler(), newRequest(http.MethodGet, "/missing", ""))
}

func TestInheritParity405MethodNotAllowed(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/only-get", stdHandler("ok"))

	app := arrow.New()
	app.GET("/only-get", func(c *arrow.Context) { c.Write([]byte("ok")) })

	req := newRequest(http.MethodDelete, "/only-get", "")
	assertParity(t, "405", baseline, app.Handler(), req)
	if r := serveAndCapture(app.Handler(), req); r.Allow == "" {
		t.Fatal("405 response missing Allow header")
	}
}

func TestInheritParityHostRouting(t *testing.T) {
	pattern := "example.com/api"
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, pattern, stdHandler("host-ok"))

	app := arrow.New()
	app.Handle(http.MethodGet, pattern, func(c *arrow.Context) { c.Write([]byte("host-ok")) })

	assertParity(t, "host", baseline, app.Handler(), newRequest(http.MethodGet, "/api", "example.com"))
}

func TestInheritParityWildcardID(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/posts/{id}", stdHandlerPathValue("id", "id="))

	app := arrow.New()
	app.GET("/posts/{id}", arrowHandlerPathValue("id", "id="))

	assertParity(t, "wildcard-id", baseline, app.Handler(), newRequest(http.MethodGet, "/posts/42", ""))
}

func TestInheritParityWildcardEllipsis(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/files/{path...}", stdHandlerPathValue("path", "path="))

	app := arrow.New()
	app.GET("/files/{path...}", arrowHandlerPathValue("path", "path="))

	assertParity(t, "wildcard-ellipsis", baseline, app.Handler(), newRequest(http.MethodGet, "/files/a/b/c", ""))
}

func TestInheritParityDollarEnd(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/posts/{$}", stdHandler("exact"))

	app := arrow.New()
	app.GET("/posts/{$}", func(c *arrow.Context) { c.Write([]byte("exact")) })

	assertParity(t, "dollar-exact", baseline, app.Handler(), newRequest(http.MethodGet, "/posts/", ""))
	assertParity(t, "dollar-no-match", baseline, app.Handler(), newRequest(http.MethodGet, "/posts/1", ""))
}

func TestInheritParityTrailingSlashSubtree(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/files/", stdHandler("subtree"))

	app := arrow.New()
	app.GET("/files/", func(c *arrow.Context) { c.Write([]byte("subtree")) })

	assertParity(t, "trailing-slash", baseline, app.Handler(), newRequest(http.MethodGet, "/files/deep/nested", ""))
}

func TestInheritParityRoutePrecedence(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/posts/latest", stdHandler("latest"))
	registerBaselineHandler(baseline, http.MethodGet, "/posts/{id}", stdHandlerPathValue("id", "id="))

	app := arrow.New()
	app.GET("/posts/latest", func(c *arrow.Context) { c.Write([]byte("latest")) })
	app.GET("/posts/{id}", arrowHandlerPathValue("id", "id="))

	assertParity(t, "precedence-specific", baseline, app.Handler(), newRequest(http.MethodGet, "/posts/latest", ""))
	assertParity(t, "precedence-wildcard", baseline, app.Handler(), newRequest(http.MethodGet, "/posts/99", ""))
}

// --- ResponseWriter delegation parity ---

func TestInheritParityRWInterfacesRecorder(t *testing.T) {
	var baseIF, arrowIF rwInterfaces

	baseH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseIF = probeRW(w)
	})

	app := arrow.New()
	app.GET("/probe", func(c *arrow.Context) {
		arrowIF = probeRW(c.Writer)
	})

	baseH.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	app.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/probe", nil))

	assertRWParity(t, "recorder", baseIF, arrowIF)
}

func TestInheritParityRWInterfacesServer(t *testing.T) {
	probeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, probeRW(w).String())
	})

	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/probe", probeHandler)

	app := arrow.New()
	app.GET("/probe", func(c *arrow.Context) {
		io.WriteString(c.Writer, probeRW(c.Writer).String())
	})

	srvBase := httptest.NewServer(baseline)
	defer srvBase.Close()
	srvApp := httptest.NewServer(app.Handler())
	defer srvApp.Close()

	for name, url := range map[string]string{"baseline": srvBase.URL, "arrow": srvApp.URL} {
		resp, err := http.Get(url + "/probe")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status=%d", name, resp.StatusCode)
		}
		if name == "baseline" {
			// store for parity compare below
			_ = body
		}
	}

	baseResp, _ := http.Get(srvBase.URL + "/probe")
	baseBody, _ := io.ReadAll(baseResp.Body)
	baseResp.Body.Close()

	arrowResp, _ := http.Get(srvApp.URL + "/probe")
	arrowBody, _ := io.ReadAll(arrowResp.Body)
	arrowResp.Body.Close()

	if string(baseBody) != string(arrowBody) {
		t.Fatalf("server interface set: baseline=%s arrow=%s", baseBody, arrowBody)
	}
}

func TestInheritParityFlusher(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		f.Flush()
		w.Write([]byte("flushed"))
	})

	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/flush", handler)

	app := arrow.New()
	app.GET("/flush", func(c *arrow.Context) {
		f, ok := c.Writer.(http.Flusher)
		if !ok {
			c.Abort(http.StatusInternalServerError)
			return
		}
		f.Flush()
		c.Write([]byte("flushed"))
	})

	assertParity(t, "flush-recorder", baseline, app.Handler(), newRequest(http.MethodGet, "/flush", ""))

	srvBase := httptest.NewServer(baseline)
	defer srvBase.Close()
	srvApp := httptest.NewServer(app.Handler())
	defer srvApp.Close()

	for name, url := range map[string]string{"baseline": srvBase.URL, "arrow": srvApp.URL} {
		resp, err := http.Get(url + "/flush")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "flushed" {
			t.Fatalf("%s: status=%d body=%q", name, resp.StatusCode, body)
		}
	}
}

func TestInheritParityHijacker(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Hijacker); !ok {
			http.Error(w, "no hijacker", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("hijack-ok"))
	})

	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/hijack", handler)

	app := arrow.New()
	app.GET("/hijack", func(c *arrow.Context) {
		if _, ok := c.Writer.(http.Hijacker); !ok {
			http.Error(c.Writer, "no hijacker", http.StatusInternalServerError)
			return
		}
		c.Write([]byte("hijack-ok"))
	})

	// recorder: neither supports Hijacker — both return 500
	assertParity(t, "hijack-recorder", baseline, app.Handler(), newRequest(http.MethodGet, "/hijack", ""))

	srvBase := httptest.NewServer(baseline)
	defer srvBase.Close()
	srvApp := httptest.NewServer(app.Handler())
	defer srvApp.Close()

	for name, url := range map[string]string{"baseline": srvBase.URL, "arrow": srvApp.URL} {
		resp, err := http.Get(url + "/hijack")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "hijack-ok" {
			t.Fatalf("%s: status=%d body=%q", name, resp.StatusCode, body)
		}
	}
}

func TestInheritParityResponseControllerFlush(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		if err := rc.Flush(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("rc-ok"))
	})

	baseline := http.NewServeMux()
	registerBaselineHandler(baseline, http.MethodGet, "/rc", handler)

	app := arrow.New()
	app.GET("/rc", func(c *arrow.Context) {
		rc := http.NewResponseController(c.Writer)
		if err := rc.Flush(); err != nil {
			c.Abort(http.StatusInternalServerError)
			return
		}
		c.Write([]byte("rc-ok"))
	})

	assertParity(t, "rc-recorder", baseline, app.Handler(), newRequest(http.MethodGet, "/rc", ""))

	srvBase := httptest.NewServer(baseline)
	defer srvBase.Close()
	srvApp := httptest.NewServer(app.Handler())
	defer srvApp.Close()

	for name, url := range map[string]string{"baseline": srvBase.URL, "arrow": srvApp.URL} {
		resp, err := http.Get(url + "/rc")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "rc-ok" {
			t.Fatalf("%s: status=%d body=%q", name, resp.StatusCode, body)
		}
	}
}

func TestInheritParityReaderFromFileServer(t *testing.T) {
	static := fstest.MapFS{"data.txt": &fstest.MapFile{Data: []byte("payload")}}

	var baseIF, arrowIF rwInterfaces
	probeBase := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseIF = probeRW(w)
	})
	probeApp := arrow.New()
	probeApp.GET("/w", func(c *arrow.Context) {
		arrowIF = probeRW(c.Writer)
	})
	probeBase.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	probeApp.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/w", nil))
	assertRWParity(t, "fileserver-recorder-rf", baseIF, arrowIF)

	baseline := http.NewServeMux()
	registerBaselineHTTP(baseline, "", "/static/", http.FileServer(http.FS(static)))

	app := arrow.New()
	app.HandleHTTP("/static/", http.FileServer(http.FS(static)))

	assertParity(t, "fileserver-body", baseline, app.Handler(), newRequest(http.MethodGet, "/static/data.txt", ""))
}

// --- Handler mounting parity ---

func TestInheritParityHandleHTTPFileServer(t *testing.T) {
	static := fstest.MapFS{"hello.txt": &fstest.MapFile{Data: []byte("file-content")}}

	baseline := http.NewServeMux()
	registerBaselineHTTP(baseline, "", "/static/", http.FileServer(http.FS(static)))

	app := arrow.New()
	app.HandleHTTP("/static/", http.FileServer(http.FS(static)))

	assertParity(t, "fileserver", baseline, app.Handler(), newRequest(http.MethodGet, "/static/hello.txt", ""))
}

func TestInheritParityHandleHTTPMethod(t *testing.T) {
	baseline := http.NewServeMux()
	registerBaselineHTTP(baseline, http.MethodGet, "/raw", stdHandler("raw"))

	app := arrow.New()
	app.HandleHTTPMethod(http.MethodGet, "/raw", stdHandler("raw"))

	assertParity(t, "handle-http-method-get", baseline, app.Handler(), newRequest(http.MethodGet, "/raw", ""))
	assertParity(t, "handle-http-method-405", baseline, app.Handler(), newRequest(http.MethodPost, "/raw", ""))
}

func TestInheritParityMuxDirectRegistration(t *testing.T) {
	baseline := http.NewServeMux()
	baseline.HandleFunc("GET /direct", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "direct")
	})

	app := arrow.New()
	app.Mux().HandleFunc("GET /direct", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "direct")
	})

	assertParity(t, "mux-direct", baseline, app.Handler(), newRequest(http.MethodGet, "/direct", ""))
}