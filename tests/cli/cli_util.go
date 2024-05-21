package clitests

import (
	"testing"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

func RunCmdLine(t *testing.T, countOutputFiles int, countTempFiles int, argsBuilder func(*curltest.TestRun) []string, successHandler func(map[string]interface{}, *curltest.TestRun), errorHandler func(*curlerrors.CurlError, *curltest.TestRun)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")

	run := &curltest.TestRun{
		OutputFiles:    outputFile,
		InputFiles:     tempFile,
		SuccessHandler: successHandler,
		ErrorHandler:   errorHandler,
	}
	runCmdLine_Real(run, argsBuilder)
}
func RunCmdLineIndexed(t *testing.T, countOutputFiles int, countTempFiles int, argsBuilder func(*curltest.TestRun) []string, successHandler func(map[string]interface{}, int, *curltest.TestRun), errorHandler func(*curlerrors.CurlError, *curltest.TestRun)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")

	run := &curltest.TestRun{
		OutputFiles:           outputFile,
		InputFiles:            tempFile,
		SuccessHandlerIndexed: successHandler,
		ErrorHandler:          errorHandler,
	}
	runCmdLine_Real(run, argsBuilder)
}
func runCmdLine_Real(run *curltest.TestRun, argsBuilder func(*curltest.TestRun) []string) {
	args := argsBuilder(run)

	ctx := &curl.CurlContext{}
	_, extraArgs, cerr := curlcli.ParseFlags(args, ctx)
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	cerr = ctx.SetupContextForRun(extraArgs)
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	curltest.RunTestRun(ctx, run)
}
