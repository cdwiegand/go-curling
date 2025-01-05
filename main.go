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
		cerr = curlerrors.NewCurlErrorFromString(curlerrors.ERROR_STATUS_CODE_FAILURE, "no valid URL was not found on the command line, try 'go-curling --help' for usage")
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
		request, cerr := ctx.BuildHttpRequest(ctx.Urls[index], index, true, true)
		if cerr != nil {
			lastErrorCode = cerr
			if ctx.FailEarly {
				reportError(cerr, ctx)
				os.Exit(cerr.ExitCode)
			}
		} else {
			resp, cerr := ctx.GetCompleteResponse(index, client, request)
			if cerr != nil {
				lastErrorCode = cerr
				if resp != nil && len(resp.Responses) > 0 && ctx.FailWithBody {
					ctx.ProcessResponseToOutputs(index, resp, request)
				}
				reportError(cerr, ctx)
				if cerr.ExitCode != 0 && ctx.FailEarly {
					os.Exit(cerr.ExitCode)
				}
			} else {
				cerrs := ctx.ProcessResponseToOutputs(index, resp, request)
				if cerrs.HasError() {
					forceExitCode := 0
					for _, h := range cerrs.Errors {
						lastErrorCode = h
						reportError(cerr, ctx)
						if h.ExitCode != 0 {
							forceExitCode = h.ExitCode
						}
					}
					if forceExitCode != 0 && ctx.FailEarly {
						os.Exit(forceExitCode)
					}
				}
			}
		}
	}

	if lastErrorCode != nil {
		os.Exit(lastErrorCode.ExitCode)
	}
}

func reportError(err *curlerrors.CurlError, ctx *curl.CurlContext) string {
	if err == nil {
		return ""
	}
	entry := "Error: " + err.ErrorString + "."

	if err.ExitCode == curlerrors.ERROR_CANNOT_WRITE_TO_STDOUT {
		// don't recurse (it called us to report the failure to write errors to a normal file)
		panic(err)
	}

	if (!ctx.IsSilent && !ctx.SilentFail) || !ctx.ShowErrorEvenIfSilent {
		oserr := ctx.WriteToFileBytes(ctx.ErrorOutput, []byte(entry))
		if oserr != nil && !ctx.SilentFail {
			panic(err)
		}
	}

	return entry
}
