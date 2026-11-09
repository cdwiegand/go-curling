package curltestharness

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseLocalCurlVersion(t *testing.T) {
	got, err1 := ParseLocalCurlVersion("curl 1.21.34")
	assert.Nil(t, err1)
	assert.Equal(t, got.Major, 1)
	assert.Equal(t, got.Minor, 21)
	assert.Equal(t, got.Patch, 34)
}

// --- rate-limit retry helpers ---

// httpbinRateLimitBody is what httpbin.org sends with a 429.
const httpbinRateLimitBody = `{
  "status_code": 429,
  "error": "Too Many Requests",
  "detail": "rate limit exceeded: 2 requests per second per client IP for body-echo requests; retry after 1 second"
}`

// recordSleeps replaces the retry sleep for the duration of a test.
func recordSleeps(t *testing.T) *[]time.Duration {
	t.Helper()
	var mu sync.Mutex
	var sleeps []time.Duration
	old := rateLimitSleep
	rateLimitSleep = func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		sleeps = append(sleeps, d)
	}
	t.Cleanup(func() { rateLimitSleep = old })
	return &sleeps
}

func Test_ParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"", 0},
		{"0", 0},
		{"1", time.Second},
		{" 5 ", 5 * time.Second},
		{"120", 2 * time.Minute},
		{"-3", 0},
		{"soon", 0},
		{now.Add(30 * time.Second).Format(http.TimeFormat), 30 * time.Second},
		{now.Add(-30 * time.Second).Format(http.TimeFormat), 0}, // a date in the past
	} {
		t.Run(tc.value, func(t *testing.T) {
			assert.Equal(t, tc.want, parseRetryAfter(tc.value, now))
		})
	}
}

func Test_RateLimitDelay(t *testing.T) {
	for _, tc := range []struct {
		attempt    int
		retryAfter time.Duration
		base       time.Duration
	}{
		{0, 0, time.Second},
		{1, 0, 2 * time.Second},
		{2, 0, 4 * time.Second},
		{3, 0, 8 * time.Second},
		{4, 0, rateLimitMaxDelay},               // 16s is capped at 15s
		{50, 0, rateLimitMaxDelay},              // large attempts do not overflow
		{0, 10 * time.Second, 10 * time.Second}, // Retry-After longer than the backoff wins
		{3, 2 * time.Second, 8 * time.Second},   // ...but a shorter one does not shorten it
		{0, time.Hour, rateLimitMaxRetryWait},   // Retry-After is capped
	} {
		t.Run(fmt.Sprintf("attempt %d retry-after %s", tc.attempt, tc.retryAfter), func(t *testing.T) {
			for i := 0; i < 50; i++ {
				d := rateLimitDelay(tc.attempt, tc.retryAfter)
				assert.GreaterOrEqual(t, d, tc.base)
				assert.LessOrEqual(t, d, tc.base+tc.base/2) // at most 50% jitter
			}
		})
	}
}

func Test_IsRateLimitedBody(t *testing.T) {
	assert.True(t, isRateLimitedBody([]byte(httpbinRateLimitBody)))
	assert.True(t, isRateLimitedBody([]byte(`{"status_code":429}`)))
	assert.False(t, isRateLimitedBody([]byte(`{"status_code":200}`)))
	assert.False(t, isRateLimitedBody([]byte(`{"status_code":"429"}`))) // not a number
	assert.False(t, isRateLimitedBody([]byte(`{"args":{},"url":"https://httpbin.org/get"}`)))
	assert.False(t, isRateLimitedBody([]byte(`[429]`)))
	assert.False(t, isRateLimitedBody([]byte("Too Many Requests")))
	assert.False(t, isRateLimitedBody(nil))
}

