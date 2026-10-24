package context

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/stretchr/testify/require"
)

func scriptingContext(t *testing.T) *CurlContext {
	t.Helper()
	ctx := &CurlContext{MaxRedirects: 50, BodyOutput: []string{filepath.Join(t.TempDir(), "body")}, ErrorOutput: "/dev/null"}
	require.Nil(t, ctx.SetupContextForRun(nil))
	return ctx
}

func scriptingTransfer(t *testing.T, ctx *CurlContext, target string) (*CurlResponses, *curlerrors.CurlError) {
	t.Helper()
	client, err := ctx.BuildClient()
	require.Nil(t, err)
	t.Cleanup(client.CloseIdleConnections)
	req, err := ctx.BuildHttpRequest(target, 0, true, true)
	require.Nil(t, err)
	resp, err := ctx.PerformTransfer(0, client, req)
	t.Cleanup(resp.Close)
	return resp, err
}

func TestSilentAndHTTPFailureModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, "diagnostic body")
	}))
	defer srv.Close()
	for _, tc := range []struct {
		name                   string
		silent, fail, withBody bool
		code                   int
		body                   string
	}{
		{"default", false, false, false, 0, "diagnostic body"},
		{"silent", true, false, false, 0, "diagnostic body"},
		{"fail", false, true, false, 22, ""},
		{"silent fail with body", true, false, true, 22, "diagnostic body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := scriptingContext(t)
			ctx.IsSilent, ctx.SilentFail, ctx.FailWithBody = tc.silent, tc.fail, tc.withBody
			require.Nil(t, ctx.SetupContextForRun(nil))
			resp, err := scriptingTransfer(t, ctx, srv.URL)
			vars, _ := ctx.BuildWriteOutData(resp)
			require.Equal(t, strconv.Itoa(tc.code), vars["exitcode"])
			if tc.code == 0 {
				require.Nil(t, err)
				require.Empty(t, vars["errormsg"])
			} else {
				require.NotNil(t, err)
				require.Equal(t, tc.code, err.ExitCode)
				require.NotEmpty(t, vars["errormsg"])
			}
			body, readErr := os.ReadFile(ctx.BodyOutput[0])
			if tc.fail {
				require.True(t, os.IsNotExist(readErr))
			} else {
				require.NoError(t, readErr)
			}
			require.Equal(t, tc.body, string(body))
		})
	}
}

func TestRedirectCredentialBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                string
		user, token, header string
		trusted             bool
	}{
		{"basic", "user:secret", "", "", false},
		{"bearer", "", "secret", "", false},
		{"custom", "", "", "Authorization: custom", false},
		{"trusted", "user:secret", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var foreignHeader http.Header
			foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				foreignHeader = r.Header.Clone()
				fmt.Fprint(w, "done")
			}))
			defer foreign.Close()
			var initialAuth, sameAuth string
			var sameCookie string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.URL.Path == "/start" {
					initialAuth = r.Header.Get("Authorization")
					http.Redirect(w, r, "/same", 302)
				} else {
					sameAuth, sameCookie = r.Header.Get("Authorization"), r.Header.Get("Cookie")
					http.Redirect(w, r, foreign.URL, 302)
				}
			}))
			defer srv.Close()
			ctx := scriptingContext(t)
			ctx.UserAuth, ctx.OAuth2_BearerToken = tc.user, tc.token
			ctx.Headers = []string{"Cookie: explicit=secret"}
			if tc.header != "" {
				ctx.Headers = append(ctx.Headers, tc.header)
			}
			ctx.FollowRedirects, ctx.RedirectsKeepAuthenticationHeaders = true, tc.trusted
			_, err := scriptingTransfer(t, ctx, srv.URL+"/start")
			require.Nil(t, err)
			mu.Lock()
			defer mu.Unlock()
			require.NotEmpty(t, initialAuth)
			require.Equal(t, initialAuth, sameAuth)
			require.Equal(t, "explicit=secret", sameCookie)
			if tc.trusted {
				require.Equal(t, initialAuth, foreignHeader.Get("Authorization"))
				require.Equal(t, sameCookie, foreignHeader.Get("Cookie"))
			} else {
				require.Empty(t, foreignHeader.Get("Authorization"))
				require.Empty(t, foreignHeader.Get("Cookie"))
			}
		})
	}
}

