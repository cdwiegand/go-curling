package main

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltestharness "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PutWithUpload_filesystemForm_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/put"},
			BodyOutput:  testrun.EnsureAtLeastOneOutputFiles(),
			Upload_File: testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandler = helper_PutWithUpload_filesystemForm_success
	testRun.RunTestRun()
}
func Test_PutWithUpload_filesystemForm_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/put", "-T", testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PutWithUpload_filesystemForm_success
	testRun.RunTestRun()
}
func helper_PutWithUpload_filesystemForm_success(json map[string]interface{}, testrun *curltestharness.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["data"])
	data := json["data"].(string)
	assert.EqualValues(t, "test=one", data)
}
