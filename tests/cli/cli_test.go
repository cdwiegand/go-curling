package clitests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

func Test_GetWithQuery_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/get?test=one", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			curltest.VerifyGot(t, "one", args["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_Headers_Cmdline(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/headers", "-H", "X-Hello: World", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "headers")
			args := json["headers"].(map[string]interface{})
			curltest.VerifyGot(t, "World", args["X-Hello"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_MultipleUrlsOnCmdLine(t *testing.T) {
	expectedResult := []string{"one", "two"}

	RunCmdLineIndexed(t, 2, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", testrun.OutputFiles[0], "-o", testrun.OutputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			curltest.VerifyGot(t, expectedResult[index], args["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithInlineForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=one", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithFilesystemForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=@" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithFilesystemBinaryForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("a&b=c"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "--data-binary", "test=@" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "a", form["test"])
			curltest.VerifyGot(t, "c", form["b"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithFilesystemForm2_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "@" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=@" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			curltest.VerifyGot(t, "one", files["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm2_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm3_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=<" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm4_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "@" + testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		}, func(json map[string]interface{}) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed as -F does not support directly pulling a @file reference!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}

func Test_PostWithUpload_filesystemForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-T", testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_PutWithUpload_filesystemForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 1,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/put", "-T", testrun.InputFiles[0], "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_PutWithUpload_filesystemFilesForm_CmdLine(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	RunCmdLineIndexed(t, 2, 2,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte(expectedResult[1]), 0666)
			return []string{"https://httpbin.org/put", "-T", testrun.InputFiles[0], "https://httpbin.org/put", "-T", testrun.InputFiles[1], "-o", testrun.OutputFiles[0], "-o", testrun.OutputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, expectedResult[index], data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_Delete_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			// no error means success, it's delete, there's no real response other than a success code
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_RawishForm_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"-o", testrun.OutputFiles[0], "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
		}, func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"]
			curltest.VerifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_GetWithCookies_CmdLine(t *testing.T) {
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/cookies", "-b", "testcookie2=value2", "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "value2", cookies["testcookie2"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-c", cookieFile, "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
	RunCmdLine(t, 1, 0,
		func(testrun *curltest.TestRun) []string {
			return []string{"https://httpbin.org/cookies", "-c", cookieFile, "-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}

func Test_CannotMixDataFormUploadArgs(t *testing.T) {
	RunCmdLine(t, 1, 2,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST",
				"-d", "@" + testrun.InputFiles[0],
				"-T", "@" + testrun.InputFiles[1],
				"-o", testrun.OutputFiles[0]}
		}, func(json map[string]interface{}) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -T are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunCmdLine(t, 1, 2,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST",
				"-d", "@" + testrun.InputFiles[0],
				"-F", "@" + testrun.InputFiles[1],
				"-o", testrun.OutputFiles[0]}
		}, func(json map[string]interface{}) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -F are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunCmdLine(t, 1, 2,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST",
				"-F", "@" + testrun.InputFiles[0],
				"-T", "@" + testrun.InputFiles[1],
				"-o", testrun.OutputFiles[0]}
		}, func(json map[string]interface{}) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -F and -T are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}

func Test_All4DataArgs(t *testing.T) {
	RunCmdLine(t, 1, 6,
		func(testrun *curltest.TestRun) []string {
			os.WriteFile(testrun.InputFiles[0], []byte("testdatastandard=a&b1=c"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("testdatabinary=a&b2=c"), 0666)
			os.WriteFile(testrun.InputFiles[2], []byte("testdataencoded=a&b"), 0666)
			os.WriteFile(testrun.InputFiles[3], []byte("a&b3=c"), 0666)
			os.WriteFile(testrun.InputFiles[4], []byte("a&b4=c"), 0666)
			os.WriteFile(testrun.InputFiles[5], []byte("a&b"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST",
				"-d", "@" + testrun.InputFiles[0],
				"--data-binary", "@" + testrun.InputFiles[1],
				"--data-urlencode", "@" + testrun.InputFiles[2],
				"--data", "testdatastandard2=@" + testrun.InputFiles[3],
				"--data-binary", "testdatabinary2=@" + testrun.InputFiles[4],
				"--data-urlencode", "testdataencoded2=@" + testrun.InputFiles[5],
				"--data-raw", "testdataraw=@" + testrun.InputFiles[5], // actual file not used, just want to make sure the "@" comes across properly
				"-o", testrun.OutputFiles[0]}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "a", form["testdatastandard"])
			curltest.VerifyGot(t, "c", form["b1"])
			curltest.VerifyGot(t, "a&b", form["testdataencoded"])
			curltest.VerifyGot(t, "a", form["testdatastandard2"])
			curltest.VerifyGot(t, "c", form["b3"])
			curltest.VerifyGot(t, "a", form["testdatabinary2"])
			curltest.VerifyGot(t, "c", form["b4"])
			curltest.VerifyGot(t, "a&b", form["testdataencoded2"])
			testdataraw := fmt.Sprintf("%v", form["testdataraw"])
			if !strings.HasPrefix(testdataraw, "@") {
				t.Errorf("testdataraw was %q - should start with @ - it should be the EXACT value, no @file support", testdataraw)
			}
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