func TestAuthenticationGateAndInvalidRequest(t *testing.T) {
	ctx := &CurlContext{UserAuth: "user:secret", Headers: []string{"Authorization: secret", "Cookie: secret=1"}}
	req, err := ctx.BuildHttpRequest("http://user:password@example.com", 0, false, false)
	require.Nil(t, err)
	require.Empty(t, req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("Cookie"))
	require.Nil(t, req.URL.User)
	ctx.HttpVerb = "INVALID METHOD"
	_, err = ctx.BuildHttpRequest("http://example.com", 0, false, true)
	require.NotNil(t, err)
}

func TestInterruptedDownloadAndCleanup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		rw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 20\r\n\r\npartial")
		rw.Flush()
	}))
	defer srv.Close()
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			ctx := scriptingContext(t)
			ctx.RemoveOnError = remove
			resp, err := scriptingTransfer(t, ctx, srv.URL)
			require.NotNil(t, err)
			require.Equal(t, curlerrors.ERROR_PARTIAL_FILE, err.ExitCode)
			vars, _ := ctx.BuildWriteOutData(resp)
			require.Equal(t, "18", vars["exitcode"])
			require.Equal(t, "7", vars["size_download"])
			body, readErr := os.ReadFile(ctx.BodyOutput[0])
			if remove {
				require.True(t, os.IsNotExist(readErr))
			} else {
				require.NoError(t, readErr)
				require.Equal(t, "partial", string(body))
			}
		})
	}
}

func TestMaxTimeCoversRedirectChainAndBody(t *testing.T) {
	for _, bodyTimeout := range []bool{false, true} {
		t.Run(fmt.Sprint(bodyTimeout), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bodyTimeout {
					fmt.Fprint(w, "start")
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				select {
				case <-time.After(60 * time.Millisecond):
				case <-r.Context().Done():
					return
				}
				if r.URL.Path != "/end" {
					http.Redirect(w, r, "/end", 302)
				} else {
					fmt.Fprint(w, "ok")
				}
			}))
			defer srv.Close()
			ctx := scriptingContext(t)
			ctx.MaxTime, ctx.FollowRedirects = 0.1, true
			_, err := scriptingTransfer(t, ctx, srv.URL)
			require.NotNil(t, err)
			require.Equal(t, curlerrors.ERROR_TIMEOUT, err.ExitCode)
		})
	}
}

func TestRedirectLimitsAndPostReplay(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/loop" {
			http.Redirect(w, r, "/loop", 302)
			return
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", 307)
			return
		}
		body, _ := io.ReadAll(r.Body)
		fmt.Fprintf(w, "%s:%s", r.Method, body)
	}))
	defer srv.Close()
	for _, limit := range []int{0, 1, 2} {
		ctx := scriptingContext(t)
		ctx.FollowRedirects, ctx.MaxRedirects = true, limit
		calls.Store(0)
		resp, err := scriptingTransfer(t, ctx, srv.URL+"/loop")
		require.NotNil(t, err)
		require.Equal(t, curlerrors.ERROR_TOO_MANY_REDIRECTS, err.ExitCode)
		require.EqualValues(t, limit+1, calls.Load())
		require.Equal(t, limit, resp.NumRedirects)
	}
	ctx := scriptingContext(t)
	ctx.FollowRedirects, ctx.Data_RawAsIs = true, []string{"payload"}
	_, err := scriptingTransfer(t, ctx, srv.URL+"/start")
	require.Nil(t, err)
	body, readErr := os.ReadFile(ctx.BodyOutput[0])
	require.NoError(t, readErr)
	require.Equal(t, "POST:payload", string(body))
}