func Test_RateLimitedResponse(t *testing.T) {
	limited, wait := rateLimitedResponse(nil)
	assert.False(t, limited)
	assert.Zero(t, wait)

	limited, _ = rateLimitedResponse(&curl.CurlResponses{})
	assert.False(t, limited)

	ok := &http.Response{StatusCode: 200, Header: http.Header{}}
	tooMany := &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": {"3"}}}
	limited, wait = rateLimitedResponse(&curl.CurlResponses{Responses: []*curl.CurlResponse{{HttpResponse: tooMany}}})
	assert.True(t, limited)
	assert.Equal(t, 3*time.Second, wait)

	// only the final response of a redirect chain counts
	limited, _ = rateLimitedResponse(&curl.CurlResponses{Responses: []*curl.CurlResponse{{HttpResponse: tooMany}, {HttpResponse: ok}}})
	assert.False(t, limited)
	limited, _ = rateLimitedResponse(&curl.CurlResponses{Responses: []*curl.CurlResponse{{HttpResponse: ok}, {HttpResponse: tooMany}}})
	assert.True(t, limited)

	limited, _ = rateLimitedResponse(&curl.CurlResponses{Responses: []*curl.CurlResponse{{HttpResponse: nil}}})
	assert.False(t, limited)
}

// rateLimitingServer answers each request according to schedule: true means
// "send httpbin's 429", false means "send a normal httpbin-style JSON echo".
// Requests beyond the schedule succeed.
type rateLimitingServer struct {
	*httptest.Server
	mu       sync.Mutex
	schedule []bool
	requests int
}

func newRateLimitingServer(t *testing.T, schedule ...bool) *rateLimitingServer {
	t.Helper()
	s := &rateLimitingServer{schedule: schedule}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := s.requests
		s.requests++
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if n < len(s.schedule) && s.schedule[n] {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, httpbinRateLimitBody)
			return
		}
		fmt.Fprintf(w, `{"args":{"test":%q},"method":%q}`, r.URL.Query().Get("test"), r.Method)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *rateLimitingServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

func Test_RunTestRun_RetriesRateLimitedGoCurlingRequests(t *testing.T) {
	sleeps := recordSleeps(t)
	srv := newRateLimitingServer(t, true, true, false)

	var got []map[string]interface{}
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{Urls: []string{srv.URL + "/get?test=one"}, BodyOutput: testrun.EnsureAtLeastOneOutputFiles()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, _ *TestRun) { got = append(got, json) }
	testRun.RunTestRun()

	require.Len(t, got, 1, "the success handler should only see the final, successful response")
	assert.Equal(t, "one", got[0]["args"].(map[string]interface{})["test"])
	assert.Equal(t, 3, srv.count())
	require.Len(t, *sleeps, 2)
	for _, d := range *sleeps {
		assert.GreaterOrEqual(t, d, time.Second, "Retry-After: 1 is honored")
	}
	assert.Equal(t, 200, testRun.Responses.Responses[0].HttpResponse.StatusCode)

	// the 429 body must not have been written to (or left in) the output file
	raw, err := os.ReadFile(testRun.ListOutputFiles[0])
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "Too Many Requests")
}

func Test_RunTestRun_RetriesEachUrlIndependently(t *testing.T) {
	recordSleeps(t)
	// first URL: limited once; second URL: limited twice
	srv := newRateLimitingServer(t, true, false, true, true, false)

	seen := map[int]string{}
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		testrun.GetOneOutputFile()
		testrun.GetOneOutputFile()
		return &curl.CurlContext{
			Urls:       []string{srv.URL + "/get?test=one", srv.URL + "/get?test=two"},
			BodyOutput: testrun.ListOutputFiles,
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, _ *TestRun) {
		seen[index] = json["args"].(map[string]interface{})["test"].(string)
	}
	testRun.RunTestRun()

	assert.Equal(t, map[int]string{0: "one", 1: "two"}, seen)
	assert.Equal(t, 5, srv.count())
}

