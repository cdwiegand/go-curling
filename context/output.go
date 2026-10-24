package context

import (
	"io"
	"os"
	"path/filepath"

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func (ctx *CurlContext) openOutput(file string) (io.Writer, io.Closer, error) {
	switch standardizeFileName(file) {
	case "/dev/null":
		return io.Discard, nil, nil
	case "/dev/stderr":
		return os.Stderr, nil, nil
	case "/dev/stdout":
		return os.Stdout, nil, nil
	}
	if ctx.CreateDirs {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			return nil, nil, err
		}
	}
	if ctx.filesAlreadyStartedWriting == nil {
		ctx.filesAlreadyStartedWriting = make(map[string]*os.File)
	}
	mode := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if ctx.filesAlreadyStartedWriting[file] != nil {
		mode = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	f, err := os.OpenFile(file, mode, 0600) // #nosec G304
	if err != nil {
		return nil, nil, err
	}
	ctx.filesAlreadyStartedWriting[file] = f
	return f, f, nil
}

func (ctx *CurlContext) WriteToFileBytes(file string, body []byte) error {
	w, closer, err := ctx.openOutput(file)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	if closer != nil {
		if closeErr := closer.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

// Distinguish destination write failures from network/body read failures.
type outputWriter struct {
	io.Writer
	err error
}

func (w *outputWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = err
	}
	return n, err
}

func (ctx *CurlContext) removeFailedOutput(file string) error {
	file = standardizeFileName(file)
	if file == "/dev/null" || file == "/dev/stdout" || file == "/dev/stderr" {
		return nil
	}
	if ctx.filesAlreadyStartedWriting[file] == nil {
		return nil
	}
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(ctx.filesAlreadyStartedWriting, file)
	return nil
}

func (ctx *CurlContext) recordBodyOutput(file string) {
	if ctx.bodyOutputsWritten == nil {
		ctx.bodyOutputsWritten = make(map[string]bool)
	}
	ctx.bodyOutputsWritten[file] = true
}

func (ctx *CurlContext) removeBodyOutputs() (errs curlerrors.CurlErrorCollection) {
	for file := range ctx.bodyOutputsWritten {
		errs.AppendError(curlerrors.ERROR_CANNOT_WRITE_FILE, ctx.removeFailedOutput(file))
	}
	return
}

// Previously opened files may contain output from an earlier URL. Preserve
// that prefix when retrying; newly opened files restart at byte zero.
func (ctx *CurlContext) outputOffsets(index int, resp *CurlResponses) (map[string]int64, error) {
	offsets := make(map[string]int64)
	_, file := ctx.GetNextOutputsFromContext(index)
	files := []string{file}
	if ctx.RemoteHeaderName && len(resp.Responses) > 0 {
		last := resp.Responses[len(resp.Responses)-1].HttpResponse
		if last != nil {
			if name := filenameFromContentDisposition(last.Header.Get("Content-Disposition")); name != "" {
				files = append(files, name)
			}
		}
	}
	for _, file := range files {
		if ctx.filesAlreadyStartedWriting[file] == nil {
			continue
		}
		info, err := os.Stat(file)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			offsets[file] = info.Size()
		}
	}
	return offsets, nil
}

func (ctx *CurlContext) rollbackBodyOutputs(offsets map[string]int64) error {
	for file := range ctx.bodyOutputsWritten {
		file = standardizeFileName(file)
		if file == "/dev/null" || file == "/dev/stdout" || file == "/dev/stderr" {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			if err := os.Truncate(file, offsets[file]); err != nil {
				return err
			}
		}
	}
	return nil
}
