package cli

// Tests for how command-line arguments and their combinations affect the
// assembled HTTP request: the method, headers, body, and URL. These focus on
// collisions and side-effects (e.g. an explicit -X overriding an auto-selected
// method, a custom -H suppressing an auto Content-Type, -G rewriting the body
// into the query string) and run without any external network by driving
// BuildHttpRequest directly and, for redirect/retry behavior, an httptest server.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	"github.com/stretchr/testify/assert"
)

// setupCtx parses args and runs the same setup main() does, failing the test on error.
func setupCtx(t *testing.T, args ...string) *curl.CurlContext {
	t.Helper()
	ctx := new(curl.CurlContext)
	extra, err := ParseFlags(args, ctx)
	if err != nil {
		t.Fatalf("ParseFlags(%v): %v", args, err)
	}
	if cerr := ctx.SetupContextForRun(extra); cerr != nil {
		t.Fatalf("SetupContextForRun(%v): %v", args, cerr)
	}
	return ctx
}

// buildReq builds the request for the first URL in the context.
func buildReq(t *testing.T, ctx *curl.CurlContext) *http.Request {
	t.Helper()
	if len(ctx.Urls) == 0 {
		t.Fatal("no URL was parsed from args")
	}
	req, cerr := ctx.BuildHttpRequest(ctx.Urls[0], 0, true, true)
	if cerr != nil {
		t.Fatalf("BuildHttpRequest: %v", cerr)
	}
	return req
}

func reqBody(t *testing.T, req *http.Request) string {
	t.Helper()
	if req.Body == nil {
		return ""
	}
	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(b)
}

// --- method side-effects: data/form/upload args auto-select a method, but an explicit -X wins ---

func Test_ArgCombo_Method_AutoSelected(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "payload.xyzzy") // unknown extension -> octet-stream
	if err := os.WriteFile(tmp, []byte("raw bytes"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		args       []string
		wantMethod string
	}{
		{"plain GET", []string{"http://localhost/"}, "GET"},
		{"-d implies POST", []string{"-d", "a=b", "http://localhost/"}, "POST"},
		{"-F implies POST", []string{"-F", "a=b", "http://localhost/"}, "POST"},
		{"--json implies POST", []string{"--json", `{"a":1}`, "http://localhost/"}, "POST"},
		{"-T implies PUT", []string{"-T", tmp, "http://localhost/"}, "PUT"},
		{"-I implies HEAD", []string{"-I", "http://localhost/"}, "HEAD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := setupCtx(t, tc.args...)
			req := buildReq(t, ctx)
			assert.Equal(t, tc.wantMethod, req.Method)
		})
	}
}

func Test_ArgCombo_ExplicitMethodOverridesAutoSelect(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "payload.xyzzy")
	if err := os.WriteFile(tmp, []byte("raw bytes"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		args       []string
		wantMethod string
	}{
		{"-X DELETE beats -d POST", []string{"-X", "DELETE", "-d", "a=b", "http://localhost/"}, "DELETE"},
		{"-X POST beats -T PUT", []string{"-X", "POST", "-T", tmp, "http://localhost/"}, "POST"},
		{"-X PATCH beats -F POST", []string{"-X", "PATCH", "-F", "a=b", "http://localhost/"}, "PATCH"},
		{"-X GET beats -I HEAD", []string{"-X", "GET", "-I", "http://localhost/"}, "GET"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := setupCtx(t, tc.args...)
			req := buildReq(t, ctx)
			assert.Equal(t, tc.wantMethod, req.Method)
		})
	}
}

// --- auto Content-Type / Accept, and custom -H suppressing them ---

func Test_ArgCombo_AutoContentType(t *testing.T) {
	t.Run("-d sets urlencoded", func(t *testing.T) {
		ctx := setupCtx(t, "-d", "a=b", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
		assert.Equal(t, "*/*", req.Header.Get("Accept"))
	})
	t.Run("--json sets json content-type and accept", func(t *testing.T) {
		ctx := setupCtx(t, "--json", `{"a":1}`, "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", req.Header.Get("Accept"))
	})
	t.Run("-F sets multipart with boundary", func(t *testing.T) {
		ctx := setupCtx(t, "-F", "a=b", "http://localhost/")
		req := buildReq(t, ctx)
		assert.True(t, strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data; boundary="),
			"got %q", req.Header.Get("Content-Type"))
	})
	t.Run("-T unknown ext falls back to octet-stream", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "payload.xyzzy")
		if err := os.WriteFile(tmp, []byte("raw"), 0600); err != nil {
			t.Fatal(err)
		}
		ctx := setupCtx(t, "-T", tmp, "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "application/octet-stream", req.Header.Get("Content-Type"))
	})
}

