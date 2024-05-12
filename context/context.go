package context

import (
	"fmt"
	"net/url"
	"strings"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	cookieJar "github.com/orirawlings/persistent-cookiejar"
	"golang.org/x/net/publicsuffix"
)

const DEFAULT_OUTPUT = "stdout"

type CurlContext struct {
	Version                    bool
	Verbose                    bool
	Method                     string
	SilentFail                 bool
	FailEarly                  bool
	Output                     []string
	HeaderOutput               []string
	UserAgent                  string
	Urls                       []string
	IgnoreBadCerts             bool
	UserAuth                   string
	IsSilent                   bool
	HeadOnly                   bool
	IncludeHeadersInMainOutput bool
	ShowErrorEvenIfSilent      bool
	Referer                    string
	ErrorOutput                string
	Cookies                    []string
	CookieJar                  string
	JunkSessionCookies         bool
	Jar                        *cookieJar.Jar
	Upload_File                []string
	Data_Standard              []string
	Data_Encoded               []string
	Data_RawAsIs               []string
	Data_Binary                []string
	Form_Multipart             []string
	Headers                    []string
}

func (ctx *CurlContext) SetupContextForRun(extraArgs []string) *curlerrors.CurlError {
	// do sanity checks and "fix" some parts left remaining from flag parsing

	if ctx.Verbose && len(ctx.HeaderOutput) == 0 {
		ctx.HeaderOutput = ctx.Output // emit headers
	}

	ctx.UserAgent = strings.ReplaceAll(ctx.UserAgent, "##DE"+"V##", "dev-branch") // split as I want to keep proper date versions unmunged

	if ctx.SilentFail || ctx.IsSilent {
		ctx.IsSilent = true   // implied
		ctx.SilentFail = true // both are the same thing right now, we only emit errors (or content)
		ctx.Output = []string{}
	}
	if ctx.HeadOnly {
		if len(ctx.HeaderOutput) == 0 {
			ctx.HeaderOutput = []string{"-"}
		}
		ctx.SetMethodIfNotSet("HEAD")
	}

	countMutuallyExclusiveActions := 0
	if len(ctx.Upload_File) > 0 {
		countMutuallyExclusiveActions += 1
	}
	if len(ctx.Form_Multipart) > 0 {
		countMutuallyExclusiveActions += 1
	}
	if ctx.HasDataArgs() {
		countMutuallyExclusiveActions += 1
	}
	if ctx.HeadOnly {
		countMutuallyExclusiveActions += 1
	}
	if countMutuallyExclusiveActions > 1 {
		return curlerrors.NewCurlError1(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: -d/--data*, -F/--form, -T/--upload, and -I/--head list")
	}

	urls := append(ctx.Urls, extraArgs...)
	ctx.Urls = []string{}

	if len(urls) > 0 {
		for _, s := range urls {
			if strings.Index(s, "/") == 0 {
				// url is /something/here - assume localhost!
				s = "http://localhost" + s
			} else if !strings.Contains(s, "://") { // ok, wasn't a root relative path, but no protocol/not a valid url, let's try to set the protocol directly
				s = "http://" + s
			}

			u, err := url.Parse(s)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", s), err)
			}

			// FIXME: do we even need these?
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			if u.Host == "" {
				u.Host = "localhost"
			}
			// FIXME_END

			ctx.Urls = append(ctx.Urls, u.String())
		}
	}

	jar, err := cookieJar.New(&cookieJar.Options{
		PublicSuffixList:      publicsuffix.List,
		Filename:              ctx.CookieJar,
		PersistSessionCookies: !ctx.JunkSessionCookies,
	})
	if err != nil {
		return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, "Unable to create cookie jar", err)
	}
	ctx.Jar = jar
	return nil
}

func (ctx *CurlContext) SetMethodIfNotSet(httpMethod string) {
	if ctx.Method == "" {
		ctx.Method = httpMethod
	}
}

func (ctx *CurlContext) getNextOutputsFromContext(index int) (headerOutput string, contentOutput string) {
	if len(ctx.Output) > index {
		contentOutput = ctx.Output[index]
	} else {
		contentOutput = DEFAULT_OUTPUT
	}
	if len(ctx.HeaderOutput) > index {
		headerOutput = ctx.HeaderOutput[index]
	} else {
		headerOutput = ""
	}
	return
}
