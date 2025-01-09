package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRootCAsPool(t *testing.T) {
	ctx := &CurlContext{}
	result, cerr := ctx.BuildRootCAsPool()
	assert.Nil(t, cerr)
	assert.NotNil(t, result)

	ctx = &CurlContext{}
	ctx.DoNotUseHostCertificateAuthorities = true
	result, cerr = ctx.BuildRootCAsPool()
	assert.Nil(t, cerr)
	assert.NotNil(t, result)
}
