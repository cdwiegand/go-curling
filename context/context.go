package context

import (
	"fmt"
	"net/url"
	"strings"

	cookieJar "github.com/orirawlings/persistent-cookiejar"
	"golang.org/x/net/publicsuffix"
)

const DEFAULT_OUTPUT = "stdout"

type CurlContext struct {
	Version                    bool
	Verbose                    bool
	Method                     string
	SilentFail                 bool
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
	Jar                        *cookieJar.Jar
	UploadFile                 []string
	Data_standard              []string
	Data_encoded               []string
	Data_rawconcat             []string
	Data_multipart             []string
	Headers                    []string
}

func (ctx *CurlContext) SetupContextForRun(extraArgs []string) {
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
			ctx.HandleErrorAndExit(err, ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", s))

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
		PersistSessionCookies: true,
	})
	ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, "Unable to create cookie jar")
	ctx.Jar = jar
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
