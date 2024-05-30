package cli

import (
	"slices"
	"strings"
	"testing"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	"github.com/stretchr/testify/assert"
)

func Test_SetupContextForRun_InvalidArgs(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"--thisisnotvalid"}
	args, errGot := ParseFlags(args, ctx)
	assert.NotNil(t, errGot)
	assert.Equal(t, curlerrors.ERROR_INVALID_ARGS, errGot.ExitCode)
}

func Test_SetupContextForRun_Test1(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"-o", "balloonbump", "-v", "-s", "--proto-default", "https"}
	args, _ = ParseFlags(args, ctx)
	ctx.SetupContextForRun(args)

	assert.EqualValues(t, []string{"balloonbump"}, ctx.BodyOutput)
	assert.NotNil(t, ctx.Jar)
	assert.True(t, ctx.SilentFail)
	assert.True(t, ctx.IsSilent)
	assert.EqualValues(t, "https", ctx.DefaultProtocolScheme)
}

func Test_SetupContextForRun_Urls(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"https://google.com", "google.com", "/local"}
	args, _ = ParseFlags(args, ctx)
	ctx.SetupContextForRun(args)
	assert.EqualValues(t, "http", ctx.DefaultProtocolScheme)
	assert.Condition(t, func() bool { return slices.Contains(ctx.Urls, "https://google.com") })
	assert.Condition(t, func() bool { return slices.Contains(ctx.Urls, "http://google.com") })
	assert.Condition(t, func() bool { return slices.Contains(ctx.Urls, "http://localhost/local") })
}

func Test_SetupContextForRun_Head(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"-I"}
	args, _ = ParseFlags(args, ctx)
	ctx.SetupContextForRun(args)
	assert.EqualValues(t, []string{"-"}, ctx.HeaderOutput)
	assert.EqualValues(t, "HEAD", ctx.HttpVerb)

	ctx = new(curl.CurlContext)
	args = []string{"-I", "-D", "loop"}
	args, _ = ParseFlags(args, ctx)
	ctx.SetupContextForRun(args)
	assert.EqualValues(t, []string{"loop"}, ctx.HeaderOutput)
	assert.EqualValues(t, "HEAD", ctx.HttpVerb)
}

func Test_SetupContextForRun_MultipleTls(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"--tlsv1.0", "--tlsv1.1"}
	args, _ = ParseFlags(args, ctx)
	errGot := ctx.SetupContextForRun(args)
	assert.NotNil(t, errGot)
	assert.EqualValues(t, errGot.ExitCode, curlerrors.ERROR_INVALID_ARGS)
	assert.True(t, strings.Contains(errGot.ErrorString, "Cannot include more than one option from"))
}

func Test_SetupContextForRun_MixedUploadArgs(t *testing.T) {
	ctx := new(curl.CurlContext)
	args := []string{"-d", "a=b", "-T", "/tmp"}
	args, _ = ParseFlags(args, ctx)
	errGot := ctx.SetupContextForRun(args)
	assert.NotNil(t, errGot)
	assert.EqualValues(t, errGot.ExitCode, curlerrors.ERROR_INVALID_ARGS)
	assert.True(t, strings.Contains(errGot.ErrorString, "Cannot include more than one option from"))

	ctx = new(curl.CurlContext)
	args = []string{"-d", "a=b", "-F", "a=b"}
	args, _ = ParseFlags(args, ctx)
	errGot = ctx.SetupContextForRun(args)
	assert.NotNil(t, errGot)
	assert.EqualValues(t, errGot.ExitCode, curlerrors.ERROR_INVALID_ARGS)
	assert.True(t, strings.Contains(errGot.ErrorString, "Cannot include more than one option from"))

	ctx = new(curl.CurlContext)
	args = []string{"-F", "a=b", "-T", "/tmp"}
	args, _ = ParseFlags(args, ctx)
	errGot = ctx.SetupContextForRun(args)
	assert.NotNil(t, errGot)
	assert.EqualValues(t, errGot.ExitCode, curlerrors.ERROR_INVALID_ARGS)
	assert.True(t, strings.Contains(errGot.ErrorString, "Cannot include more than one option from"))

	ctx = new(curl.CurlContext)
	args = []string{"-d", "a=b", "-I"}
	args, _ = ParseFlags(args, ctx)
	errGot = ctx.SetupContextForRun(args)
	assert.NotNil(t, errGot)
	assert.EqualValues(t, errGot.ExitCode, curlerrors.ERROR_INVALID_ARGS)
	assert.True(t, strings.Contains(errGot.ErrorString, "Cannot include more than one option from"))
}
