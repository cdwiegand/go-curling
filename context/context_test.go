package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SetMethodIfNotSet(t *testing.T) {
	ctx := &CurlContext{}
	assert.Empty(t, ctx.HttpVerb, "No method yet set")
	ctx.SetMethodIfNotSet("TEST")
	assert.Equal(t, "TEST", ctx.HttpVerb, "Verb should be 'TEST'")
	ctx.SetMethodIfNotSet("UMM")
	assert.Equal(t, "TEST", ctx.HttpVerb, "Verb should be 'TEST'")
}

func Test_SetHeaderIfNotSet(t *testing.T) {
	ctx := &CurlContext{}
	assert.Empty(t, ctx.GetHeadersAsDict()["X-Hello"], "Header X-Hello should not be present")
	ctx.SetHeaderIfNotSet("X-Hello", "World")
	assert.Equal(t, "World", ctx.GetHeadersAsDict()["X-Hello"], "Header X-Hello should be 'World'")
	ctx.SetHeaderIfNotSet("X-Hello", "Universe")
	assert.Equal(t, "World", ctx.GetHeadersAsDict()["X-Hello"], "Header X-Hello should be 'World'")
}

func Test_appendStrings(t *testing.T) {
	buf := []byte{}
	sepBody := []byte("&")
	ret := appendStrings(buf, sepBody, []string{"hello world"})
	assert.Equal(t, "hello world", string(ret))

	ret = appendStrings(ret, sepBody, []string{"more", "to", "come"})
	assert.Equal(t, "hello world&more\nto\ncome", string(ret))
}

func Test_appendByteArrays(t *testing.T) {
	buf := []byte{}
	sepBody := []byte("&")
	buf = appendByteArrays(buf, sepBody, []byte("hello"))
	assert.Equal(t, "hello", string(buf))
	buf = appendByteArrays(buf, sepBody, []byte("world"))
	assert.Equal(t, "hello&world", string(buf))
}

func Test_GetAndSetHeadersFromDict(t *testing.T) {
	ctx := &CurlContext{}
	dict := make(map[string]string)
	dict["hello"] = "world"
	dict["hi"] = "there"
	ctx.SetHeadersFromDict(dict)
	dict2 := ctx.GetHeadersAsDict()
	assert.NotNil(t, dict2["hello"])
	assert.Equal(t, "world", dict2["hello"])
	assert.NotNil(t, dict2["hi"])
	assert.Equal(t, "there", dict2["hi"])
	assert.Empty(t, dict2["notfound"])
}

func Test_standardizeFileName(t *testing.T) {
	assert.Equal(t, "/dev/null", standardizeFileName(""))
	assert.Equal(t, "/dev/null", standardizeFileName("null"))
	assert.Equal(t, "/dev/null", standardizeFileName("/dev/null"))

	assert.Equal(t, "/dev/stderr", standardizeFileName("/dev/stderr"))
	assert.Equal(t, "/dev/stderr", standardizeFileName("stderr"))

	assert.Equal(t, "/dev/stdout", standardizeFileName("/dev/stdout"))
	assert.Equal(t, "/dev/stdout", standardizeFileName("stdout"))
	assert.Equal(t, "/dev/stdout", standardizeFileName("-"))

	assert.Equal(t, "yo", standardizeFileName("yo"))
	assert.Equal(t, "/lollipop", standardizeFileName("/lollipop"))
	assert.Equal(t, "--", standardizeFileName("--"))
}

func Test_validateFormArgs(t *testing.T) {
	ctx := &CurlContext{}
	assert.True(t, ctx.validateFormArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Upload_File = []string{"test"}
	assert.True(t, ctx.validateFormArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.HeadOnly = true
	assert.True(t, ctx.validateFormArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Data_Ascii = []string{"hello"}
	assert.True(t, ctx.validateFormArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Form_Multipart = []string{"hello"}
	assert.True(t, ctx.validateFormArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Data_Ascii = []string{"hello"}
	ctx.Form_Multipart = []string{"hello"}
	assert.False(t, ctx.validateFormArgs(), "Form args should NOT be a valid combination")

	ctx = &CurlContext{}
	ctx.HeadOnly = true
	ctx.Form_Multipart = []string{"hello"}
	assert.False(t, ctx.validateFormArgs(), "Form args should NOT be a valid combination")

	ctx = &CurlContext{}
	ctx.Upload_File = []string{"test"}
	ctx.Form_Multipart = []string{"hello"}
	assert.False(t, ctx.validateFormArgs(), "Form args should NOT be a valid combination")
}

func Test_validateTlsArgs(t *testing.T) {
	ctx := &CurlContext{}
	assert.True(t, ctx.validateTlsArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_0 = true
	assert.True(t, ctx.validateTlsArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_1 = true
	assert.True(t, ctx.validateTlsArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_2 = true
	assert.True(t, ctx.validateTlsArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_3 = true
	assert.True(t, ctx.validateTlsArgs(), "Form args should be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_0 = true
	ctx.Tls_MinVersion_1_1 = true
	assert.False(t, ctx.validateTlsArgs(), "Form args should NOT be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_0 = true
	ctx.Tls_MinVersion_1_2 = true
	assert.False(t, ctx.validateTlsArgs(), "Form args should NOT be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_0 = true
	ctx.Tls_MinVersion_1_3 = true
	assert.False(t, ctx.validateTlsArgs(), "Form args should NOT be a valid combination")

	ctx = &CurlContext{}
	ctx.Tls_MinVersion_1_1 = true
	ctx.Tls_MinVersion_1_3 = true
	assert.False(t, ctx.validateTlsArgs(), "Form args should NOT be a valid combination")
}
