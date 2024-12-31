package curltestharness

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"

	"github.com/stretchr/testify/assert"
)

func Test_PostWithFilesystemBinaryForm_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("a&b=c"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/post"},
			HttpVerb:    "POST",
			BodyOutput:  testrun.EnsureAtLeastOneOutputFiles(),
			Data_Binary: []string{"test=@" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemBinaryForm_success
	testRun.RunTestRun()
}
func Test_PostWithFilesystemBinaryForm_CurlContext_directFile(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=a&b=c"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/post"},
			HttpVerb:    "POST",
			BodyOutput:  testrun.EnsureAtLeastOneOutputFiles(),
			Data_Binary: []string{"@" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemBinaryForm_success_directFile
	testRun.RunTestRun()
}
func Test_PostWithFilesystemBinaryForm_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("a&b=c"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "--data-binary", "test=@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemBinaryForm_success
	testRun.RunTestRun()
}
func Test_PostWithFilesystemBinaryForm_CmdLine_directFile(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=a&b=c"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "--data-binary", "@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helper_PostWithFilesystemBinaryForm_success_directFile
	testRun.RunTestRun()
}
func helper_PostWithFilesystemBinaryForm_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.True(t, form["test"].(string)[0] == '@')
}

func helper_PostWithFilesystemBinaryForm_success_directFile(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	assert.NotNil(t, json["form"])
	form := json["form"].(map[string]any)
	assert.EqualValues(t, "a", form["test"])
	assert.EqualValues(t, "c", form["b"])
}
