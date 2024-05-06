package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func (ctx *CurlContext) HandleBodyResponse(index int, resp *http.Response, request *http.Request) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	headerOutput, contentOutput := ctx.getNextOutputsFromContext(index)

	if ctx.headOnly {
		writeToFileBytes(ctx, headerOutput, GetHeaderBytes(ctx, resp, request, false))
	} else if ctx.includeHeadersInMainOutput {
		bytesOut := append(GetHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx, contentOutput, bytesOut) // do all at once
		if headerOutput != contentOutput {
			writeToFileBytes(ctx, headerOutput, GetHeaderBytes(ctx, resp, request, false)) // also emit headers to separate location??
		}
	} else if headerOutput == contentOutput {
		bytesOut := append(GetHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx, contentOutput, bytesOut) // do all at once
	} else {
		writeToFileBytes(ctx, headerOutput, GetHeaderBytes(ctx, resp, request, false))
		writeToFileBytes(ctx, contentOutput, respBody)
	}
}
func (ctx *CurlContext) ProcessResponse(index int, resp *http.Response, err error, request *http.Request) {
	HandleErrorAndExit(err, ctx, ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", ctx.urls))

	err2 := ctx._jar.Save() // is ignored if jar's filename is empty
	HandleErrorAndExit(err2, ctx, ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar")

	if resp.StatusCode >= 400 {
		// error
		if !ctx.silentFail {
			ctx.HandleBodyResponse(index, resp, request)
		}
		os.Exit(6) // arbitrary
	} else {
		// success
		ctx.HandleBodyResponse(index, resp, request)
	}
}
