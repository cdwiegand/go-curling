package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	jsonutil "github.com/cdwiegand/go-curling/jsonutil"
)

type TestRun struct {
	ListOutputFiles          []string
	ListInputFiles           []string
	ContextBuilder           func(*TestRun) *curl.CurlContext
	CmdLineBuilder           func(*TestRun) []string
	CmdLineBuilderCurl       func(*TestRun) []string
	SuccessHandler           func(map[string]interface{}, *TestRun)
	SuccessHandlerIndexed    func(map[string]interface{}, int, *TestRun)
	SuccessHandlerIndexedRaw func(map[string]interface{}, string, int, *TestRun)
	ErrorHandler             func(*curlerrors.CurlError, *TestRun)
	TempDir                  string
	Testing                  *testing.T
	DoNotTestAgainstCurl     bool
}

func BuildTestRun(t *testing.T) TestRun {
	ret := TestRun{}
	ret.TempDir = t.TempDir()
	ret.Testing = t

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

func readJson(file string) (res map[string]interface{}, raw string, err error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		return nil, "", err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, "", err
	}

	raw = string(byteValue)
	json.Unmarshal(byteValue, &res)
	return res, raw, nil
}

func (run *TestRun) Run() {
	var ctx *curl.CurlContext
	var args []string
	var extraArgs []string
	var cerr *curlerrors.CurlError

	if run.ContextBuilder != nil {
		ctx = run.ContextBuilder(run)
	} else if run.CmdLineBuilder != nil {
		ctx = &curl.CurlContext{}
		args = run.CmdLineBuilder(run)
		_, extraArgs, cerr = curlcli.ParseFlags(args, ctx)
		if cerr != nil {
			run.ErrorHandler(cerr, run)
			return
		}
	} else {
		run.Testing.Fatal("Forgot to add ContextBuilder or CmdLineBuilder to test!")
	}

	cerr = ctx.SetupContextForRun(extraArgs)
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

		if index >= len(run.ListOutputFiles) {
			run.ErrorHandler(curlerrors.NewCurlError1(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON"), run)
			return
		}

		jsonObj, rawJson, err := readJson(run.ListOutputFiles[index])
		if err != nil {
			run.ErrorHandler(curlerrors.NewCurlError2(curlerrors.ERROR_STATUS_CODE_FAILURE, "Failed to parse JSON", err), run)
			return
		}

		if run.SuccessHandler != nil {
			run.SuccessHandler(jsonObj, run)
		}
		if run.SuccessHandlerIndexed != nil {
			run.SuccessHandlerIndexed(jsonObj, index, run)
		}
		if run.SuccessHandlerIndexedRaw != nil {
			run.SuccessHandlerIndexedRaw(jsonObj, rawJson, index, run)
		}

		if run.CmdLineBuilder != nil && args != nil && !run.DoNotTestAgainstCurl {
			// test curl cli output compared to us
			if run.CmdLineBuilderCurl != nil {
				CompareCurlCliOutput(run, run.CmdLineBuilderCurl(run), jsonObj, rawJson)
			} else {
				CompareCurlCliOutput(run, args, jsonObj, rawJson)
			}
		}
	}
}

func CompareCurlCliOutput(run *TestRun, args []string, myJsonObj map[string]interface{}, myJsonRaw string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		moreargs := append([]string{"-e", "curl"}, args...)
		cmd = exec.Command("wsl", moreargs...)
	} else {
		cmd = exec.Command("curl", args...)
	}
	curlerr := cmd.Run()
	if curlerr != nil {
		return curlerr
	}

	curlJsonObj, curlJsonRaw, err := readJson(run.ListOutputFiles[0])
	if err != nil {
		return err
	}

	// compare, with specific differences permitted:
	/*
		{
		  "args": {},
		  "data": "@/tmp/0.in.tmp",
		  "files": {},
		  "form": {},
		  "headers": {
		    "Accept": "application/json",
		    "Accept-Encoding": "gzip",
		    "Content-Length": "90",
		    "Content-Type": "application/json",
		    "Host": "httpbin.org",
		    "User-Agent": "Go-http-client/2.0",                             <----- this can be different
		    "X-Amzn-Trace-Id": "Root=1-664d77a1-75a4de3a1784d8135d044b0f"   <----- this can be different
		  },
		  "json": null,
		  "origin": "73.203.21.18",
		  "url": "https://httpbin.org/post"
		}
	*/

	if !jsonutil.Equal(myJsonObj, curlJsonObj, func(path string) bool {
		if path == "X-Amzn-Trace-Id" || path == "User-Agent" {
			return true
		}
		return false
	}) {
		return errors.New("json outputs did not match between curl and go-curling:\n\ngo-curling output:\n" + myJsonRaw + "\n\ncurl output:\n" + curlJsonRaw)
	}
	return nil
}

func (run *TestRun) RunAgainstCurlCli() {
	var args []string

	if run.CmdLineBuilder != nil {
		args = run.CmdLineBuilder(run)
	} else {
		run.Testing.Fatal("Forgot to add CmdLineBuilder to 'curl' CLI test!")
	}

	cmd := exec.Command("curl", args...)
	err := cmd.Run()
	if err != nil {
		run.Testing.Fatal(err)
	}

	for index := range run.ListOutputFiles {
		json, _, err := readJson(run.ListOutputFiles[index])
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
