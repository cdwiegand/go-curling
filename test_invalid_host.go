package main

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltestharness "github.com/cdwiegand/go-curling/tests"
)

func Test_invalidUrlNoResponse_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls: []string{"http://0.0.0.0/fail"},
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed with invalid hostname, no possible response!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun() // ensure we dont crash
}
func Test_invalidUrlNoResponse_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		return []string{"http://0.0.0.0/fail", "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed with invalid hostname, no possible response!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun() // ensure we dont crash
}
