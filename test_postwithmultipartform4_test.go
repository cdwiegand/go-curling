package main

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltestharness "github.com/cdwiegand/go-curling/tests"
)

func Test_PostWithMultipartForm4_CurlContext(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Form_Multipart: []string{"@" + testrun.ListInputFiles[0]},
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed as -F does not support directly pulling a @file reference!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()
}
func Test_PostWithMultipartForm4_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{"https://httpbin.org/post", "-X", "POST", "-F", "@" + testrun.ListInputFiles[0], "-o", testrun.GetOneOutputFile()}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed as -F does not support directly pulling a @file reference!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()
}