func TestMultipartMissingFile(t *testing.T) {
	for _, prefix := range []string{"@", "<"} {
		ctx := &CurlContext{Form_Multipart: []string{"file=" + prefix + filepath.Join(t.TempDir(), "missing")}}
		_, err := ctx.HandleFormMultipart()
		require.NotNil(t, err)
		require.Equal(t, curlerrors.ERROR_CANNOT_READ_FILE, err.ExitCode)
	}
}

func TestStdinUploadsCanBeReplayed(t *testing.T) {
	for _, kind := range []string{"binary", "json", "upload"} {
		t.Run(kind, func(t *testing.T) {
			input := "{\"message\":\"hello\"}\n"
			f, err := os.CreateTemp(t.TempDir(), "stdin")
			require.NoError(t, err)
			_, err = f.WriteString(input)
			require.NoError(t, err)
			_, err = f.Seek(0, 0)
			require.NoError(t, err)
			original := os.Stdin
			os.Stdin = f
			defer func() { os.Stdin = original; f.Close() }()
			ctx := scriptingContext(t)
			switch kind {
			case "binary":
				ctx.Data_Binary = []string{"@-"}
			case "json":
				ctx.Data_Json = []string{"@-"}
			case "upload":
				ctx.Upload_File = []string{"-"}
			}
			for i := 0; i < 2; i++ {
				req, cerr := ctx.BuildHttpRequest("http://localhost", 0, true, true)
				require.Nil(t, cerr)
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.Equal(t, input, string(body))
				replayed, err := req.GetBody()
				require.NoError(t, err)
				body, err = io.ReadAll(replayed)
				replayed.Close()
				req.Body.Close()
				require.NoError(t, err)
				require.Equal(t, input, string(body))
			}
		})
	}
}

func TestDownloadLimitsAndDirectories(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, limit := range []int64{4, 5, 6} {
			t.Run(fmt.Sprintf("chunked=%v/limit=%d", chunked, limit), func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if chunked {
						w.(http.Flusher).Flush()
					}
					fmt.Fprint(w, "hello")
				}))
				defer srv.Close()
				ctx := scriptingContext(t)
				ctx.BodyOutput = []string{filepath.Join(t.TempDir(), "nested", "body")}
				ctx.CreateDirs, ctx.RemoveOnError, ctx.MaxFilesize = true, true, limit
				_, err := scriptingTransfer(t, ctx, srv.URL)
				body, readErr := os.ReadFile(ctx.BodyOutput[0])
				if limit < 5 {
					require.NotNil(t, err)
					require.Equal(t, curlerrors.ERROR_MAX_FILESIZE, err.ExitCode)
					require.True(t, os.IsNotExist(readErr))
				} else {
					require.Nil(t, err)
					require.NoError(t, readErr)
					require.Equal(t, "hello", string(body))
				}
			})
		}
	}
}

func TestRemoveOnHTTPErrorAndUntouchedFiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); fmt.Fprint(w, "error") }))
	defer srv.Close()
	for _, failWithBody := range []bool{false, true} {
		ctx := scriptingContext(t)
		require.NoError(t, os.WriteFile(ctx.BodyOutput[0], []byte("existing"), 0600))
		ctx.RemoveOnError, ctx.FailWithBody, ctx.SilentFail = true, failWithBody, !failWithBody
		_, err := scriptingTransfer(t, ctx, srv.URL)
		require.NotNil(t, err)
		body, readErr := os.ReadFile(ctx.BodyOutput[0])
		if failWithBody {
			require.True(t, os.IsNotExist(readErr))
		} else {
			require.NoError(t, readErr)
			require.Equal(t, "existing", string(body))
		}
	}
}

type scriptingRoundTripper func(*http.Request) (*http.Response, error)

