package tests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_PostWithMultipartForm3_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			Method:         "POST",
			Output:         testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=<" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandlerIndexed = helper_PostWithMultipartForm3_success
	testRun.Run()
}
func Test_PostWithMultipartForm3_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=<" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_PostWithMultipartForm3_success
	testRun.Run()
}
func helper_PostWithMultipartForm3_success(json map[string]interface{}, index int, testrun *TestRun) {
	t := testrun.Testing
	VerifyJson(t, json, "form")
	form := json["form"].(map[string]any)
	VerifyGot(t, "one", form["test"])
}
