package context

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	cookieJar "github.com/cdwiegand/persistent-cookiejar"
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
	FailWithBody                       bool
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
	Tls_MinVersion_1_3                 bool
	Tls_MinVersion_1_2                 bool
	Tls_MinVersion_1_1                 bool
	Tls_MinVersion_1_0                 bool
	Tls_MaxVersionString               string
	RetryDelaySeconds                  int
	MaxRetries                         int
	RetryAllErrors                     bool
	RetryConnRefused                   bool
	RetryMaxTime                       float64
	ForceTryHttp2                      bool
	Expect100Timeout                   float32
	WriteOut                           string
	MaxTime                            float32
	ConnectTimeout                     float32
	RemoteName                         bool
	RemoteHeaderName                   bool
	Range                              string
	ForceIPv4                          bool
	ForceIPv6                          bool
	Resolve                            []string
	Http1_0                            bool
	Http1_1                            bool
	UnixSocket                         string
	Proxy                              string
	ProxySet                           bool
	NoProxy                            string
	NoProxySet                         bool
	CreateDirs                         bool
	RemoveOnError                      bool
	MaxFilesize                        int64

	// internal:
	filesAlreadyStartedWriting map[string]*os.File
	stdinData                  []byte
	stdinRead                  bool
	bodyOutputsWritten         map[string]bool
}

type CurlOutputWriter interface {
	WriteToFileBytes(file string, body []byte) error
}

func (ctx *CurlContext) SetupContextForRun(extraArgs []string) *curlerrors.CurlError {
	// do sanity checks and "fix" some parts left remaining from flag parsing

	if strings.Contains(ctx.UserAgent, "##DE") {
		// ok, do the calc
		ctx.UserAgent = strings.ReplaceAll(ctx.UserAgent, "##DE"+"V##", "dev-branch") // split as I want to keep proper date versions unmunged in source
	}

	// -w/--write-out with an @ prefix loads the format from a file (@- reads stdin), like curl
	if strings.HasPrefix(ctx.WriteOut, "@") {
		src := ctx.WriteOut[1:]
		var data []byte
		var rerr error
		if src == "-" {
			data, rerr = io.ReadAll(os.Stdin)
		} else {
			data, rerr = os.ReadFile(src) // #nosec G304
		}
		if rerr != nil {
			return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, "Unable to read --write-out file "+src, rerr)
		}
		ctx.WriteOut = string(data)
	}

	if ctx.SilentFail && ctx.FailWithBody {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "--fail and --fail-with-body are mutually exclusive")
	}
	if ctx.RedirectsKeepAuthenticationHeaders {
		ctx.FollowRedirects = true
	}
	for _, value := range []float64{float64(ctx.MaxTime), float64(ctx.ConnectTimeout), float64(ctx.Expect100Timeout), ctx.RetryMaxTime} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) || value >= float64(math.MaxInt64)/float64(time.Second) {
			return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Timeouts must be finite, non-negative seconds within the supported duration range")
		}
	}
	if ctx.MaxRetries < 0 || ctx.RetryDelaySeconds < 0 || int64(ctx.RetryDelaySeconds) > math.MaxInt64/int64(time.Second) || ctx.MaxFilesize < 0 || ctx.MaxRedirects < -1 {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Retry counts, retry delays and file sizes must be non-negative; --max-redirs must be -1 or greater")
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

	if !ctx.validateFormArgs() {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: -d/--data*, -F/--form/--form-string, -T/--upload, or -I/--head")
	}

	if !ctx.validateTlsArgs() {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: --tls1/-1, --tlsv1.1, --tlsv1.2, --tlsv1.3")
	}

	if ctx.ForceIPv4 && ctx.ForceIPv6 {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include both -4/--ipv4 and -6/--ipv6")
	}

	httpVersions := 0
	for _, set := range []bool{ctx.Http1_0, ctx.Http1_1, ctx.ForceTryHttp2} {
		if set {
			httpVersions++
		}
	}
	if httpVersions > 1 {
		return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Cannot include more than one option from: --http1.0, --http1.1, --http2")
	}

	if len(extraArgs) > 0 {
		for _, h := range extraArgs {
			if strings.HasPrefix(h, "-") {
				return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Unrecognized argument: "+h)
			}
		}
	}

	s, err2 := ctx.setupUrlsFromArgs(extraArgs)
	if err2 != nil {
		return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", s), err2)
	}

	// -O/--remote-name (and -J, which falls back to it): derive the output file
	// name from each URL's path. With -J, the Content-Disposition header can
	// override this at emit time.
	if ctx.RemoteName || ctx.RemoteHeaderName {
		names := make([]string, len(ctx.Urls))
		for i, u := range ctx.Urls {
			name, nerr := remoteNameFromURL(u)
			if nerr != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, "Cannot use -O/--remote-name", nerr)
			}
			names[i] = name
		}
		ctx.BodyOutput = names
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

