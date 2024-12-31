package curltestharness

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_PutWithUpload_filesystemForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
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
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/put", "-T", testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PutWithUpload_filesystemForm_success
	testRun.RunTestRun()
}
func helper_PutWithUpload_filesystemForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["data"])
	data := json["data"].(string)
	assert.EqualValues(t, "test=one", data)
}
