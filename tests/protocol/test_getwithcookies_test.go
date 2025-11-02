package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltests "github.com/cdwiegand/go-curling/tests"

	"github.com/stretchr/testify/assert"
)

func Test_GetWithCookies_CurlContext(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltests.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:       []string{"https://httpbin.org/cookies"},
			BodyOutput: testrun.EnsureAtLeastOneOutputFiles(),
			Cookies:    []string{"testcookie2=value2"},
		}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.RunTestRun()
}

func Test_GetWithCookies_CmdLine(t *testing.T) {
	testRun := curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/cookies", "-b", "testcookie2=value2", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.RunTestRun()

	testRun = curltests.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltests.TestRun) []string {
		return []string{"https://httpbin.org/cookies", "--cookie", "testcookie2=value2", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_GetWithCookies_success
	testRun.RunTestRun()
}

func helper_GetWithCookies_success(json map[string]interface{}, testrun *curltests.TestRun) {
	t := testrun.Testing

	assert.NotNil(t, json["cookies"])
	cookies := json["cookies"].(map[string]interface{})
	assert.EqualValues(t, "value2", cookies["testcookie2"])
}
