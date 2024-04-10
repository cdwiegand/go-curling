package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
)

// helper functions
func verifyGot(t *testing.T, wanted any, got any) {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
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

func runCmdLine(t *testing.T, argsBuilder func(outputFile string) []string, handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.json")
	args := argsBuilder(outputFile)

	ctx := &CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := strings.Join(flags.Args(), " ")

	SetupContextForRun(ctx, extraArgs)
	helpRun_Inner(ctx, handler, outputFile)
}
func runCmdLineWithTempFile(t *testing.T, argsBuilder func(outputFile string, tempFile string) []string, handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.json")
	tempFile := filepath.Join(tmpDir, "2.json")
	args := argsBuilder(outputFile, tempFile)

	ctx := &CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := strings.Join(flags.Args(), " ")

	SetupContextForRun(ctx, extraArgs)
	helpRun_Inner(ctx, handler, outputFile)
}
func runContext(t *testing.T, contextBuilder func(outputFile string) (ctx *CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.json")

	ctx := contextBuilder(outputFile)
	SetupContextForRun(ctx, "")
	helpRun_Inner(ctx, handler, outputFile)
}
func runContextWithTempFile(t *testing.T, contextBuilder func(outputFile string, secondFile string) (ctx *CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.json")
	tempFile := filepath.Join(tmpDir, "2.json")

	ctx := contextBuilder(outputFile, tempFile)
	SetupContextForRun(ctx, "")
	helpRun_Inner(ctx, handler, outputFile)
}

func helpRun_Inner(ctx *CurlContext, handler func(map[string]interface{}), tempFile string) {
	client := BuildClient(ctx)
	request := BuildRequest(ctx)
	resp, err := client.Do(request)
	ProcessResponse(ctx, resp, err, request)

	json := readJson(tempFile)
	handler(json)
}

// Actual tests
func Test_GetWithQuery(t *testing.T) {
	runContext(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl: "https://httpbin.org/get?test=one",
			output: tempFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "args")
		args := json["args"].(map[string]any)
		verifyGot(t, "one", args["test"])
	})
}
func Test_GetWithQuery_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/get?test=one", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "args")
			args := json["args"].(map[string]any)
			verifyGot(t, "one", args["test"])
		})
}

func Test_PostWithInlineForm(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl:       "https://httpbin.org/post",
			method:       "POST",
			output:       outputFile,
			form_encoded: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithInlineForm_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=one", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=@" + tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm2(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithFilesystemForm2_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "@" + tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartInlineForm(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl:         "https://httpbin.org/post",
			method:         "POST",
			output:         outputFile,
			form_multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartFilesystemForm(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "one", files["test"])
	})
}
func Test_PostWithMultipartForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=@" + tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "files")
			files := json["files"].(map[string]any)
			verifyGot(t, "one", files["test"])
		})
}
func Test_PostWithMultipartFilesystemForm2(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithMultipartForm2_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "@" + tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}

func Test_PostWithUploadFilesystemForm(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "test=one", data)
	})
}
func Test_PostWithUploadFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-T", tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}

func Test_PutWithUploadFilesystemForm(t *testing.T) {
	runContextWithTempFile(t, func(outputFile string, tempFile string) *CurlContext {
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
		verifyGot(t, "test=one", data)
	})
}
func Test_PutWithUploadFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t,
		func(outputFile string, tempFile string) []string {
			os.WriteFile(tempFile, []byte("test=one"), 0666)
			return []string{"https://httpbin.org/put", "-X", "PUT", "-T", tempFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}

func Test_Delete(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			theUrl: "https://httpbin.org/delete",
			method: "DELETE",
			output: outputFile,
		}
	}, func(json map[string]interface{}) {
		// no error means success, it's delete, there's no real response other than a success code
	})
}
func Test_Delete_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/delete", "-X", "DELETE", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			// no error means success, it's delete, there's no real response other than a success code
		})
}

func Test_GetWithCookies(t *testing.T) {
	runContext(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:  "https://httpbin.org/cookies",
			output:  tempFile,
			cookies: []string{"testcookie2=value2"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "value2", cookies["testcookie2"])
	})
}
func Test_GetWithCookies_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies", "-b", "testcookie2=value2", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "value2", cookies["testcookie2"])
		})
}

func Test_CookieRoundTrip(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	runContext(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:    "https://httpbin.org/cookies/set/testcookie/testvalue",
			output:    tempFile,
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "testvalue", cookies["testcookie"])
	})

	runContext(t, func(tempFile string) *CurlContext {
		return &CurlContext{
			theUrl:    "https://httpbin.org/cookies",
			output:    tempFile,
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "testvalue", cookies["testcookie"])
	})
}
func Test_CookieRoundTrip_CmdLine(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies/set/testcookie/testvalue", "-c", cookieFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "testvalue", cookies["testcookie"])
		})
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies", "-c", cookieFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "testvalue", cookies["testcookie"])
		})
}
