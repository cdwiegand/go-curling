package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_Headers_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/headers"},
			Headers:    []string{"X-Hello: World", "X-Good: Times"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
	}
	testRun.SuccessHandler = helpers_Headers_success
	testRun.RunTestRun()

	// second form:
	testRun = curltests.BuildTestRun(t)
	headersDict := make(map[string]string)
	headersDict["X-Hello"] = "World"
	headersDict["X-Good"] = "Times"
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		ctx := &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/headers"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
		}
		ctx.SetHeadersFromDict(headersDict)
		return ctx
	}
	testRun.SuccessHandler = helpers_Headers_success
	testRun.RunTestRun()
}

func Test_Headers_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/headers", "-H", "X-Hello: World", "--header", "X-Good: Times", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helpers_Headers_success
	testRun.RunTestRun()
}

func helpers_Headers_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing

	assert.NotNil(t, json["headers"])
	args := json["headers"].(map[string]interface{})
	assert.EqualValues(t, "World", args["X-Hello"])
	assert.EqualValues(t, "Times", args["X-Good"])
}
