package clitests

import (
	"os"
	"path/filepath"
	"testing"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curltest "github.com/cdwiegand/go-curling/tests"
	curlcontexttest "github.com/cdwiegand/go-curling/tests/context"
	flag "github.com/spf13/pflag"
)

func RunCmdLine(t *testing.T, argsBuilder func(outputFile string) []string, handler func(map[string]interface{})) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "1.out")
	args := argsBuilder(outputFile)

	ctx := &curl.CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	curlcli.SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := flags.Args()

	ctx.SetupContextForRun(extraArgs)
	curlcontexttest.HelpRun_Inner(ctx, handler, outputFile)
}
func RunCmdLineWithTempFile(t *testing.T, countOutputFiles int, countTempFiles int, argsBuilder func(outputFiles []string, tempFiles []string) []string, handler func(map[string]interface{}, int)) {
	tmpDir := t.TempDir()

	outputFile := curltest.BuildFileList(countOutputFiles, tmpDir, "out")
	for _, s := range outputFile {
		defer os.Remove(s)
	}

	tempFile := curltest.BuildFileList(countTempFiles, tmpDir, "tmp")
	for _, s := range tempFile {
		defer os.Remove(s)
	}

	args := argsBuilder(outputFile, tempFile)
	ctx := &curl.CurlContext{}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	curlcli.SetupFlagArgs(ctx, flags)
	flags.Parse(args)
	extraArgs := flags.Args()

	ctx.SetupContextForRun(extraArgs)
	curlcontexttest.HelpRun_InnerWithFiles(ctx, handler, outputFile)
}
