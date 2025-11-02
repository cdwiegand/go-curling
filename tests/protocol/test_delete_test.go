package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"
)

func Test_Delete_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/delete"},
			HttpVerb:   "DELETE",
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltests.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()
}
func Test_Delete_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltests.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()

	testRun = curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/delete", "--request", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltests.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()
}

/* -- now part of normal Test_Delete_CmdLine using wsl in Windows!
func Test_Delete_CmdLine_ExplicitCurl(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}

	testRun.RunTestRunAgainstCurlCli()
}*/
