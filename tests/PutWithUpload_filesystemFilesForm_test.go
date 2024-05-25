package tests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_PutWithUpload_filesystemFilesForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	expectedResult := []string{"test=one", "test=two"}
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
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
	testRun.Run()
}
func Test_PutWithUpload_filesystemFilesForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	expectedResult := []string{"test=one", "test=two"}
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[0]), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte(expectedResult[1]), 0666)
		return []string{"https://httpbin.org/put", "-T", testrun.ListInputFiles[0], "https://httpbin.org/put", "-T", testrun.ListInputFiles[1], "-o", testrun.GetOneOutputFile(), "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_PutWithUpload_filesystemFilesForm_success
	testRun.Run()
}
func helper_PutWithUpload_filesystemFilesForm_success(json map[string]interface{}, index int, testrun *TestRun) {
	t := testrun.Testing
	expectedResult := []string{"test=one", "test=two"}
	VerifyJson(t, json, "data")
	data := json["data"].(string)
	VerifyGot(t, expectedResult[index], data)
}
