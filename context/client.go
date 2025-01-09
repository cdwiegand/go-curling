package context

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	cookieJar "github.com/orirawlings/persistent-cookiejar"
)

type CurlResponses struct {
	Responses []*CurlResponse
	IsError   bool
}
type CurlResponse struct {
	HttpResponse *http.Response
	Error        error
	NextUrl      *url.URL
}

func (ctx *CurlContext) BuildClient() (*http.Client, *curlerrors.CurlError) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: ctx.IgnoreBadCerts} // #nosec G402

	if ctx.Tls_MinVersion_1_0 {
		customTransport.TLSClientConfig.MinVersion = tls.VersionTLS10
	}
	if ctx.Tls_MinVersion_1_1 {
		customTransport.TLSClientConfig.MinVersion = tls.VersionTLS11
	}
	if ctx.Tls_MinVersion_1_2 {
		customTransport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	if ctx.Tls_MinVersion_1_3 {
		customTransport.TLSClientConfig.MinVersion = tls.VersionTLS13
	}
	if ctx.Tls_MaxVersionString != "" {
		maxTls, err := GetTlsVersionValue(ctx.Tls_MaxVersionString)
		if err != nil {
			return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, fmt.Sprintf("Failed to parse TLS version %s", ctx.Tls_MaxVersionString), err)
		}
		if maxTls > 0 {
			customTransport.TLSClientConfig.MaxVersion = maxTls
		}
	}

	if ctx.ForceTryHttp2 {
		customTransport.ForceAttemptHTTP2 = true
	}
	if ctx.Expect100Timeout > 0 {
		customTransport.ExpectContinueTimeout = time.Duration(ctx.Expect100Timeout)
	}

	var cerr *curlerrors.CurlError
	customTransport.TLSClientConfig.RootCAs, cerr = ctx.BuildRootCAsPool()
	if cerr != nil {
		return nil, cerr
	}

	clientCerts, cerr := ctx.BuildClientCertificates()
	if cerr != nil {
		return nil, cerr
	}
	if len(clientCerts) > 0 {
		customTransport.TLSClientConfig.Certificates = append(customTransport.TLSClientConfig.Certificates, clientCerts...)
	}

	customTransport.DisableCompression = !ctx.EnableCompression
	customTransport.DisableKeepAlives = ctx.DisableKeepalives
	if ctx.DisableBuffer {
		customTransport.ReadBufferSize = 0
		customTransport.WriteBufferSize = 0
	}

	return &http.Client{
		Transport: customTransport,
		Jar:       ctx.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // I want to handle them myself
		},
	}, nil
}

func (ctx *CurlContext) BuildHttpRequest(url string, index int, submitDataFormsPostContents bool, submitAuthenticationHeaders bool) (request *http.Request, err *curlerrors.CurlError) {
	if url == "" && index < len(ctx.Urls) {
		url = ctx.Urls[index]
	}

	var body io.Reader
	// must call these BEFORE using ctx.method (as they may set it to POST/PUT if not yet explicitly set)
	// fixme: add support for mixing them (upload file vs all others?)
	// fixme: add --data-binary support
	if submitDataFormsPostContents {
		if len(ctx.Upload_File) > index {
			body, err = ctx.HandleUploadRawFile(index)
			if err != nil {
				return nil, err // just stop now
			}
		} else if ctx.HasFormArgs() {
			body, err = ctx.HandleFormMultipart()
			if err != nil {
				return nil, err // just stop now
			}
		} else if ctx.HasDataArgs() {
			bodyData, err := ctx.HandleDataArgs(ctx.ConvertPostFormIntoGet)
			if err != nil {
				return nil, err // just stop now
			}
			if ctx.ConvertPostFormIntoGet {
				ctx.SetMethodIfNotSet("GET")
				if strings.Contains(url, "?") {
					url += "&"
				} else {
					url += "?"
				}
				url += bodyData.String()
				body = nil
			} else {
				body = io.Reader(bodyData)
			}
		}
	}

	// this should be after all other changes to method!
	ctx.SetMethodIfNotSet("GET")

	// now build
	request, _ = http.NewRequest(strings.ToUpper(ctx.HttpVerb), url, body)

	ctx.SetupInitialHeadersOnRequest(request)

	cerr := ctx.SetCookieHeadersOnRequest(request)
	if cerr != nil {
		return nil, cerr
	}

	cerr = ctx.SetAuthenticationHeadersOnRequest(request)
	if cerr != nil {
		return nil, cerr
	}

	return request, nil
}

