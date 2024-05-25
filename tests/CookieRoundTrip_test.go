package tests

import (
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_CookieRoundTrip_CurlContext(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			BodyOutput:      testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar:       cookieFile,
			FollowRedirects: true,
		}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/cookies"},
			BodyOutput:      testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar:       cookieFile,
			FollowRedirects: true,
		}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.Run()
}
func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	cookie_curlFile := filepath.Join(t.TempDir(), "cookies_curl.dat")

	testRun := BuildTestRun(t)
	testRun.GetOneOutputFile() // so we can use one output file
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-L", "-c", cookieFile, "-o", testrun.ListOutputFiles[0]}
	}
	testRun.CmdLineBuilderCurl = func(testrun *TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-L", "-c", cookie_curlFile, "-o", testrun.ListOutputFiles[0]}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.GetOneOutputFile() // so we can use one output file
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-L", "--cookie-jar", cookieFile, "-o", testrun.ListOutputFiles[0]}
	}
	testRun.CmdLineBuilderCurl = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-L", "--cookie-jar", cookie_curlFile, "-o", testrun.ListOutputFiles[0]}
	}
	testRun.SuccessHandler = helper_CookieRoundTrip_success
	testRun.Run()
}

func helper_CookieRoundTrip_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing

	VerifyJson(t, json, "cookies")
	cookies := json["cookies"].(map[string]interface{})
	VerifyGot(t, "testvalue", cookies["testcookie"])
}
