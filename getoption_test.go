package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
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
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, "one", args["test"])
	assert.EqualValues(t, "world", args["hello"])
}