func (ctx *CurlContext) setupUrlsFromArgs(extraArgs []string) (string, error) {
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
				return s, err
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return s, fmt.Errorf("unsupported protocol %q (only HTTP and HTTPS are supported)", u.Scheme)
			}

			if u.Host == "" {
				u.Host = "localhost"
			}

			ctx.Urls = append(ctx.Urls, u.String())
		}
	}
	return "", nil
}

func (ctx *CurlContext) SetMethodIfNotSet(httpMethod string) {
	if ctx.HttpVerb == "" {
		ctx.HttpVerb = httpMethod
	}
}

func (ctx *CurlContext) validateFormArgs() bool {
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
	return countMutuallyExclusiveActions <= 1
}

func (ctx *CurlContext) validateTlsArgs() bool {
	countMutuallyExclusiveActions := 0
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
	return countMutuallyExclusiveActions <= 1
}

func (ctx *CurlContext) SetHeaderIfNotSet(headerName string, headerValue string) {
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

func (ctx *CurlContext) SetHeadersFromDict(headersDict map[string]string) {
	for key, value := range headersDict {
		ctx.SetHeaderIfNotSet(key, value)
	}
}

func (ctx *CurlContext) GetHeadersAsDict() map[string]string {
	dict := make(map[string]string)
	if len(ctx.Headers) > 0 {
		for _, h := range ctx.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) != 2 {
				continue
			}
			val := parts[1]
			// as we have Header: Value - have to remove the initial string
			if strings.HasPrefix(val, " ") && len(val) > 1 {
				val = val[1:]
			}
			dict[parts[0]] = val
		}
	}
	return dict
}

