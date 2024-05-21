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

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

// -T
func (ctx *CurlContext) HandleUploadRawFile(index int) (io.Reader, *curlerrors.CurlError) {
	// DOES use index - sends a file per URL
	if len(ctx.Upload_File) > index {
		filename := ctx.Upload_File[index]
		f, err := os.ReadFile(filename)
		if err != nil {
			return nil, curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
		}
		mimeType := mime.TypeByExtension(path.Ext(filename))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		bodyBuf := &bytes.Buffer{}
		bodyBuf.Write(f)

		ctx.SetMethodIfNotSet("PUT")
		if mimeType != "" {
			ctx.SetHeaderIfNotSet("Content-Type", mimeType)
		}
		return io.Reader(bodyBuf), nil
	}
	return nil, nil
}

// -F
// -F name=@file (reads file as a FILE attachment)
// -F name=<file (reads file as the VALUE of a form field)
// -F name=value
// --form-string name=anyvalue (anyvalue can start with @ or <, they are ignored)
// Note: no -F @file support
func (ctx *CurlContext) HandleFormMultipart() (io.Reader, *curlerrors.CurlError) {
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	for _, item := range ctx.Form_Multipart {
		idxEqual := strings.Index(item, "=")
		if idxEqual > -1 {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			handleFormArg(name, value, writer)
		} else {
			return nil, curlerrors.NewCurlError1(curlerrors.ERROR_INVALID_ARGS, "I need a name=value or name=@file/path/here")
		}
	}

	for _, item := range ctx.Form_MultipartRaw {
		idxEqual := strings.Index(item, "=")
		if idxEqual > -1 {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			part, _ := writer.CreateFormField(name)
			part.Write([]byte(value))
		} else {
			return nil, curlerrors.NewCurlError1(curlerrors.ERROR_INVALID_ARGS, "I need a name=value")
		}
	}

	writer.Close()

	ctx.SetMethodIfNotSet("POST")
	ctx.SetHeaderIfNotSet("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	return bodyBuf, nil
}

func handleFormArg(name string, value string, writer *multipart.Writer) *curlerrors.CurlError {
	if strings.HasPrefix(value, "@") {
		filename := strings.TrimPrefix(value, "@")
		valueRaw, err := os.ReadFile(filename)
		if err != nil {
			return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
		}
		shortname := path.Base(filename)
		part, _ := writer.CreateFormFile(name, shortname)
		part.Write(valueRaw)
	} else if strings.HasPrefix(value, "<") {
		filename := strings.TrimPrefix(value, "<")
		valueRaw, err := os.ReadFile(filename)
		if err != nil {
			return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
		}
		part, _ := writer.CreateFormField(name)
		part.Write(valueRaw)
	} else {
		part, _ := writer.CreateFormField(name)
		part.Write([]byte(value))
	}
	return nil
}

// --data* args
// -d name=value
// -d name=@file
// -d @file (lines of name=value)
// -d (--data), --data-raw, --data-binary, --data-urlencoded
func (ctx *CurlContext) HandleDataArgs() (io.Reader, *curlerrors.CurlError) {
	bodyBuf := &bytes.Buffer{}
	if len(ctx.Data_Json) > 0 {
		err0 := handleDataArgs_Json(ctx, bodyBuf)
		if err0 != nil {
			return nil, err0
		}
		ctx.SetHeaderIfNotSet("Accept", "application/json")
		ctx.SetHeaderIfNotSet("Content-Type", "application/json")
		ctx.SetMethodIfNotSet("POST")
		return io.Reader(bodyBuf), nil
	}

	err1 := handleDataArgs_Standard(ctx, bodyBuf)
	if err1 != nil {
		return nil, err1
	}

	err2 := handleDataArgs_Encoded(ctx, bodyBuf)
	if err2 != nil {
		return nil, err2
	}

	err3 := handleDataArgs_Binary(ctx, bodyBuf)
	if err3 != nil {
		return nil, err3
	}

	err4 := handleDataArgs_RawAsIs(ctx, bodyBuf)
	if err4 != nil {
		return nil, err4
	}

	ctx.SetMethodIfNotSet("POST")
	ctx.SetHeaderIfNotSet("Content-Type", "application/x-www-form-urlencoded")
	return io.Reader(bodyBuf), nil
}
func (ctx *CurlContext) HasDataArgs() bool {
	return len(ctx.Data_Binary) > 0 || len(ctx.Data_Encoded) > 0 || len(ctx.Data_RawAsIs) > 0 || len(ctx.Data_Standard) > 0 || len(ctx.Data_Json) > 0
}
func (ctx *CurlContext) HasFormArgs() bool {
	return len(ctx.Form_Multipart) > 0 || len(ctx.Form_MultipartRaw) > 0
}

// -d / --data: includes already-URL-encoded values (or lines from a file, or a file as already-URL-encoded content with newlines stripped)
// -d name=value
// -d name=@file
// -d @file (lines of name=value)
func handleDataArgs_Standard(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	for _, item := range ctx.Data_Standard {
		idxAt, idxEqual, idxEqualAt := identifyDataReferenceIndexes(item)
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			formLines := strings.Split(string(fullForm), "\n")
			appendDataStrings(bodyBuf, formLines)
		} else if idxEqual > -1 && idxEqual == idxEqualAt { // name=@value - value is file path for the content
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			filename := strings.TrimPrefix(value, "@")
			valueRaw, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			newLine1 := []byte("\r")
			newLine2 := []byte("\n")
			empty := []byte("")
			valueRaw = bytes.Replace(valueRaw, newLine1, empty, -1)
			valueRaw = bytes.Replace(valueRaw, newLine2, empty, -1)
			appendDataString(bodyBuf, name+"="+string(valueRaw))
		} else {
			appendDataString(bodyBuf, item)
		}
	}
	return nil
}

