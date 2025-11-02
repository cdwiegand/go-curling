package curltestharnessforms

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_PutWithUpload_filesystemFilesForm_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	expectedResult := []string{"test=one", "test=two"}
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[0]), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[1]), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/put", "https://httpbin.org/put"},
			HttpVerb:    "PUT",
			BodyOutput:  testrun.GetOutputFiles(2),
			Upload_File: testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandlerIndexed = helper_PutWithUpload_filesystemFilesForm_success
	testRun.RunTestRun()
}
func Test_PutWithUpload_filesystemFilesForm_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	expectedResult := []string{"test=one", "test=two"}
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[0]), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[1]), 0666)
		return []string{"https://httpbin.org/put", "-T", testrun.ListInputFiles[0], "https://httpbin.org/put", "-T", testrun.ListInputFiles[1], "-o", testrun.GetOneOutputFile(), "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_PutWithUpload_filesystemFilesForm_success
	testRun.RunTestRun()
}
func helper_PutWithUpload_filesystemFilesForm_success(json map[string]interface{}, index int, testrun *curltests.TestRun) {
	t := testrun.Testing
	expectedResult := []string{"test=one", "test=two"}
	assert.NotNil(t, json["data"])
	data := json["data"].(string)
	assert.EqualValues(t, expectedResult[index], data)
}
