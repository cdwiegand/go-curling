package context

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
	Err         error
}

func NewCurlError0(errorString string) *CurlError {
	return &CurlError{-6, errorString, nil}
}
func NewCurlError1(exitCode int, errorString string) *CurlError {
	return &CurlError{exitCode, errorString, nil}
}
func NewCurlError2(exitCode int, errorString string, err error) *CurlError {
	return &CurlError{exitCode, errorString, err}
}
