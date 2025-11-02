package curltestharness

import (
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_CookieRoundTrip_CurlContext(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			BodyOutput:      testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar:       cookieFile,
			FollowRedirects: true,
		}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.RunTestRun()

	testRun = curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/cookies"},
			BodyOutput:      testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar:       cookieFile,
			FollowRedirects: true,
		}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.RunTestRun()
}
func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	out := testRun.GetOneOutputFile()             // normal "output"
	cookieFile := testRun.GetNextInputFile()      // cookie jar
	cookie_curlFile := testRun.GetNextInputFile() // curl's file (different format)

	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-L", "-c", cookieFile, "-o", out}
	}
	testRun.CmdLineBuilderCurl = func(testrun *curltests.TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-L", "-c", cookie_curlFile, "-o", out}
	}
	// testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.RunTestRun()

	// now run using the cookie jar
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-L", "-b", cookieFile, "-c", cookieFile, "-o", out}
	}
	testRun.CmdLineBuilderCurl = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-L", "-b", cookie_curlFile, "-c", cookieFile, "-o", out}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.RunTestRun()
}

func helper_CookieRoundTrip_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing

	assert.NotNil(t, json["cookies"])
	cookies := json["cookies"].(map[string]interface{})
	assert.EqualValues(t, "testvalue", cookies["testcookie"])
}
