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

	cookieJar "github.com/orirawlings/persistent-cookiejar"
	flag "github.com/spf13/pflag"
)

type CurlContext struct {
	version                    bool
	verbose                    bool
	method                     string
	silentFail                 bool
	output                     []string
	headerOutput               []string
	userAgent                  string
	urls                       []string
	ignoreBadCerts             bool
	userAuth                   string
	isSilent                   bool
	headOnly                   bool
	includeHeadersInMainOutput bool
	showErrorEvenIfSilent      bool
	referer                    string
	errorOutput                string
	cookies                    []string
	cookieJar                  string
	_jar                       *cookieJar.Jar
	uploadFile                 []string
	form_encoded               []string
	form_multipart             []string
	headers                    []string
}

const ERROR_STATUS_CODE_FAILURE = -6
const ERROR_NO_RESPONSE = -7
const ERROR_INVALID_URL = -8
const ERROR_CANNOT_READ_FILE = -9
const ERROR_CANNOT_WRITE_FILE = -10
const ERROR_CANNOT_WRITE_TO_STDOUT = -11

const DEFAULT_OUTPUT = "stdout"

func main() {
	ctx := &CurlContext{}

	// I want to be able to test using my own args[], so can't use default flag.Parse()..
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(os.Args[1:])

	extraArgs := flags.Args() // remaining non-parsed args
	ctx.SetupContextForRun(extraArgs)

	if ctx.version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
		return
	}

	// must be after version check
	if len(ctx.urls) == 0 {
		err := errors.New("URL was not found on the command line")
		HandleErrorAndExit(err, ctx, ERROR_STATUS_CODE_FAILURE, "Parse URL")
	}

	client := ctx.BuildClient()

	for index := range ctx.urls {
		request := ctx.BuildRequest(index)
		resp, err := client.Do(request)
		ctx.ProcessResponse(index, resp, err, request)
	}
}
func SetupFlagArgs(ctx *CurlContext, flags *flag.FlagSet) {
	empty := []string{}
	flags.BoolVarP(&ctx.version, "version", "V", false, "Return version and exit")
	flags.BoolVarP(&ctx.verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flags.StringVar(&ctx.errorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flags.StringVarP(&ctx.method, "method", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flags.StringArrayVarP(&ctx.output, "output", "o", []string{DEFAULT_OUTPUT}, "Where to output results")
	flags.StringArrayVarP(&ctx.headerOutput, "dump-header", "D", []string{}, "Where to output headers (not on by default)")
	flags.StringVarP(&ctx.userAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flags.StringVarP(&ctx.userAuth, "user", "u", "", "User:password for HTTP authentication")
	flags.StringVarP(&ctx.referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flags.StringArrayVar(&ctx.urls, "url", []string{}, "Requesting URL")
	flags.BoolVarP(&ctx.silentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flags.BoolVarP(&ctx.ignoreBadCerts, "insecure", "k", false, "Ignore invalid SSL certificates")
	flags.BoolVarP(&ctx.isSilent, "silent", "s", false, "Silence all program console output")
	flags.BoolVarP(&ctx.showErrorEvenIfSilent, "show-error", "S", false, "Show error info even if silent mode on")
	flags.BoolVarP(&ctx.headOnly, "head", "I", false, "Only return headers (ignoring body content)")
	flags.BoolVarP(&ctx.includeHeadersInMainOutput, "include", "i", false, "Include headers (prepended to body content)")
	flags.StringSliceVarP(&ctx.cookies, "cookie", "b", empty, "HTTP cookie, raw HTTP cookie only (use -c for cookie jar files)")
	flags.StringSliceVarP(&ctx.form_encoded, "data", "d", empty, "HTML form data, set mime type to 'application/x-www-form-urlencoded'")
	flags.StringSliceVarP(&ctx.form_multipart, "form", "F", empty, "HTML form data, set mime type to 'multipart/form-data'")
	flags.StringVarP(&ctx.cookieJar, "cookie-jar", "c", "", "File for storing (read and write) cookies")
	flags.StringArrayVarP(&ctx.uploadFile, "upload-file", "T", []string{}, "Raw file(s) to PUT (default) to the url(s) given, not encoded")
	flags.StringArrayVarP(&ctx.headers, "header", "H", []string{}, "Header(s) to append to request")
}

func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
func HandleErrorAndExit(err error, ctx *CurlContext, exitCode int, entry string) {
	if err == nil {
		return
	}
	if entry == "" {
		entry = "Error"
	}
	entry += ": "
	entry += err.Error()
	if exitCode == ERROR_CANNOT_WRITE_TO_STDOUT {
		// don't recurse (it called us to report the failure to write errors to a normal file)
		PanicIfError(err)
	} else if (!ctx.isSilent && !ctx.silentFail) || !ctx.showErrorEvenIfSilent {
		writeToFileBytes(ctx, ctx.errorOutput, []byte(entry+"\n"))
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
