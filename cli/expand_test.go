package cli

// Tests for --variable and --expand-<option> handling at the command-line
// level: argument ordering, rewriting of every expand form, skipping of other
// options' values, config files, errors, and the requests that result.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	"github.com/stretchr/testify/require"
)

// parseOnly runs ParseFlags without SetupContextForRun.
func parseOnly(t *testing.T, args ...string) (*curl.CurlContext, []string) {
	t.Helper()
	ctx := new(curl.CurlContext)
	extra, err := ParseFlags(args, ctx)
	require.Nil(t, err, "ParseFlags(%q)", args)
	return ctx, extra
}

// parseErr returns the error ParseFlags reports, failing if there is none.
func parseErr(t *testing.T, args ...string) string {
	t.Helper()
	ctx := new(curl.CurlContext)
	_, err := ParseFlags(args, ctx)
	require.NotNil(t, err, "ParseFlags(%q) should fail", args)
	require.Equal(t, 2, err.ExitCode)
	return err.ErrorString
}

func writeFile(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// --- the README / curl man page example ---

func TestExpand_UrlFromEnvironment(t *testing.T) {
	t.Setenv("USER_ID", "12345")
	ctx := setupCtx(t, "--variable", "%USER_ID", "--expand-url", "https://example.com{{USER_ID}}")
	require.Equal(t, []string{"https://example.com12345"}, ctx.Urls)

	ctx = setupCtx(t, "--variable", "%USER_ID", "--expand-url", "https://example.com/users/{{USER_ID}}")
	require.Equal(t, []string{"https://example.com/users/12345"}, ctx.Urls)
}

func TestExpand_UrlEncodedFromEnvironment(t *testing.T) {
	t.Setenv("USER_ID", "a b/c?d&e=f#g")
	ctx := setupCtx(t, "--variable", "%USER_ID", "--expand-url", "https://example.com/users/{{USER_ID:url}}")
	require.Equal(t, []string{"https://example.com/users/a%20b%2Fc%3Fd%26e%3Df%23g"}, ctx.Urls)
	req := buildReq(t, ctx)
	require.Equal(t, "/users/a%20b%2Fc%3Fd%26e%3Df%23g", req.URL.RequestURI())
}

// --- option syntax forms ---

func TestExpand_EqualsForms(t *testing.T) {
	ctx := setupCtx(t, "--variable=host=localhost", "--expand-url=http://{{host}}/eq")
	require.Equal(t, []string{"http://localhost/eq"}, ctx.Urls)
	require.Equal(t, "localhost", ctx.Variables["host"])
	require.Equal(t, []string{"host=localhost"}, ctx.VariableArgs)

	// an "=" inside the value of the separate-argument form is just content
	ctx = setupCtx(t, "--variable", "q=a=b", "--expand-url", "http://localhost/?{{q}}")
	require.Equal(t, []string{"http://localhost/?a=b"}, ctx.Urls)
}

func TestExpand_WithoutExpandPrefixNothingIsExpanded(t *testing.T) {
	ctx := setupCtx(t, "--variable", "a=1", "-H", "X-Raw: {{a}}", "--url", "http://localhost/{{a}}", "http://localhost/positional/{{a}}")
	require.Equal(t, "{{a}}", ctx.GetHeadersAsDict()["X-Raw"])
	require.Len(t, ctx.Urls, 2)
	for _, u := range ctx.Urls {
		require.Contains(t, u, "%7B%7Ba%7D%7D", "unexpanded braces remain (URL-escaped) in %s", u)
	}
}

func TestExpand_ValueWithoutReferencesIsUnchanged(t *testing.T) {
	ctx := setupCtx(t, "--expand-url", "http://localhost/plain", "--expand-header", "X-A: b")
	require.Equal(t, []string{"http://localhost/plain"}, ctx.Urls)
	require.Equal(t, "b", ctx.GetHeadersAsDict()["X-A"])
}

func TestExpand_ValuesThatLookLikeOptions(t *testing.T) {
	// expanded values that start with "-" must still be treated as values
	ctx := setupCtx(t, "--variable", "v=--insecure", "--expand-data", "{{v}}", "http://localhost/")
	require.Equal(t, []string{"--insecure"}, ctx.Data_Standard)
	require.False(t, ctx.IgnoreBadCerts)

	ctx = setupCtx(t, "--variable", "v=-k", "--expand-header", "{{v}}: x", "http://localhost/")
	require.Equal(t, "x", ctx.GetHeadersAsDict()["-k"])
	require.False(t, ctx.IgnoreBadCerts)
}

// --- ordering: a variable only affects expansions that come after it ---

func TestExpand_OrderMatters(t *testing.T) {
	ctx := setupCtx(t, "--expand-url", "http://localhost/before-{{late}}-after", "--variable", "late=x")
	require.Equal(t, []string{"http://localhost/before--after"}, ctx.Urls)

	ctx = setupCtx(t, "--variable", "a=first", "--expand-header", "X-One: {{a}}", "--variable", "a=second", "--expand-header", "X-Two: {{a}}", "http://localhost/")
	headers := ctx.GetHeadersAsDict()
	require.Equal(t, "first", headers["X-One"])
	require.Equal(t, "second", headers["X-Two"])
	require.Equal(t, "second", ctx.Variables["a"])
}

func TestExpand_ExpandVariableBuildsOnEarlierVariables(t *testing.T) {
	ctx := setupCtx(t,
		"--variable", "host=localhost",
		"--variable", "port=8080",
		"--expand-variable", "base=http://{{host}}:{{port}}",
		"--expand-variable", "full={{base}}/api",
		"--expand-url", "{{full}}/items")
	require.Equal(t, []string{"http://localhost:8080/api/items"}, ctx.Urls)
	require.Equal(t, []string{"host=localhost", "port=8080", "base=http://localhost:8080", "full=http://localhost:8080/api"}, ctx.VariableArgs)
}

func TestExpand_ExpandVariableWithFunctionsAndFile(t *testing.T) {
	file := writeFile(t, "token.txt", "  s3cr3t token \n")
	ctx := setupCtx(t,
		"--variable", "dir="+filepath.Dir(file),
		"--expand-variable", "tok@{{dir}}/token.txt",
		"--expand-header", "Authorization: Bearer {{tok:trim}}",
		"--expand-url", "http://localhost/?t={{tok:trim:url}}")
	require.Equal(t, "Bearer s3cr3t token", ctx.GetHeadersAsDict()["Authorization"])
	require.Equal(t, []string{"http://localhost/?t=s3cr3t%20token"}, ctx.Urls)
}

// --- every kind of option can be expanded ---

func TestExpand_AcrossOptions(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.txt")
	ctx := setupCtx(t,
		"--variable", "v=val",
		"--variable", "out="+out,
		"--expand-header", "X-Var: {{v}}",
		"--expand-user-agent", "agent/{{v}}",
		"--expand-referer", "http://ref/{{v}}",
		"--expand-user", "user:{{v}}",
		"--expand-request", "PATCH",
		"--expand-output", "{{out}}",
		"--variable", "ten= 10 ",
		"--expand-max-time", "{{ten:trim}}",
		"--expand-retry", "{{undefined}}2",
		"--expand-cookie", "c={{v}}",
		"--expand-data", "a={{v}}",
		"--expand-url", "http://localhost/{{v}}")
	require.Equal(t, "val", ctx.GetHeadersAsDict()["X-Var"])
	require.Equal(t, "agent/val", ctx.UserAgent)
	require.Equal(t, "http://ref/val", ctx.Referer)
	require.Equal(t, "user:val", ctx.UserAuth)
	require.Equal(t, "PATCH", ctx.HttpVerb)
	require.Equal(t, []string{out}, ctx.BodyOutput)
	require.EqualValues(t, 10, ctx.MaxTime) // numeric options accept expanded values
	require.Equal(t, 2, ctx.MaxRetries)     // an undefined variable expands to nothing
	require.Equal(t, []string{"c=val"}, ctx.Cookies)
	require.Equal(t, []string{"a=val"}, ctx.Data_Standard)
	require.Equal(t, []string{"http://localhost/val"}, ctx.Urls)
}

func TestExpand_DataVariants(t *testing.T) {
	for _, tc := range []struct {
		option string
		value  string
		check  func(*curl.CurlContext) []string
		want   string
	}{
		{"data", "k={{v}}", func(c *curl.CurlContext) []string { return c.Data_Standard }, "k=a b&c"},
		{"data-ascii", "k={{v}}", func(c *curl.CurlContext) []string { return c.Data_Ascii }, "k=a b&c"},
		{"data-raw", "k={{v:url}}", func(c *curl.CurlContext) []string { return c.Data_RawAsIs }, "k=a%20b%26c"},
		{"data-binary", "{{v:b64}}", func(c *curl.CurlContext) []string { return c.Data_Binary }, "YSBiJmM="},
		{"data-urlencode", "k={{v}}", func(c *curl.CurlContext) []string { return c.Data_Encoded }, "k=a b&c"},
		{"json", `{"k":"{{v:json}}"}`, func(c *curl.CurlContext) []string { return c.Data_Json }, `{"k":"a b&c"}`},
		{"form", "k={{v}}", func(c *curl.CurlContext) []string { return c.Form_Multipart }, "k=a b&c"},
		{"form-string", "k={{v}}", func(c *curl.CurlContext) []string { return c.Form_MultipartRaw }, "k=a b&c"},
	} {
		t.Run(tc.option, func(t *testing.T) {
			ctx := setupCtx(t, "--variable", "v=a b&c", "--expand-"+tc.option, tc.value, "http://localhost/")
			require.Equal(t, []string{tc.want}, tc.check(ctx))
		})
	}
}

func TestExpand_DataFileReferenceBuiltFromVariable(t *testing.T) {
	file := writeFile(t, "body.txt", "from-file")
	ctx := setupCtx(t, "--variable", "f="+file, "--expand-data-binary", "@{{f}}", "http://localhost/")
	require.Equal(t, "from-file", reqBody(t, buildReq(t, ctx)))
}

func TestExpand_JsonBodyIsSafelyQuoted(t *testing.T) {
	t.Setenv("GOCURL_MSG", "he said \"hi\"\n\\ and left")
	ctx := setupCtx(t, "--variable", "%GOCURL_MSG", "--expand-json", `{"msg":"{{GOCURL_MSG:json}}"}`, "http://localhost/")
	body := reqBody(t, buildReq(t, ctx))
	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(body), &parsed), body)
	require.Equal(t, "he said \"hi\"\n\\ and left", parsed["msg"])
}

