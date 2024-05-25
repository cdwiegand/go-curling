package functionaltests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_RawishForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			BodyOutput:    testrun.EnsureAtLeastOneOutputFiles(),
			HttpVerb:      "POST",
			Data_Standard: []string{"{'name': 'Robert J. Oppenheimer'}"},
			Headers:       []string{"Content-Type: application/json"},
		}
	}
	testRun.SuccessHandler = helper_RawishForm_success
	testRun.Run()
}

func Test_RawishForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"-o", testrun.GetOneOutputFile(), "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
	}
	testRun.SuccessHandler = helper_RawishForm_success
	testRun.Run()
}
func helper_RawishForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	curlcommontests.VerifyJson(t, json, "data")
	data := json["data"]
	curlcommontests.VerifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
}
