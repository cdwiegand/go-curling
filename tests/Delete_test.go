package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_Delete_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/delete"},
			HttpVerb:   "DELETE",
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.Run()
}
func Test_Delete_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.Run()

	testRun = BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/delete", "--request", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.Run()
}
