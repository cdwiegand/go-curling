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
			Urls:      []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			Output:    testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar: cookieFile,
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		VerifyGot(t, "testvalue", cookies["testcookie"])
	}
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:      []string{"https://httpbin.org/cookies"},
			Output:    testrun.EnsureAtLeastOneOutputFiles(),
			CookieJar: cookieFile,
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		VerifyGot(t, "testvalue", cookies["testcookie"])
	}
	testRun.Run()
}
func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-c", cookieFile, "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		VerifyGot(t, "testvalue", cookies["testcookie"])
	}
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-c", cookieFile, "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		VerifyGot(t, "testvalue", cookies["testcookie"])
	}
	testRun.Run()
}