func (ctx *CurlContext) SetupInitialHeadersOnRequest(request *http.Request) {
	// custom headers ALWAYS come first (we use `set` below to override when needed)
	if len(ctx.Headers) > 0 {
		for _, h := range ctx.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				request.Header.Set(parts[0], parts[1])
			}
		}
	}

	if ctx.UserAgent != "" {
		request.Header.Set("User-Agent", ctx.UserAgent)
	} else if request.Header.Get("User-Agent") == "" {
		request.Header.Del("User-Agent")
	}
	if ctx.Referer != "" {
		request.Header.Set("Referer", ctx.Referer)
	}
	if request.Header.Get("Accept") == "" {
		// curl default, so matching
		request.Header.Set("Accept", "*/*")
	}
}

func (ctx *CurlContext) SetCookieHeadersOnRequest(request *http.Request) *curlerrors.CurlError {
	if ctx.Cookies != nil {
		for _, cookie := range ctx.Cookies {
			if !strings.Contains(cookie, "=") { // curl does this, so... ugh, wish golang had .Net's System.IO.Path.Exists() in a safe way
				// we use cookieJar's format, not curl's
				tmp, err := cookieJar.New(&cookieJar.Options{
					Filename: ctx.CookieJar,
				})
				if err != nil {
					for _, y := range tmp.AllCookies() {
						request.AddCookie(y)
					}
				}
			} else {
				request.Header.Add("Cookie", cookie)
			}
		}
	}
	return nil
}

func (ctx *CurlContext) SetAuthenticationHeadersOnRequest(request *http.Request) *curlerrors.CurlError {
	if request.Header == nil {
		request.Header = http.Header{}
	}

	if ctx.UserAuth != "" {
		auths := strings.SplitN(ctx.UserAuth, ":", 2) // this way password can contain a :
		if len(auths) == 1 {
			if ctx.IsSilent || ctx.SilentFail {
				return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "User auth requires username:password format, operating quiet so not prompting for value.")
			}
			fmt.Print("Enter password: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n') // if unable to read, use blank instead
			auths = append(auths, input)
			ctx.UserAuth = strings.Join(auths, ":") // for next request, if any
		}
		request.SetBasicAuth(auths[0], auths[1])
	}

	if request.Header.Get("Authorization") == "" && ctx.OAuth2_BearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+ctx.OAuth2_BearerToken)
	}

	return nil
}

func (ctx *CurlContext) GetCompleteResponse(index int, client *http.Client, request *http.Request) (*CurlResponses, *curlerrors.CurlError) {
	respsReal := new(CurlResponses)

	var cerr *curlerrors.CurlError
	var urls []*http.Request
	urls = append(urls, request)
	for i := 0; i < len(urls) && (ctx.MaxRedirects <= 0 || i < ctx.MaxRedirects); i++ {
		r := urls[i]
		var respReal *CurlResponse
		for retry := 0; retry <= ctx.MaxRetries; retry++ {
			respReal = GetCurlResponse(client, r)
			respsReal.Responses = append(respsReal.Responses, respReal)

			if respReal.HttpResponse != nil && ctx.canStatusCodeRetry(respReal.HttpResponse.StatusCode) && retry < ctx.MaxRetries {
				time.Sleep(time.Duration(ctx.RetryDelaySeconds) * time.Second)
			} else {
				break
			}
		}

		respsReal.IsError = (respReal.HttpResponse == nil || respReal.HttpResponse.StatusCode >= 400)

		if respReal.Error != nil {
			respsReal.IsError = true
			cerr = curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", request.URL), respReal.Error)
			return respsReal, cerr
		}

		if ctx.FollowRedirects && respReal.NextUrl != nil &&
			respReal.HttpResponse.StatusCode >= 300 && respReal.HttpResponse.StatusCode <= 399 {
			var newReq *http.Request
			retainData := true

			if request.Method == "POST" {
				retainData = (respReal.HttpResponse.StatusCode == 301 && ctx.Allow301Post) ||
					(respReal.HttpResponse.StatusCode == 302 && ctx.Allow302Post) ||
					(respReal.HttpResponse.StatusCode == 303 && ctx.Allow303Post)
			}
			newReq, cerr = ctx.BuildHttpRequest(respReal.NextUrl.String(), index, retainData, ctx.RedirectsKeepAuthenticationHeaders)
			if cerr != nil {
				return respsReal, cerr
			}

			if !retainData {
				newReq.Method = "GET"
			}
			urls = append(urls, newReq)
		}
	}

	return respsReal, nil
}

