package contexttests

import (
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curltest "github.com/cdwiegand/go-curling/tests"
)

func RunContext(t *testing.T, contextBuilder func(outputFile string) (ctx *curl.CurlContext), handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")

	ctx := contextBuilder(outputFile)
	ctx.SetupContextForRun([]string{})
	HelpRun_Inner(ctx, handler, outputFile)
}

func RunContextWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, contextBuilder func(outputFiles []string, tempFiles []string) (ctx *curl.CurlContext), handler func(map[string]interface{}, int)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	for _, s := range outputFile {
		defer os.Remove(s)
	}

	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")
	for _, s := range tempFile {
		defer os.Remove(s)
	}

	ctx := contextBuilder(outputFile, tempFile)
	ctx.SetupContextForRun([]string{})
	HelpRun_InnerWithFiles(ctx, handler, outputFile)
}

func HelpRun_Inner(ctx *curl.CurlContext, handler func(map[string]interface{}), outputFile string) {
	client := ctx.BuildClient()

	for index := range ctx.Urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)

		json := curltest.ReadJson(outputFile)
		handler(json)
	}
}

func HelpRun_InnerWithFiles(ctx *curl.CurlContext, handler func(map[string]interface{}, int), outputFiles []string) {
	client := ctx.BuildClient()

	for index := range ctx.Urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)

		json := curltest.ReadJson(outputFiles[index])
		handler(json, index)
	}
}
