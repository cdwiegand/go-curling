package main

import (
	"fmt"
	"net/url"
	"strings"
)

func (ctx *CurlContext) SetupContextForRun(extraArgs []string) {
	// do sanity checks and "fix" some parts left remaining from flag parsing

	if ctx.verbose && len(ctx.headerOutput) == 0 {
		ctx.headerOutput = ctx.output // emit headers
	}

	ctx.userAgent = strings.ReplaceAll(ctx.userAgent, "##DE"+"V##", "dev-branch") // split as I want to keep proper date versions unmunged

	if ctx.silentFail || ctx.isSilent {
		ctx.isSilent = true   // implied
		ctx.silentFail = true // both are the same thing right now, we only emit errors (or content)
		ctx.output = []string{}
	}
	if ctx.headOnly {
		if len(ctx.headerOutput) == 0 {
			ctx.headerOutput = []string{"-"}
		}
		ctx.SetMethodIfNotSet("HEAD")
	}

	urls := append(ctx.urls, extraArgs...)
	ctx.urls = []string{}

	if len(urls) > 0 {
		for _, s := range urls {
			if strings.Index(s, "/") == 0 {
				// url is /something/here - assume localhost!
				s = "http://localhost" + s
			} else if !strings.Contains(s, "://") { // ok, wasn't a root relative path, but no protocol/not a valid url, let's try to set the protocol directly
				s = "http://" + s
			}

			u, err := url.Parse(s)
			HandleErrorAndExit(err, ctx, ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", s))

			// FIXME: do we even need these?
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			if u.Host == "" {
				u.Host = "localhost"
			}
			// FIXME_END

			ctx.urls = append(ctx.urls, u.String())
		}
	}

	ctx._jar = CreateEmptyJar(ctx)
}
func (ctx *CurlContext) getNextOutputsFromContext(index int) (headerOutput string, contentOutput string) {
	if len(ctx.output) > index {
		contentOutput = ctx.output[index]
	} else {
		contentOutput = DEFAULT_OUTPUT
	}
	if len(ctx.headerOutput) > index {
		headerOutput = ctx.headerOutput[index]
	} else {
		headerOutput = ""
	}
	return
}
