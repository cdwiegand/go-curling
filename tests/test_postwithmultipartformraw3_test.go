package curltestharness

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithMultipartFormRaw3_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:              []string{"https://httpbin.org/post"},
			HttpVerb:          "POST",
			BodyOutput:        testrun.EnsureAtLeastOneOutputFiles(),
			Form_MultipartRaw: []string{"test=<" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandler = helper_PostWithMultipartFormRaw3_success
	testRun.RunTestRun()
}
func Test_PostWithMultipartFormRaw3_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "--form-string", "test=<" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipartFormRaw3_success
	testRun.RunTestRun()
}
func helper_PostWithMultipartFormRaw3_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "<"+testrun.ListInputFiles[0], form["test"])
}
