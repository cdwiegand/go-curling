package context

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	cookieJar "github.com/orirawlings/persistent-cookiejar"
	"golang.org/x/net/publicsuffix"
)

const DEFAULT_OUTPUT = "/dev/stdout"
const DEFAULT_STDERR = "/dev/stderr"

type CurlContext struct {
	Version                            bool
	Verbose                            bool
	HttpVerb                           string
	SilentFail                         bool
	FailEarly                          bool
	FailWithBody                       bool // inherent, unless you specify --fail-early or --fail/-f
	BodyOutput                         []string
	HeaderOutput                       []string
	UserAgent                          string
	Urls                               []string
	IgnoreBadCerts                     bool
	UserAuth                           string
	IsSilent                           bool
	HeadOnly                           bool
	EnableCompression                  bool
	DisableKeepalives                  bool
	DisableBuffer                      bool
	Allow301Post                       bool
	Allow302Post                       bool
	Allow303Post                       bool
	FollowRedirects                    bool
	MaxRedirects                       int
	RedirectsKeepAuthenticationHeaders bool
	OAuth2_BearerToken                 string
	ConfigFile                         string
	DoNotUseHostCertificateAuthorities bool
	DefaultProtocolScheme              string
	ConvertPostFormIntoGet             bool
	CaCertFile                         []string
	CaCertPath                         string
	ClientCertFile                     string
	ClientCertKeyFile                  string
	ClientCertKeyPassword              string
	IncludeHeadersInMainOutput         bool
	ShowErrorEvenIfSilent              bool
	Referer                            string
	ErrorOutput                        string
	Cookies                            []string
	CookieJar                          string
	JunkSessionCookies                 bool
	Jar                                *cookieJar.Jar
	Upload_File                        []string
	Data_Standard                      []string
	Data_Ascii                         []string
	Data_Encoded                       []string
	Data_RawAsIs                       []string
	Data_Binary                        []string
	Data_Json                          []string
	Form_Multipart                     []string
	Form_MultipartRaw                  []string
	Headers                            []string
	HeadersDict                        map[string]string
	Tls_MinVersion_1_3                 bool
	Tls_MinVersion_1_2                 bool
	Tls_MinVersion_1_1                 bool
	Tls_MinVersion_1_0                 bool
	Tls_MaxVersionString               string
	RetryDelaySeconds                  int
	MaxRetries                         int
	RetryAllErrors                     bool
	ForceTryHttp2                      bool
	Expect100Timeout                   float32

	// internal:
	filesAlreadyStartedWriting map[string]*os.File
}

type CurlOutputWriter interface {
	WriteToFileBytes(file string, body []byte) error
}

func (ctx *CurlContext) SetupContextForRun(extraArgs []string) *curlerrors.CurlError {
	// do sanity checks and "fix" some parts left remaining from flag parsing

	if ctx.Verbose && len(ctx.HeaderOutput) == 0 {
		ctx.HeaderOutput = ctx.BodyOutput // emit headers
	}

	if strings.Contains(ctx.UserAgent, "##DE") {
		// ok, do the calc
		ctx.UserAgent = strings.ReplaceAll(ctx.UserAgent, "##DE"+"V##", "dev-branch") // split as I want to keep proper date versions unmunged in source
	}

	if ctx.SilentFail || ctx.IsSilent {
		ctx.IsSilent = true   // implied
		ctx.SilentFail = true // both are the same thing right now, we only emit errors (or content)
		// ctx.BodyOutput = []string{}
	}
	if ctx.DefaultProtocolScheme == "" {
		ctx.DefaultProtocolScheme = "http"
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
	if ctx.HasFormArgs() {
		countMutuallyExclusiveActions += 1
	}
	if ctx.HasDataArgs() {
		countMutuallyExclusiveActions += 1
	}
	if ctx.HeadOnly {
		countMutuallyExclusiveActions += 1
	}
	if countMutuallyExclusiveActions > 1 {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: -d/--data*, -F/--form/--form-string, -T/--upload, or -I/--head")
	}

	countMutuallyExclusiveActions = 0
	if ctx.Tls_MinVersion_1_0 {
		countMutuallyExclusiveActions += 1
	}
	if ctx.Tls_MinVersion_1_1 {
		countMutuallyExclusiveActions += 1
	}
	if ctx.Tls_MinVersion_1_2 {
		countMutuallyExclusiveActions += 1
	}
	if ctx.Tls_MinVersion_1_3 {
		countMutuallyExclusiveActions += 1
	}
	if countMutuallyExclusiveActions > 1 {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: --tls1/-1, --tlsv1.1, --tlsv1.2, --tlsv1.3")
	}

	if len(extraArgs) > 0 {
		for _, h := range extraArgs {
			if strings.HasPrefix(h, "-") {
				return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Unrecognized argument: "+h)
			}
		}
	}

	urls := append(ctx.Urls, extraArgs...)
	ctx.Urls = []string{}

	if len(urls) > 0 {
		for _, s := range urls {
			if strings.Index(s, "/") == 0 {
				// url is /something/here - assume localhost!
				s = ctx.DefaultProtocolScheme + "://localhost" + s
			} else if !strings.Contains(s, "://") { // ok, wasn't a root relative path, but no protocol/not a valid url, let's try to set the protocol directly
				s = ctx.DefaultProtocolScheme + "://" + s
			}

			u, err := url.Parse(s)
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", s), err)
			}

			if u.Host == "" {
				u.Host = "localhost"
			}

			ctx.Urls = append(ctx.Urls, u.String())
		}
	}

	jar, err := cookieJar.New(&cookieJar.Options{
		PublicSuffixList:      publicsuffix.List,
		Filename:              ctx.CookieJar,
		PersistSessionCookies: !ctx.JunkSessionCookies,
	})
	if err != nil {
		return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, "Unable to create cookie jar", err)
	}
	ctx.Jar = jar

	return nil
}

