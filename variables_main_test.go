package main

// Process-level tests for --variable / --expand-*: real exit codes, stderr
// warnings and stdin, plus a side-by-side comparison with the real curl
// binary (when curl 8.3.0+ is installed) against a local echo server.

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	curltestharness "github.com/cdwiegand/go-curling/tests"
	"github.com/stretchr/testify/require"
)

type procResult struct {
	code           int
	stdout, stderr string
}

func runProcess(t *testing.T, cmd *exec.Cmd, env []string, stdin string) procResult {
	t.Helper()
	cmd.Env = append(os.Environ(), env...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("running %v: %v", cmd.Args, err)
		}
	}
	return procResult{cmd.ProcessState.ExitCode(), stdout.String(), stderr.String()}
}

func runGoCurling(t *testing.T, args []string, env []string, stdin string) procResult {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIHelperProcess$", "--"}, args...)...) // #nosec G204
	return runProcess(t, cmd, append(env, "GO_CURLING_HELPER_PROCESS=1"), stdin)
}

func runRealCurl(t *testing.T, args []string, env []string, stdin string) procResult {
	t.Helper()
	exe := "curl"
	if runtime.GOOS == "windows" {
		exe = "curl.exe"
	}
	return runProcess(t, exec.Command(exe, args...), env, stdin) // #nosec G204
}

