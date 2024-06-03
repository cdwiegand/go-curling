package curltestharness

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseLocalCurlVersion(t *testing.T) {
	got, err1 := ParseLocalCurlVersion("curl 1.21.34")
	assert.Nil(t, err1)
	assert.Equal(t, got.Major, 1)
	assert.Equal(t, got.Minor, 21)
	assert.Equal(t, got.Patch, 34)
}
