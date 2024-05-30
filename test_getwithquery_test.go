package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_GetWithQuery_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/get?test=one"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_GetWithQuery_success
	testRun.RunTestRun()
}
func Test_GetWithQuery_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithQuery_success
	testRun.RunTestRun()
}
func helper_GetWithQuery_success(json map[string]interface{}, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, "one", args["test"])
}
