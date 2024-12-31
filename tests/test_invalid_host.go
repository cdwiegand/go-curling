package curltestharness

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func Test_invalidUrlNoResponse_CurlContext(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls: []string{"http://0.0.0.0/fail"},
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed with invalid hostname, no possible response!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun() // ensure we dont crash
}
func Test_invalidUrlNoResponse_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{"http://0.0.0.0/fail", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed with invalid hostname, no possible response!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun() // ensure we dont crash
}