func TestExpand_AliasedOptionNames(t *testing.T) {
	// --ca-cert is normalized to --cacert; the expand form follows suit
	ctx := setupCtx(t, "--variable", "p=/certs", "--expand-ca-cert", "{{p}}/a.pem", "--expand-ca-path", "{{p}}", "http://localhost/")
	require.Equal(t, []string{"/certs/a.pem"}, ctx.CaCertFile)
	require.Equal(t, "/certs", ctx.CaCertPath)
}

func TestExpand_RepeatableOptionsKeepEveryValue(t *testing.T) {
	ctx := setupCtx(t, "--variable", "n=1", "-H", "X-A: plain", "--expand-header", "X-B: {{n}}", "-H", "X-C: 3", "--expand-url", "http://localhost/{{n}}", "--url", "http://localhost/2")
	require.Equal(t, []string{"X-A: plain", "X-B: 1", "X-C: 3"}, ctx.Headers)
	require.Equal(t, []string{"http://localhost/1", "http://localhost/2"}, ctx.Urls)
}

// --- other options' values are never treated as --variable / --expand-* ---

func TestExpand_ValuesOfOtherOptionsAreSkipped(t *testing.T) {
	ctx, _ := parseOnly(t, "-d", "--variable", "--url", "http://localhost/")
	require.Equal(t, []string{"--variable"}, ctx.Data_Standard)
	require.Empty(t, ctx.Variables)

	ctx, _ = parseOnly(t, "--data", "--expand-url", "--url", "http://localhost/")
	require.Equal(t, []string{"--expand-url"}, ctx.Data_Standard)

	ctx, _ = parseOnly(t, "-H", "--variable", "--url", "http://localhost/")
	require.Equal(t, []string{"--variable"}, ctx.Headers)

	// short option clusters: the last option takes the next argument as its value
	ctx, _ = parseOnly(t, "-sd", "--variable", "--url", "http://localhost/")
	require.True(t, ctx.IsSilent)
	require.Equal(t, []string{"--variable"}, ctx.Data_Standard)
	require.Empty(t, ctx.Variables)

	// boolean clusters do not consume the next argument
	ctx, _ = parseOnly(t, "-sS", "--variable", "a=1", "--expand-url", "http://localhost/{{a}}")
	require.True(t, ctx.IsSilent)
	require.True(t, ctx.ShowErrorEvenIfSilent)
	require.Equal(t, "1", ctx.Variables["a"])
	require.Equal(t, []string{"http://localhost/1"}, ctx.Urls)
}

