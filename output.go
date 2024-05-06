package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

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
