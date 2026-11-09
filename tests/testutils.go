package curltestharness

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	jsonutil "github.com/cdwiegand/go-curling/jsonutil"
)

type TestRun struct {
	ListOutputFiles           []string
	ListInputFiles            []string
	ContextBuilder            func(*TestRun) *curl.CurlContext
	CmdLineBuilder            func(*TestRun) []string
	CmdLineBuilderCurl        func(*TestRun) []string
	SuccessHandler            func(map[string]interface{}, *TestRun)
	SuccessHandlerIndexed     func(map[string]interface{}, int, *TestRun)
	SuccessHandlerIndexedRaw  func(map[string]interface{}, string, int, *TestRun)
	ErrorHandler              func(*curlerrors.CurlError, *TestRun)
	TempDir                   string
	Testing                   *testing.T
	DoNotTestAgainstCurl      bool
	SkipCompareJsonToRealCurl bool
	Responses                 *curl.CurlResponses
}

func BuildTestRun(t *testing.T) *TestRun {
	ret := new(TestRun)
	ret.TempDir = t.TempDir()
	ret.Testing = t

	// default error handler
	ret.ErrorHandler = func(err *curlerrors.CurlError, testrun *TestRun) {
		GenericTestErrorHandler(t, err)
	}
	return ret
}

func (run *TestRun) GetNextInputFile() (ret string) {
	i := len(run.ListInputFiles)
	ret = filepath.Join(run.TempDir, fmt.Sprintf("%d.in.tmp", i))
	run.ListInputFiles = append(run.ListInputFiles, ret)
	return
}

func (run *TestRun) GetOneOutputFiles() []string {
	// this is for cleaner code in the context tests, which needs an array of output files that is usually just 1 long
	ret := run.GetOneOutputFile()
	return []string{ret}
}
func (run *TestRun) GetOutputFiles(count int) []string {
	i := len(run.ListOutputFiles)
	var ret []string
	for i2 := i + 1; i2 <= count; i2++ {
		file := filepath.Join(run.TempDir, fmt.Sprintf("%d.out.tmp", i2))
		ret = append(ret, file)
		run.ListOutputFiles = append(run.ListOutputFiles, file)
	}
	return ret
}
func (run *TestRun) GetOneOutputFile() (ret string) {
	i := len(run.ListOutputFiles)
	ret = filepath.Join(run.TempDir, fmt.Sprintf("%d.out.tmp", i))
	run.ListOutputFiles = append(run.ListOutputFiles, ret)
	return
}
func (run *TestRun) EnsureAtLeastOneOutputFiles() (ret []string) {
	if len(run.ListOutputFiles) == 0 {
		run.GetOneOutputFile()
	}
	return run.ListOutputFiles
}

func GenericTestErrorHandler(t *testing.T, err *curlerrors.CurlError) {
	t.Errorf("Got error %v", err)
}