// variableEchoServer reports the method, request URI, selected headers and body.
func variableEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var names []string
		for k := range r.Header {
			if strings.HasPrefix(k, "X-") || k == "Authorization" || k == "Content-Type" || k == "Referer" {
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

type variableCase struct {
	name    string
	args    []string // "{{SRV}}" in an argument is replaced with the server URL before running
	env     []string
	stdin   string
	code    int
	stdout  string // expected stdout; "{{SRV}}" is replaced as well
	stderr  string // substring expected on stderr from go-curling ("" means stderr must be empty)
	minCurl curltestharness.VersionInfo
	noCurl  bool // skip the real curl comparison (see the case for why)
}

var curl8_3 = curltestharness.NewVersionInfo(8, 3, 0)
var curl8_12 = curltestharness.NewVersionInfo(8, 12, 0) // byte ranges
var curl8_13 = curltestharness.NewVersionInfo(8, 13, 0) // :64dec

func variableCases(t *testing.T) []variableCase {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token.txt")
	require.NoError(t, os.WriteFile(tokenFile, []byte("  file-token\n"), 0600))
	digitsFile := filepath.Join(dir, "digits.txt")
	require.NoError(t, os.WriteFile(digitsFile, []byte("0123456789"), 0600))
	nulFile := filepath.Join(dir, "nul.bin")
	require.NoError(t, os.WriteFile(nulFile, []byte("a\x00b"), 0600))

	return []variableCase{
		{
			name:   "readme example",
			args:   []string{"-s", "--variable", "%USER_ID", "--expand-url", "{{SRV}}/{{USER_ID}}"},
			env:    []string{"USER_ID=12345"},
			stdout: "GET /12345\n\n",
		},
		{
			name:   "readme example with url function",
			args:   []string{"-s", "--variable", "%USER_ID", "--expand-url", "{{SRV}}/users/{{USER_ID:url}}"},
			env:    []string{"USER_ID=a b/c&d=e"},
			stdout: "GET /users/a%20b%2Fc%26d%3De\n\n",
		},
		{
			name:   "environment default used when unset",
			args:   []string{"-s", "--variable", "%GOCURL_UNSET_VAR_X=fallback", "--expand-url", "{{SRV}}/{{GOCURL_UNSET_VAR_X}}"},
			stdout: "GET /fallback\n\n",
		},
		{
			// the curl man page says the default applies only when the variable
			// is unset; curl 8.6 uses the default even when it is set
			name:   "environment wins over default",
			noCurl: true,
			args:   []string{"-s", "--variable", "%GOCURL_SET_VAR=dflt", "--expand-url", "{{SRV}}/{{GOCURL_SET_VAR}}"},
			env:    []string{"GOCURL_SET_VAR=env"},
			stdout: "GET /env\n\n",
		},
		{
			name:   "unset environment is an error",
			args:   []string{"-s", "--variable", "%GOCURL_UNSET_VAR_Y", "--expand-url", "{{SRV}}/"},
			code:   2,
			stderr: "",
		},
		{
			name:   "unset environment error is shown with -S",
			args:   []string{"-sS", "--variable", "%GOCURL_UNSET_VAR_Y", "--expand-url", "{{SRV}}/"},
			code:   2,
			stderr: "not set",
		},
		{
			// options are applied in order, so a -s after the failing option is too late
			name:   "silent after the error does not hide it",
			args:   []string{"--variable", "%GOCURL_UNSET_VAR_Y", "-s", "--expand-url", "{{SRV}}/"},
			code:   2,
			stderr: "not set",
			noCurl: true, // curl's stderr is not compared
		},
		{
			name:   "stderr redirect before the error is honored",
			args:   []string{"--stderr", "-", "--variable", "%GOCURL_UNSET_VAR_Y", "--expand-url", "{{SRV}}/"},
			code:   2,
			stdout: "Error: Invalid --variable: variable \"GOCURL_UNSET_VAR_Y\" is not set in the environment.\n",
			noCurl: true, // curl words its error differently
		},
		{
			name:   "undefined variable expands to nothing",
			args:   []string{"-s", "--expand-url", "{{SRV}}/a{{nope}}b"},
			stdout: "GET /ab\n\n",
		},
		{
			name:   "later definition wins and order matters",
			args:   []string{"-s", "--variable", "a=1", "--expand-header", "X-First: {{a}}", "--variable", "a=2", "--expand-header", "X-Second: {{a}}", "--expand-url", "{{SRV}}/{{a}}"},
			stdout: "GET /2\nX-First: 1\nX-Second: 2\n\n",
		},
		{
			name:   "variable defined after use is not seen",
			args:   []string{"-s", "--expand-url", "{{SRV}}/x{{late}}y", "--variable", "late=nope"},
			stdout: "GET /xy\n\n",
		},
		{
			name:   "variable from file with trim",
			args:   []string{"-s", "--variable", "tok@" + tokenFile, "--expand-header", "Authorization: Bearer {{tok:trim}}", "--expand-url", "{{SRV}}/"},
			stdout: "GET /\nAuthorization: Bearer file-token\n\n",
		},
		{
			name:   "variable from stdin",
			args:   []string{"-s", "--variable", "body@-", "--expand-data-raw", "got={{body:trim:url}}", "--expand-url", "{{SRV}}/"},
			stdin:  "piped value\n",
			stdout: "POST /\nContent-Type: application/x-www-form-urlencoded\n\ngot=piped%20value",
		},
		{
			name:   "all functions",
			args:   []string{"-s", "--variable", "v= a\"b\\c&d ", "--expand-header", "X-Trim: [{{v:trim}}]", "--expand-header", "X-Url: {{v:url}}", "--expand-header", "X-Json: {{v:json}}", "--expand-header", "X-B64: {{v:b64}}", "--expand-url", "{{SRV}}/"},
			stdout: "GET /\nX-B64: IGEiYlxjJmQg\nX-Json: a\\\"b\\\\c&d\nX-Trim: [a\"b\\c&d]\nX-Url: %20a%22b%5Cc%26d%20\n\n",
		},
		{
			name:   "chained functions",
			args:   []string{"-s", "--variable", "v=  x y  ", "--expand-url", "{{SRV}}/{{v:trim:url}}/{{v:url:trim}}/{{v:trim:b64:url}}"},
			stdout: "GET /x%20y/%20%20x%20y%20%20/eCB5\n\n",
		},
		{
			name:   "json body",
			args:   []string{"-s", "--variable", "msg=line1\nsaid \"hi\"\t\\", "--expand-json", `{"m":"{{msg:json}}"}`, "--expand-url", "{{SRV}}/"},
			stdout: "POST /\nContent-Type: application/json\n\n{\"m\":\"line1\\nsaid \\\"hi\\\"\\t\\\\\"}",
		},
		{
			name:   "expand-variable",
			args:   []string{"-s", "--variable", "a=1", "--expand-variable", "b={{a}}2", "--expand-variable", "c={{b}}3", "--expand-url", "{{SRV}}/{{c}}"},
			stdout: "GET /123\n\n",
		},
		{
			name:   "many option kinds",
			args:   []string{"-s", "--variable", "v=val", "--expand-request", "PUT", "--expand-referer", "http://ref/{{v}}", "--expand-user", "u:{{v}}", "--expand-data-binary", "d={{v}}", "--expand-url", "{{SRV}}/{{v}}"},
			stdout: "PUT /val\nAuthorization: Basic dTp2YWw=\nContent-Type: application/x-www-form-urlencoded\nReferer: http://ref/val\n\nd=val",
		},
		{
			name:   "get with expanded data",
			args:   []string{"-s", "-G", "--variable", "q=a&b", "--expand-data", "q={{q:url}}", "--expand-url", "{{SRV}}/search"},
			stdout: "GET /search?q=a%26b\n\n",
		},
		{
			name:   "write-out format expanded",
			args:   []string{"-s", "--variable", "fmt=%{http_code}", "--expand-write-out", "[code={{fmt}}]", "--expand-url", "{{SRV}}/w"},
			stdout: "GET /w\n\n[code=200]",
		},
		{
			name:   "escaped braces next to a real reference",
			args:   []string{"-s", "--variable", "a=1", "--expand-header", `X-E: q{{a}}\{{a}}`, "--expand-url", "{{SRV}}/"},
			stdout: "GET /\nX-E: q1{{a}}\n\n",
		},
		{
			// curl 8.x leaves a value untouched (backslash included) when its only
			// {{ is escaped; go-curling follows the documented rule and drops it
			name:   "escaped braces alone",
			args:   []string{"-s", "--variable", "a=1", "--expand-header", `X-E: \{{a}}`, "--expand-url", "{{SRV}}/"},
			stdout: "GET /\nX-E: {{a}}\n\n",
			noCurl: true,
		},
		{
			name:   "malformed references kept verbatim",
			args:   []string{"-s", "--variable", "a=1", "--expand-header", "X-M: {{a b}}|{{}}|{{a-b}}|{{a}}|{{a", "--expand-url", "{{SRV}}/"},
			stdout: "GET /\nX-M: {{a b}}|{{}}|{{a-b}}|1|{{a\n\n",
		},
		{
			name:   "malformed references warn on stderr",
			args:   []string{"--variable", "a=1", "--expand-header", "X-M: {{a b}}", "--expand-url", "{{SRV}}/", "-o", "-"},
			stdout: "GET /\nX-M: {{a b}}\n\n",
			stderr: "Warning: bad variable name: a b",
			noCurl: true, // curl's progress meter also writes to stderr
		},
		{
			name:   "bad variable syntax is skipped silently with -s",
			args:   []string{"-s", "--variable", "bad-name=1", "--variable", "novalue", "--expand-url", "{{SRV}}/[{{bad}}{{novalue}}]"},
			stdout: "GET /[]\n\n",
		},
		{
			name:   "bad variable syntax warns without -s",
			args:   []string{"--variable", "bad-name=1", "--expand-url", "{{SRV}}/"},
			stdout: "GET /\n\n",
			stderr: "Warning: Bad --variable syntax, skipping: bad-name=1",
			noCurl: true, // curl's progress meter also writes to stderr
		},
		{
			name: "unknown function is an error",
			args: []string{"-s", "--variable", "a=1", "--expand-url", "{{SRV}}/{{a:nope}}"},
			code: 2,
		},
		{
			name: "empty function is an error",
			args: []string{"-s", "--variable", "a=1", "--expand-url", "{{SRV}}/{{a:}}"},
			code: 2,
		},
		{
			name: "expand on an option without a value is an error",
			args: []string{"-s", "--expand-insecure", "x", "{{SRV}}/"},
			code: 2,
		},
		{
			name: "null byte without encoding is an error",
			args: []string{"-s", "--variable", "n@" + nulFile, "--expand-url", "{{SRV}}/{{n}}"},
			code: 2,
		},
		{
			name:   "null byte with encoding is fine",
			args:   []string{"-s", "--variable", "n@" + nulFile, "--expand-url", "{{SRV}}/{{n:url}}/{{n:b64}}"},
			stdout: "GET /a%00b/YQBi\n\n",
		},
		{
			name:    "byte range from file",
			args:    []string{"-s", "--variable", "d[2-4]@" + digitsFile, "--expand-url", "{{SRV}}/{{d}}"},
			stdout:  "GET /234\n\n",
			minCurl: curl8_12,
		},
		{
			name:    "64dec",
			args:    []string{"-s", "--variable", "e=aGVsbG8=", "--variable", "bad=***", "--expand-url", "{{SRV}}/{{e:64dec}}/{{bad:64dec:url}}"},
			stdout:  "GET /hello/%5B64dec-fail%5D\n\n",
			minCurl: curl8_13,
		},
	}
}

func TestVariablesProcess(t *testing.T) {
	srv := variableEchoServer(t)
	localCurl, verr := curltestharness.GetLocalCurlVersion()
	if verr != nil {
		t.Logf("real curl not available for comparison: %v", verr)
	}
	for _, tc := range variableCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			args := make([]string, len(tc.args))
			for i, a := range tc.args {
				args[i] = strings.ReplaceAll(a, "{{SRV}}", srv.URL)
			}
			want := strings.ReplaceAll(tc.stdout, "{{SRV}}", srv.URL)

			got := runGoCurling(t, args, tc.env, tc.stdin)
			require.Equal(t, tc.code, got.code, "go-curling exit code (stderr: %s)", got.stderr)
			require.Equal(t, want, got.stdout, "go-curling stdout")
			if tc.stderr == "" {
				require.Empty(t, got.stderr, "go-curling stderr")
			} else {
				require.Contains(t, got.stderr, tc.stderr, "go-curling stderr")
			}

			minCurl := tc.minCurl
			if minCurl == (curltestharness.VersionInfo{}) {
				minCurl = curl8_3
			}
			if tc.noCurl || verr != nil || !curltestharness.CompareVersionMinimum(minCurl, localCurl) {
				return
			}
			real := runRealCurl(t, args, tc.env, tc.stdin)
			require.Equal(t, real.code, got.code, "exit code differs from curl %d.%d.%d (curl stderr: %s)", localCurl.Major, localCurl.Minor, localCurl.Patch, real.stderr)
			require.Equal(t, real.stdout, got.stdout, "stdout differs from curl %d.%d.%d", localCurl.Major, localCurl.Minor, localCurl.Patch)
			t.Logf("matches curl %d.%d.%d", localCurl.Major, localCurl.Minor, localCurl.Patch)
		})
	}
}

