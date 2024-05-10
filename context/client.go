package context

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
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

func (ctx *CurlContext) BuildRequest(index int) (request *http.Request) {
	url := ctx.Urls[index]

	upload := &UploadInformation{}
	// must call these BEFORE using ctx.method (as they may set it to POST/PUT if not yet explicitly set)
	// fixme: add support for mixing them (upload file vs all others?)
	// fixme: add --data-binary support
	if len(ctx.UploadFile) > 0 {
		upload = ctx.HandleUploadRawFile(index)
	} else if len(ctx.Data_standard) > 0 {
		upload = ctx.HandleFormRawWithAtFileSupport()
	} else if len(ctx.Data_encoded) > 0 {
		upload = ctx.HandleFormEncoded()
	} else if len(ctx.Data_rawconcat) > 0 {
		upload = ctx.HandleFormRawConcat()
	} else if len(ctx.Data_multipart) > 0 {
		upload = ctx.HandleFormMultipart()
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
			request.Header.Add("Cookie", cookie)
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

	return request
}

func (ctx *CurlContext) ProcessResponse(index int, resp *http.Response, err error, request *http.Request) {
	ctx.HandleErrorAndExit(err, ERROR_NO_RESPONSE, fmt.Sprintf("Was unable to query URL %v", ctx.Urls[index]))

	err2 := ctx.Jar.Save() // is ignored if jar's filename is empty
	ctx.HandleErrorAndExit(err2, ERROR_CANNOT_WRITE_FILE, "Failed to save cookies to jar")

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
}
