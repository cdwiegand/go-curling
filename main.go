package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type CurlContext struct {
	method         string
	silentFail     bool
	output         string
	headerout      string
	agentout       string
	the_url        string
	ignoreBadCerts bool
}

func main() {
	ctx := &CurlContext{
		the_url: "",
	}

	flag.StringVar(&ctx.method, "X", "GET", "HTTP method to use")
	flag.StringVar(&ctx.output, "o", "-", "Where to output results")
	flag.StringVar(&ctx.headerout, "D", "/dev/null", "Where to output headers")
	flag.StringVar(&ctx.agentout, "A", "go-curling/1", "User-agent to use")
	flag.BoolVar(&ctx.silentFail, "f", false, "If fail do not emit contents just return fail exit code (-6)")
	flag.BoolVar(&ctx.ignoreBadCerts, "k", false, "Ignore invalid SSL certificates")
	flag.Parse()

	for _, val := range flag.Args() {
		val2 := strings.TrimSpace(strings.ToLower(val))
		if strings.HasPrefix(val2, "https://") || strings.HasPrefix(val2, "http://") {
			ctx.the_url = val
		}
	}

	if ctx.the_url == "" {
		if !ctx.silentFail {
			log.Fatalln("URL must be specified last.")
		}
		os.Exit(-8)
	}

	run(ctx)
}
func run(ctx *CurlContext) {
	request, err := http.NewRequest(ctx.method, ctx.the_url, nil)
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	if ctx.ignoreBadCerts {
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: customTransport}
	request.Header.Set("User-Agent", ctx.agentout)
	resp, err := client.Do(request)

	if resp != nil {
		if err == nil || !ctx.silentFail {
			// emit body
			var respBody []byte
			if resp.Body != nil {
				defer resp.Body.Close()
				respBody, _ = io.ReadAll(resp.Body)
			}

			headerString := strings.Join(formatResponseHeaders(resp), "\n")
			if ctx.headerout == ctx.output {
				bytesOut := []byte(headerString)
				bytesOut = append(bytesOut, respBody...)
				writeToFileBytes(ctx.headerout, bytesOut)
			} else {
				writeToFileBytes(ctx.headerout, []byte(headerString))
				writeToFileBytes(ctx.output, respBody)
			}
		}
	}

	if err != nil {
		if !ctx.silentFail {
			if resp == nil {
				log.Fatalf("Was unable to query URL %v", ctx.the_url)
			} else {
				log.Fatalf("Failed with error code %d", resp.StatusCode)
			}
		}
		os.Exit(-6) // arbitrary
	}
}
func formatResponseHeaders(resp *http.Response) (res []string) {
	proto := resp.Request.Proto
	if resp.Request.Proto == "" {
		proto = "HTTP/?" // default, sometimes golang won't let you have the HTTP protocol version in the response
	}
	res = append(res, fmt.Sprintf("%s %d %v", proto, resp.StatusCode, resp.Request.URL))
	for name, values := range resp.Header {
		for _, value := range values {
			res = append(res, fmt.Sprintf("%s: %s", name, value))
		}
	}
	return
}
func writeToFileBytes(file string, body []byte) {
	if file == "/dev/null" {
		// do nothing
	} else if file == "/dev/stderr" {
		os.Stderr.Write(body)
	} else if file == "-" || file == "/dev/stdout" {
		// stdout
		os.Stdout.Write(body)
	} else {
		// output to file
		os.WriteFile(file, body, 0644)
	}
}