func TestExpand_DoubleDashStopsProcessing(t *testing.T) {
	ctx, extra := parseOnly(t, "--variable", "a=1", "--", "--expand-url", "--variable")
	require.Equal(t, []string{"--expand-url", "--variable"}, extra)
	require.Equal(t, map[string]string{"a": "1"}, ctx.Variables)
	require.Empty(t, ctx.Urls)
}

// --- errors ---

func TestExpand_Errors(t *testing.T) {
	os.Unsetenv("GOCURL_DEFINITELY_UNSET")
	nul := writeFile(t, "nul.bin", "a\x00b")
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown option", []string{"--expand-nope", "x"}, "Unknown option: --expand-nope"},
		{"boolean option", []string{"--expand-insecure", "x"}, "does not take a value"},
		{"boolean option equals", []string{"--expand-silent=1"}, "does not take a value"},
		{"missing value", []string{"--expand-url"}, "requires a value"},
		{"unknown function", []string{"--variable", "a=1", "--expand-url", "http://localhost/{{a:nope}}"}, "unknown variable function"},
		{"empty function", []string{"--variable", "a=1", "--expand-url", "http://localhost/{{a:}}"}, "unknown variable function"},
		{"null byte", []string{"--variable", "n@" + nul, "--expand-header", "X: {{n}}"}, "null byte"},
		{"unset environment", []string{"--variable", "%GOCURL_DEFINITELY_UNSET"}, "not set"},
		{"missing file", []string{"--variable", "a@" + filepath.Join(t.TempDir(), "nope")}, "unable to read variable file"},
		{"bad range", []string{"--variable", "a[5-1]=abcdef"}, "bad byte range"},
		{"expanded variable error", []string{"--expand-variable", "%{{missing}}GOCURL_DEFINITELY_UNSET"}, "not set"},
		{"error inside expand-variable", []string{"--expand-variable", "a={{x:bad}}"}, "unknown variable function"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Contains(t, parseErr(t, tc.args...), tc.want)
		})
	}
}

