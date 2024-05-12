package clitests

import (
	"path/filepath"
	"testing"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

func RunCmdLine(t *testing.T, argsBuilder func(outputFile string) []string, successHandler func(map[string]interface{}), errorHandler func(*curlerrors.CurlError)) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")
	args := argsBuilder(outputFile)

	ctx := &curl.CurlContext{}
	_, extraArgs, cerr := curlcli.ParseFlags(args, ctx)
	if cerr != nil {
		errorHandler(cerr)
		return
	}

	cerr = ctx.SetupContextForRun(extraArgs)
	if cerr != nil {
		errorHandler(cerr)
		return
	}

	curltest.HelpRun_Inner(ctx, successHandler, outputFile, errorHandler)
}
func RunCmdLineWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, argsBuilder func(outputFiles []string, tempFiles []string) []string, successHandler func(map[string]interface{}, int), errorHandler func(*curlerrors.CurlError)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")
	args := argsBuilder(outputFile, tempFile)

	ctx := &curl.CurlContext{}
	_, extraArgs, cerr := curlcli.ParseFlags(args, ctx)
	if cerr != nil {
		errorHandler(cerr)
		return
	}

	cerr = ctx.SetupContextForRun(extraArgs)
	if cerr != nil {
		errorHandler(cerr)
		return
	}

	curltest.HelpRun_InnerWithFiles(ctx, successHandler, outputFile, errorHandler)
}
