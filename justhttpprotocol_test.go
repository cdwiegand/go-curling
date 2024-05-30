package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_JustCallHttp_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"http://httpbin.org/get?health=ok"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_JustCallHttp_success
	testRun.Run()
}
func Test_JustCallHttp_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"http://httpbin.org/get?health=ok", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_JustCallHttp_success
	testRun.Run()
}

func helper_JustCallHttp_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, "ok", args["health"])
}
