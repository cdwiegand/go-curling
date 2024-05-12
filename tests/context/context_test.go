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
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/get?test=one"},
			Output: []string{outputFile},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "args")
		args := json["args"].(map[string]any)
		curltest.VerifyGot(t, "one", args["test"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_Headers_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:    []string{"https://httpbin.org/headers"},
			Headers: []string{"X-Hello: World"},
			Output:  []string{outputFile},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "headers")
		args := json["headers"].(map[string]interface{})
		curltest.VerifyGot(t, "World", args["X-Hello"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithInlineForm_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        []string{outputFile},
			Data_standard: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(outputFiles []string, tempFiles []string) *curl.CurlContext {
		os.WriteFile(tempFiles[0], []byte("one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        outputFiles,
			Data_standard: []string{"test=@" + tempFiles[0]},
		}
	}, func(json map[string]interface{}, index int) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemBinaryForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(outputFiles []string, tempFiles []string) *curl.CurlContext {
		os.WriteFile(tempFiles[0], []byte("a&b=c"), 0666)
		return &curl.CurlContext{
			Urls:        []string{"https://httpbin.org/post"},
			Method:      "POST",
			Output:      outputFiles,
			Data_binary: []string{"test=@" + tempFiles[0]},
		}
	}, func(json map[string]interface{}, index int) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "a", form["test"])
		curltest.VerifyGot(t, "c", form["b"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_PostWithFilesystemForm2_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1, func(outputFiles []string, tempFiles []string) *curl.CurlContext {
		os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
		return &curl.CurlContext{
			Urls:          []string{"https://httpbin.org/post"},
			Method:        "POST",
			Output:        outputFiles,
			Data_standard: []string{"@" + tempFiles[0]},
		}
	}, func(json map[string]interface{}, index int) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			Method:         "POST",
			Output:         []string{outputFile},
			Form_multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_PostWithMultipartForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Form_multipart: []string{"test=@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			curltest.VerifyGot(t, "one", files["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm2_CurlContext(t *testing.T) {
	RunContext(t,
		func(outputFile string) *curl.CurlContext {
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         []string{outputFile},
				Form_multipart: []string{"test=one"},
			}
		}, func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm3_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Form_multipart: []string{"test=<" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PostWithMultipartForm4_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Form_multipart: []string{"@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed as -F does not support directly pulling a @file reference!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}
func Test_PostWithUpload_filesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/post"},
				Method:      "POST",
				Output:      outputFiles,
				Upload_file: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PutWithUpload_filesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/put"},
				Output:      outputFiles,
				Upload_file: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_PutWithUpload_filesystemFilesForm_CurlContext(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	RunContextWithTempFile(t, 2, 2,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(tempFiles[1], []byte(expectedResult[1]), 0666)
			return &curl.CurlContext{
				Urls:        []string{"https://httpbin.org/put", "https://httpbin.org/put"},
				Method:      "PUT",
				Output:      outputFiles,
				Upload_file: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, expectedResult[index], data)
		}, func(err *curlerrors.CurlError) {
			curltest.GenericErrorHandler(t, err)
		})
}
func Test_Delete_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:   []string{"https://httpbin.org/delete"},
			Method: "DELETE",
			Output: []string{outputFile},
		}
	}, func(json map[string]interface{}) {
		// no error means success, it's delete, there's no real response other than a success code
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_GetWithCookies_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:    []string{"https://httpbin.org/cookies"},
			Output:  []string{outputFile},
			Cookies: []string{"testcookie2=value2"},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "value2", cookies["testcookie2"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}
func Test_CookieRoundTrip_CurlContext(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:      []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			Output:    []string{outputFile},
			CookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})

	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:      []string{"https://httpbin.org/cookies"},
			Output:    []string{outputFile},
			CookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
	}, func(err *curlerrors.CurlError) {
		curltest.GenericErrorHandler(t, err)
	})
}

func Test_CannotMixDataFormUploadArgs(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:          []string{"https://httpbin.org/post"},
				Method:        "POST",
				Output:        outputFiles,
				Data_standard: []string{"test=one"},
				Upload_file:   tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -T are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Data_standard:  []string{"test=one"},
				Form_multipart: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -d and -F are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
	RunContextWithTempFile(t, 1, 2,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			os.WriteFile(tempFiles[1], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Upload_file:    []string{tempFiles[0]},
				Form_multipart: []string{tempFiles[1]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.GenericErrorHandler(t, curlerrors.NewCurlError0("Should not succeed if -F and -T are mixed!"))
		}, func(err *curlerrors.CurlError) {
			// ok, it SHOULD fail, this is not a valid request!
		})
}

func Test_All4DataArgs(t *testing.T) {
	RunContextWithTempFile(t, 1, 6,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("testdatastandard=a&b1=c"), 0666)
			os.WriteFile(tempFiles[1], []byte("testdatabinary=a&b2=c"), 0666)
			os.WriteFile(tempFiles[2], []byte("testdataencoded=a&b"), 0666)
			os.WriteFile(tempFiles[3], []byte("a&b3=c"), 0666)
			os.WriteFile(tempFiles[4], []byte("a&b4=c"), 0666)
			os.WriteFile(tempFiles[5], []byte("a&b"), 0666)
			return &curl.CurlContext{
				Urls:          []string{"https://httpbin.org/post"},
				Method:        "POST",
				Output:        outputFiles,
				Data_standard: []string{"@" + tempFiles[0], "testdatastandard2=@" + tempFiles[3]},
				Data_binary:   []string{"@" + tempFiles[1], "testdatabinary2=@" + tempFiles[4]},
				Data_encoded:  []string{"@" + tempFiles[2], "testdataencoded2=@" + tempFiles[5]},
				Data_rawasis:  []string{"testdataraw=@" + tempFiles[5]}, // actual file not used, just want to make sure the "@" comes across properly
			}
		},
		func(json map[string]interface{}, index int) {
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
