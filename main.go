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
	output                     []string
	headerOutput               []string
	userAgent                  string
	urls                       []string
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
	uploadFile                 []string
	form_encoded               []string
	form_multipart             []string
	_bodies                    []io.Reader
	_bodies_contentType        []string
}

func (ctx *CurlContext) AddBody(body io.Reader, mimeType string, httpMethod string) {
	ctx._bodies = append(ctx._bodies, body)
	ctx._bodies_contentType = append(ctx._bodies_contentType, mimeType)
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

const DEFAULT_OUTPUT = "stdout"

func main() {
	ctx := &CurlContext{}

	// I want to be able to test using my own args[], so can't use default flag.Parse()..
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	SetupFlagArgs(ctx, flags)
	flags.Parse(os.Args[1:])

	extraArgs := flags.Args() // remaining non-parsed args
	SetupContextForRun(ctx, extraArgs)

	if ctx.version {
		os.Stdout.WriteString("go-curling build ##DEV##")
		os.Exit(0)
		return
	}

	// must be after version check
	if len(ctx.urls) == 0 {
		err := errors.New("URL was not found on the command line")
		HandleErrorAndExit(err, ctx, ERROR_STATUS_CODE_FAILURE, "Parse URL")
	}

	client := BuildClient(ctx)

	for index := range ctx.urls {
		request := BuildRequest(ctx, index)
		resp, err := client.Do(request)
		ProcessResponse(ctx, index, resp, err, request)
	}
}
func ProcessResponse(ctx *CurlContext, index int, resp *http.Response, err error, request *http.Request) {
	HandleErrorAndExit(err, ctx, ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", ctx.urls))

	err2 := ctx._jar.Save() // is ignored if jar's filename is empty
	HandleErrorAndExit(err2, ctx, ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar")

	if resp.StatusCode >= 400 {
		// error
		if !ctx.silentFail {
			HandleBodyResponse(ctx, index, resp, request)
		}
		os.Exit(6) // arbitrary
	} else {
		// success
		HandleBodyResponse(ctx, index, resp, request)
	}
}
func SetupFlagArgs(ctx *CurlContext, flags *flag.FlagSet) {
	empty := []string{}
	flags.BoolVarP(&ctx.version, "version", "V", false, "Return version and exit")
	flags.BoolVarP(&ctx.verbose, "verbose", "v", false, "Logs all headers, and body to output")
	flags.StringVar(&ctx.errorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flags.StringVarP(&ctx.method, "method", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flags.StringArrayVarP(&ctx.output, "output", "o", []string{DEFAULT_OUTPUT}, "Where to output results")
	flags.StringArrayVarP(&ctx.headerOutput, "dump-header", "D", []string{}, "Where to output headers (not on by default)")
	flags.StringVarP(&ctx.userAgent, "user-agent", "A", "go-curling/##DEV##", "User-agent to use")
	flags.StringVarP(&ctx.userAuth, "user", "u", "", "User:password for HTTP authentication")
	flags.StringVarP(&ctx.referer, "referer", "e", "", "Referer URL to use with HTTP request")
	flags.StringArrayVar(&ctx.urls, "url", []string{}, "Requesting URL")
	flags.BoolVarP(&ctx.silentFail, "fail", "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flags.BoolVarP(&ctx.ignoreBadCerts, "insecure", "k", false, "Ignore invalid SSL certificates")
	flags.BoolVarP(&ctx.isSilent, "silent", "s", false, "Silence all program console output")
	flags.BoolVarP(&ctx.showErrorEvenIfSilent, "show-error", "S", false, "Show error info even if silent mode on")
	flags.BoolVarP(&ctx.headOnly, "head", "I", false, "Only return headers (ignoring body content)")
	flags.BoolVarP(&ctx.includeHeadersInMainOutput, "include", "i", false, "Include headers (prepended to body content)")
	flags.StringSliceVarP(&ctx.cookies, "cookie", "b", empty, "HTTP cookie, raw HTTP cookie only (use -c for cookie jar files)")
	flags.StringSliceVarP(&ctx.form_encoded, "data", "d", empty, "HTML form data, set mime type to 'application/x-www-form-urlencoded'")
	flags.StringSliceVarP(&ctx.form_multipart, "form", "F", empty, "HTML form data, set mime type to 'multipart/form-data'")
	flags.StringVarP(&ctx.cookieJar, "cookie-jar", "c", "", "File for storing (read and write) cookies")
	flags.StringArrayVarP(&ctx.uploadFile, "upload-file", "T", []string{}, "Raw file(s) to PUT (default) to the url(s) given, not encoded")
}
func SetupContextForRun(ctx *CurlContext, extraArgs []string) {
	if ctx.verbose && len(ctx.headerOutput) == 0 {
		ctx.headerOutput = ctx.output // emit headers
	}

	// do sanity checks and "fix" some parts left remaining from flag parsing
	urls := append(ctx.urls, extraArgs...)

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

	if len(ctx.uploadFile) > 0 {
		HandleUploadFile(ctx)
	} else if len(ctx.form_encoded) > 0 {
		HandleFormEncoded(ctx)
	} else if len(ctx.form_multipart) > 0 {
		HandleFormMultipart(ctx)
	}

	// this should be after all other changes to method!
	ctx.SetMethodIfNotSet("GET")
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
	for _, file := range ctx.uploadFile {
		f, err := os.ReadFile(file)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", ctx.uploadFile))
		mime := mime.TypeByExtension(path.Ext(file))
		if mime == "" {
			mime = "application/octet-stream"
		}
		body := &bytes.Buffer{}
		body.Write(f)
		ctx.AddBody(body, mime, "PUT")
	}
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
	ctx.AddBody(body, "application/x-www-form-urlencoded", "POST")
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

	ctx.AddBody(body, "multipart/form-data; boundary="+writer.Boundary(), "POST")
}
func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
func HandleErrorAndExit(err error, ctx *CurlContext, exitCode int, entry string) {
	if err == nil {
		return
	}
	if entry == "" {
		entry = "Error"
	}
	entry += ": "
	entry += err.Error()
	if exitCode == ERROR_CANNOT_WRITE_TO_STDOUT {
		// don't recurse (it called us to report the failure to write errors to a normal file)
		PanicIfError(err)
	} else if (!ctx.isSilent && !ctx.silentFail) || !ctx.showErrorEvenIfSilent {
		writeToFileBytes(ctx, ctx.errorOutput, []byte(entry+"\n"))
	}
	if exitCode != 0 {
		os.Exit(exitCode)
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
func BuildRequest(ctx *CurlContext, index int) (request *http.Request) {
	url := ctx.urls[index]
	body, mime := getNextInputsFromContext(ctx, index)
	request, _ = http.NewRequest(strings.ToUpper(ctx.method), url, body)
	if mime != "" {
		request.Header.Add("Content-Type", mime)
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
		auths := strings.SplitN(ctx.userAuth, ":", 2) // this way password can contain a :
		if len(auths) == 1 {
			fmt.Print("Enter password: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n') // if unable to read, use blank instead
			auths = append(auths, input)
			ctx.userAuth = strings.Join(auths, ":") // for next request, if any
		}
		request.SetBasicAuth(auths[0], auths[1])
	}

	return request
}
func getNextInputsFromContext(ctx *CurlContext, index int) (body io.Reader, mime string) {
	if len(ctx._bodies) > index {
		body = ctx._bodies[index]
	} else {
		body = nil
	}
	if len(ctx._bodies_contentType) > index {
		mime = ctx._bodies_contentType[index]
	} else {
		mime = ""
	}
	return
}
func HandleBodyResponse(ctx *CurlContext, index int, resp *http.Response, request *http.Request) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	headerOutput, contentOutput := getNextOutputsFromContext(ctx, index)

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
func getNextOutputsFromContext(ctx *CurlContext, index int) (headerOutput string, contentOutput string) {
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
func writeToFileBytes(ctx *CurlContext, file string, body []byte) {
	if file == "/dev/null" || file == "null" || file == "" {
		// do nothing
	} else if file == "/dev/stderr" || file == "stderr" {
		_, err := os.Stderr.Write(body)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_TO_STDOUT, "Could not write to stderr")
	} else if file == "/dev/stdout" || file == "stdout" || file == "-" {
		_, err := os.Stdout.Write(body)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_TO_STDOUT, "Could not write to stdout")
	} else {
		err := os.WriteFile(file, body, 0644)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_WRITE_FILE, fmt.Sprintf("Could not write to file %q", file))
		// ^^ could call us back, but with stderr as the output, so it's not recursive
	}
}
