package curltestharnesscli

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_MultipleUrls_Context(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		testrun.GetOneOutputFile()
		testrun.GetOneOutputFile()
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandlerIndexed = helper_MultipleUrls_success
	testRun.RunTestRun()
}

func Test_MultipleUrls_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", testrun.GetOneOutputFile(), "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = helper_MultipleUrls_success
	testRun.RunTestRun()
}
func helper_MultipleUrls_success(json map[string]interface{}, index int, testrun *curltests.TestRun) {
	t := testrun.Testing
	expectedResult := []string{"one", "two"}
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, expectedResult[index], args["test"])
}
