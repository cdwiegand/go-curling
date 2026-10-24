package main

import (
	"os"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func main() {
	ctx := new(curl.CurlContext)
	var cerr *curlerrors.CurlError

	nonFlagArgs, cerr := curlcli.ParseFlags(os.Args[1:], ctx)
	if cerr != nil {
		reportError(cerr, ctx)
		os.Exit(cerr.ExitCode)
		return
	}

	if ctx.Version {
		_, err := os.Stdout.WriteString("go-curling build ##DEV##")
		if err != nil {
			panic("Unable to write to stdout")
		}
		os.Exit(0)
		return
	}

	cerr = ctx.SetupContextForRun(nonFlagArgs)
	if cerr != nil {
		reportError(cerr, ctx)
		os.Exit(cerr.ExitCode)
		return
	}

	// must be after version check
	if len(ctx.Urls) == 0 {
		cerr = curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_URL, "No URL supplied; try 'go-curling --help' for usage")
		reportError(cerr, ctx)
		os.Exit(cerr.ExitCode)
		return
	}

	client, cerr := ctx.BuildClient()
	if cerr != nil {
		reportError(cerr, ctx)
		os.Exit(cerr.ExitCode)
		return
	}

	var lastErrorCode *curlerrors.CurlError
	for index := range ctx.Urls {
		request, transferErr := ctx.BuildHttpRequest(ctx.Urls[index], index, true, true)
		var resp *curl.CurlResponses
		if transferErr == nil {
			resp, transferErr = ctx.PerformTransfer(index, client, request)
		} else {
			resp = &curl.CurlResponses{Error: transferErr}
		}
		if transferErr != nil {
			lastErrorCode = transferErr
			reportError(transferErr, ctx)
		}
		if outErr := ctx.EmitWriteOut(resp); outErr != nil {
			lastErrorCode, transferErr = outErr, outErr
			reportError(outErr, ctx)
		}
		if transferErr != nil && ctx.FailEarly {
			client.CloseIdleConnections()
			os.Exit(transferErr.ExitCode)
		}
	}
	client.CloseIdleConnections()
	if lastErrorCode != nil {
		os.Exit(lastErrorCode.ExitCode)
	}
}

func reportError(err *curlerrors.CurlError, ctx *curl.CurlContext) string {
	if err == nil {
		return ""
	}
	entry := "Error: " + err.ErrorString + ".\n"

	if !ctx.IsSilent || ctx.ShowErrorEvenIfSilent {
		destination := ctx.ErrorOutput
		if destination == "" {
			destination = curl.DEFAULT_STDERR
		}
		oserr := ctx.WriteToFileBytes(destination, []byte(entry))
		if oserr != nil {
			_, _ = os.Stderr.WriteString(entry)
		}
	}

	return entry
}
