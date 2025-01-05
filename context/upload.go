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
		f, err := os.ReadFile(filename) // #nosec G304
		if err != nil {
			return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
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
			return nil, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "I need a name=value or name=@file/path/here")
		}
	}

	for _, item := range ctx.Form_MultipartRaw {
		idxEqual := strings.Index(item, "=")
		if idxEqual > -1 {
			splits := strings.SplitN(item, "=", 2)
			name := splits[0]
			value := splits[1]
			part, _ := writer.CreateFormField(name)
			_, err := part.Write([]byte(value))
			if err != nil {
				return nil, curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
			}
		} else {
			return nil, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "I need a name=value")
		}
	}

	err := writer.Close()
	if err != nil {
		cerr := curlerrors.NewCurlErrorFromError(curlerrors.ERROR_INTERNAL, err)
		return nil, cerr
	}

	ctx.SetMethodIfNotSet("POST")
	ctx.SetHeaderIfNotSet("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	return bodyBuf, nil
}

func handleFormArg(name string, value string, writer *multipart.Writer) *curlerrors.CurlError {
	var err error
	if strings.HasPrefix(value, "@") {
		filename := strings.TrimPrefix(value, "@")
		valueRaw, err := os.ReadFile(filename) // #nosec G304
		if err != nil {
			return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
		}
		shortname := path.Base(filename)
		part, _ := writer.CreateFormFile(name, shortname)
		_, err = part.Write(valueRaw)
		if err != nil {
			return curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
		}
	} else if strings.HasPrefix(value, "<") {
		filename := strings.TrimPrefix(value, "<")
		valueRaw, err := os.ReadFile(filename) // #nosec G304
		if err != nil {
			return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
		}
		part, _ := writer.CreateFormField(name)
		_, err = part.Write(valueRaw)
		if err != nil {
			return curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
		}
	} else {
		part, _ := writer.CreateFormField(name)
		_, err = part.Write([]byte(value))
		if err != nil {
			return curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, err)
		}
	}
	return nil
}

// --data* args
// -d name=value
// -d name=@file
// -d @file (lines of name=value)
// -d (--data), --data-raw, --data-binary, --data-urlencoded
func (ctx *CurlContext) HandleDataArgs(returnAsGetParams bool) (*bytes.Buffer, *curlerrors.CurlError) {
	bodyBuf := &bytes.Buffer{}
	if len(ctx.Data_Json) > 0 {
		err0 := handleDataArgs_Json(ctx, bodyBuf)
		if err0 != nil {
			return nil, err0
		}
		ctx.SetHeaderIfNotSet("Accept", "application/json")
		ctx.SetHeaderIfNotSet("Content-Type", "application/json")
		ctx.SetMethodIfNotSet("POST")
		return bodyBuf, nil
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

	if !returnAsGetParams {
		ctx.SetMethodIfNotSet("POST")
		ctx.SetHeaderIfNotSet("Content-Type", "application/x-www-form-urlencoded")
	}

	return bodyBuf, nil
}
func (ctx *CurlContext) HasDataArgs() bool {
	return len(ctx.Data_Binary) > 0 ||
		len(ctx.Data_Encoded) > 0 ||
		len(ctx.Data_RawAsIs) > 0 ||
		len(ctx.Data_Standard) > 0 ||
		len(ctx.Data_Ascii) > 0 ||
		len(ctx.Data_Json) > 0
}
func (ctx *CurlContext) HasFormArgs() bool {
	return len(ctx.Form_Multipart) > 0 || len(ctx.Form_MultipartRaw) > 0
}

// -d / --data: includes already-URL-encoded values (or lines from a file, or a file as already-URL-encoded content with newlines stripped)
// -d name=value
// NO NO NO -d name=@file NO NO NO NOT SUPPORTED IN UPSTREAM!!
// -d @file (lines of name=value)
func handleDataArgs_Standard(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	data_combined := append(ctx.Data_Standard, ctx.Data_Ascii...)
	for _, item := range data_combined {
		idxAt := strings.Index(item, "@")
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename) // #nosec G304
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			formLines := strings.Split(string(fullForm), "\n")
			for _, item := range formLines {
				appendDataString(bodyBuf, item)
			}
		} else {
			appendDataString(bodyBuf, item)
		}
	}
	return nil
}

// --data-urlencoded: includes to-be-URL-encoded values (or lines from a file, or a file as to-bo-URL-encoded content with newlines encoded)
// --data-urlencoded name=value
// --data-urlencoded name@file (note: NOT name=@file)
// --data-urlencoded @file (lines of name=value)
func handleDataArgs_Encoded(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	formBody := url.Values{}
	for _, item := range ctx.Data_Encoded {
		idxAt := strings.Index(item, "@")
		idxEqual := strings.Index(item, "=")
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename) // #nosec G304
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			formLines := strings.Split(string(fullForm), "\n")
			for _, item2 := range formLines {
				//item2parts := strings.SplitN(item2, "=", 2)
				//if len(item2parts) == 2 {
				//	formBody.Add(item2parts[0], item2parts[1])
				//} else {
				//	panic("I need a name=value, @file of lines name=value format, or name@file/path/here")
				//}
				formBody.Add(item2, "")
				// this is ... weird, but it's how curl on my machine (curl 7.81.0) actually works..
				/*
									echo "hello=world" > /tmp/d
									curl -D - -o - --data-urlencode @/tmp/d https://httpbin.org/post returns (relavant part):
									"form": {
					    				"hello=world\n": ""
					  				},
									which is WEIRD, but I want to be compatible, so I'll reproduce it..
				*/
			}
		} else if idxAt > 0 { // name@value
			splits := strings.SplitN(item, "@", 2)
			name := splits[0]
			filename := splits[1]
			valueRaw, err := os.ReadFile(filename) // #nosec G304
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
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
		if item[0] == '@' {
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename) // #nosec G304
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			appendDataBytes(bodyBuf, fullForm)
		} else {
			appendDataString(bodyBuf, item)
		}
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
// NO NO NO --data-binary name=@file NO NO NO NOT SUPPORTED IN UPSTREAM!!
// --data-binary @file (lines of name=value)
func handleDataArgs_Binary(ctx *CurlContext, bodyBuf *bytes.Buffer) *curlerrors.CurlError {
	for _, item := range ctx.Data_Binary {
		idxAt := strings.Index(item, "@")
		if idxAt == 0 { // @file/path/here - file containing name=value lines
			filename := strings.TrimPrefix(item, "@")
			fullForm, err := os.ReadFile(filename) // #nosec G304
			if err != nil {
				return curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to read file %s", filename), err)
			}
			appendDataBytes(bodyBuf, fullForm)
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
	if bodyBuf.Len() > 0 {
		bodyBuf.WriteString("&")
	}
	bodyBuf.WriteString(content)
}
