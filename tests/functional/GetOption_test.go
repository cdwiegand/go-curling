package functionaltests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_GetArg_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:                   []string{"https://httpbin.org/get?test=one"},
			ConvertPostFormIntoGet: true,
			Data_Standard:          []string{"hello=world"},
			BodyOutput:             testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_GetArg_success
	testRun.Run()
}
func Test_GetArg_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "-d", "hello=world", "-G", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetArg_success
	testRun.Run()
}
func helper_GetArg_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	curlcommontests.VerifyJson(t, json, "args")
	args := json["args"].(map[string]any)
	curlcommontests.VerifyGot(t, "one", args["test"])
	curlcommontests.VerifyGot(t, "world", args["hello"])
}
