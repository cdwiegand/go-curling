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

func GenericErrorHandler(t *testing.T, err *curlerrors.CurlError) {
	t.Errorf("Got error %v", err)
}

// helper functions
func VerifyGot(t *testing.T, wanted any, got any) {
	if got != wanted {
		t.Errorf("got %q wanted %q", got, wanted)
	}
}
func VerifyJson(t *testing.T, json map[string]interface{}, arg string) {
	if json[arg] == nil {
		err := fmt.Sprintf("%v was not present in json response", arg)
		t.Errorf(err)
		panic(err)
	}
}

func ReadJson(file string) (res map[string]interface{}) {
	jsonFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	json.Unmarshal([]byte(byteValue), &res)
	return
}

func BuildFileList(count int, outputDir string, ext string) (files []string) {
	files = []string{}
	for i := 0; i < count; i++ {
		files = append(files, filepath.Join(outputDir, fmt.Sprintf("%d.%s", i, ext)))
	}
	return
}

func HelpRun_Inner(ctx *curl.CurlContext, successHandler func(map[string]interface{}), outputFile string, errorHandler func(*curlerrors.CurlError)) {
	client := ctx.BuildClient()

	for index := range ctx.Urls {
		request, err := ctx.BuildRequest(index)
		if err != nil {
			errorHandler(err)
		}
		resp, err := ctx.Do(client, request)
		if err != nil {
			errorHandler(err)
		}
		ctx.ProcessResponse(index, resp, request)
		if err != nil {
			errorHandler(err)
		}

		json := ReadJson(outputFile)
		successHandler(json)
	}
}

func HelpRun_InnerWithFiles(ctx *curl.CurlContext, successHandler func(map[string]interface{}, int), outputFiles []string, errorHandler func(*curlerrors.CurlError)) {
	client := ctx.BuildClient()

	for index := range ctx.Urls {
		request, err := ctx.BuildRequest(index)
		if err != nil {
			errorHandler(err)
			return
		}
		resp, err := ctx.Do(client, request)
		if err != nil {
			errorHandler(err)
			return
		}
		err = ctx.ProcessResponse(index, resp, request)
		if err != nil {
			errorHandler(err)
		}

		json := ReadJson(outputFiles[index])
		successHandler(json, index)
	}
}
