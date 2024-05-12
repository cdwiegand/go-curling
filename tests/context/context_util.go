package contexttests

import (
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	curltest "github.com/cdwiegand/go-curling/tests"
)

// FIXME: make a test context? and then clean up these almost-duplicate functions

func RunContext(t *testing.T, contextBuilder func(testrun *curltest.TestRun) (ctx *curl.CurlContext), successHandler func(map[string]interface{}), errorHandler func(*curlerrors.CurlError)) {
	tmpDir := t.TempDir()
	outputFile := curltest.BuildFileList(1, tmpDir, "out")
	tempFile := curltest.BuildFileList(0, tmpDir, "tmp")

	run := &curltest.TestRun{
		OutputFiles:    outputFile,
		InputFiles:     tempFile,
		SuccessHandler: successHandler,
		ErrorHandler:   errorHandler,
	}
	runContext_Real(run, contextBuilder)
}

func RunContextWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, contextBuilder func(testrun *curltest.TestRun) (ctx *curl.CurlContext), successHandler func(map[string]interface{}, int), errorHandler func(*curlerrors.CurlError)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")

	run := &curltest.TestRun{
		OutputFiles:           outputFile,
		InputFiles:            tempFile,
		SuccessHandlerIndexed: successHandler,
		ErrorHandler:          errorHandler,
	}
	runContext_Real(run, contextBuilder)
}

func runContext_Real(run *curltest.TestRun, contextBuilder func(testrun *curltest.TestRun) (ctx *curl.CurlContext)) {
	ctx := contextBuilder(run)

	cerr := ctx.SetupContextForRun([]string{})
	if cerr != nil {
		run.ErrorHandler(cerr)
		return
	}

	curltest.RunTestRun(ctx, run)
}
