package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// helper functions
func verifyGot(t *testing.T, name string, args string, wanted any, got any) {
	if got != wanted {
		t.Errorf("%v failed, got %q wanted %q for %q", name, got, wanted, args)
	}
}
func verifyJson(json map[string]interface{}, arg string) {
	if json[arg] == nil {
		panic(fmt.Errorf("%v was not present in json response", arg))
	}
}

func readJson(file string) (res map[string]interface{}) {
	jsonFile, err := os.Open(file)
	PanicIfError(err)
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	PanicIfError(err)
	json.Unmarshal([]byte(byteValue), &res)
	return
}

func helpRun(t *testing.T, contextBuilder func(outputFile string) (ctx *CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	tempFile := filepath.Join(tmpDir, "1.json")

	ctx := contextBuilder(tempFile)
	helpRun_Inner(ctx, handler, tempFile)
}
func helpRun_2file(t *testing.T, contextBuilder func(outputFile string, secondFile string) (ctx *CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	tempFile := filepath.Join(tmpDir, "1.json")
	tempFile2 := filepath.Join(tmpDir, "2.json")

	ctx := contextBuilder(tempFile, tempFile2)
	helpRun_Inner(ctx, handler, tempFile)
}
func helpRun_Inner(ctx *CurlContext, handler func(map[string]interface{}), tempFile string) {
	SetupContextForRun(ctx)
	client := BuildClient(ctx)
	request := BuildRequest(ctx)
	resp, err := client.Do(request)
	ProcessResponse(ctx, resp, err, request)

	json := readJson(tempFile)
	handler(json)
}

// Actual tests
func Test_GetWithQuery(t *testing.T) {
	helpRun(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl: "https://httpbin.org/get?test=one",
			output: tempFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "args")
		args := json["args"].(map[string]any)
		verifyGot(t, "GetWithQuery", "", "one", args["test"])
	})
}

func Test_PostWithInlineForm(t *testing.T) {
	helpRun(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl:       "https://httpbin.org/post",
			method:       "POST",
			output:       outputFile,
			form_encoded: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "PostWithInlineForm", "", "one", form["test"])
	})
}
func Test_PostWithFilesystemForm(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("one"), 0666)
		return &CurlContext{
			theUrl:       "https://httpbin.org/post",
			method:       "POST",
			output:       outputFile,
			form_encoded: []string{"test=@" + tempFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "PostWithFilesystemForm", "", "one", form["test"])
	})
}
func Test_PostWithFilesystemForm2(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("test=one"), 0666)
		return &CurlContext{
			theUrl:       "https://httpbin.org/post",
			method:       "POST",
			output:       outputFile,
			form_encoded: []string{"@" + tempFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "PostWithFilesystemForm2", "", "one", form["test"])
	})
}
func Test_PostWithMultipartInlineForm(t *testing.T) {
	helpRun(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl:         "https://httpbin.org/post",
			method:         "POST",
			output:         outputFile,
			form_multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "PostWithMultipartInlineForm", "", "one", form["test"])
	})
}
func Test_PostWithMultipartFilesystemForm(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("one"), 0666)
		return &CurlContext{
			theUrl:         "https://httpbin.org/post",
			method:         "POST",
			output:         outputFile,
			form_multipart: []string{"test=@" + tempFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "files")
		files := json["files"].(map[string]any)
		verifyGot(t, "PostWithMultipartFilesystemForm", "", "one", files["test"])
	})
}
func Test_PostWithMultipartFilesystemForm2(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("test=one"), 0666)
		return &CurlContext{
			theUrl:         "https://httpbin.org/post",
			method:         "POST",
			output:         outputFile,
			form_multipart: []string{"@" + tempFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "PostWithMultipartFilesystemForm2", "", "one", form["test"])
	})
}

func Test_PostWithUploadFilesystemForm(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("test=one"), 0666)
		return &CurlContext{
			theUrl:     "https://httpbin.org/post",
			method:     "POST",
			output:     outputFile,
			uploadFile: tempFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "data")
		data := json["data"].(string)
		verifyGot(t, "PostWithUploadFilesystemForm", "", "test=one", data)
	})
}
func Test_PutWithUploadFilesystemForm(t *testing.T) {
	helpRun_2file(t, func(outputFile string, tempFile string) *CurlContext {
		os.WriteFile(tempFile, []byte("test=one"), 0666)
		return &CurlContext{
			theUrl:     "https://httpbin.org/put",
			method:     "PUT",
			output:     outputFile,
			uploadFile: tempFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "data")
		data := json["data"].(string)
		verifyGot(t, "PostWithUploadFilesystemForm", "", "test=one", data)
	})
}

func Test_Delete(t *testing.T) {
	helpRun(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl: "https://httpbin.org/delete",
			method: "DELETE",
			output: outputFile,
		}
	}, func(json map[string]interface{}) {
		// no error means success, it's delete, there's no real response other than a success code
	})
}

func Test_GetWithCookies(t *testing.T) {
	helpRun(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:  "https://httpbin.org/cookies",
			output:  tempFile,
			cookies: []string{"testcookie2=value2"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "GetWithCookies", "", "value2", cookies["testcookie2"])
	})
}

func Test_CookieRoundTrip(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	helpRun(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:    "https://httpbin.org/cookies/set/testcookie/testvalue",
			output:    tempFile,
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "GetWithCookies", "", "testvalue", cookies["testcookie"])
	})

	helpRun(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:    "https://httpbin.org/cookies",
			output:    tempFile,
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "GetWithCookies", "", "testvalue", cookies["testcookie"])
	})
}

func Test_standardizeFileRef(t *testing.T) {
	got := standardizeFileRef("/dev/null")
	verifyGot(t, "standardizeFileRef", "/dev/null", "/dev/null", got)
	got = standardizeFileRef("null")
	verifyGot(t, "standardizeFileRef", "null", "/dev/null", got)
	got = standardizeFileRef("")
	verifyGot(t, "standardizeFileRef", "", "/dev/null", got)

	got = standardizeFileRef("/dev/stdout")
	verifyGot(t, "standardizeFileRef", "/dev/stdout", "/dev/stdout", got)
	got = standardizeFileRef("stdout")
	verifyGot(t, "standardizeFileRef", "stdout", "/dev/stdout", got)
	got = standardizeFileRef("-")
	verifyGot(t, "standardizeFileRef", "-", "/dev/stdout", got)

	got = standardizeFileRef("/dev/stderr")
	verifyGot(t, "standardizeFileRef", "/dev/stderr", "/dev/stderr", got)
	got = standardizeFileRef("stderr")
	verifyGot(t, "standardizeFileRef", "stderr", "/dev/stderr", got)

	got = standardizeFileRef("/boo")
	verifyGot(t, "standardizeFileRef", "/boo", "/boo", got)
}
