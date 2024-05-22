package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

type TestRun struct {
	ListOutputFiles       []string
	ListInputFiles        []string
	ContextBuilder        func(*TestRun) *curl.CurlContext
	CmdLineBuilder        func(*TestRun) []string
	SuccessHandler        func(map[string]interface{}, *TestRun)
	SuccessHandlerIndexed func(map[string]interface{}, int, *TestRun)
	ErrorHandler          func(*curlerrors.CurlError, *TestRun)
	TempDir               string
}

func BuildTestRun(t *testing.T) TestRun {
	ret := TestRun{}
	ret.TempDir = t.TempDir()

	// default error handler
	ret.ErrorHandler = func(err *curlerrors.CurlError, testrun *TestRun) {
		GenericTestErrorHandler(t, err)
	}
	return ret
}

func (run *TestRun) GetNextInputFile() (ret string) {
	i := len(run.ListInputFiles)
	ret = filepath.Join(run.TempDir, fmt.Sprintf("%d.in.tmp", i))
	run.ListInputFiles = append(run.ListInputFiles, ret)
	return
}
func (run *TestRun) GetNextOutputFile() (ret string) {
	i := len(run.ListOutputFiles)
	ret = filepath.Join(run.TempDir, fmt.Sprintf("%d.out.tmp", i))
	run.ListOutputFiles = append(run.ListOutputFiles, ret)
	return
}

func (run *TestRun) GetOneOutputFiles() []string {
	// this is for cleaner code in the context tests, which needs an array of output files that is usually just 1 long
	ret := run.GetOneOutputFile()
	return []string{ret}
}
func (run *TestRun) GetOutputFiles(count int) []string {
	i := len(run.ListOutputFiles)
	var ret []string
	for i2 := i + 1; i2 <= count; i2++ {
		file := filepath.Join(run.TempDir, fmt.Sprintf("%d.out.tmp", i2))
		ret = append(ret, file)
		run.ListOutputFiles = append(run.ListOutputFiles, file)
	}
	return ret
}
func (run *TestRun) GetOneOutputFile() (ret string) {
	i := len(run.ListOutputFiles)
	ret = filepath.Join(run.TempDir, fmt.Sprintf("%d.out.tmp", i))
	run.ListOutputFiles = append(run.ListOutputFiles, ret)
	return
}
func (run *TestRun) EnsureAtLeastOneOutputFiles() (ret []string) {
	if len(run.ListOutputFiles) == 0 {
		run.GetOneOutputFile()
	}
	return run.ListOutputFiles
}

func GenericTestErrorHandler(t *testing.T, err *curlerrors.CurlError) {
	t.Errorf("Got error %v", err)
}

// helper functions
func VerifyGot(t *testing.T, wanted any, got any) bool {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
		return false
	}
	return true
}
func VerifyJson(t *testing.T, json map[string]interface{}, arg string) bool {
	if json[arg] == nil {
		err := fmt.Sprintf("%v was not present in json response", arg)
		t.Errorf(err)
		return false
	}
	return true
}

func readJson(file string) (res map[string]interface{}, err error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(byteValue), &res)
	return res, nil
}

func (run *TestRun) Run() {
	var ctx *curl.CurlContext
	if run.ContextBuilder != nil {
		ctx = run.ContextBuilder(run)
	} else if run.CmdLineBuilder != nil {
		args := run.CmdLineBuilder(run)

		ctx = &curl.CurlContext{}
		_, _, cerr := curlcli.ParseFlags(args, ctx)
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}
	}

	cerr := ctx.SetupContextForRun([]string{})
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	client, cerr := ctx.BuildClient()
	if cerr != nil {
		run.ErrorHandler(cerr, run)
		return
	}

	for index := range ctx.Urls {
		request, cerr := ctx.BuildRequest(index)
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}

		resp, cerr := ctx.Do(client, request)
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}

		cerr = ctx.ProcessResponse(index, resp, request)
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}

		json, err := readJson(run.ListOutputFiles[index])
		if err != nil {
			run.ErrorHandler(curlerrors.NewCurlError2(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON", err), run)
			return
		}

		if run.SuccessHandler != nil {
			run.SuccessHandler(json, run)
		}
		if run.SuccessHandlerIndexed != nil {
			run.SuccessHandlerIndexed(json, index, run)
		}
	}
}
