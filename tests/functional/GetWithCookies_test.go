package functionaltests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_GetWithCookies_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/cookies"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
			Cookies:    []string{"testcookie2=value2"},
		}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.Run()
}

func Test_GetWithCookies_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-b", "testcookie2=value2", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/cookies", "--cookie", "testcookie2=value2", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.Run()
}

func helper_GetWithCookies_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing

	curlcommontests.VerifyJson(t, json, "cookies")
	cookies := json["cookies"].(map[string]interface{})
	curlcommontests.VerifyGot(t, "value2", cookies["testcookie2"])
}