func (ctx *CurlContext) GetNextOutputsFromContext(index int) (headerOutput string, contentOutput string) {
	if len(ctx.BodyOutput) > index {
		contentOutput = standardizeFileName(ctx.BodyOutput[index])
	} else {
		// more URLs than -o entries (or none given): curl sends the extras to stdout
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
	ctx.bodyOutputsWritten = make(map[string]bool)
	defer resp.Close()
	defer func() {
		if ctx.RemoveOnError && cerrs.HasError() {
			cerrs.AppendCurlErrors(ctx.removeBodyOutputs())
		}
		if cerrs.HasError() {
			resp.Error = cerrs.Errors[len(cerrs.Errors)-1]
			resp.IsError = true
		}
	}()
	for i := 0; i < len(resp.Responses); i++ {
		isLast := i == len(resp.Responses)-1
		cr := resp.Responses[i]
		bodyLen, cerr := ctx.EmitSingleHttpResponseToOutputs(index, cr.HttpResponse, request, !isLast)
		if isLast {
			// record the final transfer size/time for --write-out
			resp.BodyBytes = bodyLen
			if cr.Timings != nil {
				cr.Timings.BodyDone = time.Now()
			}
		}
		cerrs.AppendCurlErrors(cerr)
		request = nil
	}
	return cerrs
}

func (ctx *CurlContext) EmitSingleHttpResponseToOutputs(index int, resp *http.Response, request *http.Request, headersOnly bool) (bodyLen int, cerrs curlerrors.CurlErrorCollection) {
	if resp == nil {
		return
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	headerOutput, contentOutput := ctx.GetNextOutputsFromContext(index)
	if ctx.RemoteHeaderName && !headersOnly {
		if fn := filenameFromContentDisposition(resp.Header.Get("Content-Disposition")); fn != "" {
			contentOutput = fn
		}
	}
	bodyOpened := false
	defer func() {
		if ctx.RemoveOnError && bodyOpened && cerrs.HasError() {
			cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.removeFailedOutput(contentOutput))
		}
	}()
	if ctx.Verbose {
		var lines []string
		if request != nil {
			lines = append(lines, DumpRequestHeaders(request)...)
		}
		if resp.TLS != nil {
			lines = append(lines, DumpTlsDetails(resp.TLS)...)
		}
		lines = append(lines, DumpResponseHeaders(resp, true)...)
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(ctx.ErrorOutput, []byte(strings.Join(lines, "\n")+"\n")))
	}
	headerBody := []byte(strings.Join(DumpResponseHeaders(resp, false), "\n") + "\n\n")
	if ctx.HeadOnly {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(headerOutput, headerBody))
		return
	}
	if headerOutput != contentOutput {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.WriteToFileBytes(headerOutput, headerBody))
	}
	if cerrs.HasError() {
		return
	}
	if !headersOnly && ctx.MaxFilesize > 0 && resp.ContentLength > ctx.MaxFilesize {
		cerrs.AppendCurlError(curlerrors.NewCurlErrorFromString(curlerrors.ERROR_MAX_FILESIZE, "Response exceeds --max-filesize"))
		return
	}
	if ctx.IncludeHeadersInMainOutput || headerOutput == contentOutput {
		err := ctx.WriteToFileBytes(contentOutput, headerBody)
		bodyOpened = err == nil
		if bodyOpened {
			ctx.recordBodyOutput(contentOutput)
		}
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
	}
	if headersOnly || cerrs.HasError() || resp.Body == nil {
		return
	}
	w, closer, err := ctx.openOutput(contentOutput)
	if err != nil {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
		return
	}
	bodyOpened = true
	ctx.recordBodyOutput(contentOutput)
	tracked := &outputWriter{Writer: w}
	reader := io.Reader(resp.Body)
	if ctx.MaxFilesize > 0 {
		reader = io.LimitReader(reader, ctx.MaxFilesize)
	}
	n, readErr := io.Copy(tracked, reader)
	bodyLen = int(n)
	if readErr != nil {
		if tracked.err != nil {
			cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, tracked.err)
		} else {
			bodyErr := transferError(readErr)
			if bodyErr.ExitCode == curlerrors.ERROR_NO_RESPONSE {
				bodyErr.ExitCode = curlerrors.ERROR_RECEIVE
				if resp.ContentLength >= 0 && n < resp.ContentLength {
					bodyErr.ExitCode = curlerrors.ERROR_PARTIAL_FILE
				}
			}
			cerrs.AppendCurlError(bodyErr)
		}
	} else if ctx.MaxFilesize > 0 && n == ctx.MaxFilesize {
		var extra [1]byte
		count, probeErr := io.ReadFull(resp.Body, extra[:])
		if count > 0 {
			cerrs.AppendCurlError(curlerrors.NewCurlErrorFromString(curlerrors.ERROR_MAX_FILESIZE, "Response exceeds --max-filesize"))
		} else if probeErr != nil && probeErr != io.EOF {
			cerrs.AppendCurlError(transferError(probeErr))
		}
	}
	if closer != nil {
		cerrs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, closer.Close())
	}
	return
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
