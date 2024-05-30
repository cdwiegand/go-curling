package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"
)

func Test_Delete_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/delete"},
			HttpVerb:   "DELETE",
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()
}
func Test_Delete_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()

	testRun = curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"https://httpbin.org/delete", "--request", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}
	testRun.RunTestRun()
}

/* -- now part of normal Test_Delete_CmdLine using wsl in Windows!
func Test_Delete_CmdLine_ExplicitCurl(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}

	testRun.RunTestRunAgainstCurlCli()
}*/
