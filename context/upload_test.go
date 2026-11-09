package context

import (
	"bytes"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/stretchr/testify/assert"
)

// -T
func Test_HandleUploadRawFile(t *testing.T) {
}

// -F
// -F name=@file (reads file as a FILE attachment)
// -F name=<file (reads file as the VALUE of a form field)
// -F name=value
// --form-string name=anyvalue (anyvalue can start with @ or <, they are ignored)
// Note: no -F @file support
func Test_HandleFormMultipart(t *testing.T) {
}

// Every request built from one context (one per URL, or rebuilt for a retry)
// must carry a multipart body whose boundary matches its Content-Type header.
func Test_HandleFormMultipart_RepeatedBuildsUseMatchingBoundary(t *testing.T) {
	valueFile := filepath.Join(t.TempDir(), "value.txt")
	assert.NoError(t, os.WriteFile(valueFile, []byte("from-file"), 0600))
	ctx := &CurlContext{
		Urls:              []string{"http://localhost/1", "http://localhost/2", "http://localhost/3"},
		Form_Multipart:    []string{"a=b", "f=<" + valueFile},
		Form_MultipartRaw: []string{"raw=@literal"},
	}
	for i, u := range append(ctx.Urls, ctx.Urls[0]) {
		req, cerr := ctx.BuildHttpRequest(u, i%len(ctx.Urls), true, true)
		if !assert.Nil(t, cerr) {
			return
		}
		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		assert.NoError(t, err)
		assert.Equal(t, "multipart/form-data", mediaType)
		form, err := multipart.NewReader(req.Body, params["boundary"]).ReadForm(1 << 20)
		if !assert.NoError(t, err, "request %d body does not match its boundary", i+1) {
			return
		}
		assert.Equal(t, []string{"b"}, form.Value["a"], "request %d", i+1)
		assert.Equal(t, []string{"from-file"}, form.Value["f"], "request %d", i+1)
		assert.Equal(t, []string{"@literal"}, form.Value["raw"], "request %d", i+1)
		assert.Equal(t, http.MethodPost, req.Method)
	}
	contentTypes := 0
	for key := range ctx.GetHeadersAsDict() {
		if key == "Content-Type" {
			contentTypes++
		}
	}
	assert.Equal(t, 1, contentTypes, "the Content-Type header is stored once")
}

func Test_handleFormArg(t *testing.T) {
}

func Test_HasDataArgs(t *testing.T) {
	ctx := &CurlContext{}
	assert.False(t, ctx.HasDataArgs(), "No data args should be present")
	ctx.Data_Ascii = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "Ascii data arg should be present")
	ctx.Data_Ascii = nil
	ctx.Data_Binary = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "Binary data arg should be present")
	ctx.Data_Binary = nil
	ctx.Data_Encoded = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "Encoded data arg should be present")
	ctx.Data_Encoded = nil
	ctx.Data_Json = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "Json data arg should be present")
	ctx.Data_Json = nil
	ctx.Data_RawAsIs = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "RawAsIs data arg should be present")
	ctx.Data_RawAsIs = nil
	ctx.Data_Standard = []string{"hello"}
	assert.True(t, ctx.HasDataArgs(), "Standard data arg should be present")
	ctx.Data_Standard = nil
	assert.False(t, ctx.HasDataArgs(), "No data args should be present")
}

func Test_HasFormArgs(t *testing.T) {
	ctx := &CurlContext{}
	assert.False(t, ctx.HasFormArgs(), "No form args are yet present")

	ctx = &CurlContext{}
	ctx.Form_Multipart = []string{"hello"}
	assert.True(t, ctx.HasFormArgs(), "Should have form args")

	ctx = &CurlContext{}
	ctx.Form_MultipartRaw = []string{"hello"}
	assert.True(t, ctx.HasFormArgs(), "Should have form args")
}

// -d / --data: includes already-URL-encoded values (or lines from a file, or a file as already-URL-encoded content with newlines stripped)
// -d name=value
// NO NO NO -d name=@file NO NO NO NOT SUPPORTED IN UPSTREAM!!
// -d @file (lines of name=value)
func Test_handleDataArgs_Standard(t *testing.T) {
}

