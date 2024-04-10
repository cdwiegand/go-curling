package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	cookieJar "github.com/orirawlings/persistent-cookiejar"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/publicsuffix"
)

type CurlContext struct {
	version                    bool
	verbose                    bool
	method                     string
	silentFail                 bool
	output                     string
	headerOutput               string
	userAgent                  string
	theUrl                     string
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
	uploadFile                 string
	form_encoded               []string
	form_multipart             []string
	_body_contentType          string
	_body                      io.Reader
}

func (ctx *CurlContext) SetBody(body io.Reader, mimeType string, httpMethod string) {
	ctx._body = body
	ctx._body_contentType = mimeType
	ctx.SetMethodIfNotSet(httpMethod)
}
func (ctx *CurlContext) SetMethodIfNotSet(httpMethod string) {
	if ctx.method == "" {
		ctx.method = httpMethod
	}
}

func main() {
	ctx := &CurlContext{}

	parseArgs(ctx)

	if ctx.theUrl == "" {
		logError(ctx, "URL was not found in command line.")
		os.Exit(8)
	}

	request := buildRequest(ctx)
	client := buildClient(ctx)
	resp, err := client.Do(request)
	processResponse(ctx, resp, err, request)
}
func processResponse(ctx *CurlContext, resp *http.Response, err error, request *http.Request) {
	if resp != nil {
		err := ctx._jar.Save() // is ignored if jar's filename is empty
		if err != nil {
			logErrorF(ctx, "Failed to save cookies to jar %d", err.Error())
		}

		if resp.StatusCode >= 400 {
			// error
			if !ctx.silentFail {
				handleBodyResponse(ctx, resp, err, request)
			} else {
				logErrorF(ctx, "Failed with error code %d", resp.StatusCode)
			}
			os.Exit(6) // arbitrary
		} else {
			// success
			handleBodyResponse(ctx, resp, err, request)
		}
	} else if err != nil {
		if resp == nil {
			logErrorF(ctx, "Was unable to query URL %v", ctx.theUrl)
		} else {
			logErrorF(ctx, "Failed with error code %d", resp.StatusCode)
		}
		os.Exit(7) // arbitrary
	}
}
func parseArgs(ctx *CurlContext) {
	empty := []string{}
	flag.BoolVarP(&ctx.version, "version", "V", false, "Return version and exit")
	flag.BoolVarP(&ctx.verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flag.StringVar(&ctx.errorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flag.StringVarP(&ctx.method, "method", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flag.StringVarP(&ctx.output, "output", "o", "stdout", "Where to output results")
	flag.StringVarP(&ctx.headerOutput, "dump-header", "D", "", "Where to output headers (not on by default)")
	flag.StringVarP(&ctx.userAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flag.StringVarP(&ctx.userAuth, "user", "u", "", "User:password for HTTP authentication")
	flag.StringVarP(&ctx.referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flag.StringVar(&ctx.theUrl, "url", "", "Requesting URL")
	flag.BoolVarP(&ctx.silentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flag.BoolVarP(&ctx.ignoreBadCerts, "insecure", "k", false, "Ignore invalid SSL certificates")
	flag.BoolVarP(&ctx.isSilent, "silent", "s", false, "Silence all program console output")
	flag.BoolVarP(&ctx.showErrorEvenIfSilent, "show-error", "S", false, "Show error info even if silent mode on")
	flag.BoolVarP(&ctx.headOnly, "head", "I", false, "Only return headers (ignoring body content)")
	flag.BoolVarP(&ctx.includeHeadersInMainOutput, "include", "i", false, "Include headers (prepended to body content)")
	flag.StringSliceVarP(&ctx.cookies, "cookie", "b", empty, "HTTP cookie, raw HTTP cookie only (use -c for cookie jar files)")
	flag.StringSliceVarP(&ctx.form_encoded, "data", "d", empty, "HTML form data, set mime type to 'application/x-www-form-urlencoded'")
	flag.StringSliceVarP(&ctx.form_multipart, "form", "F", empty, "HTML form data, set mime type to 'multipart/form-data'")
	flag.StringVarP(&ctx.cookieJar, "cookie-jar", "c", "", "File for storing (read and write) cookies")
	flag.StringVarP(&ctx.uploadFile, "upload-file", "T", "", "Raw file to PUT (default) to the url given, not encoded")
	flag.Parse()

	if ctx.version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
	}

	if ctx.verbose {
		if ctx.headerOutput == "" {
			ctx.headerOutput = ctx.output // emit headers
		}
	}

	// do sanity checks and "fix" some parts left remaining from flag parsing
	tempUrl := strings.Join(flag.Args(), " ")
	if ctx.theUrl == "" && tempUrl != "" {
		ctx.theUrl = tempUrl
	}
	ctx.userAgent = strings.ReplaceAll(ctx.userAgent, "##DE"+"V##", "dev-branch") // split as I want to keep proper date versions unmunged

	if ctx.silentFail || ctx.isSilent {
		ctx.isSilent = true   // implied
		ctx.silentFail = true // both are the same thing right now, we only emit errors (or content)
		if ctx.output == "stdout" {
			ctx.output = "null"
		}
	}
	if ctx.headOnly {
		if ctx.headerOutput == "" {
			ctx.headerOutput = "-"
		}
		ctx.SetMethodIfNotSet("HEAD")
	}

	if ctx.theUrl != "" {
		u, err := url.Parse(ctx.theUrl)
		changed := false
		if err != nil {
			panic(err)
		}
		if u.Scheme == "" {
			u.Scheme = "http"
			changed = true
		}
		if u.Host == "" {
			u.Host = "localhost"
			changed = true
		}
		if changed {
			ctx.theUrl = u.String()
		}
	}

	ctx._jar = createEmptyJar(ctx)

	handleFormsAndFiles(ctx)

	// this should be LAST!
	ctx.SetMethodIfNotSet("GET")

	ctx.headerOutput = standardizeFileRef(ctx.headerOutput)
	ctx.output = standardizeFileRef(ctx.output)
	ctx.errorOutput = standardizeFileRef(ctx.errorOutput)
}
func createEmptyJar(ctx *CurlContext) (jar *cookieJar.Jar) {
	jar, _ = cookieJar.New(&cookieJar.Options{
		PublicSuffixList:      publicsuffix.List,
		Filename:              ctx.cookieJar,
		PersistSessionCookies: true,
	})
	return
}
func handleFormsAndFiles(ctx *CurlContext) {
	if ctx.uploadFile != "" {
		f, err := os.ReadFile(ctx.uploadFile)
		if err != nil {
			logErrorF(ctx, "Failed to read file %s", ctx.uploadFile)
			os.Exit(9)
		}
		mime := mime.TypeByExtension(path.Ext(ctx.uploadFile))
		if mime == "" {
			mime = "application/octet-stream"
		}
		body := &bytes.Buffer{}
		body.Write(f)
		ctx.SetBody(body, mime, "POST")

	} else if len(ctx.form_encoded) > 0 {
		formBody := url.Values{}
		for _, item := range ctx.form_encoded {
			if strings.HasPrefix(item, "@") {
				filename := strings.TrimPrefix(item, "@")
				fullForm, err := os.ReadFile(filename)
				if err != nil {
					logErrorF(ctx, "Failed to read file %s", filename)
					os.Exit(9)
				}
				formLines := strings.Split(string(fullForm), "\n")
				for _, line := range formLines {
					splits := strings.SplitN(line, "=", 2)
					name := splits[0]
					value := splits[1]
					formBody.Set(name, value)
				}
			} else {
				splits := strings.SplitN(item, "=", 2)
				os.Stdout.WriteString(item)
				name := splits[0]
				value := splits[1]

				if strings.HasPrefix(value, "@") {
					filename := strings.TrimPrefix(value, "@")
					valueRaw, err := os.ReadFile(filename)
					if err != nil {
						logErrorF(ctx, "Failed to read file %s", filename)
						os.Exit(9)
					}
					//formBody.Set(name, base64.StdEncoding.EncodeToString(valueRaw))
					formBody.Set(name, string(valueRaw))
				} else {
					formBody.Set(name, value)
				}
			}
		}
		body := strings.NewReader(formBody.Encode())
		ctx.SetBody(body, "application/x-www-form-urlencoded", "POST")

	} else if len(ctx.form_multipart) > 0 {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for _, item := range ctx.form_multipart {
			if strings.HasPrefix(item, "@") {
				filename := strings.TrimPrefix(item, "@")
				fullForm, err := os.ReadFile(filename)
				if err != nil {
					logErrorF(ctx, "Failed to read file %s", filename)
					os.Exit(9)
				}
				formLines := strings.Split(string(fullForm), "\n")
				for _, line := range formLines {
					splits := strings.SplitN(line, "=", 2)
					name := splits[0]
					value := splits[1]
					part, _ := writer.CreateFormField(name)
					part.Write([]byte(value))
				}
			} else {
				splits := strings.SplitN(item, "=", 2)
				name := splits[0]
				value := splits[1]

				if strings.HasPrefix(value, "@") {
					filename := strings.TrimPrefix(value, "@")
					valueRaw, err := os.ReadFile(filename)
					if err != nil {
						logErrorF(ctx, "Failed to read file %s", filename)
						os.Exit(9)
					}
					part, _ := writer.CreateFormFile(name, path.Base(filename))
					part.Write(valueRaw)
				} else {
					part, _ := writer.CreateFormField(name)
					part.Write([]byte(value))
				}
			}
		}
		writer.Close()

		ctx.SetBody(body, "multipart/form-data; boundary="+writer.Boundary(), "POST")
	}
}
func logErrorF(ctx *CurlContext, entry string, value interface{}) {
	logError(ctx, fmt.Sprintf(entry, value))
}
func logError(ctx *CurlContext, entry string) {
	if (!ctx.isSilent && !ctx.silentFail) || !ctx.showErrorEvenIfSilent {
		writeToFileBytes(ctx.errorOutput, []byte(entry+"\n"))
	}
}
func buildClient(ctx *CurlContext) (client *http.Client) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	if ctx.ignoreBadCerts {
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client = &http.Client{
		Transport: customTransport,
		Jar:       ctx._jar,
	}
	return
}
func buildRequest(ctx *CurlContext) (request *http.Request) {
	request, _ = http.NewRequest(strings.ToUpper(ctx.method), ctx.theUrl, ctx._body)
	if ctx._body_contentType != "" {
		request.Header.Add("Content-Type", ctx._body_contentType)
	}
	if ctx.userAgent != "" {
		request.Header.Set("User-Agent", ctx.userAgent)
	} else {
		request.Header.Del("User-Agent")
	}
	if ctx.referer != "" {
		request.Header.Set("Referer", ctx.referer)
	}
	if ctx.cookies != nil {
		for _, cookie := range ctx.cookies {
			request.Header.Add("Cookie", cookie)
		}
	}
	if ctx.userAuth != "" {
		auths := strings.SplitN(ctx.userAuth, ":", 2)
		if len(auths) == 1 {
			fmt.Print("Enter password: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n') // if unable to read, use blank instead
			auths = append(auths, input)
		}
		request.SetBasicAuth(auths[0], auths[1])
	}

	return request
}
func handleBodyResponse(ctx *CurlContext, resp *http.Response, err error, request *http.Request) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	if ctx.headOnly {
		writeToFileBytes(ctx.headerOutput, getHeaderBytes(ctx, resp, request, false))
	} else if ctx.includeHeadersInMainOutput {
		bytesOut := append(getHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx.output, bytesOut) // do all at once
		if ctx.headerOutput != ctx.output {
			writeToFileBytes(ctx.headerOutput, getHeaderBytes(ctx, resp, request, false)) // also emit headers to separate location??
		}
	} else if ctx.headerOutput == ctx.output {
		bytesOut := append(getHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx.output, bytesOut) // do all at once
	} else {
		writeToFileBytes(ctx.headerOutput, getHeaderBytes(ctx, resp, request, false))
		writeToFileBytes(ctx.output, respBody)
	}
}
func getTlsVersionString(version uint16) (res string) {
	switch version {
	case tls.VersionSSL30:
		res = "SSL 3.0"
	case tls.VersionTLS10:
		res = "TLS 1.0"
	case tls.VersionTLS11:
		res = "TLS 1.1"
	case tls.VersionTLS12:
		res = "TLS 1.2"
	case tls.VersionTLS13:
		res = "TLS 1.3"
	case 0x0305:
		res = "TLS 1.4?"
	default:
		res = fmt.Sprintf("Unknown (%v)", version)
	}
	return
}
func getTlsDetails(conn *tls.ConnectionState) (res []string) {
	res = append(res, fmt.Sprintf("TLS Version: %v", getTlsVersionString(conn.Version)))
	res = append(res, fmt.Sprintf("TLS Cipher Suite: %v", tls.CipherSuiteName(conn.CipherSuite)))
	if conn.NegotiatedProtocol != "" {
		res = append(res, fmt.Sprintf("TLS Negotiated Protocol: %v", conn.NegotiatedProtocol))
	}
	if conn.ServerName != "" {
		res = append(res, fmt.Sprintf("TLS Server Name: %v", conn.ServerName))
	}
	return
}
func getHeaderBytes(ctx *CurlContext, resp *http.Response, req *http.Request, appendLine bool) (res []byte) {
	headerString := ""
	if ctx.verbose {
		if resp.TLS != nil {
			headerString = strings.Join(formatRequestHeaders(req), "\n") + "\n\n" + strings.Join(getTlsDetails(resp.TLS), "\n") + "\n\n" + strings.Join(formatResponseHeaders(resp, true), "\n")
		} else {
			headerString = strings.Join(formatRequestHeaders(req), "\n") + "\n\n" + strings.Join(formatResponseHeaders(resp, true), "\n")
		}
	} else {
		headerString = strings.Join(formatResponseHeaders(resp, false), "\n")
	}
	if appendLine {
		headerString = headerString + "\n\n"
	}
	res = []byte(headerString) // now contains verbose details, if necessary
	return
}
func formatResponseHeaders(resp *http.Response, verboseFormat bool) (res []string) {
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
func formatRequestHeaders(req *http.Request) (res []string) {
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
func standardizeFileRef(file string) string {
	if file == "/dev/null" || file == "null" || file == "" {
		return "/dev/null"
	}
	if file == "/dev/stderr" || file == "stderr" {
		return "/dev/stderr"
	}
	if file == "/dev/stdout" || file == "stdout" || file == "-" {
		return "/dev/stdout"
	}
	return file // no change
}
func writeToFileBytes(file string, body []byte) {
	if file == "/dev/null" {
		// do nothing
	} else if file == "/dev/stderr" {
		os.Stderr.Write(body)
	} else if file == "/dev/stdout" {
		os.Stdout.Write(body)
	} else {
		os.WriteFile(file, body, 0644)
	}
}