func TestExpand_ShortOptionsHaveNoExpandForm(t *testing.T) {
	// curl only offers --expand- for long option names
	msg := parseErr(t, "--expand-H", "X: y")
	require.Contains(t, msg, "Unknown option")
}

func TestExpand_PlainUnknownOptionsStillReportedByParser(t *testing.T) {
	msg := parseErr(t, "--not-an-option", "http://localhost/")
	require.Contains(t, msg, "not-an-option")
}

func TestExpand_BadVariableSyntaxWarnsButContinues(t *testing.T) {
	ctx := setupCtx(t, "--variable", "bad-name=1", "--variable", "novalue", "--variable", "ok=1", "--expand-url", "http://localhost/{{bad}}{{ok}}")
	require.Equal(t, []string{"http://localhost/1"}, ctx.Urls)
	require.Len(t, ctx.Warnings, 2)
	require.Contains(t, ctx.Warnings[0], "bad-name=1")
	require.Contains(t, ctx.Warnings[1], "novalue")
}

func TestExpand_MalformedReferenceWarnsButContinues(t *testing.T) {
	ctx, _ := parseOnly(t, "--variable", "a=1", "--expand-header", "X: {{a b}}{{a}}")
	require.Equal(t, []string{"X: {{a b}}1"}, ctx.Headers)
	require.Len(t, ctx.Warnings, 1)
	require.Contains(t, ctx.Warnings[0], "bad variable name")
}

// --- config files ---

func TestExpand_FromConfigFile(t *testing.T) {
	t.Setenv("GOCURL_CFG_ID", "42")
	config := writeFile(t, "curl.conf", strings.Join([]string{
		`# variables in a config file`,
		`variable = "host=localhost"`,
		`variable %GOCURL_CFG_ID`,
		`--variable path=items`,
		`expand-url = "http://{{host}}/{{path}}/{{GOCURL_CFG_ID}}"`,
		`expand-header = "X-Id: {{GOCURL_CFG_ID:b64}}"`,
	}, "\n"))
	ctx := setupCtx(t, "-K", config)
	require.Equal(t, []string{"http://localhost/items/42"}, ctx.Urls)
	require.Equal(t, "NDI=", ctx.GetHeadersAsDict()["X-Id"])
}