func (ctx *CurlContext) SetMethodIfNotSet(httpMethod string) {
	if ctx.HttpVerb == "" {
		ctx.HttpVerb = httpMethod
	}
}

func (ctx *CurlContext) SetHeaderIfNotSet(headerName string, headerValue string) {
	if ctx.HeadersDict[headerName] != "" {
		return
	}
	if len(ctx.Headers) > 0 {
		for _, h := range ctx.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], headerName) {
				return
			}
		}
	}
	ctx.Headers = append(ctx.Headers, headerName+": "+headerValue) // subsequent ones will override
}

func (ctx *CurlContext) GetNextOutputsFromContext(index int) (headerOutput string, contentOutput string) {
	if len(ctx.BodyOutput) > index {
		contentOutput = standardizeFileName(ctx.BodyOutput[index])
	} else if len(ctx.BodyOutput) == 1 {
		contentOutput = standardizeFileName(ctx.BodyOutput[0])
	} else if len(ctx.BodyOutput) == 1 {
		contentOutput = DEFAULT_OUTPUT
	}
	if len(ctx.HeaderOutput) > index {
		headerOutput = standardizeFileName(ctx.HeaderOutput[index])
	} else if len(ctx.HeaderOutput) == 1 {
		headerOutput = standardizeFileName(ctx.HeaderOutput[0])
	} else {
		headerOutput = ""
	}
	return
}

func (ctx *CurlContext) EmitResponseToOutputs(index int, resp *CurlResponses, request *http.Request) (cerrs curlerrors.CurlErrorCollection) {
	for i := 0; i < len(resp.Responses); i++ {
		isLast := i == len(resp.Responses)-1
		cerr := ctx.EmitSingleHttpResponseToOutputs(index, resp.Responses[i].HttpResponse, request, !isLast)
		cerrs.AppendCurlErrors(cerr)
		request = nil
	}
	return cerrs
}

func (ctx *CurlContext) EmitSingleHttpResponseToOutputs(index int, resp *http.Response, request *http.Request, headersOnly bool) (cerrs curlerrors.CurlErrorCollection) {
	// emit body
	var respBody []byte
	if !headersOnly && resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	seperator := []byte("\n\n")
	headerBody := []byte("")
	if ctx.Verbose {
		if request != nil {
			headerBody = appendStrings(headerBody, seperator, DumpRequestHeaders(request))
		}
		if resp.TLS != nil {
			headerBody = appendStrings(headerBody, seperator, DumpTlsDetails(resp.TLS))
		}
	}
	headerBody = appendStrings(headerBody, seperator, DumpResponseHeaders(resp, ctx.Verbose))
	headerOutput, contentOutput := ctx.GetNextOutputsFromContext(index)

	if ctx.HeadOnly {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(headerOutput, headerBody))
	} else if ctx.IncludeHeadersInMainOutput {
		bytesOut := appendByteArrays(headerBody, seperator, respBody)
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(contentOutput, bytesOut)) // do all at once
		if headerOutput != contentOutput && respBody != nil {
			cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(headerOutput, headerBody))
		}
	} else if headerOutput == contentOutput {
		bytesOut := appendByteArrays(headerBody, seperator, respBody)
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(contentOutput, bytesOut)) // do all at once
	} else {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(headerOutput, headerBody))
		if respBody != nil {
			cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(contentOutput, respBody))
		}
	}
	return cerrs
}

func appendStrings(resp []byte, sepBody []byte, lines []string) []byte {
	vb := []byte(strings.Join(lines, "\n"))
	return appendByteArrays(resp, sepBody, vb)
}

func appendByteArrays(resp []byte, sepBody []byte, secondBody []byte) []byte {
	if len(resp) > 0 {
		resp = append(resp, sepBody...)
	}
	return append(resp, secondBody...)
}

// standardize the filename, so OutputWriter implementations only need /dev/null, /dev/stdout, /dev/stderr, and anything else they can support
func standardizeFileName(file string) string {
	if file == "/dev/null" || file == "null" || file == "" {
		return "/dev/null"
	}
	if file == "/dev/stderr" || file == "stderr" {
		return "/dev/stderr"
	}
	if file == "/dev/stdout" || file == "stdout" || file == "-" {
		return "/dev/stdout"
	}
	return file
}
