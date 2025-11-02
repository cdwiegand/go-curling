package curltestharnessforms

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithUpload_filesystemForm_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/post"},
			HttpVerb:    "POST",
			BodyOutput:  testrun.EnsureAtLeastOneOutputFiles(),
			Upload_File: testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandler = helper_PostWithUpload_filesystemForm_success
	testRun.RunTestRun()
}
func Test_PostWithUpload_filesystemForm_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-T", testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithUpload_filesystemForm_success
	testRun.RunTestRun()
}
func helper_PostWithUpload_filesystemForm_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["data"])
	data := json["data"].(string)
	assert.EqualValues(t, "test=one", data)
}
