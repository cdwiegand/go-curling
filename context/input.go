package context

import (
	"io"
	"os"
)

// Cache stdin so redirects, retries and repeated URL transfers can replay it.
// Request payloads are buffered, as are the existing file/data upload paths.
func (ctx *CurlContext) readInputFile(name string) ([]byte, error) {
	if name != "-" {
		return os.ReadFile(name) // #nosec G304
	}
	if !ctx.stdinRead {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		ctx.stdinData, ctx.stdinRead = data, true
	}
	return ctx.stdinData, nil
}
