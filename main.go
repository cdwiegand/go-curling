/*
  go-curling - an implementation of curl in golang
  Copyright (C) 2022 Christopher Wiegand

  This library is free software; you can redistribute it and/or
  modify it under the terms of the GNU Lesser General Public
  License as published by the Free Software Foundation; either
  version 2.1 of the License, or (at your option) any later version.

  This library is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
  Lesser General Public License for more details.

  You should have received a copy of the GNU Lesser General Public
  License along with this library; if not, write to the Free Software
  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110
*/

package main

import (
	"errors"
	"os"

	curlcli "github.com/cdwiegand/go-curling/cli"
	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func main() {
	ctx := &curl.CurlContext{}
	var cerr *curlerrors.CurlError

	_, extraArgs, cerr := curlcli.ParseFlags(os.Args[1:], ctx)
	if cerr != nil {
		handleErrorAndExit(cerr, ctx)
		return
	}

	if ctx.Version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
		return
	}

	cerr = ctx.SetupContextForRun(extraArgs)
	if cerr != nil {
		handleErrorAndExit(cerr, ctx)
		return
	}

	// must be after version check
	if len(ctx.Urls) == 0 {
		err := errors.New("no valid URL was not found on the command line")
		cerr = curlerrors.NewCurlError2(curlerrors.ERROR_STATUS_CODE_FAILURE, "Parse URL failed", err)
		handleErrorAndExit(cerr, ctx)
		return
	}

	client := ctx.BuildClient()

	var lastErrorCode *curlerrors.CurlError
	for index := range ctx.Urls {
		request, cerr := ctx.BuildRequest(index)
		if cerr != nil {
			lastErrorCode = cerr
			if ctx.FailEarly {
				handleError(cerr, ctx)
			}
		} else {
			resp, cerr := ctx.Do(client, request)
			if cerr != nil {
				lastErrorCode = cerr
				handleError(cerr, ctx)
			} else {
				cerr = ctx.ProcessResponse(index, resp, request)
				if cerr != nil {
					lastErrorCode = cerr
					handleError(cerr, ctx)
				}
			}
		}
	}
	if lastErrorCode != nil {
		handleErrorAndExit(lastErrorCode, ctx)
	}
}

func handleErrorAndExit(err *curlerrors.CurlError, ctx *curl.CurlContext) {
	handleError(err, ctx)
	os.Exit(err.ExitCode)
}
func handleError(err *curlerrors.CurlError, ctx *curl.CurlContext) {
	if err == nil {
		return
	}
	entry := err.ErrorString
	if entry == "" {
		entry = "Error"
	}
	if err.Err != nil {
		entry += ": "
		entry += err.Err.Error()
	}

	if err.ExitCode == curlerrors.ERROR_CANNOT_WRITE_TO_STDOUT {
		// don't recurse (it called us to report the failure to write errors to a normal file)
		panic(err)
	}

	if (!ctx.IsSilent && !ctx.SilentFail) || !ctx.ShowErrorEvenIfSilent {
		curl.WriteToFileBytes(ctx.ErrorOutput, []byte(entry)) // this feels... bad practice - can we move this to some kind of helper or something??
	}

	if err.ExitCode != 0 && ctx.FailEarly {
		os.Exit(err.ExitCode)
	}
}
