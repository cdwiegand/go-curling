package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_MultipleUrls_Context(t *testing.T) {
	expectedResult := []string{"one", "two"}

	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		testrun.GetOneOutputFile()
		testrun.GetOneOutputFile()
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two"},
			Output: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		VerifyJson(t, json, "args")
		args := json["args"].(map[string]any)
		VerifyGot(t, expectedResult[index], args["test"])
	}
	testRun.Run()
}

func Test_MultipleUrls_CmdLine(t *testing.T) {
	expectedResult := []string{"one", "two"}

	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", testrun.GetOneOutputFile(), "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		VerifyJson(t, json, "args")
		args := json["args"].(map[string]any)
		VerifyGot(t, expectedResult[index], args["test"])
	}
	testRun.Run()
}