func Test_ArgCombo_CustomHeaderSuppressesAutoContentType(t *testing.T) {
	t.Run("custom Content-Type wins over -d default", func(t *testing.T) {
		ctx := setupCtx(t, "-d", "a=b", "-H", "Content-Type: application/vnd.custom+xml", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "application/vnd.custom+xml", req.Header.Get("Content-Type"))
	})
	t.Run("custom Content-Type wins over --json but Accept stays json", func(t *testing.T) {
		ctx := setupCtx(t, "--json", `{"a":1}`, "-H", "Content-Type: application/vnd.custom+json", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "application/vnd.custom+json", req.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", req.Header.Get("Accept"))
	})
	t.Run("custom Accept wins over default */*", func(t *testing.T) {
		ctx := setupCtx(t, "-H", "Accept: text/plain", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "text/plain", req.Header.Get("Accept"))
	})
}

// --- body assembly: multiple values, comma-safety (StringArray, not StringSlice), url-encoding ---

func Test_ArgCombo_DataBodyAssembly(t *testing.T) {
	t.Run("multiple -d joined with &", func(t *testing.T) {
		ctx := setupCtx(t, "-d", "a=1", "-d", "b=2", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "a=1&b=2", reqBody(t, req))
	})
	t.Run("comma in -d value is NOT split", func(t *testing.T) {
		// regression guard for the StringArray (vs StringSlice) fix
		ctx := setupCtx(t, "-d", "a=1,b=2", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "a=1,b=2", reqBody(t, req))
	})
	t.Run("--data-urlencode encodes spaces", func(t *testing.T) {
		ctx := setupCtx(t, "--data-urlencode", "a=b c", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "a=b+c", reqBody(t, req))
	})
}

// --- -G / --get: data is moved into the query string, body is dropped, no Content-Type ---

func Test_ArgCombo_GetConvertsDataToQuery(t *testing.T) {
	t.Run("no existing query", func(t *testing.T) {
		ctx := setupCtx(t, "-G", "-d", "a=b", "http://localhost/path")
		req := buildReq(t, ctx)
		assert.Equal(t, "GET", req.Method)
		assert.Nil(t, req.Body)
		assert.Equal(t, "a=b", req.URL.RawQuery)
		assert.Empty(t, req.Header.Get("Content-Type"))
	})
	t.Run("appends to existing query with &", func(t *testing.T) {
		ctx := setupCtx(t, "-G", "-d", "a=b", "http://localhost/path?x=1")
		req := buildReq(t, ctx)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "x=1&a=b", req.URL.RawQuery)
	})
}

// --- header-ish single-value args: -A user-agent, -e referer, -b cookie, --oauth2-bearer ---

func Test_ArgCombo_SimpleHeaderArgs(t *testing.T) {
	t.Run("default user-agent present", func(t *testing.T) {
		ctx := setupCtx(t, "http://localhost/")
		req := buildReq(t, ctx)
		assert.Contains(t, req.Header.Get("User-Agent"), "go-curling/")
	})
	t.Run("-A overrides user-agent", func(t *testing.T) {
		ctx := setupCtx(t, "-A", "my-agent/9", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "my-agent/9", req.Header.Get("User-Agent"))
	})
	t.Run("-e sets referer", func(t *testing.T) {
		ctx := setupCtx(t, "-e", "http://ref.example/", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "http://ref.example/", req.Header.Get("Referer"))
	})
	t.Run("--oauth2-bearer sets Authorization", func(t *testing.T) {
		ctx := setupCtx(t, "--oauth2-bearer", "tok123", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "Bearer tok123", req.Header.Get("Authorization"))
	})
	t.Run("-b adds raw cookie header", func(t *testing.T) {
		ctx := setupCtx(t, "-b", "a=b", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, "a=b", req.Header.Get("Cookie"))
	})
	t.Run("multiple -b add multiple cookies, comma not split", func(t *testing.T) {
		ctx := setupCtx(t, "-b", "a=1", "-b", "b=2,c=3", "http://localhost/")
		req := buildReq(t, ctx)
		assert.Equal(t, []string{"a=1", "b=2,c=3"}, req.Header.Values("Cookie"))
	})
}

// --- basic auth vs bearer collision: -u produces Basic; an explicit Authorization is not clobbered ---

func Test_ArgCombo_AuthCollision(t *testing.T) {
	t.Run("-u sets Basic auth", func(t *testing.T) {
		ctx := setupCtx(t, "-u", "alice:secret", "http://localhost/")
		req := buildReq(t, ctx)
		user, pass, ok := req.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "alice", user)
		assert.Equal(t, "secret", pass)
	})
	t.Run("-u wins over --oauth2-bearer (bearer only fills empty Authorization)", func(t *testing.T) {
		ctx := setupCtx(t, "-u", "alice:secret", "--oauth2-bearer", "tok", "http://localhost/")
		req := buildReq(t, ctx)
		user, _, ok := req.BasicAuth()
		assert.True(t, ok, "Authorization should be Basic, not Bearer")
		assert.Equal(t, "alice", user)
	})
}

