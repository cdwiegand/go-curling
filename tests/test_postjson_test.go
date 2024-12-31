package curltestharness

import (
	"encoding/json"
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
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
	testRun.RunTestRun()
}
func Test_PostJsonInclude_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.DoNotTestAgainstCurl = !EnsureLocalCurlMinVersion(NewVersionInfo(7, 82, 0))
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("{\"test\": \"one\"}"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostJsonInclude_success
	testRun.RunTestRun()
}
func Test_PostJsonSingleQuotes_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.DoNotTestAgainstCurl = !EnsureLocalCurlMinVersion(NewVersionInfo(7, 82, 0))
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "{ 'test': 'one' }", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		assert.NotNil(t, json["data"])
		data := json["data"].(string)
		assert.EqualValues(t, "{ 'test': 'one' }", data)
	}
	testRun.RunTestRun()
}
func Test_PostJsonDoubleQuotes_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.DoNotTestAgainstCurl = !EnsureLocalCurlMinVersionAndLog(t, NewVersionInfo(7, 82, 0))
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "--json", "{ \"test\": \"one\" }", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		assert.NotNil(t, json["data"])
		data := json["data"].(string)
		assert.EqualValues(t, "{ \"test\": \"one\" }", data)
	}
	testRun.RunTestRun()
}
func helper_PostJsonInclude_success(jsonIn map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing

	assert.NotNil(t, jsonIn["json"])
	jsonJson := jsonIn["json"].(map[string]interface{})
	assert.EqualValues(t, "one", jsonJson["test"])

	assert.NotNil(t, jsonIn["data"])
	dataJsonStr := jsonIn["data"].(string)
	var dataJsonObj map[string]interface{}
	json.Unmarshal([]byte(dataJsonStr), &dataJsonObj)
	assert.EqualValues(t, "one", dataJsonObj["test"])
}