func (run *TestRun) GetTestRunReady() (ctx *curl.CurlContext, args []string, cerr *curlerrors.CurlError) {
	var nonFlagArgs []string

	if run.ContextBuilder != nil {
		ctx = run.ContextBuilder(run)
	} else if run.CmdLineBuilder != nil {
		ctx = new(curl.CurlContext)
		args = run.CmdLineBuilder(run)
		nonFlagArgs, cerr = curlcli.ParseFlags(args, ctx)
		if cerr != nil {
			return
		}
	} else {
		run.Testing.Fatal("Forgot to add ContextBuilder or CmdLineBuilder to test!")
	}

	cerr = ctx.SetupContextForRun(nonFlagArgs)
	return
}
func (run *TestRun) RunTestRun() {
	var ctx *curl.CurlContext
	var args []string
	var cerr *curlerrors.CurlError

	ctx, args, cerr = run.GetTestRunReady()
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	client, cerr := ctx.BuildClient()
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	var jsonGot []map[string]interface{}
	var rawJsonsGot []string

	for index := range ctx.Urls {
		resp, request, cerr := run.getCompleteResponseRetryingRateLimits(ctx, client, index)
		if request == nil {
			run.ErrorHandler(cerr, run)
			return
		}
		run.Responses = resp
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}

		cerrs := ctx.ProcessResponseToOutputs(index, resp, request)
		if cerrs.HasError() {
			for _, h := range cerrs.Errors {
				run.ErrorHandler(h, run)
			}
			return
		}

		if index >= len(run.ListOutputFiles) {
			run.ErrorHandler(curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON"), run)
			return
		}

		jsonObj, rawJson, err := ReadJson(run.ListOutputFiles[index])
		jsonGot = append(jsonGot, jsonObj)
		rawJsonsGot = append(rawJsonsGot, rawJson)

		if err != nil {
			run.ErrorHandler(curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON", err), run)
			return
		}

		if run.SuccessHandler != nil {
			run.SuccessHandler(jsonObj, run)
		}
		if run.SuccessHandlerIndexed != nil {
			run.SuccessHandlerIndexed(jsonObj, index, run)
		}
		if run.SuccessHandlerIndexedRaw != nil {
			run.SuccessHandlerIndexedRaw(jsonObj, rawJson, index, run)
		}
	}

	if run.CmdLineBuilder != nil && args != nil && !run.DoNotTestAgainstCurl {
		// test curl cli output compared to us
		var errCurl error
		if run.CmdLineBuilderCurl != nil {
			errCurl = CompareCurlCliOutput(run, run.CmdLineBuilderCurl(run), jsonGot, rawJsonsGot)
		} else {
			errCurl = CompareCurlCliOutput(run, args, jsonGot, rawJsonsGot)
		}
		if errCurl != nil {
			run.ErrorHandler(curlerrors.NewCurlErrorFromError(curlerrors.ERROR_STATUS_CODE_FAILURE, errCurl), run)
			return
		}
	}
}

func ReadJson(file string) (res map[string]interface{}, raw string, err error) {
	jsonFile, err := os.Open(file) // #nosec G304
	if err != nil {
		return nil, "", err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, "", err
	}

	raw = string(byteValue)
	if raw != "" {
		err = json.Unmarshal(byteValue, &res)
	}

	return res, raw, err
}

func CompareCurlCliOutput(run *TestRun, args []string, myJsonObjs []map[string]interface{}, myJsonRaws []string) error {
	outputs, curlerr := run.runCurlRetryingRateLimits(args)
	if curlerr != nil {
		return curlerr
	}

	for i, h := range outputs {
		curlJsonObj, curlJsonRaw, err := ReadJson(h)
		if err != nil {
			return err
		}

		if run.SuccessHandler != nil {
			run.SuccessHandler(curlJsonObj, run)
		}
		if run.SuccessHandlerIndexed != nil {
			run.SuccessHandlerIndexed(curlJsonObj, i, run)
		}
		if run.SuccessHandlerIndexedRaw != nil {
			run.SuccessHandlerIndexedRaw(curlJsonObj, curlJsonRaw, i, run)
		}

		if !run.SkipCompareJsonToRealCurl {
			// compare, with specific differences permitted:
			/*
				{
				  "args": {},
				  "data": "@/tmp/0.in.tmp",
				  "files": {},
				  "form": {},
				  "headers": {
				    "Accept": "application/json",
				    "Accept-Encoding": "gzip",
				    "Content-Length": "90",
				    "Content-Type": "application/json",
				    "Host": "httpbin.org",
				    "User-Agent": "Go-http-client/2.0",                             <----- this can be different
				    "X-Amzn-Trace-Id": "Root=1-664d77a1-75a4de3a1784d8135d044b0f"   <----- this can be different
				  },
				  "json": null,
				  "origin": "73.203.21.18",
				  "url": "https://httpbin.org/post"
				}
			*/

			if !jsonutil.Equal(myJsonObjs[i], curlJsonObj, func(path string) bool {
				if path == "X-Amzn-Trace-Id" || path == "User-Agent" || path == "Content-Type" || path == "Content-Length" {
					return true
				}
				return false
			}) {
				return errors.New("json outputs did not match between curl and go-curling:\n\ngo-curling output:\n" + myJsonRaws[i] + "\n\ncurl output:\n" + curlJsonRaw)
			}
		}
	}
	return nil
}

func (run *TestRun) FixLinuxRunIfWindowsToWslCurlRun(args []string) (cmd *exec.Cmd, inputs []string, outputs []string) {
	// test-only harness: args are built by the test cases (not untrusted input) and we
	// intentionally run the real curl binary to compare its output against go-curling.
	cmd = exec.Command("curl", args...) // #nosec G204
	if runtime.GOOS == "windows" {      // seriously, Microsoft?? Kill the curl powershell "alias"
		cmd = exec.Command("curl.exe", args...) // #nosec G204
	}
	inputs = run.ListInputFiles
	outputs = run.ListOutputFiles
	return
}

func RunCurlExe(args []string) (exitCode int, stdOut bytes.Buffer, stdErr bytes.Buffer, err error) {
	var cmd *exec.Cmd
	// test-only harness: args are built by the test cases (not untrusted input) and we
	// intentionally run the real curl binary to compare its output against go-curling.
	cmd = exec.Command("curl", args...) // #nosec G204
	if runtime.GOOS == "windows" {      // seriously, Microsoft?? Kill the curl powershell "alias"
		cmd = exec.Command("curl.exe", args...) // #nosec G204
	}

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	if err != nil {
		return 0, outb, errb, err
	}
	return cmd.ProcessState.ExitCode(), outb, errb, nil
}

type VersionInfo struct {
	Major int
	Minor int
	Patch int
}

func NewVersionInfo(major int, minor int, patch int) (ret VersionInfo) {
	ret.Major = major
	ret.Minor = minor
	ret.Patch = patch
	return
}

func GetLocalCurlVersion() (version VersionInfo, err error) {
	exitCode, stdout, _, err := RunCurlExe([]string{"--version"})
	if exitCode != 0 {
		return version, fmt.Errorf("curl exit code was failure: %d", exitCode)
	}
	if err != nil {
		return version, err
	}
	if stdout.Len() == 0 {
		return version, errors.New("curl output was empty")
	}

	firstLine := strings.Split(stdout.String(), "\n")[0]
	return ParseLocalCurlVersion(firstLine)
}
func EnsureLocalCurlMinVersion(wantAtLeast VersionInfo) bool {
	gotVersion, _ := GetLocalCurlVersion()
	return CompareVersionMinimum(wantAtLeast, gotVersion)
}
func EnsureLocalCurlMinVersionAndLog(t *testing.T, wantAtLeast VersionInfo) bool {
	gotVersion, _ := GetLocalCurlVersion()
	ret := CompareVersionMinimum(wantAtLeast, gotVersion)
	if !ret {
		t.Logf("local curl version was %d.%d.%d, I wanted at least %d.%d.%d",
			gotVersion.Major, gotVersion.Minor, gotVersion.Patch,
			wantAtLeast.Major, wantAtLeast.Minor, wantAtLeast.Patch)
	}
	return ret
}
func CompareVersionMinimum(wantAtLeast VersionInfo, gotVersion VersionInfo) bool {
	if gotVersion.Major > wantAtLeast.Major ||
		(gotVersion.Major == wantAtLeast.Major && (gotVersion.Minor > wantAtLeast.Minor ||
			(gotVersion.Minor == wantAtLeast.Minor && gotVersion.Patch >= wantAtLeast.Patch))) {
		return true
	}
	return false
}

func ParseLocalCurlVersion(firstLine string) (VersionInfo, error) {
	var ret VersionInfo
	if !strings.HasPrefix(firstLine, "curl ") || len(firstLine) < 6 { // "curl 1" is 6 long, so that's a minimum
		return ret, errors.New("curl output was not sane")
	}
	versionWord := strings.Split(firstLine, " ")[1] // second "word" - we know that 'curl ' with the space is present
	versionParts := strings.Split(versionWord, ".") // should be 3 parts long

	Major, majorErr := strconv.Atoi(versionParts[0])
	Minor, minorErr := strconv.Atoi(versionParts[1])
	Patch, patchErr := strconv.Atoi(versionParts[2])
	ret.Major = Major
	ret.Minor = Minor
	ret.Patch = Patch
	return ret, cmp.Or(majorErr, minorErr, patchErr)
}

/*
	func (run *TestRun) FixLinuxRunIfWindowsToWslCurlRun(args []string) (cmd *exec.Cmd, inputs []string, outputs []string) {
		cmd = exec.Command("curl", args...)
		inputs = run.ListInputFiles
		outputs = run.ListOutputFiles

		if runtime.GOOS == "windows" {
			moreargs := []string{"--cd", run.TempDir, "--shell-type", "none", "curl"} // --shell-type none required for zsh at least
			for _, origParam := range args {
				filePrefix := ""
				paramValueNoPrefix := origParam
				if origParam[0:1] == "@" || origParam[0:1] == ">" {
					filePrefix = origParam[0:1]
					paramValueNoPrefix = origParam[1:]
				}
				if slices.Index(outputs, paramValueNoPrefix) > -1 || slices.Index(inputs, paramValueNoPrefix) > -1 {
					moreargs = append(moreargs, filePrefix+filepath.Base(origParam)) // must remap
				} else {
					moreargs = append(moreargs, origParam)
				}
			}
			args = moreargs
			cmd = exec.Command("wsl", moreargs...)
		}
		return cmd, inputs, outputs
	}
*/
func (run *TestRun) RunTestRunAgainstCurlCli() {
	var args []string

	if run.CmdLineBuilder != nil {
		args = run.CmdLineBuilder(run)
	} else {
		run.Testing.Fatal("Forgot to add CmdLineBuilder to 'curl' CLI test!")
	}

	outputs, err := run.runCurlRetryingRateLimits(args)
	if err != nil {
		run.Testing.Fatal(err)
	}

	for index := range outputs {
		json, rawJson, err := ReadJson(outputs[index])
		if err != nil {
			run.ErrorHandler(curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON", err), run)
			return
		}

		if run.SuccessHandler != nil {
			run.SuccessHandler(json, run)
		}
		if run.SuccessHandlerIndexed != nil {
			run.SuccessHandlerIndexed(json, index, run)
		}
		if run.SuccessHandlerIndexedRaw != nil {
			run.SuccessHandlerIndexedRaw(json, rawJson, index, run)
		}
	}
}

// httpbin.org rate-limits clients by answering 429 Too Many Requests, which
// made the network tests fail at random. The harness retries rate-limited
// requests, for both go-curling and the real curl binary, with exponential
// backoff plus jitter (so parallel test packages spread out), honoring a
// Retry-After header when one is visible.
const (
	rateLimitMaxAttempts  = 8
	rateLimitBaseDelay    = time.Second
	rateLimitMaxDelay     = 15 * time.Second
	rateLimitMaxRetryWait = time.Minute
)

// rateLimitSleep is replaced in tests to keep them fast.
var rateLimitSleep = time.Sleep

// rateLimitDelay returns how long to wait before retry number attempt+1: an
// exponential backoff capped at rateLimitMaxDelay, raised to retryAfter when the
// server asked for longer, plus up to 50% random jitter.
func rateLimitDelay(attempt int, retryAfter time.Duration) time.Duration {
	delay := rateLimitMaxDelay
	if attempt < 8 {
		delay = min(rateLimitBaseDelay<<attempt, rateLimitMaxDelay)
	}
	delay = max(delay, min(retryAfter, rateLimitMaxRetryWait))
	return delay + time.Duration(rand.Int64N(int64(delay/2)+1)) // #nosec G404 -- jitter only
}

// parseRetryAfter reads a Retry-After header given in seconds or as an HTTP date.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return 0
}

// rateLimitedResponse reports whether go-curling's final response was a 429,
// and how long the server asked us to wait.
func rateLimitedResponse(resp *curl.CurlResponses) (bool, time.Duration) {
	if resp == nil || len(resp.Responses) == 0 {
		return false, 0
	}
	last := resp.Responses[len(resp.Responses)-1].HttpResponse
	if last == nil || last.StatusCode != http.StatusTooManyRequests {
		return false, 0
	}
	return true, parseRetryAfter(last.Header.Get("Retry-After"), time.Now())
}

// isRateLimitedBody recognizes httpbin's 429 reply body. That body is all the
// harness can see when the real curl binary saves a response to a file.
func isRateLimitedBody(raw []byte) bool {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	code, ok := obj["status_code"].(float64)
	return ok && code == http.StatusTooManyRequests
}

func outputsRateLimited(outputs []string) bool {
	for _, file := range outputs {
		raw, err := os.ReadFile(file) // #nosec G304 -- test output files
		if err == nil && isRateLimitedBody(raw) {
			return true
		}
	}
	return false
}

// getCompleteResponseRetryingRateLimits performs go-curling's request for one
// URL, retrying while the server answers 429. The status is checked before any
// output is written, so a rate-limited body never reaches the output files. It
// returns a nil request when the request could not be built, and an error once
// rateLimitMaxAttempts attempts were all rate-limited.
func (run *TestRun) getCompleteResponseRetryingRateLimits(ctx *curl.CurlContext, client *http.Client, index int) (*curl.CurlResponses, *http.Request, *curlerrors.CurlError) {
	for attempt := 0; ; attempt++ {
		request, cerr := ctx.BuildHttpRequest(ctx.Urls[index], index, true, true)
		if cerr != nil {
			return nil, nil, cerr
		}
		resp, cerr := ctx.GetCompleteResponse(index, client, request)
		limited, retryAfter := rateLimitedResponse(resp)
		if !limited {
			return resp, request, cerr
		}
		resp.Close()
		if attempt+1 >= rateLimitMaxAttempts {
			return resp, request, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, fmt.Sprintf("%s is still rate-limited (HTTP 429) after %d attempts", ctx.Urls[index], rateLimitMaxAttempts))
		}
		delay := rateLimitDelay(attempt, retryAfter)
		run.Testing.Logf("go-curling: %s was rate-limited (HTTP 429); retrying in %s (attempt %d of %d)", ctx.Urls[index], delay.Round(time.Millisecond), attempt+2, rateLimitMaxAttempts)
		rateLimitSleep(delay)
	}
}

// runCurlRetryingRateLimits runs the real curl binary, running it again while
// any output file holds a rate-limited reply. curl's -o truncates its output
// files, so each attempt starts clean.
func (run *TestRun) runCurlRetryingRateLimits(args []string) ([]string, error) {
	for attempt := 0; ; attempt++ {
		cmd, _, outputs := run.FixLinuxRunIfWindowsToWslCurlRun(args)
		if err := cmd.Run(); err != nil {
			return nil, errors.New("Error running @[" + cmd.Path + "] " + strings.Join(cmd.Args, " ") + ": " + err.Error())
		}
		if !outputsRateLimited(outputs) {
			return outputs, nil
		}
		if attempt+1 >= rateLimitMaxAttempts {
			return nil, fmt.Errorf("curl %s is still rate-limited (HTTP 429) after %d attempts", strings.Join(args, " "), rateLimitMaxAttempts)
		}
		delay := rateLimitDelay(attempt, 0)
		run.Testing.Logf("curl: rate-limited (HTTP 429); retrying in %s (attempt %d of %d)", delay.Round(time.Millisecond), attempt+2, rateLimitMaxAttempts)
		rateLimitSleep(delay)
	}
}
