package context

// Helpers for -O/--remote-name and -J/--remote-header-name: deriving a local
// output file name from a URL path or from a Content-Disposition header.

import (
	"fmt"
	"mime"
	"net/url"
	"path"
	"strings"
)

// remoteNameFromURL returns the file name to use for -O: the last path segment
// of the URL. It errors when the URL has no usable file name (curl behaves the
// same, refusing with "Remote file name has no length").
func remoteNameFromURL(rawurl string) (string, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	if u.Path == "" || strings.HasSuffix(u.Path, "/") {
		return "", fmt.Errorf("remote file name has no length for URL %q", rawurl)
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "", fmt.Errorf("remote file name has no length for URL %q", rawurl)
	}
	return base, nil
}

// filenameFromContentDisposition extracts a safe base file name from a
// Content-Disposition header's filename parameter, or "" if none/unsafe. Any
// directory components are stripped so the value cannot escape the working dir.
func filenameFromContentDisposition(headerValue string) string {
	if headerValue == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(headerValue)
	if err != nil {
		return ""
	}
	fn := params["filename"]
	if fn == "" {
		return ""
	}
	// strip any path (both separators) so "../x" or "/etc/x" collapses to the base name
	fn = strings.ReplaceAll(fn, "\\", "/")
	base := path.Base(fn)
	if base == "" || base == "." || base == "/" {
		return ""
	}
	return base
}
