package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	flag "github.com/spf13/pflag"
)

// helper functions
func verifyGot(t *testing.T, wanted any, got any) {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
	}
}
func verifyJson(t *testing.T, json map[string]interface{}, arg string) {
	if json[arg] == nil {
		err := fmt.Sprintf("%v was not present in json response", arg)
		t.Errorf(err)
		panic(err)
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

func buildFileList(count int, outputDir string, ext string) (files []string) {
	files = []string{}
	for i := 0; i < count; i++ {
		files = append(files, filepath.Join(outputDir, fmt.Sprintf("%d.%s", i, ext)))
	}
	return
}
func runCmdLine(t *testing.T, argsBuilder func(outputFile string) []string, handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")
	args := argsBuilder(outputFile)

	ctx := &CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := flags.Args()

	ctx.SetupContextForRun(extraArgs)
	helpRun_Inner(ctx, handler, outputFile)
}
func runCmdLineWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, argsBuilder func(outputFiles []string, tempFiles []string) []string, handler func(map[string]interface{}, int)) {
	tmpDir := t.TempDir()

	outputFile := buildFileList(countOutputFiles, tmpDir, "out")
	for _, s := range outputFile {
		defer os.Remove(s)
	}

	tempFile := buildFileList(countTempFiles, tmpDir, "tmp")
	for _, s := range tempFile {
		defer os.Remove(s)
	}

	args := argsBuilder(outputFile, tempFile)
	ctx := &CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := flags.Args()

	ctx.SetupContextForRun(extraArgs)
	helpRun_InnerWithFiles(ctx, handler, outputFile)
}
func runContext(t *testing.T, contextBuilder func(outputFile string) (ctx *CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")

	ctx := contextBuilder(outputFile)
	ctx.SetupContextForRun([]string{})
	helpRun_Inner(ctx, handler, outputFile)
}
func runContextWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, contextBuilder func(outputFiles []string, tempFiles []string) (ctx *CurlContext), handler func(map[string]interface{}, int)) {
	tmpDir := t.TempDir()

	outputFile := buildFileList(countOutputFiles, tmpDir, "out")
	for _, s := range outputFile {
		defer os.Remove(s)
	}

	tempFile := buildFileList(countTempFiles, tmpDir, "tmp")
	for _, s := range tempFile {
		defer os.Remove(s)
	}

	ctx := contextBuilder(outputFile, tempFile)
	ctx.SetupContextForRun([]string{})
	helpRun_InnerWithFiles(ctx, handler, outputFile)
}

func helpRun_Inner(ctx *CurlContext, handler func(map[string]interface{}), outputFile string) {
	client := ctx.BuildClient()

	for index := range ctx.urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)

		json := readJson(outputFile)
		handler(json)
	}
}
func helpRun_InnerWithFiles(ctx *CurlContext, handler func(map[string]interface{}, int), outputFiles []string) {
	client := ctx.BuildClient()

	for index := range ctx.urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)

		json := readJson(outputFiles[index])
		handler(json, index)
	}
}

// Actual tests
func Test_GetWithQuery_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:   []string{"https://httpbin.org/get?test=one"},
			output: []string{outputFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "args")
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
			verifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			verifyGot(t, "one", args["test"])
		})
}

func Test_Headers_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:    []string{"https://httpbin.org/headers"},
			headers: []string{"X-Hello: World"},
			output:  []string{outputFile},
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "headers")
		args := json["headers"].(map[string]interface{})
		verifyGot(t, "World", args["X-Hello"])
	})
}
func Test_Headers_Cmdline(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/headers", "-H", "X-Hello: World", "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(t, json, "headers")
			args := json["headers"].(map[string]interface{})
			verifyGot(t, "World", args["X-Hello"])
		})
}

func Test_MultipleUrlsOnCmdLine(t *testing.T) {
	expectedResult := []string{"one", "two"}

	runCmdLineWithTempFile(t, 2, 0,
		func(outputFiles []string, tempFiles []string) []string {
			return []string{"https://httpbin.org/get?test=one", "https://httpbin.org/get?test=two", "-o", outputFiles[0], "-o", outputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "args")
			args := json["args"].(map[string]any)
			verifyGot(t, expectedResult[index], args["test"])
		})
}

func Test_PostWithInlineForm_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:          []string{"https://httpbin.org/post"},
			method:        "POST",
			output:        []string{outputFile},
			data_standard: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "form")
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
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1, func(outputFiles []string, tempFiles []string) *CurlContext {
		os.WriteFile(tempFiles[0], []byte("one"), 0666)
		return &CurlContext{
			urls:          []string{"https://httpbin.org/post"},
			method:        "POST",
			output:        outputFiles,
			data_standard: []string{"test=@" + tempFiles[0]},
		}
	}, func(json map[string]interface{}, index int) {
		verifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "test=@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithFilesystemForm2_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1, func(outputFiles []string, tempFiles []string) *CurlContext {
		os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
		return &CurlContext{
			urls:          []string{"https://httpbin.org/post"},
			method:        "POST",
			output:        outputFiles,
			data_standard: []string{"@" + tempFiles[0]},
		}
	}, func(json map[string]interface{}, index int) {
		verifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithFilesystemForm2_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-d", "@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartInlineForm_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:           []string{"https://httpbin.org/post"},
			method:         "POST",
			output:         []string{outputFile},
			data_multipart: []string{"test=one"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "form")
		form := json["form"].(map[string]any)
		verifyGot(t, "one", form["test"])
	})
}
func Test_PostWithMultipartInlineForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=one", "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartFilesystemForm_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *CurlContext {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return &CurlContext{
				urls:           []string{"https://httpbin.org/post"},
				method:         "POST",
				output:         outputFiles,
				data_multipart: []string{"test=@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			verifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			verifyGot(t, "one", files["test"])
		})
}
func Test_PostWithMultipartForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "test=@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "files")
			files := json["files"].(map[string]any)
			verifyGot(t, "one", files["test"])
		})
}
func Test_PostWithMultipartFilesystemForm2_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &CurlContext{
				urls:           []string{"https://httpbin.org/post"},
				method:         "POST",
				output:         outputFiles,
				data_multipart: []string{"@" + tempFiles[0]},
			}
		}, func(json map[string]interface{}, index int) {
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}
func Test_PostWithMultipartForm2_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-F", "@" + tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "form")
			form := json["form"].(map[string]any)
			verifyGot(t, "one", form["test"])
		})
}

