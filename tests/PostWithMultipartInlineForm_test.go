package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			Method:         "POST",
			Output:         testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=one"},
		}
	}
	testRun.SuccessHandler = helper_PostWithMultipartInlineForm_success
	testRun.Run()
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipartInlineForm_success
	testRun.Run()
}
func helper_PostWithMultipartInlineForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	VerifyJson(t, json, "form")
	form := json["form"].(map[string]any)
	VerifyGot(t, "one", form["test"])
}
