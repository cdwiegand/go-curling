package cli

// Tests for the networking-related flags: --range, -O/-J, -4/-6, --resolve,
// --max-time/--connect-timeout, and --http1.0/--http1.1. Functional cases use
// an httptest server (no external network).

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/stretchr/testify/assert"
)

// parseSetup runs ParseFlags + SetupContextForRun and returns any error (for
// the validation cases, which setupCtx would turn into a fatal).
func parseSetup(args ...string) (*curl.CurlContext, *curlerrors.CurlError) {
	ctx := new(curl.CurlContext)
	extra, err := ParseFlags(args, ctx)
	if err != nil {
		return ctx, err
	}
	return ctx, ctx.SetupContextForRun(extra)
}

func Test_Range_SetsHeader(t *testing.T) {
	ctx := setupCtx(t, "-r", "0-499", "http://localhost/")
	req := buildReq(t, ctx)
	assert.Equal(t, "bytes=0-499", req.Header.Get("Range"))
}

func Test_Range_CustomHeaderWins(t *testing.T) {
	ctx := setupCtx(t, "-r", "0-499", "-H", "Range: bytes=10-20", "http://localhost/")
	req := buildReq(t, ctx)
	assert.Equal(t, "bytes=10-20", req.Header.Get("Range"))
}

func Test_RemoteName_SetsOutputFromURL(t *testing.T) {
	ctx := setupCtx(t, "-O", "http://example.com/path/archive.tar.gz")
	assert.Equal(t, []string{"archive.tar.gz"}, ctx.BodyOutput)
}

func Test_RemoteName_ErrorsWithoutFilename(t *testing.T) {
	_, cerr := parseSetup("-O", "http://example.com/")
	assert.NotNil(t, cerr)
	assert.Equal(t, curlerrors.ERROR_INVALID_ARGS, cerr.ExitCode)
}

func Test_IPVersion_MutuallyExclusive(t *testing.T) {
	_, cerr := parseSetup("-4", "-6", "http://localhost/")
	assert.NotNil(t, cerr)
	assert.Equal(t, curlerrors.ERROR_INVALID_ARGS, cerr.ExitCode)
}

func Test_HttpVersion_MutuallyExclusive(t *testing.T) {
	_, cerr := parseSetup("--http1.1", "--http2", "http://localhost/")
	assert.NotNil(t, cerr)
	assert.Equal(t, curlerrors.ERROR_INVALID_ARGS, cerr.ExitCode)

	_, cerr = parseSetup("--http1.0", "--http1.1", "http://localhost/")
	assert.NotNil(t, cerr)
	assert.Equal(t, curlerrors.ERROR_INVALID_ARGS, cerr.ExitCode)
}

func Test_Resolve_DialsMappedAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	port := mustPort(t, srv.URL)

	// request a name that has no DNS entry, pinned to the test server's loopback address
	ctx := setupCtx(t, "--resolve", "pinned.test:"+port+":127.0.0.1", "http://pinned.test:"+port+"/")
	resp := runToCompletion(t, ctx)
	last := resp.Responses[len(resp.Responses)-1]
	assert.Equal(t, 200, last.HttpResponse.StatusCode)
}

func Test_IPv4_WorksAgainstLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()
	ctx := setupCtx(t, "-4", srv.URL+"/")
	resp := runToCompletion(t, ctx)
	last := resp.Responses[len(resp.Responses)-1]
	assert.Equal(t, 204, last.HttpResponse.StatusCode)
}

func Test_Http11_RequestSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("hi"))
	}))
	defer srv.Close()
	ctx := setupCtx(t, "--http1.1", srv.URL+"/")
	resp := runToCompletion(t, ctx)
	last := resp.Responses[len(resp.Responses)-1]
	assert.Equal(t, 200, last.HttpResponse.StatusCode)
}

func Test_MaxTime_TimesOutSlowServer(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // never responds until the test ends
	}))
	defer srv.Close()
	defer close(block)

	ctx := setupCtx(t, "-m", "0.3", srv.URL+"/slow")
	client, cerr := ctx.BuildClient()
	assert.Nil(t, cerr)
	req := buildReq(t, ctx)
	_, cerr = ctx.GetCompleteResponse(0, client, req)
	assert.NotNil(t, cerr, "should time out")
	assert.Equal(t, curlerrors.ERROR_NO_RESPONSE, cerr.ExitCode)
}

func mustPort(t *testing.T, rawurl string) string {
	t.Helper()
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("parse %s: %v", rawurl, err)
	}
	return u.Port()
}
