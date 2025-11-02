package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_JustCallHttp_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"http://httpbin.org/get?health=ok"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helper_JustCallHttp_success
	testRun.RunTestRun()
}
func Test_JustCallHttp_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"http://httpbin.org/get?health=ok", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_JustCallHttp_success
	testRun.RunTestRun()
}

func helper_JustCallHttp_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["args"])
	args := json["args"].(map[string]any)
	assert.EqualValues(t, "ok", args["health"])
}