func Test_RunTestRun_GivesUpAfterMaxAttempts(t *testing.T) {
	sleeps := recordSleeps(t)
	schedule := make([]bool, rateLimitMaxAttempts+5)
	for i := range schedule {
		schedule[i] = true
	}
	srv := newRateLimitingServer(t, schedule...)

	var errs []*curlerrors.CurlError
	handlerCalled := false
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{Urls: []string{srv.URL + "/get"}, BodyOutput: testrun.EnsureAtLeastOneOutputFiles()}
	}
	testRun.SuccessHandler = func(map[string]interface{}, *TestRun) { handlerCalled = true }
	testRun.ErrorHandler = func(err *curlerrors.CurlError, _ *TestRun) { errs = append(errs, err) }
	testRun.RunTestRun()

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].ErrorString, "still rate-limited")
	assert.False(t, handlerCalled)
	assert.Equal(t, rateLimitMaxAttempts, srv.count())
	assert.Len(t, *sleeps, rateLimitMaxAttempts-1)
}

func Test_RunTestRun_DoesNotRetryOtherErrorStatuses(t *testing.T) {
	sleeps := recordSleeps(t)
	var requests int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status_code":503}`)
	}))
	defer srv.Close()

	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{Urls: []string{srv.URL + "/get"}, BodyOutput: testrun.EnsureAtLeastOneOutputFiles()}
	}
	testRun.SuccessHandler = func(map[string]interface{}, *TestRun) {}
	testRun.RunTestRun()

	assert.Equal(t, 1, requests)
	assert.Empty(t, *sleeps)
}

func Test_RunTestRun_RetriesRateLimitedCurlRuns(t *testing.T) {
	if !EnsureLocalCurlMinVersionAndLog(t, NewVersionInfo(7, 0, 0)) {
		t.Skip("real curl is not installed")
	}
	sleeps := recordSleeps(t)
	// go-curling: limited once, then OK; real curl: limited twice, then OK
	srv := newRateLimitingServer(t, true, false, true, true, false)

	calls := 0
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{srv.URL + "/get?test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, _ *TestRun) {
		calls++
		assert.Equal(t, "one", json["args"].(map[string]interface{})["test"])
		assert.Nil(t, json["status_code"], "a rate-limited body must never reach the handler")
	}
	testRun.RunTestRun()

	assert.Equal(t, 2, calls, "once for go-curling and once for curl")
	assert.Equal(t, 5, srv.count())
	assert.Len(t, *sleeps, 3)
}

func Test_RunCurlRetryingRateLimits_GivesUp(t *testing.T) {
	if !EnsureLocalCurlMinVersionAndLog(t, NewVersionInfo(7, 0, 0)) {
		t.Skip("real curl is not installed")
	}
	sleeps := recordSleeps(t)
	schedule := make([]bool, rateLimitMaxAttempts+5)
	for i := range schedule {
		schedule[i] = true
	}
	srv := newRateLimitingServer(t, schedule...)

	run := BuildTestRun(t)
	_, err := run.runCurlRetryingRateLimits([]string{"-s", srv.URL + "/get", "-o", run.GetOneOutputFile()})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "still rate-limited"), err.Error())
	assert.Equal(t, rateLimitMaxAttempts, srv.count())
	assert.Len(t, *sleeps, rateLimitMaxAttempts-1)
}

func Test_RunCurlRetryingRateLimits_NoOutputFilesMeansNoRetry(t *testing.T) {
	if !EnsureLocalCurlMinVersionAndLog(t, NewVersionInfo(7, 0, 0)) {
		t.Skip("real curl is not installed")
	}
	sleeps := recordSleeps(t)
	srv := newRateLimitingServer(t, true, true, true)

	run := BuildTestRun(t)
	null := os.DevNull
	_, err := run.runCurlRetryingRateLimits([]string{"-s", srv.URL + "/get", "-o", null})
	require.NoError(t, err)
	assert.Equal(t, 1, srv.count())
	assert.Empty(t, *sleeps)
}
