package main

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithMultipartForm3_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=<" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandlerIndexed = helper_PostWithMultipartForm3_success
	testRun.RunTestRun()
}
func Test_PostWithMultipartForm3_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=<" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_PostWithMultipartForm3_success
	testRun.RunTestRun()
}
func helper_PostWithMultipartForm3_success(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test"])
}
