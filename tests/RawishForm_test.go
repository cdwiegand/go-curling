package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_RawishForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Output:        testrun.EnsureAtLeastOneOutputFiles(),
			Method:        "POST",
			Data_Standard: []string{"{'name': 'Robert J. Oppenheimer'}"},
			Headers:       []string{"Content-Type: application/json"},
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "data")
		data := json["data"]
		VerifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
	}
	testRun.Run()
}

func Test_RawishForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"-o", testrun.GetOneOutputFile(), "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "data")
		data := json["data"]
		VerifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
	}
	testRun.Run()
}
