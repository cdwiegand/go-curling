package contexttests

import (
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
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
	})
}
func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	RunContext(t, func(outputFile string) *curl.CurlContext {
		return &curl.CurlContext{
			Urls:           []string{"https://httpbin.org/post"},
			Method:         "POST",
			Output:         []string{outputFile},
			Data_multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		curltest.VerifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		curltest.VerifyGot(t, "one", form["test"])
	})
}
func Test_PostWithMultipartFilesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Data_multipart: []string{"test=@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			curltest.VerifyGot(t, "one", files["test"])
		})
}
func Test_PostWithMultipartFilesystemForm2_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:           []string{"https://httpbin.org/post"},
				Method:         "POST",
				Output:         outputFiles,
				Data_multipart: []string{"@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}
func Test_PostWithUploadFilesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:       []string{"https://httpbin.org/post"},
				Method:     "POST",
				Output:     outputFiles,
				UploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		})
}
func Test_PutWithUploadFilesystemForm_CurlContext(t *testing.T) {
	RunContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &curl.CurlContext{
				Urls:       []string{"https://httpbin.org/put"},
				Output:     outputFiles,
				UploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		})
}
func Test_PutWithUploadFilesystemFilesForm_CurlContext(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	RunContextWithTempFile(t, 2, 2,
		func(outputFiles []string, tempFiles []string) *curl.CurlContext {
			os.WriteFile(tempFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(tempFiles[1], []byte(expectedResult[1]), 0666)
			return &curl.CurlContext{
				Urls:       []string{"https://httpbin.org/put", "https://httpbin.org/put"},
				Method:     "PUT",
				Output:     outputFiles,
				UploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, expectedResult[index], data)
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
	})
}
