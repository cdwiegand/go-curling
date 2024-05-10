package context

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

func (ctx *CurlContext) EmitResponseToOutputs(index int, resp *http.Response, request *http.Request) {
	// emit body
	var respBody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)
	}

	sepBody := []byte("\n\n")
	headerBody := []byte("")
	if ctx.Verbose {
		headerBody = appendStrings(headerBody, sepBody, dumpRequestHeaders(request))
		if resp.TLS != nil {
			headerBody = appendStrings(headerBody, sepBody, dumpTlsDetails(resp.TLS))
		}
	}
	headerBody = appendStrings(headerBody, sepBody, dumpResponseHeaders(resp, ctx.Verbose))
	headerOutput, contentOutput := ctx.getNextOutputsFromContext(index)

	if ctx.HeadOnly {
		writeToFileBytes(headerOutput, headerBody)
	} else if ctx.IncludeHeadersInMainOutput {
		bytesOut := appendByteArrays(headerBody, sepBody, respBody)
		writeToFileBytes(contentOutput, bytesOut) // do all at once
		if headerOutput != contentOutput {
			writeToFileBytes(headerOutput, headerBody)
		}
	} else if headerOutput == contentOutput {
		bytesOut := appendByteArrays(headerBody, sepBody, respBody)
		writeToFileBytes(contentOutput, bytesOut) // do all at once
	} else {
		writeToFileBytes(headerOutput, headerBody)
		writeToFileBytes(contentOutput, respBody)
	}
}

func writeToFileBytes(file string, body []byte) (err error) {
	if file == "/dev/null" || file == "null" || file == "" {
		// do nothing
	} else if file == "/dev/stderr" || file == "stderr" {
		_, err = os.Stderr.Write(body)
	} else if file == "/dev/stdout" || file == "stdout" || file == "-" {
		_, err = os.Stdout.Write(body)
	} else {
		err = os.WriteFile(file, body, 0644)
	}
	return
}

func appendStrings(resp []byte, sepBody []byte, lines []string) (respOut []byte) {
	vb := []byte(strings.Join(lines, "\n"))
	respOut = appendByteArrays(resp, sepBody, vb)
	return
}

func appendByteArrays(resp []byte, sepBody []byte, secondBody []byte) (respOut []byte) {
	if len(resp) > 0 {
		resp = append(resp, sepBody...)
	}
	respOut = append(resp, secondBody...)
	return
}

func dumpResponseHeaders(resp *http.Response, verboseFormat bool) (res []string) {
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

func dumpRequestHeaders(req *http.Request) (res []string) {
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

func getTlsVersionString(version uint16) (res string) {
	switch version {
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
		res = fmt.Sprintf("Unknown TLS version: %v", version)
	}
	return
}

func dumpTlsDetails(conn *tls.ConnectionState) (res []string) {
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
