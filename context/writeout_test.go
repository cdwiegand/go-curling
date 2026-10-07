package context

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FormatWriteOut(t *testing.T) {
	vars := map[string]string{
		"http_code":     "200",
		"size_download": "18",
		"content_type":  "text/plain",
	}
	headers := map[string]string{
		"content-type": "text/plain; charset=utf-8",
		"x-custom":     "hi",
	}

	cases := []struct {
		name   string
		format string
		want   string
	}{
		{"plain text", "hello", "hello"},
		{"single var", "%{http_code}", "200"},
		{"var in context", "code=%{http_code} size=%{size_download}", "code=200 size=18"},
		{"case-insensitive var", "%{HTTP_CODE}", "200"},
		{"literal percent", "100%% done", "100% done"},
		{"newline and tab escapes", "a\\tb\\nc", "a\tb\nc"},
		{"carriage return and backslash", "a\\rb\\\\c", "a\rb\\c"},
		{"unknown var renders empty", "x=[%{nope}]", "x=[]"},
		{"header lookup", "%header{Content-Type}", "text/plain; charset=utf-8"},
		{"header lookup case-insensitive", "%header{X-CUSTOM}", "hi"},
		{"unterminated var brace is literal percent", "%{oops", "%{oops"},
		{"trailing backslash is literal", "end\\", "end\\"},
		{"unknown escape kept verbatim", "a\\qb", "a\\qb"},
		{"bare percent kept", "50% off", "50% off"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatWriteOut(tc.format, vars, headers))
		})
	}
}

func Test_normalizeHTTPVersion(t *testing.T) {
	assert.Equal(t, "1", normalizeHTTPVersion("HTTP/1.0"))
	assert.Equal(t, "1.1", normalizeHTTPVersion("HTTP/1.1"))
	assert.Equal(t, "2", normalizeHTTPVersion("HTTP/2.0"))
	assert.Equal(t, "3", normalizeHTTPVersion("HTTP/3.0"))
}

func Test_splitHostPort(t *testing.T) {
	h, p, ok := splitHostPort("127.0.0.1:8080")
	assert.True(t, ok)
	assert.Equal(t, "127.0.0.1", h)
	assert.Equal(t, "8080", p)

	_, _, ok = splitHostPort("")
	assert.False(t, ok)
}

// drives a real (local) transfer through the client and asserts the variables
// --write-out would produce.
func Test_BuildWriteOutData(t *testing.T) {
	body := []byte("hello body content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Thing", "yes")
		w.WriteHeader(201)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	ctx := &CurlContext{BodyOutput: []string{"/dev/null"}}
	assert.Nil(t, ctx.SetupContextForRun(nil)) // sets up the cookie jar the client needs
	client, cerr := ctx.BuildClient()
	assert.Nil(t, cerr)
	req, cerr := ctx.BuildHttpRequest(srv.URL+"/thing", 0, true, true)
	assert.Nil(t, cerr)
	resp, cerr := ctx.GetCompleteResponse(0, client, req)
	assert.Nil(t, cerr)
	// emit populates BodyBytes and the final BodyDone timing
	ctx.EmitResponseToOutputs(0, resp, req)

	vars, headers := ctx.BuildWriteOutData(resp)

	assert.Equal(t, "201", vars["http_code"])
	assert.Equal(t, "201", vars["response_code"])
	assert.Equal(t, "GET", vars["method"])
	assert.Equal(t, "HTTP", vars["scheme"])
	assert.Equal(t, "text/plain; charset=utf-8", vars["content_type"])
	assert.Equal(t, strconv.Itoa(len(body)), vars["size_download"])
	assert.Equal(t, "0", vars["num_redirects"])
	assert.Equal(t, srv.URL+"/thing", vars["url_effective"])
	assert.Equal(t, "1.1", vars["http_version"])
	assert.NotEmpty(t, vars["remote_ip"])
	assert.NotEmpty(t, vars["remote_port"])
	assert.Equal(t, "text/plain; charset=utf-8", headers["content-type"])
	assert.Equal(t, "yes", headers["x-thing"])

	// time_total should parse as a non-negative float
	total, err := strconv.ParseFloat(vars["time_total"], 64)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 0.0)
}

func Test_BuildWriteOutData_CountsRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/dest")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx := &CurlContext{BodyOutput: []string{"/dev/null"}, FollowRedirects: true, MaxRedirects: 10}
	assert.Nil(t, ctx.SetupContextForRun(nil)) // sets up the cookie jar the client needs
	client, cerr := ctx.BuildClient()
	assert.Nil(t, cerr)
	req, cerr := ctx.BuildHttpRequest(srv.URL+"/start", 0, true, true)
	assert.Nil(t, cerr)
	resp, cerr := ctx.GetCompleteResponse(0, client, req)
	assert.Nil(t, cerr)
	ctx.EmitResponseToOutputs(0, resp, req)

	vars, _ := ctx.BuildWriteOutData(resp)
	assert.Equal(t, "1", vars["num_redirects"])
	assert.Equal(t, "200", vars["http_code"])
	assert.Equal(t, srv.URL+"/dest", vars["url_effective"])
	assert.Equal(t, srv.URL+"/start", vars["url"]) // original URL
}

func Test_BuildWriteOutData_NoResponse(t *testing.T) {
	ctx := &CurlContext{}
	vars, headers := ctx.BuildWriteOutData(&CurlResponses{})
	assert.Equal(t, "000", vars["http_code"])
	assert.Equal(t, "0", vars["num_redirects"])
	assert.Empty(t, headers)
}
