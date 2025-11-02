package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_GetArg_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:                   []string{"https://httpbin.org/get?test=one"},
			ConvertPostFormIntoGet: true,
			Data_Standard:          []string{"hello=world"},
			BodyOutput:             testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_GetArg_success
	testRun.RunTestRun()
}
func Test_GetArg_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/get?test=one", "-d", "hello=world", "-G", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetArg_success
	testRun.RunTestRun()
}
func helper_GetArg_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, "one", args["test"])
	assert.EqualValues(t, "world", args["hello"])
}
