package clitests

import (
	"os"
	"path/filepath"
	"testing"

	curltest "github.com/cdwiegand/go-curling/tests"
)

func Test_GetWithQuery_CmdLine(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/get?test=one", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			curltest.VerifyGot(t, "one", args["test"])
		})
}

func Test_Headers_Cmdline(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/headers", "-H", "X-Hello: World", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "headers")
			args := json["headers"].(map[string]interface{})
			curltest.VerifyGot(t, "World", args["X-Hello"])
		})
}

func Test_MultipleUrlsOnCmdLine(t *testing.T) {
	expectedResult := []string{"one", "two"}

	RunCmdLineWithTempFile(t, 2, 0,
		func(outputFiles []string, tempFiles []string) []string {
			return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", outputFiles[0], "-o", outputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			curltest.VerifyGot(t, expectedResult[index], args["test"])
		})
}
func Test_PostWithInlineForm_CmdLine(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=one", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm2_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartForm_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			curltest.VerifyGot(t, "one", files["test"])
		})
}
func Test_PostWithMultipartForm2_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			curltest.VerifyGot(t, "one", form["test"])
		})
}

func Test_PostWithUploadFilesystemForm_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-T", tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		})
}

func Test_PutWithUploadFilesystemForm_CmdLine(t *testing.T) {
	RunCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/put", "-T", tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, "test=one", data)
		})
}

func Test_PutWithUploadFilesystemFilesForm_CmdLine(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	RunCmdLineWithTempFile(t, 2, 2,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(tempFiles[1], []byte(expectedResult[1]), 0666)
			return []string{"https://httpbin.org/put", "-T", tempFiles[0], "https://httpbin.org/put", "-T", tempFiles[1], "-o", outputFiles[0], "-o", outputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"].(string)
			curltest.VerifyGot(t, expectedResult[index], data)
		})
}

func Test_Delete_CmdLine(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			// no error means success, it's delete, there's no real response other than a success code
		})
}

func Test_RawishForm_CmdLine(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"-o", outputFile, "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
		}, func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "data")
			data := json["data"]
			curltest.VerifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
		})
}

func Test_GetWithCookies_CmdLine(t *testing.T) {
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies", "-b", "testcookie2=value2", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "value2", cookies["testcookie2"])
		})
}

func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-c", cookieFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
		})
	RunCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies", "-c", cookieFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			curltest.VerifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			curltest.VerifyGot(t, "testvalue", cookies["testcookie"])
		})
}