func TestExpand_ConfigFileSeesEarlierCommandLineVariables(t *testing.T) {
	config := writeFile(t, "curl.conf", `expand-url = "http://localhost/{{fromcli}}"`)
	ctx := setupCtx(t, "--variable", "fromcli=yes", "-K", config)
	require.Equal(t, []string{"http://localhost/yes"}, ctx.Urls)

	// ...but not command-line variables defined after the -K
	ctx = setupCtx(t, "-K", config, "--variable", "fromcli=too-late")
	require.Equal(t, []string{"http://localhost/"}, ctx.Urls)
}

func TestExpand_CommandLineSeesConfigFileVariables(t *testing.T) {
	config := writeFile(t, "curl.conf", `variable = "fromcfg=cfg"`)
	ctx := setupCtx(t, "-K", config, "--expand-url", "http://localhost/{{fromcfg}}")
	require.Equal(t, []string{"http://localhost/cfg"}, ctx.Urls)
}

func TestConfigFileOptionsApplyWhereKAppears(t *testing.T) {
	// like curl, later arguments override earlier ones, including those from -K
	config := writeFile(t, "curl.conf", "request POST\n")
	ctx := setupCtx(t, "-K", config, "-X", "PUT", "http://localhost/")
	require.Equal(t, "PUT", ctx.HttpVerb)
	ctx = setupCtx(t, "-X", "PUT", "-K", config, "http://localhost/")
	require.Equal(t, "POST", ctx.HttpVerb)
}

func TestNestedConfigFiles(t *testing.T) {
	inner := writeFile(t, "inner.conf", `variable = "deep=inner"`)
	outer := writeFile(t, "outer.conf", "-K "+inner+"\n"+`expand-url = "http://localhost/{{deep}}"`)
	ctx := setupCtx(t, "-K", outer)
	require.Equal(t, []string{"http://localhost/inner"}, ctx.Urls)
}

// --- end to end against a local server ---

func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var names []string
		for k := range r.Header {
			if strings.HasPrefix(k, "X-") || k == "Authorization" || k == "Content-Type" || k == "Cookie" {
				names = append(names, k)
			}
		}
		sort.Strings(names)
		fmt.Fprintf(w, "%s %s\n", r.Method, r.RequestURI)
		for _, k := range names {
			fmt.Fprintf(w, "%s: %s\n", k, strings.Join(r.Header[k], ", "))
		}
		fmt.Fprintf(w, "\n%s", body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func doRequest(t *testing.T, ctx *curl.CurlContext) string {
	t.Helper()
	client, cerr := ctx.BuildClient()
	require.Nil(t, cerr)
	defer client.CloseIdleConnections()
	resp, err := client.Do(buildReq(t, ctx))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func TestExpand_EndToEndRequest(t *testing.T) {
	srv := echoServer(t)
	t.Setenv("GOCURL_TOKEN", "  abc123  ")
	t.Setenv("USER_ID", "id with/slash")
	got := doRequest(t, setupCtx(t,
		"--variable", "base="+srv.URL,
		"--variable", "%USER_ID",
		"--variable", "%GOCURL_TOKEN",
		"--variable", "note=line1\nline2 \"q\"",
		"--expand-url", "{{base}}/users/{{USER_ID:url}}?n={{note:url}}",
		"--expand-header", "Authorization: Bearer {{GOCURL_TOKEN:trim}}",
		"--expand-header", "X-Encoded: {{GOCURL_TOKEN:trim:b64}}",
		"--expand-json", `{"user":"{{USER_ID:json}}","note":"{{note:json}}"}`,
	))
	require.Equal(t, "POST /users/id%20with%2Fslash?n=line1%0Aline2%20%22q%22\n"+
		"Authorization: Bearer abc123\n"+
		"Content-Type: application/json\n"+
		"X-Encoded: YWJjMTIz\n"+
		"\n"+
		`{"user":"id with/slash","note":"line1\nline2 \"q\""}`, got)
}

func TestExpand_EndToEndGetWithExpandedQueryData(t *testing.T) {
	srv := echoServer(t)
	got := doRequest(t, setupCtx(t,
		"--variable", "term=go & curl",
		"-G",
		"--expand-data", "raw={{term:url}}",
		"--expand-data", "n={{term:b64}}",
		"--expand-url", srv.URL+"/search"))
	require.Equal(t, "GET /search?raw=go%20%26%20curl&n=Z28gJiBjdXJs\n\n", got)

	got = doRequest(t, setupCtx(t,
		"--variable", "term=go & curl",
		"-G",
		"--expand-data-urlencode", "q={{term}}",
		"--expand-url", srv.URL+"/search"))
	require.Equal(t, "GET /search?q=go+%26+curl\n\n", got)
}
