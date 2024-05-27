package functionaltests

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_PostWithMultipartForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"test=@" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandler = helper_PostWithMultipartForm_success
	testRun.Run()
}
func Test_PostWithMultipartForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithMultipartForm_success
	testRun.Run()
}
func helper_PostWithMultipartForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	curlcommontests.VerifyJson(t, json, "files")
	files := json["files"].(map[string]any)
	curlcommontests.VerifyGot(t, "one", files["test"])
}