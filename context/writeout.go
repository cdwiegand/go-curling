package context

// Support for curl's -w / --write-out option: after a transfer completes, emit
// a user-supplied format string with %{variable} placeholders substituted from
// information about the request/response (status, sizes, timings, the final URL,
// connection details, and individual response headers via %header{name}).
//
// Timings are captured with net/http/httptrace and are measured relative to the
// start of the final request; for a reused (keep-alive) connection the phase
// timers that correspond to establishing a connection report 0, and for a
// redirect chain time_redirect holds the time spent before the final request.

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RequestTimings records the wall-clock instants of a single request's lifecycle.
type RequestTimings struct {
	Start       time.Time // client.Do called
	DNSStart    time.Time
	DNSDone     time.Time
	ConnectDone time.Time
	TLSDone     time.Time
	GotConn     time.Time
	FirstByte   time.Time
	BodyDone    time.Time // body fully read (set during output emission)
	RemoteAddr  string
	LocalAddr   string
}

// EmitWriteOut renders the --write-out format (if any) and writes it to stdout,
// matching curl, which sends -w output to stdout regardless of -o and is not
// suppressed by -s.
func (ctx *CurlContext) EmitWriteOut(resp *CurlResponses) {
	if ctx.WriteOut == "" {
		return
	}
	vars, headers := ctx.BuildWriteOutData(resp)
	out := FormatWriteOut(ctx.WriteOut, vars, headers)
	_ = ctx.WriteToFileBytes(DEFAULT_OUTPUT, []byte(out))
}

// BuildWriteOutData collects the --write-out variables and the per-header map
// from the final response of an operation.
func (ctx *CurlContext) BuildWriteOutData(resp *CurlResponses) (vars map[string]string, headers map[string]string) {
	vars = map[string]string{
		"http_code":          "000",
		"response_code":      "000",
		"num_redirects":      "0",
		"num_headers":        "0",
		"size_download":      "0",
		"size_header":        "0",
		"size_upload":        "0",
		"size_request":       "0",
		"ssl_verify_result":  "0",
		"exitcode":           "0",
		"errormsg":           "",
		"content_type":       "",
		"http_version":       "",
		"scheme":             "",
		"method":             "",
		"url":                "",
		"url_effective":      "",
		"remote_ip":          "",
		"remote_port":        "",
		"local_ip":           "",
		"local_port":         "",
		"time_namelookup":    "0.000000",
		"time_connect":       "0.000000",
		"time_appconnect":    "0.000000",
		"time_pretransfer":   "0.000000",
		"time_starttransfer": "0.000000",
		"time_total":         "0.000000",
		"time_redirect":      "0.000000",
	}
	headers = map[string]string{}

	if resp == nil || len(resp.Responses) == 0 {
		return vars, headers
	}

	vars["num_redirects"] = strconv.Itoa(resp.NumRedirects)
	vars["size_download"] = strconv.Itoa(resp.BodyBytes)

	// the original (first) URL requested
	if first := resp.Responses[0].HttpResponse; first != nil && first.Request != nil && first.Request.URL != nil {
		vars["url"] = first.Request.URL.String()
	}

	final := resp.Responses[len(resp.Responses)-1]
	hr := final.HttpResponse
	if hr != nil {
		vars["http_code"] = fmt.Sprintf("%03d", hr.StatusCode)
		vars["response_code"] = vars["http_code"]
		vars["content_type"] = hr.Header.Get("Content-Type")
		vars["http_version"] = normalizeHTTPVersion(hr.Proto)
		vars["num_headers"] = strconv.Itoa(countHeaderLines(hr.Header))
		vars["size_header"] = strconv.Itoa(headerByteSize(hr))
		if hr.TLS != nil {
			vars["ssl_verify_result"] = "0" // a response over TLS means verification passed (or -k was used)
		}
		for name, vals := range hr.Header {
			if len(vals) > 0 {
				headers[strings.ToLower(name)] = vals[len(vals)-1]
			}
		}
		if hr.Request != nil {
			vars["method"] = hr.Request.Method
			if hr.Request.URL != nil {
				vars["url_effective"] = hr.Request.URL.String()
				vars["scheme"] = strings.ToUpper(hr.Request.URL.Scheme)
			}
			if hr.Request.ContentLength > 0 {
				vars["size_upload"] = strconv.FormatInt(hr.Request.ContentLength, 10)
				vars["size_request"] = strconv.FormatInt(hr.Request.ContentLength, 10)
			}
		}
	}

	if t := final.Timings; t != nil {
		vars["time_namelookup"] = fmtSeconds(secondsBetween(t.Start, t.DNSDone))
		vars["time_connect"] = fmtSeconds(secondsBetween(t.Start, t.ConnectDone))
		vars["time_appconnect"] = fmtSeconds(secondsBetween(t.Start, t.TLSDone))
		vars["time_pretransfer"] = fmtSeconds(secondsBetween(t.Start, t.GotConn))
		vars["time_starttransfer"] = fmtSeconds(secondsBetween(t.Start, t.FirstByte))
		vars["time_total"] = fmtSeconds(secondsBetween(t.Start, t.BodyDone))
		vars["time_redirect"] = fmtSeconds(secondsBetween(resp.OperationStart, t.Start))
		if host, port, ok := splitHostPort(t.RemoteAddr); ok {
			vars["remote_ip"] = host
			vars["remote_port"] = port
		}
		if host, port, ok := splitHostPort(t.LocalAddr); ok {
			vars["local_ip"] = host
			vars["local_port"] = port
		}
	}

	return vars, headers
}

