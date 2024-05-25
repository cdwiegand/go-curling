package functionaltests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_PostJsonInclude_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("{\"test\": \"one\"}"), 0666)
		return &curl.CurlContext{
			HttpVerb:   "POST",
			Data_Json:  []string{"@" + testrun.ListInputFiles[0]},
			Urls:       []string{"https://httpbin.org/post"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
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
		curlcommontests.VerifyJson(t, json, "form")
		data := json["data"].(string)
		curlcommontests.VerifyGot(t, "{ 'test': 'one' }", data)
	}
	testRun.Run()
}
func Test_PostJsonDoubleQuotes_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "{ \"test\": \"one\" }", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		curlcommontests.VerifyJson(t, json, "form")
		data := json["data"].(string)
		curlcommontests.VerifyGot(t, "{ \"test\": \"one\" }", data)
	}
	testRun.Run()
}
func helper_PostJsonInclude_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	curlcommontests.VerifyJson(t, json, "form")
	data := json["data"].(string)
	curlcommontests.VerifyGot(t, "@"+testrun.ListInputFiles[0], data)
}
