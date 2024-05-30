package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_RawishForm_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			BodyOutput:    testrun.EnsureAtLeastOneOutputFiles(),
			HttpVerb:      "POST",
			Data_Standard: []string{"{'name': 'Robert J. Oppenheimer'}"},
			Headers:       []string{"Content-Type: application/json"},
		}
	}
	testRun.SuccessHandler = helper_RawishForm_success
	testRun.RunTestRun()
}

func Test_RawishForm_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"-o", testrun.GetOneOutputFile(), "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
	}
	testRun.SuccessHandler = helper_RawishForm_success
	testRun.RunTestRun()
}
func helper_RawishForm_success(json map[string]interface{}, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["data"])
	data := json["data"]
	assert.EqualValues(t, "{'name': 'Robert J. Oppenheimer'}", data)
}
