package curltestharness

import (
	"fmt"
	"os"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	"github.com/stretchr/testify/assert"
)

func Test_All4DataArgs_Context(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.SkipCompareJsonToRealCurl = true // we will test below, json contents will differ due to file paths
	testRun.ContextBuilder = func(testrun *TestRun) *curl.CurlContext {
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatastandardfile=a&b1=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatabinaryfile=a&b2=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdataencodedfile=a&b"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b3=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b4=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			HttpVerb:      "POST",
			BodyOutput:    testRun.GetOneOutputFiles(),
			Data_Standard: []string{"@" + testrun.ListInputFiles[0], "testdatastandardinline=@" + testrun.ListInputFiles[3]},
			Data_Binary:   []string{"@" + testrun.ListInputFiles[1], "testdatabinaryinline=@" + testrun.ListInputFiles[4]},
			Data_Encoded:  []string{"@" + testrun.ListInputFiles[2], "testdataencodedinline@" + testrun.ListInputFiles[5]},
			Data_RawAsIs:  []string{"testdataraw=@" + testrun.ListInputFiles[5]}, // actual file not used, just want to make sure the "@" comes across properly
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		t := testrun.Testing

		assert.NotNil(t, json["form"])
		form := json["form"].(map[string]any)
		assert.EqualValues(t, "a", form["testdatastandardfile"])
		assert.EqualValues(t, "a", form["testdatabinaryfile"])
		assert.EqualValues(t, "", form["testdataencodedfile=a&b"]) // I disagree that curl is supposed to do this, but it DOES on my machine (curl 7.81.0)
		assert.True(t, form["testdatastandardinline"].(string)[0:1] == "@")
		assert.True(t, form["testdatabinaryinline"].(string)[0:1] == "@")
		assert.EqualValues(t, "a&b", form["testdataencodedinline"])
		assert.EqualValues(t, "c", form["b1"])
		assert.EqualValues(t, "c", form["b2"])
		assert.Nil(t, form["b3"])
		assert.Nil(t, form["b4"])
		// no b3, b4, they are false flags
		testdataraw := fmt.Sprintf("%v", form["testdataraw"])
		if !strings.HasPrefix(testdataraw, "@") {
			t.Errorf("testdataraw was %q - should start with @ - it should be the EXACT value, no @file support", testdataraw)
		}
	}
	testRun.RunTestRun()
}

func Test_All4DataArgs_CmdLine(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-d", "testdatastandardfile=a",
			"-d", "b1=c",
			"--data-binary", "testdatabinaryfile=a",
			"--data-binary", "b2=c",
			"--data-urlencode", "testdataencodedfile=a&b",
			"--data", "testdatastandardinline=a",
			"--data", "b3=c",
			"--data-binary", "testdatabinaryinline=a",
			"--data-binary", "b4=c",
			"--data-urlencode", "testdataencodedinline=a&b",
			"--data-raw", "testdataraw=@/1/2/3", // actual file not used, just want to make sure the "@" comes across properly
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		t := testrun.Testing

		assert.NotNil(t, json["form"])
		form := json["form"].(map[string]any)
		assert.EqualValues(t, "a", form["testdatastandardfile"])
		assert.EqualValues(t, "a", form["testdatabinaryfile"])
		assert.EqualValues(t, "a&b", form["testdataencodedfile"])
		assert.EqualValues(t, "a", form["testdatastandardinline"])
		assert.EqualValues(t, "a", form["testdatabinaryinline"])
		assert.EqualValues(t, "a&b", form["testdataencodedinline"])
		assert.EqualValues(t, "c", form["b1"])
		assert.EqualValues(t, "c", form["b2"])
		assert.EqualValues(t, "c", form["b3"])
		assert.EqualValues(t, "c", form["b4"])
		testdataraw := fmt.Sprintf("%v", form["testdataraw"])
		if !strings.HasPrefix(testdataraw, "@") {
			t.Errorf("testdataraw was %q - should start with @ - it should be the EXACT value, no @file support", testdataraw)
		}
	}
	testRun.RunTestRun()

	ctx, _, cerr := testRun.GetTestRunReady()
	if cerr != nil {
		testRun.ErrorHandler(cerr, testRun)
	}

	bodyData, cerr := ctx.HandleDataArgs(ctx.ConvertPostFormIntoGet)
	if cerr != nil {
		testRun.ErrorHandler(cerr, testRun)
	}

	got := bodyData.String()
	// parts may not always be in a specific order
	lines := strings.Split(got, "&")
	assert.Contains(t, lines, "testdatastandardfile=a")
	assert.Contains(t, lines, "testdatabinaryfile=a")
	assert.Contains(t, lines, "testdataencodedfile=a%26b")
	assert.Contains(t, lines, "testdatastandardinline=a")
	assert.Contains(t, lines, "testdatabinaryinline=a")
	assert.Contains(t, lines, "testdataencodedinline=a%26b")
	assert.Contains(t, lines, "b1=c")
	assert.Contains(t, lines, "b2=c")
	assert.Contains(t, lines, "b3=c")
	assert.Contains(t, lines, "b4=c")
	found := false
	for _, h := range lines {
		if strings.Contains(h, "testdataraw=@/") {
			found = true
		}
	}
	assert.True(t, found)
}

func Test_All4DataArgs_CmdLine2(t *testing.T) {
	testRun := BuildTestRun(t)
	testRun.SkipCompareJsonToRealCurl = true // we will test below, json contents will differ due to file paths
	testRun.CmdLineBuilder = func(testrun *TestRun) []string {
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatastandardfile=a&b1=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdatabinaryfile=a&b2=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("testdataencodedfile=a&b"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b3=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b4=c"), 0666)
		os.WriteFile(testRun.GetNextInputFile(), []byte("a&b"), 0666)
		return []string{
			"https://httpbin.org/post", "-X", "POST",
			"-d", "@" + testrun.ListInputFiles[0],
			"--data-binary", "@" + testrun.ListInputFiles[1],
			"--data-urlencode", "@" + testrun.ListInputFiles[2],
			"--data", "testdatastandardinline=@" + testrun.ListInputFiles[3],
			"--data-binary", "testdatabinaryinline=@" + testrun.ListInputFiles[4],
			"--data-urlencode", "testdataencodedinline@" + testrun.ListInputFiles[5],
			"--data-raw", "testdataraw=@" + testrun.ListInputFiles[5], // actual file not used, just want to make sure the "@" comes across properly
			"-o", testrun.GetOneOutputFile(),
		}
	}
	testRun.SuccessHandler = func(json map[string]interface{}, testrun *TestRun) {
		t := testrun.Testing

		assert.NotNil(t, json["form"])
		form := json["form"].(map[string]any)
		assert.EqualValues(t, "a", form["testdatastandardfile"])
		assert.EqualValues(t, "a", form["testdatabinaryfile"])
		assert.EqualValues(t, "", form["testdataencodedfile=a&b"]) // I disagree that curl is supposed to do this, but it DOES on my machine (curl 7.81.0)
		assert.True(t, form["testdatastandardinline"].(string)[0:1] == "@")
		assert.True(t, form["testdatabinaryinline"].(string)[0:1] == "@")
		assert.EqualValues(t, "a&b", form["testdataencodedinline"])
		assert.EqualValues(t, "c", form["b1"])
		assert.EqualValues(t, "c", form["b2"])
		assert.Nil(t, form["b3"]) // false flag
		assert.Nil(t, form["b4"]) // false flag
		assert.True(t, form["testdataraw"].(string)[0:1] == "@")
	}
	testRun.RunTestRun()
}
