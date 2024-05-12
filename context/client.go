package context

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func (ctx *CurlContext) BuildClient() (client *http.Client) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	if ctx.IgnoreBadCerts {
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client = &http.Client{
		Transport: customTransport,
		Jar:       ctx.Jar,
	}
	return
}

func (ctx *CurlContext) BuildRequest(index int) (request *http.Request, err *curlerrors.CurlError) {
	url := ctx.Urls[index]

	var upload *UploadInformation
	// must call these BEFORE using ctx.method (as they may set it to POST/PUT if not yet explicitly set)
	// fixme: add support for mixing them (upload file vs all others?)
	// fixme: add --data-binary support
	if len(ctx.Upload_File) > 0 {
		upload, err = ctx.HandleUploadRawFile(index)
		if err != nil {
			return nil, err // just stop now
		}
	}
	if len(ctx.Form_Multipart) > 0 {
		upload, err = ctx.HandleFormMultipart()
		if err != nil {
			return nil, err // just stop now
		}
	}
	if ctx.HasDataArgs() {
		upload, err = ctx.HandleDataArgs()
		if err != nil {
			return nil, err // just stop now
		}
	}

	if upload == nil {
		upload = &UploadInformation{} // null-safe default please
	}

	// this should be after all other changes to method!
	if upload.RecommendedMethod != "" {
		ctx.SetMethodIfNotSet(upload.RecommendedMethod)
	} else {
		ctx.SetMethodIfNotSet("GET")
	}

	// now build
	request, _ = http.NewRequest(strings.ToUpper(ctx.Method), url, upload.Body)

	// custom headers ALWAYS come first (we use `set` below to override when needed)
	if len(ctx.Headers) > 0 {
		for _, h := range ctx.Headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				request.Header.Add(parts[0], parts[1])
			}
		}
	}
	if upload.RecommendedMimeType != "" && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", upload.RecommendedMimeType)
	}
	if ctx.UserAgent != "" {
		request.Header.Set("User-Agent", ctx.UserAgent)
	} else if request.Header.Get("User-Agent") == "" {
		request.Header.Del("User-Agent")
	}
	if ctx.Referer != "" {
		request.Header.Set("Referer", ctx.Referer)
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

func (ctx *CurlContext) Do(client *http.Client, request *http.Request) (*http.Response, *curlerrors.CurlError) {
	resp, err := client.Do(request)
	if err != nil {
		cerr := curlerrors.NewCurlError2(curlerrors.ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", request.URL), err)
		return resp, cerr
	}
	return resp, nil
}

func (ctx *CurlContext) ProcessResponse(index int, resp *http.Response, request *http.Request) (cerr *curlerrors.CurlError) {
	err2 := ctx.Jar.Save() // is ignored if jar's filename is empty
	if err2 != nil {
		cerr = curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar", err2)
		// continue anyways!
	}

	if resp.StatusCode >= 400 {
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
