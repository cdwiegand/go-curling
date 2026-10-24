package errors

// Use curl's positive, shell-portable exit codes.
const ERROR_INTERNAL = 2
const ERROR_SSL_SYSTEM_FAILURE = 60
const ERROR_STATUS_CODE_FAILURE = 22
const ERROR_NO_RESPONSE = 7
const ERROR_INVALID_URL = 3
const ERROR_CANNOT_READ_FILE = 26
const ERROR_CANNOT_WRITE_FILE = 23
const ERROR_CANNOT_WRITE_TO_STDOUT = 23
const ERROR_INVALID_ARGS = 2
const ERROR_RESOLVE_HOST = 6
const ERROR_PARTIAL_FILE = 18
const ERROR_TIMEOUT = 28
const ERROR_TOO_MANY_REDIRECTS = 47
const ERROR_MAX_FILESIZE = 63
const ERROR_CA_CERT = 77
const ERROR_RECEIVE = 56

type CurlError struct {
	ExitCode    int
	ErrorString string
}

type CurlErrorCollection struct {
	Errors []*CurlError
}

func (err *CurlError) Error() string {
	return err.ErrorString
}

func NewCurlErrorFromError(exitCode int, err error) *CurlError {
	return &CurlError{exitCode, err.Error()}
}
func NewCurlErrorFromString(exitCode int, errorString string) *CurlError {
	return &CurlError{exitCode, errorString}
}
func NewCurlErrorFromStringAndError(exitCode int, errorString string, err error) *CurlError {
	return &CurlError{exitCode, errorString + ": " + err.Error()}
}

func (cerrs *CurlErrorCollection) AppendError(exitCode int, err error) {
	if err != nil {
		cerr := NewCurlErrorFromError(exitCode, err)
		cerrs.Errors = append(cerrs.Errors, cerr)
	}
}
func (cerrs *CurlErrorCollection) AppendCurlError(cerr *CurlError) {
	if cerr != nil {
		cerrs.Errors = append(cerrs.Errors, cerr)
	}
}
func (cerrs *CurlErrorCollection) AppendCurlErrors(cerr CurlErrorCollection) {
	if len(cerr.Errors) > 0 {
		cerrs.Errors = append(cerrs.Errors, cerr.Errors...)
	}
}
func (cerrs *CurlErrorCollection) HasError() bool {
	return len(cerrs.Errors) > 0
}