func (f scriptingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRetriesReplayBodiesAndExcludeFailedResponses(t *testing.T) {
	ctx := scriptingContext(t)
	ctx.MaxRetries = 1
	ctx.Data_RawAsIs = []string{"payload"}
	var calls int
	client := &http.Client{Transport: scriptingRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "payload", string(body))
		code, content := 503, "retry body"
		if calls == 2 {
			code, content = 200, "success"
		}
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(content)), Request: r}, nil
	})}
	req, err := ctx.BuildHttpRequest("http://localhost", 0, true, true)
	require.Nil(t, err)
	resp, err := ctx.GetCompleteResponse(0, client, req)
	require.Nil(t, err)
	defer resp.Close()
	require.Len(t, resp.Responses, 1)
	require.Equal(t, 2, calls)
	errs := ctx.ProcessResponseToOutputs(0, resp, req)
	require.False(t, errs.HasError())
	body, readErr := os.ReadFile(ctx.BodyOutput[0])
	require.NoError(t, readErr)
	require.Equal(t, "success", string(body))
}

func TestRetryConnectionRefusedAndBudget(t *testing.T) {
	for _, tc := range []struct {
		name         string
		refused, all bool
		budget       float64
		calls        int
	}{
		{"default", false, false, 0, 1},
		{"refused", true, false, 0, 2},
		{"all", false, true, 0, 2},
		{"budget", true, false, .1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := scriptingContext(t)
			ctx.MaxRetries, ctx.RetryConnRefused, ctx.RetryAllErrors, ctx.RetryMaxTime = 1, tc.refused, tc.all, tc.budget
			calls := 0
			client := &http.Client{Transport: scriptingRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody, Request: r}, nil
			})}
			req, err := ctx.BuildHttpRequest("http://localhost", 0, true, true)
			require.Nil(t, err)
			resp, err := ctx.GetCompleteResponse(0, client, req)
			defer resp.Close()
			require.Equal(t, tc.calls, calls)
			if calls == 1 {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestRetryAfterAndBackoff(t *testing.T) {
	ctx := &CurlContext{}
	require.Equal(t, time.Second, ctx.retryDelay(0, nil))
	require.Equal(t, 4*time.Second, ctx.retryDelay(2, nil))
	require.Equal(t, 600*time.Second, ctx.retryDelay(20, nil))
	ctx.RetryDelaySeconds = 3
	require.Equal(t, 3*time.Second, ctx.retryDelay(5, nil))
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"8"}}}
	require.Equal(t, 8*time.Second, ctx.retryDelay(0, resp))
	resp.Header.Set("Retry-After", time.Now().Add(20*time.Second).UTC().Format(http.TimeFormat))
	require.Greater(t, ctx.retryDelay(0, resp), 18*time.Second)
}

func TestBodyRetriesReplacePartialFile(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			ctx := scriptingContext(t)
			ctx.MaxRetries, ctx.RetryAllErrors, ctx.RemoveOnError = 1, true, remove
			// Preserve output from an earlier URL when rolling back a retry.
			require.NoError(t, ctx.WriteToFileBytes(ctx.BodyOutput[0], []byte("prefix:")))
			calls := 0
			client := &http.Client{Transport: scriptingRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				body := io.Reader(strings.NewReader("complete"))
				if calls == 1 {
					body = io.MultiReader(strings.NewReader("partial"), brokenDownloadReader{})
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(body), ContentLength: 8, Request: r}, nil
			})}
			req, err := ctx.BuildHttpRequest("http://localhost", 0, true, true)
			require.Nil(t, err)
			resp, err := ctx.PerformTransfer(0, client, req)
			defer resp.Close()
			require.Nil(t, err)
			require.Equal(t, 2, calls)
			body, readErr := os.ReadFile(ctx.BodyOutput[0])
			require.NoError(t, readErr)
			require.Equal(t, "prefix:complete", string(body))
			vars, _ := ctx.BuildWriteOutData(resp)
			require.Equal(t, "0", vars["exitcode"])
			require.Equal(t, "8", vars["size_download"])
		})
	}
}

