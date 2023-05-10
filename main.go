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
	"net/textproto"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	cookieJar "github.com/juju/persistent-cookiejar"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/publicsuffix"
)

type CurlContext struct {
	version                    bool
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
	_jar                       cookieJar.Jar
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
	processResponse(ctx, resp, err)
}
func processResponse(ctx *CurlContext, resp *http.Response, err error) {
	if resp != nil {
		ctx._jar.Save() // is ignored if jar's filename is empty

		if resp.StatusCode >= 400 {
			// error
			if !ctx.silentFail {
				handleBodyResponse(ctx, resp, err)
			} else {
				logErrorF(ctx, "Failed with error code %d", resp.StatusCode)
			}
			os.Exit(6) // arbitrary
		} else {
			// success
			handleBodyResponse(ctx, resp, err)
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
	flag.BoolVarP(&ctx.version, "version", "v", false, "Return version and exit")
	flag.StringVar(&ctx.errorOutput, "stderr", "stderr", "Log errors to this replacement for stderr")
	flag.StringVarP(&ctx.method, "method", "X", "", "HTTP method to use (usually GET unless otherwise modified by other parameters)")
	flag.StringVarP(&ctx.output, "output", "o", "stdout", "Where to output results")
	flag.StringVarP(&ctx.headerOutput, "dump-header", "D", "/dev/null", "Where to output headers")
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
	flag.StringSliceVarP(&ctx.cookies, "cookie", "b", empty, "HTTP cookie, raw, can be repeated")
	flag.StringSliceVarP(&ctx.form_encoded, "data", "d", empty, "HTML form data, set mime type to 'application/x-www-form-urlencoded'")
	flag.StringSliceVarP(&ctx.form_multipart, "form", "F", empty, "HTML form data, set mime type to 'multipart/form-data'")
	flag.StringVarP(&ctx.cookieJar, "cookie-jar", "c", "", "File for storing (and reading) cookies")
	flag.StringVarP(&ctx.uploadFile, "upload-file", "T", "", "Raw file to PUT (default) to the url given, not encoded")
	flag.Parse()

	if ctx.version {
		os.Stdout.WriteString("go-curling build ##DEV##\n")
		os.Exit(0)
	}

	// do sanity checks and "fix" some parts left remaining from flag parsing
	tempUrl := strings.Join(flag.Args(), " ")
	if ctx.theUrl == "" && tempUrl != "" {
		ctx.theUrl = tempUrl
	}

	if ctx.silentFail || ctx.isSilent {
		ctx.isSilent = true   // implied
		ctx.silentFail = true // both are the same thing right now, we only emit errors (or content)
	}
	if ctx.headOnly {
		if ctx.headerOutput == "/dev/null" {
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

	handleFormsAndFiles(ctx)

	// this should be LAST!
	ctx.SetMethodIfNotSet("GET")
}
func handleFormsAndFiles(ctx *CurlContext) {
	if ctx.uploadFile != "" {
		f, err := os.Open(ctx.uploadFile)
		if err != nil {
			logErrorF(ctx, "Failed to read file %s", ctx.uploadFile)
			os.Exit(9)
		}
		defer f.Close()
		mime := mime.TypeByExtension(path.Ext(ctx.uploadFile))
		if mime == "" {
			mime = "application/octet-stream"
		}
		ctx.SetBody(f, mime, "POST")

	} else if len(ctx.form_encoded) > 0 {
		formBody := url.Values{}
		for _, item := range ctx.form_encoded {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]

			if strings.HasPrefix(value, "@") {
				valueRaw, err := os.ReadFile(value)
				if err != nil {
					logErrorF(ctx, "Failed to read file %s", value)
					os.Exit(9)
				}
				//formBody.Set(name, base64.StdEncoding.EncodeToString(valueRaw))
				formBody.Set(name, string(valueRaw))
			}
			formBody.Set(name, value)
		}
		body := strings.NewReader(formBody.Encode())
		ctx.SetBody(body, "application/x-www-form-urlencoded", "POST")

	} else if len(ctx.form_multipart) > 0 {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for _, item := range ctx.form_multipart {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]

			part, _ := writer.CreatePart(textproto.MIMEHeader{
				"Name": []string{name},
			})
			if strings.HasPrefix(value, "@") {
				valueRaw, err := os.ReadFile(value)
				if err != nil {
					logErrorF(ctx, "Failed to read file %s", value)
					os.Exit(9)
				}
				part.Write(valueRaw)
			}
			writer.Close()
		}

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

	newjar, _ := cookieJar.New(&cookieJar.Options{
		PublicSuffixList: publicsuffix.List,
		Filename:         ctx.cookieJar,
	})
	ctx._jar = *newjar // save for later!

	client = &http.Client{
		Transport: customTransport,
		Jar:       newjar,
	}
	return
}
func buildRequest(ctx *CurlContext) (request *http.Request) {
	request, _ = http.NewRequest(ctx.method, ctx.theUrl, ctx._body)
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
func handleBodyResponse(ctx *CurlContext, resp *http.Response, err error) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	headerString := strings.Join(formatResponseHeaders(resp), "\n")
	headerBytesOut := []byte(headerString)
	if ctx.headOnly {
		writeToFileBytes(ctx.headerOutput, headerBytesOut)
	} else if ctx.includeHeadersInMainOutput {
		bytesOut := append(headerBytesOut, respBody...)
		writeToFileBytes(ctx.output, bytesOut) // do all at once
		if ctx.headerOutput != ctx.output {
			writeToFileBytes(ctx.headerOutput, headerBytesOut) // also emit headers to separate location??
		}
	} else if ctx.headerOutput == ctx.output {
		bytesOut := append(headerBytesOut, respBody...)
		writeToFileBytes(ctx.output, bytesOut) // do all at once
	} else {
		writeToFileBytes(ctx.headerOutput, headerBytesOut)
		writeToFileBytes(ctx.output, respBody)
	}
}
func formatResponseHeaders(resp *http.Response) (res []string) {
	proto := resp.Request.Proto
	if resp.Request.Proto == "" {
		proto = "HTTP/?" // default, sometimes golang won't let you have the HTTP protocol version in the response
	}
	res = append(res, fmt.Sprintf("%s %d %v", proto, resp.StatusCode, resp.Request.URL))
	dict := make(map[string]string)
	keys := make([]string, 0, len(resp.Header))
	for name, values := range resp.Header {
		keys = append(keys, name)
		for _, value := range values {
			dict[name] = value
		}
	}
	sort.Strings(keys)
	for _, name := range keys { // I want them alphabetical
		res = append(res, fmt.Sprintf("%s: %s", name, dict[name]))
	}

	return
}
func writeToFileBytes(file string, body []byte) {
	if file == "/dev/null" || file == "null" {
		// do nothing
	} else if file == "/dev/stderr" || file == "stderr" {
		os.Stderr.Write(body)
	} else if file == "-" || file == "/dev/stdout" || file == "stdout" {
		// stdout
		os.Stdout.Write(body)
	} else if file == "" {
		// do nothing, no file to push to..
	} else {
		// output to file
		os.WriteFile(file, body, 0644)
	}
}