func GetCurlResponse(client *http.Client, request *http.Request) *CurlResponse {
	resp, err := client.Do(request)

	respReal := new(CurlResponse)
	respReal.Error = err
	respReal.HttpResponse = resp

	if respReal.HttpResponse != nil && respReal.HttpResponse.StatusCode >= 300 && respReal.HttpResponse.StatusCode <= 399 {
		location := respReal.HttpResponse.Header.Get("Location")
		if location != "" {
			newURL, err := url.Parse(location)
			if newURL != nil && err == nil {
				if !newURL.IsAbs() {
					newURL = request.URL.ResolveReference(newURL)
				}
				respReal.NextUrl = newURL
			}
		}
	}

	return respReal
}

func (ctx *CurlContext) canStatusCodeRetry(statusCode int) bool {
	if ctx.RetryAllErrors {
		return statusCode >= 400
	} else {
		return statusCode == 408 || statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
	}
}

func (ctx *CurlContext) ProcessResponseToOutputs(index int, resp *CurlResponses, request *http.Request) (cerrs curlerrors.CurlErrorCollection) {
	err2 := ctx.Jar.Save() // is ignored if jar's filename is empty
	if err2 != nil {
		cerrs.AppendCurlError(curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar", err2))
		// continue anyways!
	}

	if resp.IsError {
		// error
		if !ctx.SilentFail {
			cerrs.AppendCurlErrors(ctx.EmitResponseToOutputs(index, resp, request))
		}
		os.Exit(curlerrors.ERROR_STATUS_CODE_FAILURE) // arbitrary
	} else {
		// success
		cerrs.AppendCurlErrors(ctx.EmitResponseToOutputs(index, resp, request))
	}
	return
}

func DumpResponseHeaders(resp *http.Response, verboseFormat bool) (res []string) {
	proto := resp.Proto
	if resp.Proto == "" {
		proto = "HTTP/?" // default, sometimes golang won't let you have the HTTP protocol version in the response
	}
	res = append(res, fmt.Sprintf("%s %d", proto, resp.StatusCode))
	dict := make(map[string]string)
	keys := make([]string, 0, len(resp.Header))
	for name, values := range resp.Header {
		keys = append(keys, name)
		for _, value := range values {
			dict[name] = value
		}
	}
	sort.Strings(keys)
	prefix := ""
	if verboseFormat {
		prefix = "< "
	}
	for _, name := range keys { // I want them alphabetical
		res = append(res, fmt.Sprintf("%s%s: %s", prefix, name, dict[name]))
	}

	return
}

func DumpRequestHeaders(req *http.Request) (res []string) {
	res = append(res, fmt.Sprintf("%v %v", req.Method, req.URL))
	dict := make(map[string]string)
	keys := make([]string, 0, len(req.Header))
	for name, values := range req.Header {
		keys = append(keys, name)
		for _, value := range values {
			dict[name] = value
		}
	}
	sort.Strings(keys)
	for _, name := range keys { // I want them alphabetical
		res = append(res, fmt.Sprintf("> %s: %s", name, dict[name]))
	}

	return
}

func GetTlsVersionString(version uint16) (res string) {
	switch version {
	case tls.VersionTLS10:
		res = "TLS 1.0"
	case tls.VersionTLS11:
		res = "TLS 1.1"
	case tls.VersionTLS12:
		res = "TLS 1.2"
	case tls.VersionTLS13:
		res = "TLS 1.3"
	case AssumedVersionTLS14:
		res = "TLS 1.4?"
	default:
		res = fmt.Sprintf("Unknown TLS version: %v", version)
	}
	return
}

func GetTlsVersionValue(value string) (uint16, error) {
	switch value {
	case "default":
		return 0, nil
	case "":
		return 0, nil
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	case "1.4":
		return AssumedVersionTLS14, nil
	default:
		return 0, fmt.Errorf("unknown TLS version: %q", value)
	}
}

const (
	AssumedVersionTLS14 = 0x0305
)

func DumpTlsDetails(conn *tls.ConnectionState) (res []string) {
	res = append(res, fmt.Sprintf("TLS Version: %v", GetTlsVersionString(conn.Version)))
	res = append(res, fmt.Sprintf("TLS Cipher Suite: %v", tls.CipherSuiteName(conn.CipherSuite)))
	if conn.NegotiatedProtocol != "" {
		res = append(res, fmt.Sprintf("TLS Negotiated Protocol: %v", conn.NegotiatedProtocol))
	}
	if conn.ServerName != "" {
		res = append(res, fmt.Sprintf("TLS Server Name: %v", conn.ServerName))
	}
	return
}
