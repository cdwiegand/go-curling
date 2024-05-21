package contexttests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

func Test_GetWithQuery_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/get?test=one"},
			Output: testrun.OutputFiles,
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "args")
		args := json["args"].(map[string]any)
		curltest.VerifyGot(t, "one", args["test"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_Headers_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:    []string{"https://httpbin.org/headers"},
			Headers: []string{"X-Hello: World"},
			Output:  testrun.OutputFiles,
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "headers")
		args := json["headers"].(map[string]interface{})
		curltest.VerifyGot(t, "World", args["X-Hello"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithInlineForm_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        testrun.OutputFiles,
			Data_Standard: []string{"test=one"},
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(testrun *curltest.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        testrun.OutputFiles,
			Data_Standard: []string{"test=@" + testrun.InputFiles[0]},
		}
	}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemBinaryForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(testrun *curltest.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.InputFiles[0], []byte("a&b=c"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/post"},
			Method:      "POST",
			Output:      testrun.OutputFiles,
			Data_Binary: []string{"test=@" + testrun.InputFiles[0]},
		}
	}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "a", form["test"])
		curltest.VerifyGot(t, "c", form["b"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemForm2_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(testrun *curltest.TestRun) *curl.CurlContext {
		os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        testrun.OutputFiles,
			Data_Standard: []string{"@" + testrun.InputFiles[0]},
		}
	}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			Method:         "POST",
			Output:         testrun.OutputFiles,
			Form_Multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_PostWithMultipartForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Form_Multipart: []string{"test=@" + testrun.InputFiles[0]},
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			curltest.VerifyGot(t, "one", files["test"])
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm2_CurlContext(t *testing.T) {
	RunContext(t,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Form_Multipart: []string{"test=one"},
			}
		}, func(json map[string]interface{}, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm3_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Form_Multipart: []string{"test=<" + testrun.InputFiles[0]},
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartFormRaw3_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:              []string{"https://httpbin.org/post"},
				Method:            "POST",
				Output:            testrun.OutputFiles,
				Form_MultipartRaw: []string{"test=<" + testrun.InputFiles[0]},
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "<"+testrun.InputFiles[0], form["test"])
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm4_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Form_Multipart: []string{"@" + testrun.InputFiles[0]},
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed as -F does not support directly pulling a @file reference!"))
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}
func Test_PostWithUpload_filesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/post"},
				Method:      "POST",
				Output:      testrun.OutputFiles,
				Upload_File: testrun.InputFiles,
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PutWithUpload_filesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/put"},
				Output:      testrun.OutputFiles,
				Upload_File: testrun.InputFiles,
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PutWithUpload_filesystemFilesForm_CurlContext(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	RunContextWithTempFile(t, 2, 2,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte(expectedResult[1]), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/put", "https://httpbin.org/put"},
				Method:      "PUT",
				Output:      testrun.OutputFiles,
				Upload_File: testrun.InputFiles,
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, expectedResult[index], data)
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_Delete_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/delete"},
			Method: "DELETE",
			Output: testrun.OutputFiles,
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		// no error means success, it's delete, there's no real response other than a success code
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_GetWithCookies_CurlContext(t *testing.T) {
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:    []string{"https://httpbin.org/cookies"},
			Output:  testrun.OutputFiles,
			Cookies: []string{"testcookie2=value2"},
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "value2", cookies["testcookie2"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_CookieRoundTrip_CurlContext(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:      []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			Output:    testrun.OutputFiles,
			CookieJar: cookieFile,
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})

	RunContext(t, func(testrun *curltest.TestRun) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:      []string{"https://httpbin.org/cookies"},
			Output:    testrun.OutputFiles,
			CookieJar: cookieFile,
		}
	}, func(json map[string]interface{}, testrun *curltest.TestRun) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
	}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_CannotMixDataFormUploadArgs(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:          []string{"https://httpbin.org/post"},
				Method:        "POST",
				Output:        testrun.OutputFiles,
				Data_Standard: []string{"test=one"},
				Upload_File:   testrun.InputFiles,
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -T are mixed!"))
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunContextWithTempFile(t, 1, 1,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Data_Standard:  []string{"test=one"},
				Form_Multipart: testrun.InputFiles,
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -F are mixed!"))
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunContextWithTempFile(t, 1, 2,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("test=one"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         testrun.OutputFiles,
				Upload_File:    []string{testrun.InputFiles[0]},
				Form_Multipart: []string{testrun.InputFiles[1]},
			}
		}, func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -F and -T are mixed!"))
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}

func Test_All4DataArgs(t *testing.T) {
	RunContextWithTempFile(t, 1, 6,
		func(testrun *curltest.TestRun) *curl.CurlContext {
			os.WriteFile(testrun.InputFiles[0], []byte("testdatastandard=a&b1=c"), 0666)
			os.WriteFile(testrun.InputFiles[1], []byte("testdatabinary=a&b2=c"), 0666)
			os.WriteFile(testrun.InputFiles[2], []byte("testdataencoded=a&b"), 0666)
			os.WriteFile(testrun.InputFiles[3], []byte("a&b3=c"), 0666)
			os.WriteFile(testrun.InputFiles[4], []byte("a&b4=c"), 0666)
			os.WriteFile(testrun.InputFiles[5], []byte("a&b"), 0666)
			return &curl.CurlContext{
				Urls:          []string{"https://httpbin.org/post"},
				Method:        "POST",
				Output:        testrun.OutputFiles,
				Data_Standard: []string{"@" + testrun.InputFiles[0], "testdatastandard2=@" + testrun.InputFiles[3]},
				Data_Binary:   []string{"@" + testrun.InputFiles[1], "testdatabinary2=@" + testrun.InputFiles[4]},
				Data_Encoded:  []string{"@" + testrun.InputFiles[2], "testdataencoded2=@" + testrun.InputFiles[5]},
				Data_RawAsIs:  []string{"testdataraw=@" + testrun.InputFiles[5]}, // actual file not used, just want to make sure the "@" comes across properly
			}
		},
		func(json map[string]interface{}, index int, testrun *curltest.TestRun) {
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
		}, func(err *curlerrors.CurlError, testrun *curltest.TestRun) {
			curltest.GenericErrorHandler(t, err)
		})
}