// --- output routing: URLs beyond the -o list go to stdout (regression guard for the dead-branch fix) ---

func Test_ArgCombo_ExtraUrlOutputsToStdout(t *testing.T) {
	ctx := setupCtx(t, "-o", "a.out", "-o", "b.out",
		"http://localhost/1", "http://localhost/2", "http://localhost/3")
	_, out0 := ctx.GetNextOutputsFromContext(0)
	_, out1 := ctx.GetNextOutputsFromContext(1)
	_, out2 := ctx.GetNextOutputsFromContext(2) // beyond the -o list
	assert.Equal(t, "a.out", out0)
	assert.Equal(t, "b.out", out1)
	assert.Equal(t, curl.DEFAULT_OUTPUT, out2)
}

// --- redirect method handling via httptest (no external network) ---

// recordingRedirectServer returns a server that redirects /start -> /dest with the
// given status, records "METHOD PATH" for each hit, and 200s on /dest.
func recordingRedirectServer(t *testing.T, status int, hits *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits = append(*hits, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/dest")
			w.WriteHeader(status)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	}))
}

func runToCompletion(t *testing.T, ctx *curl.CurlContext) *curl.CurlResponses {
	t.Helper()
	client, cerr := ctx.BuildClient()
	if cerr != nil {
		t.Fatalf("BuildClient: %v", cerr)
	}
	req := buildReq(t, ctx)
	resp, cerr := ctx.GetCompleteResponse(0, client, req)
	if cerr != nil {
		t.Fatalf("GetCompleteResponse: %v", cerr)
	}
	return resp
}

func Test_ArgCombo_Redirect_303_PostBecomesGet(t *testing.T) {
	var hits []string
	srv := recordingRedirectServer(t, http.StatusSeeOther, &hits)
	defer srv.Close()

	ctx := setupCtx(t, "-L", "-d", "a=b", srv.URL+"/start")
	resp := runToCompletion(t, ctx)

	assert.Equal(t, []string{"POST /start", "GET /dest"}, hits)
	last := resp.Responses[len(resp.Responses)-1]
	assert.Equal(t, http.StatusOK, last.HttpResponse.StatusCode)
}

func Test_ArgCombo_Redirect_302_PostBecomesGetByDefault(t *testing.T) {
	var hits []string
	srv := recordingRedirectServer(t, http.StatusFound, &hits)
	defer srv.Close()

	ctx := setupCtx(t, "-L", "-d", "a=b", srv.URL+"/start")
	_ = runToCompletion(t, ctx)

	assert.Equal(t, []string{"POST /start", "GET /dest"}, hits)
}

func Test_ArgCombo_Redirect_301_Post301RetainsPost(t *testing.T) {
	var hits []string
	srv := recordingRedirectServer(t, http.StatusMovedPermanently, &hits)
	defer srv.Close()

	ctx := setupCtx(t, "-L", "--post301", "-d", "a=b", srv.URL+"/start")
	_ = runToCompletion(t, ctx)

	// with --post301 the method (and body) are retained across the redirect
	assert.Equal(t, []string{"POST /start", "POST /dest"}, hits)
}

func Test_ArgCombo_NoLocationFlag_DoesNotFollow(t *testing.T) {
	var hits []string
	srv := recordingRedirectServer(t, http.StatusFound, &hits)
	defer srv.Close()

	ctx := setupCtx(t, "-d", "a=b", srv.URL+"/start") // no -L
	_ = runToCompletion(t, ctx)

	assert.Equal(t, []string{"POST /start"}, hits, "should not follow the redirect without -L")
}

// --- retry via httptest: exercises the newly-wired --retry flag ---

func Test_ArgCombo_RetryOnTransientStatus(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503, transient
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx := setupCtx(t, "--retry", "2", "--retry-delay", "0", srv.URL+"/x")
	resp := runToCompletion(t, ctx)

	assert.Equal(t, 3, hits, "1 initial attempt + 2 retries")
	last := resp.Responses[len(resp.Responses)-1]
	assert.Equal(t, http.StatusOK, last.HttpResponse.StatusCode)
}

func Test_ArgCombo_NoRetryByDefault(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := setupCtx(t, srv.URL+"/x") // no --retry
	_ = runToCompletion(t, ctx)

	assert.Equal(t, 1, hits, "without --retry a transient error is not retried")
}
