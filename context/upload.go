package context

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

type UploadInformation struct {
	Body                io.Reader
	RecommendedMimeType string
	RecommendedMethod   string
}

func (ctx *CurlContext) HandleUploadRawFile(index int) *UploadInformation {
	// DOES use index - sends a file per URL
	ret := &UploadInformation{}
	if len(ctx.UploadFile) > index {
		file := ctx.UploadFile[index]
		f, err := os.ReadFile(file)
		ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", file))
		mimeType := mime.TypeByExtension(path.Ext(file))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		bodyBuf := &bytes.Buffer{}
		bodyBuf.Write(f)

		ret.Body = io.Reader(bodyBuf)
		ret.RecommendedMethod = "PUT"
	}
	return ret
}

func (ctx *CurlContext) HandleFormRawWithAtFileSupport() *UploadInformation {
	ret := &UploadInformation{}
	lines := []string{}
	for _, item := range ctx.Data_standard {
		idxAt := strings.Index(item, "@")
		idxEqual := strings.Index(item, "=")
		idxEqualAt := strings.Index(item, "=@")
		if idxAt == 0 { // @file/path/here
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
			formLines := strings.Split(string(fullForm), "\n")
			lines = append(lines, formLines...)
		} else if idxEqual > -1 && idxEqual == idxEqualAt { // name=@value
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			filename := strings.TrimPrefix(value, "@")
			valueRaw, err := os.ReadFile(filename)
			ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
			lines = append(lines, name+"="+string(valueRaw))
		} else {
			lines = append(lines, item)
		}
	}

	bodyBuf := &bytes.Buffer{}
	bodyBuf.Write([]byte(strings.Join(lines, "&")))

	ret.Body = io.Reader(bodyBuf)
	ret.RecommendedMimeType = "application/x-www-form-urlencoded"
	ret.RecommendedMethod = "POST"

	return ret
}

func (ctx *CurlContext) HandleFormEncoded() *UploadInformation {
	ret := &UploadInformation{}
	formBody := url.Values{}
	for _, item := range ctx.Data_encoded {
		// fixme: =xxxxx means ignore any = in xxxxx (content) - just URL encode directly? https://curl.se/docs/manpage.html#--data-urlencode
		idxAt := strings.Index(item, "@")
		idxEqual := strings.Index(item, "=")
		idxEqualAt := strings.Index(item, "=@")
		if idxAt == 0 { // @file/path/here
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
			formLines := strings.Split(string(fullForm), "\n")
			for _, line := range formLines {
				splits := strings.SplitN(line, "=", 2)
				name := splits[0]
				value := splits[1]
				formBody.Set(name, value)
			}
		} else if idxEqual > -1 && idxEqualAt == idxEqual { // name=@file/path/here
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]

			filename := strings.TrimPrefix(value, "@")
			valueRaw, err := os.ReadFile(filename)
			ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
			//formBody.Set(name, base64.StdEncoding.EncodeToString(valueRaw))
			formBody.Set(name, string(valueRaw))
		} else if idxEqual > -1 { // name=value
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			formBody.Set(name, value)
		} else {
			panic("I need a name=value, @file of lines name=value format, or name=@file/path/here")
		}
	}

	ret.Body = strings.NewReader(formBody.Encode())
	ret.RecommendedMimeType = "application/x-www-form-urlencoded"
	ret.RecommendedMethod = "POST"

	return ret
}

func (ctx *CurlContext) HandleFormRawConcat() *UploadInformation {
	ret := &UploadInformation{}
	bodyBuf := &bytes.Buffer{}
	bodyBuf.Write([]byte(strings.Join(ctx.Data_rawconcat, "&")))

	ret.Body = io.Reader(bodyBuf)
	ret.RecommendedMimeType = "application/x-www-form-urlencoded"
	ret.RecommendedMethod = "POST"

	return ret
}

func (ctx *CurlContext) HandleFormMultipart() *UploadInformation {
	ret := &UploadInformation{}
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)
	for _, item := range ctx.Data_multipart {
		if strings.HasPrefix(item, "@") {
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
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
				ctx.HandleErrorAndExit(err, ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename))
				part, _ := writer.CreateFormFile(name, path.Base(filename))
				part.Write(valueRaw)
			} else {
				part, _ := writer.CreateFormField(name)
				part.Write([]byte(value))
			}
		}
	}
	writer.Close()

	ret.Body = bodyBuf
	ret.RecommendedMimeType = "multipart/form-data; boundary=" + writer.Boundary()
	ret.RecommendedMethod = "POST"

	return ret
}
