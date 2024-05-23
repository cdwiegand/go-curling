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

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func (ctx *CurlContext) BuildClient() (*http.Client, *curlerrors.CurlError) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: ctx.IgnoreBadCerts}

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

	customTransport.DisableCompression = ctx.DisableCompression

	return &http.Client{
		Transport: customTransport,
		Jar:       ctx.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // I want to handle them myself
		},
	}, nil
}

func (ctx *CurlContext) BuildRequest(index int) (request *http.Request, err *curlerrors.CurlError) {
	url := ctx.Urls[index]

	var upload io.Reader
	// must call these BEFORE using ctx.method (as they may set it to POST/PUT if not yet explicitly set)
	// fixme: add support for mixing them (upload file vs all others?)
	// fixme: add --data-binary support
	if len(ctx.Upload_File) > index {
		upload, err = ctx.HandleUploadRawFile(index)
		if err != nil {
			return nil, err // just stop now
		}
	} else if ctx.HasFormArgs() {
		upload, err = ctx.HandleFormMultipart()
		if err != nil {
			return nil, err // just stop now
		}
	} else if ctx.HasDataArgs() {
		upload, err = ctx.HandleDataArgs()
		if err != nil {
			return nil, err // just stop now
		}
	}

	// this should be after all other changes to method!
	ctx.SetMethodIfNotSet("GET")

	// now build
	request, _ = http.NewRequest(strings.ToUpper(ctx.Method), url, upload)

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
	if ctx.DisableCompression {
		request.Header.Del("Accept-Encoding")
	}
	if request.Header.Get("Accept") == "" {
		// curl default, so matching
		request.Header.Set("Accept", "*/*")
	}
	if ctx.Cookies != nil {
		for _, cookie := range ctx.Cookies {
			if !strings.Contains(cookie, "=") { // curl does this, so... ugh, wish golang had .Net's System.IO.Path.Exists() in a safe way
				f, err := os.ReadFile(cookie)
				if err != nil {
					return nil, curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", cookie), err)
				}
				request.Header.Add("Cookie", string(f))
			} else {
				request.Header.Add("Cookie", cookie)
			}
		}
	}
	if ctx.UserAuth != "" {
		auths := strings.SplitN(ctx.UserAuth, ":", 2) // this way password can contain a :
		if len(auths) == 1 {
			fmt.Print("Enter password: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n') // if unable to read, use blank instead
			auths = append(auths, input)
			ctx.UserAuth = strings.Join(auths, ":") // for next request, if any
		}
		request.SetBasicAuth(auths[0], auths[1])
	}

	return request, nil
}

type CurlResponses struct {
	Responses []*CurlResponse
	IsError   bool
}
type CurlResponse struct {
	HttpResponse *http.Response
	Error        error
	NextUrl      *url.URL
}

func (ctx *CurlContext) GetCompleteResponse(client *http.Client, request *http.Request) (*CurlResponses, *curlerrors.CurlError) {
	respsReal := new(CurlResponses)

	var urls []*http.Request
	urls = append(urls, request)
	for i := 0; i < len(urls); i++ {
		r := urls[i]
		respReal := GetRawResponse(client, r)
		respsReal.Responses = append(respsReal.Responses, respReal)
		respsReal.IsError = (respReal.HttpResponse == nil || respReal.HttpResponse.StatusCode >= 400)

		if respReal.Error != nil {
			respsReal.IsError = true
			cerr := curlerrors.NewCurlError2(curlerrors.ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", request.URL), respReal.Error)
			return respsReal, cerr
		}

		if ctx.FollowRedirects && respReal.NextUrl != nil {
			request.URL = respReal.NextUrl
			urls = append(urls, request)
		}
	}

	return respsReal, nil
}

func GetRawResponse(client *http.Client, request *http.Request) *CurlResponse {
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

func (ctx *CurlContext) ProcessResponse(index int, resp *CurlResponses, request *http.Request) (cerr *curlerrors.CurlError) {
	err2 := ctx.Jar.Save() // is ignored if jar's filename is empty
	if err2 != nil {
		cerr = curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar", err2)
		// continue anyways!
	}

	if resp.IsError {
		// error
		if !ctx.SilentFail {
			ctx.EmitResponseToOutputs(index, resp, request)
		}
		os.Exit(6) // arbitrary
	} else {
		// success
		ctx.EmitResponseToOutputs(index, resp, request)
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
