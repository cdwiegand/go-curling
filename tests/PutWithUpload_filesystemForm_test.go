package tests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_PutWithUpload_filesystemForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/put"},
			Output:      testrun.EnsureAtLeastOneOutputFiles(),
			Upload_File: testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		VerifyJson(t, json, "data")
		data := json["data"].(string)
		VerifyGot(t, "test=one", data)
	}
	testRun.Run()
}
func Test_PutWithUpload_filesystemForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/put", "-T", testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "data")
		data := json["data"].(string)
		VerifyGot(t, "test=one", data)
	}
	testRun.Run()
}