type brokenDownloadReader struct{}

func (brokenDownloadReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type streamingDownloadReader struct {
	t    *testing.T
	file string
	sent bool
}

func (r *streamingDownloadReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "streamed"), nil
	}
	body, err := os.ReadFile(r.file)
	require.NoError(r.t, err)
	require.Equal(r.t, "streamed", string(body), "first chunk must be written before reading EOF")
	return 0, io.EOF
}

func TestResponseOutputStreamsBeforeEOF(t *testing.T) {
	ctx := scriptingContext(t)
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, ContentLength: -1, Body: io.NopCloser(&streamingDownloadReader{t: t, file: ctx.BodyOutput[0]})}
	n, errs := ctx.EmitSingleHttpResponseToOutputs(0, resp, nil, false)
	require.False(t, errs.HasError())
	require.Equal(t, 8, n)
}

func TestPrivateCADirectoryAndFile(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "trusted") }))
	defer srv.Close()
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600))
	for _, directory := range []bool{false, true} {
		ctx := scriptingContext(t)
		ctx.DoNotUseHostCertificateAuthorities = true
		if directory {
			ctx.CaCertPath = dir
		} else {
			ctx.CaCertFile = []string{certPath}
		}
		_, err := scriptingTransfer(t, ctx, srv.URL)
		require.Nil(t, err)
	}
	for _, path := range []bool{false, true} {
		ctx := scriptingContext(t)
		if path {
			ctx.CaCertPath = filepath.Join(t.TempDir(), "missing")
		} else {
			ctx.CaCertFile = []string{filepath.Join(t.TempDir(), "missing")}
		}
		_, err := ctx.BuildRootCAsPool()
		require.NotNil(t, err)
		require.Equal(t, curlerrors.ERROR_CA_CERT, err.ExitCode)
	}
}

func TestUnixSocketTransfer(t *testing.T) {
	// Use the system temporary directory to avoid Unix socket path length limits.
	dir, err := os.MkdirTemp("", "curl-sock-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", socket)
	if err != nil && runtime.GOOS == "windows" {
		t.Skipf("Unix domain sockets are unavailable on this host: %v", err)
	}
	require.NoError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, r.Host) })}
	defer srv.Close()
	go srv.Serve(listener)
	ctx := scriptingContext(t)
	ctx.UnixSocket, ctx.ProxySet, ctx.Proxy = socket, true, "http://unreachable.invalid:1234"
	_, cerr := scriptingTransfer(t, ctx, "http://service.test/")
	require.Nil(t, cerr)
	body, err := os.ReadFile(ctx.BodyOutput[0])
	require.NoError(t, err)
	require.Equal(t, "service.test", string(body))
}

func TestTransferErrorClassification(t *testing.T) {
	require.Equal(t, 28, transferError(context.DeadlineExceeded).ExitCode)
	require.Equal(t, 18, transferError(io.ErrUnexpectedEOF).ExitCode)
	require.Equal(t, 6, transferError(&net.DNSError{Err: "no such host", Name: "missing.invalid"}).ExitCode)
	require.Equal(t, 7, transferError(errors.New("connection error")).ExitCode)
}

func TestProxyBypassRules(t *testing.T) {
	for _, tc := range []struct {
		host, rules string
		bypass      bool
	}{
		{"service.example.com", "example.com", true}, {"example.com", ".example.com", true},
		{"notexample.com", "example.com", false}, {"EXAMPLE.COM.", "example.com", true},
		{"127.0.0.1", "127.0.0.0/8", true}, {"10.1.2.3", "127.0.0.0/8", false},
		{"::1", "::1", true}, {"2001:db8::1", "2001:db8::/32", true},
		{"localhost", "", false}, {"anything", "*", true},
	} {
		require.Equal(t, tc.bypass, bypassProxy(tc.host, tc.rules), "%s / %s", tc.host, tc.rules)
	}
}
