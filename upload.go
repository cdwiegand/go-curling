package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"strings"
)

func (ctx *CurlContext) HandleUploadFile(index int) (body io.Reader, mimeType string) {
	// DOES use index - sends a file per URL
	if len(ctx.uploadFile) > index {
		file := ctx.uploadFile[index]
		f, err := os.ReadFile(file)
		HandleErrorAndExit(err, ctx, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", ctx.uploadFile))
		mimeType := mime.TypeByExtension(path.Ext(file))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		bodyBuf := &bytes.Buffer{}
		bodyBuf.Write(f)
		body = io.Reader(bodyBuf)
		ctx.SetMethodIfNotSet("PUT")
	}
	return
}
func (ctx *CurlContext) HandleFormEncoded() (body io.Reader, mimeType string) {
	// does not use index - form encoded sends the whole form to every URL given
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
	body = strings.NewReader(formBody.Encode())
	mimeType = "application/x-www-form-urlencoded"
	ctx.SetMethodIfNotSet("POST")
	return
}
func (ctx *CurlContext) HandleFormMultipart() (body io.Reader, mimeType string) {
	// does not use index - form multipart sends the whole form to every URL given
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)
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

	body = bodyBuf
	mimeType = "multipart/form-data; boundary=" + writer.Boundary()
	ctx.SetMethodIfNotSet("POST")
	return
}
