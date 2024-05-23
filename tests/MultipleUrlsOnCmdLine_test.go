package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_MultipleUrls_Context(t *testing.T) {

	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		testrun.GetOneOutputFile()
		testrun.GetOneOutputFile()
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two"},
			Output: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandlerIndexed = helper_MultipleUrls_success
	testRun.Run()
}

func Test_MultipleUrls_CmdLine(t *testing.T) {

	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", testrun.GetOneOutputFile(), "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_MultipleUrls_success
	testRun.Run()
}
func helper_MultipleUrls_success(json map[string]interface{}, index int, testrun *TestRun) {
	t := testrun.Testing
	expectedResult := []string{"one", "two"}
	VerifyJson(t, json, "args")
	args := json["args"].(map[string]any)
	VerifyGot(t, expectedResult[index], args["test"])
}
