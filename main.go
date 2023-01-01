package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	the_url := ""
	var method string
	var silentFail bool
	var output string
	var headerout string

	flag.StringVar(&method, "X", "GET", "HTTP method to use")
	flag.StringVar(&output, "o", "/dev/null", "Where to output results")
	flag.StringVar(&headerout, "D", "/dev/null", "Where to output headers")
	flag.BoolVar(&silentFail, "f", false, "If fail do not emit contents just return fail exit code (-6).")
	flag.Parse()

	for _, val := range flag.Args() {
		val2 := strings.TrimSpace(strings.ToLower(val))
		if strings.HasPrefix(val2, "https://") || strings.HasPrefix(val2, "http://") {
			the_url = val
		}
	}

	if the_url == "" {
		if !silentFail {
			log.Fatalln("URL must be specified last.")
		}
		os.Exit(-8)
	}

	request, err := http.NewRequest(method, the_url, nil)
	client := new(http.Client)
	resp, err := client.Do(request)

	if resp != nil {
		if err == nil || !silentFail {
			// emit body
			respBodyStr := ""
			if resp.Body != nil {
				defer resp.Body.Close()
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}
				respBodyStr = string(respBody)
			}

			headerString := strings.Join(formatResponseHeaders(resp), "\n")
			if headerout == output {
				writeToFile(headerout, headerString+"\n\n"+respBodyStr)
			} else {
				writeToFile(headerout, headerString)
				writeToFile(output, respBodyStr)
			}
		}
	}

	if err != nil {
		if !silentFail {
			if resp == nil {
				log.Fatalf("Was unable to query URL %v", the_url)
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
func writeToFile(file string, body string) {
	if file == "/dev/null" {
		// do nothing
	} else if file == "/dev/stderr" {
		os.Stderr.WriteString(body)
	} else if file == "-" || file == "/dev/stdout" {
		// stdout
		fmt.Println(body)
	} else {
		// output to file
		os.WriteFile(file, []byte(body), 0644)
	}
}
