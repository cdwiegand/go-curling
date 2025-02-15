package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithMultipartForm2_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=one"},
		}
	}
	testRun.SuccessHandler = helper_PostWithMultipartForm2_success
	testRun.RunTestRun()
}
func Test_PostWithMultipartForm2_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipartForm2_success
	testRun.RunTestRun()
}
func helper_PostWithMultipartForm2_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "one", form["test"])
}
