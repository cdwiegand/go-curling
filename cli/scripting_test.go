package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScriptingFlags(t *testing.T) {
	ctx := setupCtx(t, "--retry-connrefused", "--retry-max-time", "3.5", "--proxy", "", "--noproxy", "*", "--unix-socket", "/tmp/service.sock", "--create-dirs", "--remove-on-error", "--max-filesize", "1000", "--cacert", "first.pem", "--cacert", "second.pem", "--capath", "/certs", "http://localhost")
	require.True(t, ctx.RetryConnRefused)
	require.Equal(t, 3.5, ctx.RetryMaxTime)
	require.True(t, ctx.ProxySet)
	require.Empty(t, ctx.Proxy)
	require.True(t, ctx.NoProxySet)
	require.Equal(t, "*", ctx.NoProxy)
	require.Equal(t, "/tmp/service.sock", ctx.UnixSocket)
	require.True(t, ctx.CreateDirs)
	require.True(t, ctx.RemoveOnError)
	require.EqualValues(t, 1000, ctx.MaxFilesize)
	require.Equal(t, []string{"first.pem", "second.pem"}, ctx.CaCertFile)
	require.Equal(t, "/certs", ctx.CaCertPath)
	ctx = setupCtx(t, "--ca-cert", "old.pem", "--ca-path", "/old", "http://localhost")
	require.Equal(t, []string{"old.pem"}, ctx.CaCertFile)
	require.Equal(t, "/old", ctx.CaCertPath)
	ctx = setupCtx(t, "--ca-cert", "first.pem", "--cacert", "second.pem", "http://localhost")
	require.Equal(t, []string{"first.pem", "second.pem"}, ctx.CaCertFile)
}

func TestInvalidScriptingArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--retry", "-1"}, {"--retry-delay", "-1"}, {"--retry-max-time", "-1"}, {"--max-redirs", "-2"},
		{"--max-time", "NaN"}, {"--max-time", "Inf"}, {"--max-time", "-1"}, {"--max-time", "1e20"}, {"--connect-timeout", "-1"},
		{"--max-filesize", "-1"}, {"--fail", "--fail-with-body"}, {"--config"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			_, err := parseSetup(args...)
			require.NotNil(t, err)
			require.Equal(t, 2, err.ExitCode)
		})
	}
}

func TestProxyOverridesAndBypass(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "proxied:%s", r.URL.Host)
	}))
	defer proxy.Close()
	// An explicit proxy must override an inherited environment proxy.
	t.Setenv("HTTP_PROXY", "http://bad-proxy.invalid:1")
	t.Setenv("HTTPS_PROXY", "http://bad-proxy.invalid:1")
	t.Setenv("NO_PROXY", "*")
	ctx := setupCtx(t, "-x", proxy.URL, "--noproxy", "", "http://service.test/data")
	client, cerr := ctx.BuildClient()
	require.Nil(t, cerr)
	defer client.CloseIdleConnections()
	req := buildReq(t, ctx)
	resp, err := client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, "proxied:service.test", string(body))
	ctx = setupCtx(t, "--proxy", proxy.URL, "--noproxy", "", "http://127.0.0.1:1")
	client, cerr = ctx.BuildClient()
	require.Nil(t, cerr)
	resp, err = client.Do(buildReq(t, ctx))
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	client.CloseIdleConnections()
	require.NoError(t, err)
	require.Equal(t, "proxied:127.0.0.1:1", string(body))
	for _, bypass := range []string{"service.test", ".test", "*"} {
		ctx = setupCtx(t, "--proxy", proxy.URL, "--noproxy", bypass, "http://service.test")
		client, cerr = ctx.BuildClient()
		require.Nil(t, cerr)
		p, err := client.Transport.(*http.Transport).Proxy(buildReq(t, ctx))
		require.NoError(t, err)
		require.Nil(t, p)
		client.CloseIdleConnections()
	}
	ctx = setupCtx(t, "--proxy", "", "http://service.test")
	client, cerr = ctx.BuildClient()
	require.Nil(t, cerr)
	defer client.CloseIdleConnections()
	p, err := client.Transport.(*http.Transport).Proxy(buildReq(t, ctx))
	require.NoError(t, err)
	require.Nil(t, p)
}

func TestScriptingFlagsFromConfig(t *testing.T) {
	config := filepath.Join(t.TempDir(), "curl.conf")
	require.NoError(t, os.WriteFile(config, []byte("proxy = \"\"\nnoproxy = \"*\"\nretry-max-time = 2\ncreate-dirs\n"), 0600))
	ctx := setupCtx(t, "--config", config, "http://localhost")
	require.True(t, ctx.ProxySet)
	require.True(t, ctx.NoProxySet)
	require.Equal(t, "*", ctx.NoProxy)
	require.Equal(t, 2.0, ctx.RetryMaxTime)
	require.True(t, ctx.CreateDirs)
}
