package tests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_RedirectTest_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:            []string{"https://httpbin.org/redirect-to?url=https://httpbin.org/get%3Ftest%3Done"},
			Output:          testrun.EnsureAtLeastOneOutputFiles(),
			FollowRedirects: true,
		}
	}
	testRun.SuccessHandler = helper_RedirectTest_Success

	testRun.Run()
}
func Test_RedirectTest_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.GetOneOutputFile() // so we can use one output file
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		// adding -L so we act like curl and follow the redirect
		return []string{"-L", "https://httpbin.org/redirect-to?url=https://httpbin.org/get%3Ftest%3Done", "-o", testrun.ListOutputFiles[0]}
	}
	testRun.SuccessHandler = helper_RedirectTest_Success
	testRun.Run()
}
func helper_RedirectTest_Success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing

	VerifyJson(t, json, "args")
	args := json["args"].(map[string]interface{})
	VerifyGot(t, "one", args["test"])

	if len(testrun.Responses.Responses) != 2 {
		t.Errorf("Should have 2 responses, got %d", len(testrun.Responses.Responses))
	}
	firstResponse := testrun.Responses.Responses[0]
	if firstResponse.HttpResponse.StatusCode < 300 || firstResponse.HttpResponse.StatusCode > 399 {
		t.Errorf("Should have 3xx response code, got %d", firstResponse.HttpResponse.StatusCode)
	}
	if firstResponse.HttpResponse.Header.Get("Location") == "" {
		t.Errorf("Should have a Location header, did not get one")
	}
	secondResponse := testrun.Responses.Responses[1]
	if secondResponse.HttpResponse.StatusCode != 200 {
		t.Errorf("Should have 200 response code, got %d", secondResponse.HttpResponse.StatusCode)
	}
}
