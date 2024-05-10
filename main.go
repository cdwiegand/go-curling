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
	flag "github.com/spf13/pflag"
)

func main() {
	ctx := &curl.CurlContext{}

	// I want to be able to test using my own args[], so can't use default flag.Parse()..
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	curlcli.SetupFlagArgs(ctx, flags)
	flags.Parse(os.Args[1:])

	extraArgs := flags.Args() // remaining non-parsed args
	ctx.SetupContextForRun(extraArgs)

	if ctx.Version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
		return
	}

	// must be after version check
	if len(ctx.Urls) == 0 {
		err := errors.New("URL was not found on the command line")
		ctx.HandleErrorAndExit(err, curl.ERROR_STATUS_CODE_FAILURE, "Parse URL")
	}

	client := ctx.BuildClient()

	for index := range ctx.Urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)
	}
}
