package functionaltests

import (
	"testing"

	curlcommontests "github.com/cdwiegand/go-curling/tests/common"
)

func Test_MultipleSuperShortOptions_CmdLine(t *testing.T) {
	// test multiple short args at once
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"-sL", "https://httpbin.org/redirect-to?url=https://httpbin.org/get%3Ftest%3Done", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = helpers_CliTest1_success
	testRun.Run()
}

func helpers_CliTest1_success(json map[string]interface{}, testrun *TestRun) {
	t := testrun.Testing
	curlcommontests.VerifyJson(t, json, "args")
	args := json["args"].(map[string]any)
	curlcommontests.VerifyGot(t, "one", args["test"])
}