// --data-urlencoded: includes to-be-URL-encoded values (or lines from a file, or a file as to-bo-URL-encoded content with newlines encoded)
// --data-urlencoded name=value
// --data-urlencoded name=@file
// --data-urlencoded @file (lines of name=value)
func handleDataArgs_Encoded(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	formBody := url.Values{}
	for _, item := range ctx.Data_Encoded {
		idxAt, idxEqual, idxEqualAt := identifyDataReferenceIndexes(item)
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			formLines := strings.Split(string(fullForm), "\n")
			for _, item2 := range formLines {
				item2parts := strings.SplitN(item2, "=", 2)
				if len(item2parts) == 2 {
					formBody.Add(item2parts[0], item2parts[1])
				} else {
					panic("I need a name=value, @file of lines name=value format, or name=@file/path/here")
				}
			}
		} else if idxEqual > -1 && idxEqual == idxEqualAt { // name=@value - value is file path for the content
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			filename := strings.TrimPrefix(value, "@")
			valueRaw, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			formBody.Add(name, string(valueRaw))
		} else if idxEqual > -1 { // name=value
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			formBody.Set(name, value)
		} else {
			panic("I need a name=value, @file of lines name=value format, or name=@file/path/here")
		}
	}
	if len(formBody) > 0 {
		appendDataString(bodyBuf, formBody.Encode())
	}
	return nil
}

// --json: send JSON to server as input (mutually incompatible with other --data-* parameters)
// --json '{ "name": "John Doe" }'
// --json @file (raw JSON)
func handleDataArgs_Json(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	for _, item := range ctx.Data_Json {
		appendDataString(bodyBuf, item)
	}
	return nil
}

// --data-raw: append EXACTLY what is specified as value
// --data-raw name=value
// note: no name=@file or @file support
func handleDataArgs_RawAsIs(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	for _, item := range ctx.Data_RawAsIs {
		appendDataString(bodyBuf, item)
	}
	return nil
}

// --data-binary: includes exactly-as-presented values (or lines from a file, or a file as exactly-as-presented content with newlines retained)
// --data-binary name=value
// --data-binary name=@file
// --data-binary @file (lines of name=value)
func handleDataArgs_Binary(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	for _, item := range ctx.Data_Binary {
		idxAt, idxEqual, idxEqualAt := identifyDataReferenceIndexes(item)
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			appendDataBytes(bodyBuf, fullForm)
		} else if idxEqual > -1 && idxEqual == idxEqualAt { // name=@value - value is file path for the content
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			filename := strings.TrimPrefix(value, "@")
			valueRaw, err := os.ReadFile(filename)
			if err != nil {
				return curlerrors.NewCurlError2(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			appendDataString(bodyBuf, name+"=")
			bodyBuf.Write(valueRaw) // already wrote the [&]name= part, just add value directly
		} else {
			appendDataString(bodyBuf, item)
		}
	}
	return nil
}

func appendDataBytes(bodyBuf *bytes.Buffer, content []byte) {
	if bodyBuf.Len() > 0 {
		bodyBuf.WriteString("&")
	}
	bodyBuf.Write(content)
}

func appendDataString(bodyBuf *bytes.Buffer, content string) {
	appendDataBytes(bodyBuf, []byte(content))
}

func appendDataStrings(bodyBuf *bytes.Buffer, lines []string) {
	for _, item := range lines {
		appendDataString(bodyBuf, item)
	}
}

func identifyDataReferenceIndexes(item string) (idxAt int, idxEqual int, idxEqualAt int) {
	idxAt = strings.Index(item, "@")
	idxEqual = strings.Index(item, "=")
	idxEqualAt = strings.Index(item, "=@")
	return
}