func TestVariablesProcess_StdinSharedWithData(t *testing.T) {
	// go-curling reads stdin once, so a variable and a -d @- body can both use it
	srv := variableEchoServer(t)
	got := runGoCurling(t, []string{"-s", "--variable", "in@-", "--expand-header", "X-In: {{in:trim}}", "--data-binary", "@-", srv.URL + "/"}, nil, "shared\n")
	require.Equal(t, 0, got.code, got.stderr)
	require.Equal(t, "POST /\nContent-Type: application/x-www-form-urlencoded\nX-In: shared\n\nshared\n", got.stdout)
}

func TestVariablesProcess_ConfigFile(t *testing.T) {
	srv := variableEchoServer(t)
	config := filepath.Join(t.TempDir(), "vars.conf")
	require.NoError(t, os.WriteFile(config, []byte(strings.Join([]string{
		`variable = "base=` + srv.URL + `"`,
		`variable %GOCURL_CFG_USER`,
		`expand-url = "{{base}}/u/{{GOCURL_CFG_USER:url}}"`,
		`expand-header = "X-User: {{GOCURL_CFG_USER}}"`,
		`silent`,
	}, "\n")), 0600))
	env := []string{"GOCURL_CFG_USER=jo smith"}
	want := "GET /u/jo%20smith\nX-User: jo smith\n\n"
	got := runGoCurling(t, []string{"-K", config}, env, "")
	require.Equal(t, 0, got.code, got.stderr)
	require.Equal(t, want, got.stdout)

	if v, err := curltestharness.GetLocalCurlVersion(); err == nil && curltestharness.CompareVersionMinimum(curl8_3, v) {
		real := runRealCurl(t, []string{"-K", config}, env, "")
		require.Equal(t, real.code, got.code, real.stderr)
		require.Equal(t, real.stdout, got.stdout)
	}
}
