package main

import (
	"os"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltestharness "github.com/cdwiegand/go-curling/tests"
)

func Test_CannotMixDataFormUploadArgs_Context(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			HttpVerb:      "POST",
			BodyOutput:    testrun.EnsureAtLeastOneOutputFiles(),
			Data_Standard: []string{"test=one"},
			Upload_File:   testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -d and -T are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()

	testRun = curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Data_Standard:  []string{"test=one"},
			Form_Multipart: testrun.ListInputFiles,
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -d and -F are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()

	testRun = curltestharness.BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *curltestharness.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			HttpVerb:       "POST",
			BodyOutput:     testrun.EnsureAtLeastOneOutputFiles(),
			Upload_File:    []string{testrun.ListInputFiles[0]},
			Form_Multipart: []string{testrun.ListInputFiles[1]},
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -F and -T are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()
}

func Test_CannotMixDataFormUploadArgs_CmdLine(t *testing.T) {
	testRun := curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-d", "@" + testrun.ListInputFiles[0],
			"-T", "@" + testrun.ListInputFiles[1],
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -d and -T are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()

	testRun = curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-d", "@" + testrun.ListInputFiles[0],
			"-F", "@" + testrun.ListInputFiles[1],
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -d and -F are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()

	testRun = curltestharness.BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *curltestharness.TestRun) []string {
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		os.WriteFile(testrun.GetNextInputFile(), []byte("test=one"), 0666)
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-F", "@" + testrun.ListInputFiles[0],
			"-T", "@" + testrun.ListInputFiles[1],
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *curltestharness.TestRun) {
		curltestharness.GenericTestErrorHandler(t, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "Should not succeed if -F and -T are mixed!"))
	}
	testRun.ErrorHandler = func(err *curlerrors.CurlError, testrun *curltestharness.TestRun) {
		// ok, it SHOULD fail, this is not a valid request!
	}
	testRun.RunTestRun()
}
