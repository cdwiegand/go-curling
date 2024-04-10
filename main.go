package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
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

const ERROR_STATUS_CODE_FAILURE = -6
const ERROR_NO_RESPONSE = -7
const ERROR_INVALID_URL = -8
const ERROR_CANNOT_READ_FILE = -9
const ERROR_CANNOT_WRITE_FILE = -10
const ERROR_CANNOT_WRITE_TO_STDOUT = -11

func main() {
	ctx := &CurlContext{}

	parseArgs(ctx)
	SetupContextForRun(ctx)

	if ctx.version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
		return
	}

	// must be after version check
	if ctx.theUrl == "" {
		err := errors.New("URL was not found on the command line")
		HandleErrorAndExit(err, ctx, ERROR_STATUS_CODE_FAILURE, "Parse URL")
	}

	request := BuildRequest(ctx)
	client := BuildClient(ctx)
	resp, err := client.Do(request)
	ProcessResponse(ctx, resp, err, request)
}
func ProcessResponse(ctx *CurlContext, resp *http.Response, err error, request *http.Request) {
	HandleErrorAndExit(err, ctx, ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", ctx.theUrl))

	err2 := ctx._jar.Save() // is ignored if jar's filename is empty
	HandleErrorAndExit(err2, ctx, ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar")

	if resp.StatusCode >= 400 {
		// error
		if !ctx.silentFail {
			HandleBodyResponse(ctx, resp, request)
		}
		os.Exit(6) // arbitrary
	} else {
		// success
		HandleBodyResponse(ctx, resp, request)
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
}
func SetupContextForRun(ctx *CurlContext) {
	if ctx.verbose && ctx.headerOutput == "" {
		ctx.headerOutput = ctx.output // emit headers
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
		HandleErrorAndExit(err, ctx, ERROR_INVALID_URL, fmt.Sprintf("Could not parse url: %q", ctx.theUrl))
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

	ctx._jar = CreateEmptyJar(ctx)

	if ctx.uploadFile != "" {
		HandleUploadFile(ctx)
	} else if len(ctx.form_encoded) > 0 {
		HandleFormEncoded(ctx)
	} else if len(ctx.form_multipart) > 0 {
		HandleFormMultipart(ctx)
	}

	// this should be after all other changes to method!
	ctx.SetMethodIfNotSet("GET")

	ctx.headerOutput = standardizeFileRef(ctx.headerOutput)
	ctx.output = standardizeFileRef(ctx.output)
	ctx.errorOutput = standardizeFileRef(ctx.errorOutput)
}
func CreateEmptyJar(ctx *CurlContext) (jar *cookieJar.Jar) {
	jar, err := cookieJar.New(&cookieJar.Options{
		PublicSuffixList:      publicsuffix.List,
		Filename:              ctx.cookieJar,
		PersistSessionCookies: true,
	})
	HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, "Unable to create cookie jar")
	return
}

func HandleUploadFile(ctx *CurlContext) {
	f, err := os.ReadFile(ctx.uploadFile)
	HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", ctx.uploadFile))
	mime := mime.TypeByExtension(path.Ext(ctx.uploadFile))
	if mime == "" {
		mime = "application/octet-stream"
	}
	body := &bytes.Buffer{}
	body.Write(f)
	ctx.SetBody(body, mime, "POST")
}
func HandleFormEncoded(ctx *CurlContext) {
	formBody := url.Values{}
	for _, item := range ctx.form_encoded {
		if strings.HasPrefix(item, "@") {
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
			formLines := strings.Split(string(fullForm), "\n")
			for _, line := range formLines {
				splits := strings.SplitN(line, "=", 2)
				name := splits[0]
				value := splits[1]
				formBody.Set(name, value)
			}
		} else {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]

			if strings.HasPrefix(value, "@") {
				filename := strings.TrimPrefix(value, "@")
				valueRaw, err := os.ReadFile(filename)
				HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
				//formBody.Set(name, base64.StdEncoding.EncodeToString(valueRaw))
				formBody.Set(name, string(valueRaw))
			} else {
				formBody.Set(name, value)
			}
		}
	}
	body := strings.NewReader(formBody.Encode())
	ctx.SetBody(body, "application/x-www-form-urlencoded", "POST")
}
func HandleFormMultipart(ctx *CurlContext) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, item := range ctx.form_multipart {
		if strings.HasPrefix(item, "@") {
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
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
				HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
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
func HandleErrorAndExit(err error, ctx *CurlContext, exitCode int, entry string) {
	if err != nil {
		if entry == "" {
			entry = "Error"
		}
		entry += ": "
		entry += err.Error()
		if exitCode == ERROR_CANNOT_WRITE_TO_STDOUT {
			// don't recurse (it called us to report the failure to write errors to a normal file)
			panic(err)
		} else if (!ctx.isSilent && !ctx.silentFail) || !ctx.showErrorEvenIfSilent {
			writeToFileBytes(ctx, ctx.errorOutput, []byte(entry+"\n"))
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}
}
func BuildClient(ctx *CurlContext) (client *http.Client) {
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
func BuildRequest(ctx *CurlContext) (request *http.Request) {
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
func HandleBodyResponse(ctx *CurlContext, resp *http.Response, request *http.Request) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	if ctx.headOnly {
		writeToFileBytes(ctx, ctx.headerOutput, GetHeaderBytes(ctx, resp, request, false))
	} else if ctx.includeHeadersInMainOutput {
		bytesOut := append(GetHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx, ctx.output, bytesOut) // do all at once
		if ctx.headerOutput != ctx.output {
			writeToFileBytes(ctx, ctx.headerOutput, GetHeaderBytes(ctx, resp, request, false)) // also emit headers to separate location??
		}
	} else if ctx.headerOutput == ctx.output {
		bytesOut := append(GetHeaderBytes(ctx, resp, request, true), respBody...)
		writeToFileBytes(ctx, ctx.output, bytesOut) // do all at once
	} else {
		writeToFileBytes(ctx, ctx.headerOutput, GetHeaderBytes(ctx, resp, request, false))
		writeToFileBytes(ctx, ctx.output, respBody)
	}
}
func GetTlsVersionString(version uint16) (res string) {
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
func GetTlsDetails(conn *tls.ConnectionState) (res []string) {
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
func GetHeaderBytes(ctx *CurlContext, resp *http.Response, req *http.Request, appendLine bool) (res []byte) {
	headerString := ""
	if ctx.verbose {
		if resp.TLS != nil {
			headerString = strings.Join(FormatRequestHeaders(req), "\n") + "\n\n" + strings.Join(GetTlsDetails(resp.TLS), "\n") + "\n\n" + strings.Join(FormatResponseHeaders(resp, true), "\n")
		} else {
			headerString = strings.Join(FormatRequestHeaders(req), "\n") + "\n\n" + strings.Join(FormatResponseHeaders(resp, true), "\n")
		}
	} else {
		headerString = strings.Join(FormatResponseHeaders(resp, false), "\n")
	}
	if appendLine {
		headerString = headerString + "\n\n"
	}
	res = []byte(headerString) // now contains verbose details, if necessary
	return
}
func FormatResponseHeaders(resp *http.Response, verboseFormat bool) (res []string) {
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
func FormatRequestHeaders(req *http.Request) (res []string) {
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
func writeToFileBytes(ctx *CurlContext, file string, body []byte) {
	if file == "/dev/null" {
		// do nothing
	} else if file == "/dev/stderr" {
		_, err := os.Stderr.Write(body)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_TO_STDOUT, "Could not write to stderr")
	} else if file == "/dev/stdout" {
		_, err := os.Stdout.Write(body)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_TO_STDOUT, "Could not write to stdout")
	} else {
		err := os.WriteFile(file, body, 0644)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_FILE, fmt.Sprintf("Could not write to file %q", file))
		// ^^ could call us back, but with stderr as the output, so it's not recursive
	}
}
