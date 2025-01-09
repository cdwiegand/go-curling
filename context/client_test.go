package context

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SetAuthenticationHeadersOnRequest(t *testing.T) {

	// request *http.Request) *curlerrors.CurlError
	req := &http.Request{}
	ctx := &CurlContext{}
	ctx.UserAuth = ""
	cerr := ctx.SetAuthenticationHeadersOnRequest(req)
	assert.Nil(t, cerr)
	user, pass, ok := req.BasicAuth()
	assert.Empty(t, user)
	assert.Empty(t, pass)
	assert.False(t, ok)

	req = &http.Request{}
	ctx = &CurlContext{}
	ctx.UserAuth = "hello:world"
	cerr = ctx.SetAuthenticationHeadersOnRequest(req)
	assert.Nil(t, cerr)
	user, pass, ok = req.BasicAuth()
	assert.Equal(t, "hello", user)
	assert.Equal(t, "world", pass)
	assert.True(t, ok)

	req = &http.Request{}
	ctx = &CurlContext{}
	ctx.UserAuth = "empty"
	ctx.SilentFail = true
	cerr = ctx.SetAuthenticationHeadersOnRequest(req)
	assert.NotNil(t, cerr)

	req = &http.Request{}
	ctx = &CurlContext{}
	ctx.OAuth2_BearerToken = "Artwork-Mountain-Underscore1"
	cerr = ctx.SetAuthenticationHeadersOnRequest(req)
	assert.Nil(t, cerr)
	user, pass, ok = req.BasicAuth()
	assert.Empty(t, user)
	assert.Empty(t, pass)
	assert.False(t, ok)
	foundHeader := req.Header.Get("Authorization")
	assert.Equal(t, "Bearer Artwork-Mountain-Underscore1", foundHeader)

	req = &http.Request{}
	ctx = &CurlContext{}
	ctx.OAuth2_BearerToken = "Artwork-Mountain-Underscore1"
	ctx.UserAuth = "hello:world"
	cerr = ctx.SetAuthenticationHeadersOnRequest(req)
	assert.Nil(t, cerr)
	user, pass, ok = req.BasicAuth()
	assert.Equal(t, "hello", user)
	assert.Equal(t, "world", pass)
	assert.True(t, ok)
	foundHeader = req.Header.Get("Authorization")
	assert.NotNil(t, foundHeader)
	assert.Equal(t, "Basic aGVsbG86d29ybGQ=", foundHeader)
}

func Test_DumpResponseHeaders(t *testing.T) {
	resp := &http.Response{}
	resp.Proto = "HTTP/2"
	resp.StatusCode = 418
	resp.Header = make(http.Header)
	resp.Header.Add("X-Hello", "World2")
	resp.Header.Add("Y-Hello", "World3")
	resp.Header.Add("H-Hello", "World1")

	test := DumpResponseHeaders(resp, true)
	test2 := strings.Join(test, "\n")

	assert.True(t, strings.HasPrefix(test2, "HTTP/2 418\n"))
	assert.Equal(t, "< H-Hello: World1", test[1])
	assert.Equal(t, "< X-Hello: World2", test[2])
	assert.Equal(t, "< Y-Hello: World3", test[3])

	resp.Proto = ""
	test = DumpResponseHeaders(resp, true)
	test2 = strings.Join(test, "\n")
	assert.True(t, strings.HasPrefix(test2, "HTTP/? 418\n"))
}

func Test_DumpRequestHeaders(t *testing.T) {
	req := &http.Request{}
	req.Method = "PUT"
	req.URL, _ = url.Parse("https://github.com/cdwiegand/go-curling/")
	req.Header = make(http.Header)
	req.Header.Add("X-Hello", "World2")
	req.Header.Add("Y-Hello", "World3")
	req.Header.Add("H-Hello", "World1")

	test := DumpRequestHeaders(req)
	test2 := strings.Join(test, "\n")

	assert.True(t, strings.HasPrefix(test2, "PUT https://github.com/cdwiegand/go-curling/\n"))
	assert.Equal(t, "> H-Hello: World1", test[1])
	assert.Equal(t, "> X-Hello: World2", test[2])
	assert.Equal(t, "> Y-Hello: World3", test[3])
}

func Test_GetTlsVersionString(t *testing.T) {
	assert.Equal(t, "TLS 1.0", GetTlsVersionString(tls.VersionTLS10))
	assert.Equal(t, "TLS 1.1", GetTlsVersionString(tls.VersionTLS11))
	assert.Equal(t, "TLS 1.2", GetTlsVersionString(tls.VersionTLS12))
	assert.Equal(t, "TLS 1.3", GetTlsVersionString(tls.VersionTLS13))
	assert.Equal(t, "TLS 1.4?", GetTlsVersionString(AssumedVersionTLS14))
	assert.Contains(t, GetTlsVersionString(99), "Unknown TLS version:")
}

func Test_GetTlsVersionValue(t *testing.T) {
	res, err := GetTlsVersionValue("default")
	assert.EqualValues(t, 0, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("")
	assert.EqualValues(t, 0, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("")
	assert.EqualValues(t, 0, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("1.0")
	assert.EqualValues(t, tls.VersionTLS10, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("1.1")
	assert.EqualValues(t, tls.VersionTLS11, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("1.2")
	assert.EqualValues(t, tls.VersionTLS12, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("1.3")
	assert.EqualValues(t, tls.VersionTLS13, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("1.4")
	assert.EqualValues(t, AssumedVersionTLS14, res)
	assert.Nil(t, err)

	res, err = GetTlsVersionValue("nope")
	assert.EqualValues(t, 0, res)
	assert.Error(t, err)
}

func Test_DumpTlsDetails(t *testing.T) {
	conn := &tls.ConnectionState{}

	conn.Version = tls.VersionTLS13
	conn.CipherSuite = tls.TLS_AES_128_GCM_SHA256
	conn.NegotiatedProtocol = "Awesome/1.0"
	conn.ServerName = "www.hello.world.com"

	res := DumpTlsDetails(conn)
	res2 := strings.Join(res, "\n")
	assert.Contains(t, res2, "TLS Version: TLS 1.3")
	assert.Contains(t, res2, "TLS Cipher Suite: TLS_AES_128_GCM_SHA256")
	assert.Contains(t, res2, "TLS Negotiated Protocol: Awesome/1.0")
	assert.Contains(t, res2, "TLS Server Name: www.hello.world.com")

	conn.NegotiatedProtocol = ""
	conn.ServerName = ""
	res = DumpTlsDetails(conn)
	res2 = strings.Join(res, "\n")
	assert.Contains(t, res2, "TLS Version: TLS 1.3")
	assert.Contains(t, res2, "TLS Cipher Suite: TLS_AES_128_GCM_SHA256")
	assert.NotContains(t, res2, "TLS Negotiated Protocol:")
	assert.NotContains(t, res2, "TLS Server Name:")
}
