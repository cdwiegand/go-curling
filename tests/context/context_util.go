package contexttests

import (
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

// FIXME: make a test context? and then clean up these almost-duplicate functions

func RunContext(t *testing.T, contextBuilder func(outputFile string) (ctx *curl.CurlContext), successHandler func(map[string]interface{}), errorHandler func(*curlerrors.CurlError)) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")

	ctx := contextBuilder(outputFile)
	cerr := ctx.SetupContextForRun([]string{})
	if cerr != nil {
		errorHandler(cerr)
		return
	}
	curltest.HelpRun_Inner(ctx, successHandler, outputFile, errorHandler)
}

func RunContextWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, contextBuilder func(outputFiles []string, tempFiles []string) (ctx *curl.CurlContext), successHandler func(map[string]interface{}, int), errorHandler func(*curlerrors.CurlError)) {
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
	cerr := ctx.SetupContextForRun([]string{})
	if cerr != nil {
		errorHandler(cerr)
		return
	}
	curltest.HelpRun_InnerWithFiles(ctx, successHandler, outputFile, errorHandler)
}
