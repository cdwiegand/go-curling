package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	cookieJar "github.com/orirawlings/persistent-cookiejar"
	"golang.org/x/net/publicsuffix"
)

func (ctx *CurlContext) BuildRequest(index int) (request *http.Request) {
	url := ctx.urls[index]

	var body io.Reader // nil
	mime := ""
	// must call these BEFORE using ctx.method (as they may set it to POST/PUT if not yet explicitly set)
	// fixme: add support for mixing them (upload file vs all others?)
	// fixme: add --data-binary support
	if len(ctx.uploadFile) > 0 {
		body, mime = ctx.HandleUploadRawFile(index)
	} else if len(ctx.data_standard) > 0 {
		body, mime = ctx.HandleFormRawWithAtFileSupport()
	} else if len(ctx.data_encoded) > 0 {
		body, mime = ctx.HandleFormEncoded()
	} else if len(ctx.data_rawconcat) > 0 {
		body, mime = ctx.HandleFormRawConcat()
	} else if len(ctx.data_multipart) > 0 {
		body, mime = ctx.HandleFormMultipart()
	}

	// this should be after all other changes to method!
	ctx.SetMethodIfNotSet("GET")

	// now build
	request, _ = http.NewRequest(strings.ToUpper(ctx.method), url, body)

	// custom headers ALWAYS come first (we use `set` below to override when needed)
	if len(ctx.headers) > 0 {
		for _, h := range ctx.headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				request.Header.Add(parts[0], parts[1])
			}
		}
	}
	if mime != "" && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", mime)
	}
	if ctx.userAgent != "" {
		request.Header.Set("User-Agent", ctx.userAgent)
	} else if request.Header.Get("User-Agent") == "" {
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
func (ctx *CurlContext) SetMethodIfNotSet(httpMethod string) {
	if ctx.method == "" {
		ctx.method = httpMethod
	}
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
