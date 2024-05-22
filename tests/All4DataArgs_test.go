package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
)

func Test_All4DataArgs_Context(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatastandard=a&b1=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatabinary=a&b2=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdataencoded=a&b"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b3=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b4=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        testRun.GetOneOutputFiles(),
			Data_Standard: []string{"@" + testrun.ListInputFiles[0], "testdatastandard2=@" + testrun.ListInputFiles[3]},
			Data_Binary:   []string{"@" + testrun.ListInputFiles[1], "testdatabinary2=@" + testrun.ListInputFiles[4]},
			Data_Encoded:  []string{"@" + testrun.ListInputFiles[2], "testdataencoded2=@" + testrun.ListInputFiles[5]},
			Data_RawAsIs:  []string{"testdataraw=@" + testrun.ListInputFiles[5]}, // actual file not used, just want to make sure the "@" comes across properly
		}
	}
	testRun.SuccessHandlerIndexed = func(json map[string]interface{}, index int, testrun *TestRun) {
		VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		VerifyGot(t, "a", form["testdatastandard"])
		VerifyGot(t, "c", form["b1"])
		VerifyGot(t, "a&b", form["testdataencoded"])
		VerifyGot(t, "a", form["testdatastandard2"])
		VerifyGot(t, "c", form["b3"])
		VerifyGot(t, "a", form["testdatabinary2"])
		VerifyGot(t, "c", form["b4"])
		VerifyGot(t, "a&b", form["testdataencoded2"])
		testdataraw := fmt.Sprintf("%v", form["testdataraw"])
		if !strings.HasPrefix(testdataraw, "@") {
			t.Errorf("testdataraw was %q - should start with @ - it should be the EXACT value, no @file support", testdataraw)
		}
	}
	testRun.Run()
}

func Test_All4DataArgs_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatastandard=a&b1=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatabinary=a&b2=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdataencoded=a&b"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b3=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b4=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b"), 0666)
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-d", "@" + testrun.ListInputFiles[0],
			"--data-binary", "@" + testrun.ListInputFiles[1],
			"--data-urlencode", "@" + testrun.ListInputFiles[2],
			"--data", "testdatastandard2=@" + testrun.ListInputFiles[3],
			"--data-binary", "testdatabinary2=@" + testrun.ListInputFiles[4],
			"--data-urlencode", "testdataencoded2=@" + testrun.ListInputFiles[5],
			"--data-raw", "testdataraw=@" + testrun.ListInputFiles[5], // actual file not used, just want to make sure the "@" comes across properly
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		VerifyGot(t, "a", form["testdatastandard"])
		VerifyGot(t, "c", form["b1"])
		VerifyGot(t, "a&b", form["testdataencoded"])
		VerifyGot(t, "a", form["testdatastandard2"])
		VerifyGot(t, "c", form["b3"])
		VerifyGot(t, "a", form["testdatabinary2"])
		VerifyGot(t, "c", form["b4"])
		VerifyGot(t, "a&b", form["testdataencoded2"])
		testdataraw := fmt.Sprintf("%v", form["testdataraw"])
		if !strings.HasPrefix(testdataraw, "@") {
			t.Errorf("testdataraw was %q - should start with @ - it should be the EXACT value, no @file support", testdataraw)
		}
	}
	testRun.Run()
}
