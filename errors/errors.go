package errors

const ERROR_INTERNAL = -4
const ERROR_SSL_SYSTEM_FAILURE = -5
const ERROR_STATUS_CODE_FAILURE = -6
const ERROR_NO_RESPONSE = -7
const ERROR_INVALID_URL = -8
const ERROR_CANNOT_READ_FILE = -9
const ERROR_CANNOT_WRITE_FILE = -10
const ERROR_CANNOT_WRITE_TO_STDOUT = -11
const ERROR_INVALID_ARGS = -12

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
	if cerr.Errors != nil && len(cerr.Errors) > 0 {
		cerrs.Errors = append(cerrs.Errors, cerr.Errors...)
	}
}
func (cerrs *CurlErrorCollection) HasError() bool {
	return cerrs.Errors != nil && len(cerrs.Errors) > 0
}
