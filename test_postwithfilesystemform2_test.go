package main

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithFilesystemForm2_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			HttpVerb:      "POST",
			BodyOutput:    testrun.EnsureAtLeastOneOutputFiles(),
			Data_Standard: []string{"@" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemForm2_success
	testRun.RunTestRun()
}
func Test_PostWithFilesystemForm2_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-d", "@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemForm2_success
	testRun.RunTestRun()
}
func helper_PostWithFilesystemForm2_success(json map[string]interface{}, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test"])
}
