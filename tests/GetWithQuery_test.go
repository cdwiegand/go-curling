package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_GetWithQuery_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/get?test=one"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_GetWithQuery_success
	testRun.Run()
}
func Test_GetWithQuery_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithQuery_success
	testRun.Run()
}
func helper_GetWithQuery_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	VerifyJson(t, json, "args")
	args := json["args"].(map[string]any)
	VerifyGot(t, "one", args["test"])
}
