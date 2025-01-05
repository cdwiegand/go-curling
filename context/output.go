package context

import (
	"os"
)

func (ctx *CurlContext) WriteToFileBytes(file string, body []byte) (err error) {
	if ctx.filesAlreadyStartedWriting == nil {
		ctx.filesAlreadyStartedWriting = make(map[string]*os.File)
	}

	if file == "" || file == "/dev/null" {
		// do nothing
	} else if file == "/dev/stderr" {
		_, err = os.Stderr.Write(body)
	} else if file == "/dev/stdout" {
		_, err = os.Stdout.Write(body)
	} else {
		fileref, found := ctx.filesAlreadyStartedWriting[file]
		if !found || fileref == nil {
			fileref, err = os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0600) // #nosec G304
			if err != nil {
				return err
			}
			ctx.filesAlreadyStartedWriting[file] = fileref
		} else {
			fileref, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304
			if err != nil {
				return err
			}
		}
		defer fileref.Close()
		_, err = fileref.Write(body)
	}
	return
}
