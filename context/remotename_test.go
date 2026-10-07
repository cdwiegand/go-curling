package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_remoteNameFromURL(t *testing.T) {
	cases := []struct {
		url     string
		want    string
		wantErr bool
	}{
		{"http://example.com/path/file.zip", "file.zip", false},
		{"http://example.com/file.bin", "file.bin", false},
		{"https://example.com/a/b/c/report.pdf?x=1", "report.pdf", false},
		{"http://example.com/", "", true},
		{"http://example.com", "", true},
		{"http://example.com/dir/", "", true},
	}
	for _, tc := range cases {
		got, err := remoteNameFromURL(tc.url)
		if tc.wantErr {
			assert.Error(t, err, tc.url)
		} else {
			assert.NoError(t, err, tc.url)
			assert.Equal(t, tc.want, got, tc.url)
		}
	}
}

func Test_filenameFromContentDisposition(t *testing.T) {
	assert.Equal(t, "report.pdf", filenameFromContentDisposition(`attachment; filename="report.pdf"`))
	assert.Equal(t, "report.pdf", filenameFromContentDisposition(`attachment; filename=report.pdf`))
	assert.Equal(t, "", filenameFromContentDisposition(""))
	assert.Equal(t, "", filenameFromContentDisposition("inline")) // no filename param
	// path components are stripped for safety
	assert.Equal(t, "evil.sh", filenameFromContentDisposition(`attachment; filename="../../evil.sh"`))
	assert.Equal(t, "evil.sh", filenameFromContentDisposition(`attachment; filename="/etc/evil.sh"`))
}

func Test_buildResolveMaps(t *testing.T) {
	ctx := &CurlContext{Resolve: []string{
		"example.com:443:127.0.0.1",
		"*:8080:10.0.0.5",
		"-remove.me:80:1.2.3.4", // leading '-' entries are ignored
		"malformed-entry",
	}}
	exact, wild := ctx.buildResolveMaps()
	assert.Equal(t, "127.0.0.1:443", exact["example.com:443"])
	assert.Equal(t, "10.0.0.5:8080", wild["8080"])
	assert.NotContains(t, exact, "remove.me:80")
	assert.Len(t, exact, 1)
	assert.Len(t, wild, 1)
}