func Test_PostWithUploadFilesystemForm_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &CurlContext{
				urls:       []string{"https://httpbin.org/post"},
				method:     "POST",
				output:     outputFiles,
				uploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}
func Test_PostWithUploadFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/post", "-X", "POST", "-T", tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}

func Test_PutWithUploadFilesystemForm_CurlContext(t *testing.T) {
	runContextWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) *CurlContext {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return &CurlContext{
				urls:       []string{"https://httpbin.org/put"},
				output:     outputFiles,
				uploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}
func Test_PutWithUploadFilesystemForm_CmdLine(t *testing.T) {
	runCmdLineWithTempFile(t, 1, 1,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte("test=one"), 0666)
			return []string{"https://httpbin.org/put", "-T", tempFiles[0], "-o", outputFiles[0]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, "test=one", data)
		})
}

func Test_PutWithUploadFilesystemFilesForm_CurlContext(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	runContextWithTempFile(t, 2, 2,
		func(outputFiles []string, tempFiles []string) *CurlContext {
			os.WriteFile(tempFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(tempFiles[1], []byte(expectedResult[1]), 0666)
			return &CurlContext{
				urls:       []string{"https://httpbin.org/put", "https://httpbin.org/put"},
				method:     "PUT",
				output:     outputFiles,
				uploadFile: tempFiles,
			}
		}, func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, expectedResult[index], data)
		})
}
func Test_PutWithUploadFilesystemFilesForm_CmdLine(t *testing.T) {
	expectedResult := []string{"test=one", "test=two"}
	runCmdLineWithTempFile(t, 2, 2,
		func(outputFiles []string, tempFiles []string) []string {
			os.WriteFile(tempFiles[0], []byte(expectedResult[0]), 0666)
			os.WriteFile(tempFiles[1], []byte(expectedResult[1]), 0666)
			return []string{"https://httpbin.org/put", "-T", tempFiles[0], "https://httpbin.org/put", "-T", tempFiles[1], "-o", outputFiles[0], "-o", outputFiles[1]}
		},
		func(json map[string]interface{}, index int) {
			verifyJson(t, json, "data")
			data := json["data"].(string)
			verifyGot(t, expectedResult[index], data)
		})
}

func Test_Delete_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:   []string{"https://httpbin.org/delete"},
			method: "DELETE",
			output: []string{outputFile},
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

func Test_RawishForm_CmdLine(t *testing.T) {
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"-o", outputFile, "https://httpbin.org/post", "-X", "POST", "-d", "{'name': 'Robert J. Oppenheimer'}", "-H", "Content-Type: application/json"}
		}, func(json map[string]interface{}) {
			verifyJson(t, json, "data")
			data := json["data"]
			verifyGot(t, "{'name': 'Robert J. Oppenheimer'}", data)
		})
}

func Test_GetWithCookies_CurlContext(t *testing.T) {
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:    []string{"https://httpbin.org/cookies"},
			output:  []string{outputFile},
			cookies: []string{"testcookie2=value2"},
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "cookies")
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
			verifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "value2", cookies["testcookie2"])
		})
}

func Test_CookieRoundTrip_CurlContext(t *testing.T) {
	cookieFile := filepath.Join(t.TempDir(), "cookies.dat")
	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:      []string{"https://httpbin.org/cookies/set/testcookie/testvalue"},
			output:    []string{outputFile},
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "cookies")
		cookies := json["cookies"].(map[string]interface{})
		verifyGot(t, "testvalue", cookies["testcookie"])
	})

	runContext(t, func(outputFile string) *CurlContext {
		return &CurlContext{
			urls:      []string{"https://httpbin.org/cookies"},
			output:    []string{outputFile},
			cookieJar: cookieFile,
		}
	}, func(json map[string]interface{}) {
		verifyJson(t, json, "cookies")
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
			verifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "testvalue", cookies["testcookie"])
		})
	runCmdLine(t,
		func(outputFile string) []string {
			return []string{"https://httpbin.org/cookies", "-c", cookieFile, "-o", outputFile}
		},
		func(json map[string]interface{}) {
			verifyJson(t, json, "cookies")
			cookies := json["cookies"].(map[string]interface{})
			verifyGot(t, "testvalue", cookies["testcookie"])
		})
}
