package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_Headers_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:    []string{"https://httpbin.org/headers"},
			Headers: []string{"X-Hello: World"},
			Output:  testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "headers")
		args := json["headers"].(map[string]interface{})
		VerifyGot(t, "World", args["X-Hello"])
	}
	testRun.Run()
}

func Test_Headers_Cmdline(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/headers", "-H", "X-Hello: World", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "headers")
		args := json["headers"].(map[string]interface{})
		VerifyGot(t, "World", args["X-Hello"])
	}
	testRun.Run()
}
