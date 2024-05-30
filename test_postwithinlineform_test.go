package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithInlineForm_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			HttpVerb:      "POST",
			BodyOutput:    testrun.EnsureAtLeastOneOutputFiles(),
			Data_Standard: []string{"test=one"},
		}
	}
	testRun.SuccessHandler = helper_PostWithInlineForm_success
	testRun.RunTestRun()
}

func Test_PostWithInlineForm_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithInlineForm_success
	testRun.RunTestRun()
}
func helper_PostWithInlineForm_success(json map[string]interface{}, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test"])
}
