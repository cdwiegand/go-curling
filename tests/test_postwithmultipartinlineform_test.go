package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=one"},
		}
	}
	testRun.SuccessHandler = helper_PostWithMultipartInlineForm_success
	testRun.RunTestRun()
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipartInlineForm_success
	testRun.RunTestRun()
}
func helper_PostWithMultipartInlineForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test"])
}

func Test_PostWithMultipleDataArgs_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test1=one", "-d", "test2=two", "-d", "test3=three", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipleDataArgs_success
	testRun.RunTestRun()
}
func helper_PostWithMultipleDataArgs_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test1"])
	assert.EqualValues(t, "two", form["test2"])
	assert.EqualValues(t, "three", form["test3"])
}
