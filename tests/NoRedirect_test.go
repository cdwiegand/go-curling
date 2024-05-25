package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_NoRedirectTest_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/redirect-to?url=https://httpbin.org/get%3Ftest%3Done"},
			BodyOutput:      testrun.EnsureAtLeastOneOutputFiles(),
			FollowRedirects: false,
		}
	}
	testRun.SuccessHandler = helper_NoRedirectTest_Success

	testRun.Run()
}
func Test_NoRedirectTest_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.GetOneOutputFile() // so we can use one output file
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"https://httpbin.org/redirect-to?url=https://httpbin.org/get%3Ftest%3Done", "-o", testrun.ListOutputFiles[0]}
	}
	testRun.SuccessHandler = helper_NoRedirectTest_Success
	testRun.Run()
}
func helper_NoRedirectTest_Success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing

	if len(testrun.Responses.Responses) != 1 {
		t.Error("Should only have gotten 1 response")
	}

	firstResponse := testrun.Responses.Responses[0]
	if firstResponse.HttpResponse.StatusCode < 300 || firstResponse.HttpResponse.StatusCode > 399 {
		t.Errorf("Should have 3xx response code, got %d", firstResponse.HttpResponse.StatusCode)
	}
	if firstResponse.HttpResponse.Header.Get("Location") == "" {
		t.Errorf("Should have a Location header, did not get one")
	}
}