// --data-urlencoded: includes to-be-URL-encoded values (or lines from a file, or a file as to-bo-URL-encoded content with newlines encoded)
// --data-urlencoded name=value
// --data-urlencoded name@file (note: NOT name=@file)
// --data-urlencoded @file (lines of name=value)
func Test_handleDataArgs_Encoded(t *testing.T) {
}

// --json: send JSON to server as input (mutually incompatible with other --data-* parameters)
// --json '{ "name": "John Doe" }'
// --json @file (raw JSON)
func Test_handleDataArgs_Json(t *testing.T) {
	ctx := &CurlContext{}
	bodyBuf := &bytes.Buffer{}

	ctx.Data_Json = []string{"@/this-does/not-exist"}
	cerr := handleDataArgs_Json(ctx, bodyBuf)
	assert.Equal(t, curlerrors.ERROR_CANNOT_READ_FILE, cerr.ExitCode, "Should return ERROR_CANNOT_READ_FILE")

	testFile := filepath.Join(t.TempDir(), "config.test")
	err := os.WriteFile(testFile, []byte("{ \"hello\": \"world\" }"), 0666)
	assert.NoError(t, err, "Could not write test file")
	ctx.Data_Json = []string{"@" + testFile}
	cerr = handleDataArgs_Json(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "{ \"hello\": \"world\" }", bodyBuf.String())

	bodyBuf = &bytes.Buffer{}
	ctx.Data_Json = []string{"{ \"hello\": \"world\" }"}
	cerr = handleDataArgs_Json(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "{ \"hello\": \"world\" }", bodyBuf.String())
}

// --data-raw: append EXACTLY what is specified as value
// --data-raw name=value
// note: no name=@file or @file support
func Test_handleDataArgs_RawAsIs(t *testing.T) {
	ctx := &CurlContext{}
	bodyBuf := &bytes.Buffer{}

	ctx.Data_RawAsIs = []string{"@/this-does/not-exist"}
	cerr := handleDataArgs_RawAsIs(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "@/this-does/not-exist", bodyBuf.String())

	bodyBuf = &bytes.Buffer{}
	ctx.Data_RawAsIs = []string{"hello=world"}
	cerr = handleDataArgs_RawAsIs(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "hello=world", bodyBuf.String())
}

// --data-binary: includes exactly-as-presented values (or lines from a file, or a file as exactly-as-presented content with newlines retained)
// --data-binary name=value
// NO NO NO --data-binary name=@file NO NO NO NOT SUPPORTED IN UPSTREAM!!
// --data-binary @file (lines of name=value)
func Test_handleDataArgs_Binary(t *testing.T) {
	ctx := &CurlContext{}
	bodyBuf := &bytes.Buffer{}

	ctx.Data_Binary = []string{"@/this-does/not-exist"}
	cerr := handleDataArgs_Binary(ctx, bodyBuf)
	assert.Equal(t, curlerrors.ERROR_CANNOT_READ_FILE, cerr.ExitCode, "Should return ERROR_CANNOT_READ_FILE")

	testFile := filepath.Join(t.TempDir(), "config.test")
	err := os.WriteFile(testFile, []byte("hello=world"), 0666)
	assert.NoError(t, err, "Could not write test file")
	ctx.Data_Binary = []string{"@" + testFile}
	cerr = handleDataArgs_Binary(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "hello=world", bodyBuf.String())

	bodyBuf = &bytes.Buffer{}
	ctx.Data_Binary = []string{"hello=world"}
	cerr = handleDataArgs_Binary(ctx, bodyBuf)
	assert.Nil(t, cerr, "Should not return an error")
	assert.Equal(t, "hello=world", bodyBuf.String())
}

func Test_appendDataBytes(t *testing.T) {
	bodyBuf := &bytes.Buffer{}

	appendDataBytes(bodyBuf, []byte("hello=world"))
	assert.Equal(t, "hello=world", bodyBuf.String())

	appendDataBytes(bodyBuf, []byte("hi=there"))
	assert.Equal(t, "hello=world&hi=there", bodyBuf.String())
}

func Test_appendDataString(t *testing.T) {
	bodyBuf := &bytes.Buffer{}

	appendDataString(bodyBuf, "hello=world")
	assert.Equal(t, "hello=world", bodyBuf.String())

	appendDataString(bodyBuf, "hi=there")
	assert.Equal(t, "hello=world&hi=there", bodyBuf.String())
}
