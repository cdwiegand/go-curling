package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

type TestRun struct {
	OutputFiles           []string
	InputFiles            []string
	SuccessHandler        func(map[string]interface{}, *TestRun)
	SuccessHandlerIndexed func(map[string]interface{}, int, *TestRun)
	ErrorHandler          func(*curlerrors.CurlError, *TestRun)
}

func GenericErrorHandler(t *testing.T, err *curlerrors.CurlError) {
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

func ReadJson(file string) (res map[string]interface{}, err error) {
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

func BuildFileList(count int, outputDir string, ext string) (files []string) {
	files = []string{}
	for i := 0; i < count; i++ {
		files = append(files, filepath.Join(outputDir, fmt.Sprintf("%d.%s", i, ext)))
	}
	return
}

func RunTestRun(ctx *curl.CurlContext, run *TestRun) {
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

		json, err := ReadJson(run.OutputFiles[index])
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
