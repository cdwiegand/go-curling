package tests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_PostJsonInclude_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("{\"test\": \"one\"}"), 0666)
		return &curl.CurlContext{
			Method:    "POST",
			Data_Json: []string{"@" + testrun.ListInputFiles[0]},
			Urls:      []string{"https://httpbin.org/post"},
			Output:    testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_PostJsonInclude_success
	testRun.Run()
}
func Test_PostJsonInclude_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("{\"test\": \"one\"}"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostJsonInclude_success
	testRun.Run()
}
func Test_PostJsonSingleQuotes_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "{ 'test': 'one' }", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "form")
		data := json["data"].(string)
		VerifyGot(t, "{ 'test': 'one' }", data)
	}
	testRun.Run()
}
func Test_PostJsonDoubleQuotes_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "{ \"test\": \"one\" }", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "form")
		data := json["data"].(string)
		VerifyGot(t, "{ \"test\": \"one\" }", data)
	}
	testRun.Run()
}
func helper_PostJsonInclude_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	VerifyJson(t, json, "form")
	data := json["data"].(string)
	VerifyGot(t, "@"+testrun.ListInputFiles[0], data)
}