// FormatWriteOut renders a curl --write-out format string. It supports
// %{variable}, %header{name}, %% (a literal %), and the \n \t \r \\ escapes.
// Unknown %{variable} references render as empty, as curl does.
func FormatWriteOut(format string, vars map[string]string, headers map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		switch c {
		case '\\':
			if i+1 >= len(format) {
				b.WriteByte('\\')
				continue
			}
			i++
			switch format[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(format[i])
			}
		case '%':
			switch {
			case i+1 < len(format) && format[i+1] == '%':
				b.WriteByte('%')
				i++
			case i+1 < len(format) && format[i+1] == '{':
				end := strings.IndexByte(format[i+2:], '}')
				if end == -1 {
					b.WriteByte('%')
					continue
				}
				name := strings.ToLower(format[i+2 : i+2+end])
				b.WriteString(vars[name])
				i = i + 2 + end
			case strings.HasPrefix(format[i+1:], "header{"):
				rest := format[i+1+len("header{"):]
				end := strings.IndexByte(rest, '}')
				if end == -1 {
					b.WriteByte('%')
					continue
				}
				name := strings.ToLower(rest[:end])
				b.WriteString(headers[name])
				i = i + 1 + len("header{") + end
			default:
				b.WriteByte('%')
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func normalizeHTTPVersion(proto string) string {
	v := strings.TrimPrefix(proto, "HTTP/")
	switch v {
	case "1.0":
		return "1" // match curl, which reports HTTP/1.0 as "1"
	case "2.0":
		return "2"
	case "3.0":
		return "3"
	default:
		return v
	}
}

func countHeaderLines(h http.Header) int {
	n := 0
	for _, vals := range h {
		n += len(vals)
	}
	return n
}

// headerByteSize approximates the on-wire size of the response headers:
// the status line plus each "Name: value\r\n" plus the terminating CRLF.
func headerByteSize(resp *http.Response) int {
	status := resp.Status
	if status == "" {
		status = strconv.Itoa(resp.StatusCode)
	}
	n := len(resp.Proto) + 1 + len(status) + 2 // "HTTP/1.1 200 OK\r\n"
	for name, vals := range resp.Header {
		for _, v := range vals {
			n += len(name) + 2 + len(v) + 2 // "Name: value\r\n"
		}
	}
	n += 2 // final CRLF
	return n
}

func secondsBetween(start, end time.Time) float64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Seconds()
}

func fmtSeconds(s float64) string {
	return strconv.FormatFloat(s, 'f', 6, 64)
}

func splitHostPort(addr string) (host string, port string, ok bool) {
	if addr == "" {
		return "", "", false
	}
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, "", true
	}
	return h, p, true
}
