package context

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

func transferError(err error) *curlerrors.CurlError {
	code := curlerrors.ERROR_NO_RESPONSE
	var netErr net.Error
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	var unknownCA x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	switch {
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()):
		code = curlerrors.ERROR_TIMEOUT
	case errors.As(err, &dnsErr):
		code = curlerrors.ERROR_RESOLVE_HOST
	case errors.As(err, &certErr), errors.As(err, &unknownCA), errors.As(err, &hostnameErr):
		code = curlerrors.ERROR_SSL_SYSTEM_FAILURE
	case errors.Is(err, io.ErrUnexpectedEOF):
		code = curlerrors.ERROR_PARTIAL_FILE
	}
	return curlerrors.NewCurlErrorFromError(code, err)
}
