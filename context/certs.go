package context

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	curlerrors "github.com/cdwiegand/go-curling/errors"
	pkcs8 "github.com/youmark/pkcs8"
)

func (ctx *CurlContext) BuildClientCertificates() ([]tls.Certificate, *curlerrors.CurlError) {
	var pemBlocks []pem.Block

	if ctx.ClientCertFile != "" {
		if strings.Contains(ctx.ClientCertFile, ":") {
			parts := strings.SplitN(ctx.ClientCertFile, ":", 2)
			ctx.ClientCertFile = parts[0]
			ctx.ClientCertKeyPassword = parts[1]
		}
		pemBytes, error := os.ReadFile(ctx.ClientCertFile)
		if error != nil && ctx.FailEarly {
			return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to open file %s", ctx.ClientCertFile), error)
		}
		pemBlocks = append(pemBlocks, extractPemBlocks(pemBytes, true)...)
	}

	if ctx.ClientCertKeyFile != "" {
		if strings.Contains(ctx.ClientCertKeyFile, ":") {
			parts := strings.SplitN(ctx.ClientCertKeyFile, ":", 2)
			ctx.ClientCertKeyFile = parts[0]
			ctx.ClientCertKeyPassword = parts[1]
		}
		pemBytes, error := os.ReadFile(ctx.ClientCertKeyFile)
		if error != nil && ctx.FailEarly {
			return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_CANNOT_READ_FILE, fmt.Sprintf("Failed to open file %s", ctx.ClientCertKeyFile), error)
		}
		pemBlocks = append(pemBlocks, extractPemBlocks(pemBytes, true)...)
	}

	ret := []tls.Certificate{}
	if pemBlocks != nil {
		var keyErr error
		var cert tls.Certificate
		for _, h := range pemBlocks {
			cert.PrivateKey, keyErr = convertPrivateKeyBlock(h, ctx.ClientCertKeyPassword)
			if keyErr != nil && ctx.FailEarly {
				return nil, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_SSL_SYSTEM_FAILURE, "Unable to decrypt private key", keyErr)
			}
			if cert.PrivateKey != nil {
				ret = append(ret, cert)
			}
		}
	}

	return ret, nil
}

func (ctx *CurlContext) BuildRootCAsPool() (*x509.CertPool, *curlerrors.CurlError) {
	var pool *x509.CertPool
	if ctx.DoNotUseHostCertificateAuthorities {
		pool = x509.NewCertPool()
	} else {
		var err error
		pool, err = x509.SystemCertPool()
		if err != nil {
			return nil, curlerrors.NewCurlErrorFromError(curlerrors.ERROR_SSL_SYSTEM_FAILURE, err)
		}
	}
	if pool == nil {
		pool = x509.NewCertPool()
	}
	load := func(filename string, required bool) *curlerrors.CurlError {
		data, err := os.ReadFile(filename) // #nosec G304
		if err != nil {
			return curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CA_CERT, err)
		}
		if !pool.AppendCertsFromPEM(data) && required {
			return curlerrors.NewCurlErrorFromString(curlerrors.ERROR_CA_CERT, "No CA certificates found in "+filename)
		}
		return nil
	}
	for _, filename := range ctx.CaCertFile {
		if err := load(filename, true); err != nil {
			return nil, err
		}
	}
	if ctx.CaCertPath != "" {
		err := filepath.Walk(ctx.CaCertPath, func(filename string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if err := load(filename, false); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CA_CERT, err)
		}
	}
	return pool, nil
}

func convertPrivateKeyBlock(pem pem.Block, ClientCertKeyPassword string) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(pem.Bytes); err == nil {
		return key, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(pem.Bytes); err == nil {
		if strings.Contains(pem.Type, "ENCRYPTED") {
			switch keyType := key.(type) {
			case *rsa.PrivateKey, *ecdsa.PrivateKey:
				decrypted, _, err := pkcs8.ParsePrivateKey(pem.Bytes, []byte(ClientCertKeyPassword))
				if err != nil {
					return nil, err
				}
				return decrypted, nil
			default:
				return nil, fmt.Errorf("unable to decrypt %s %s private key", pem.Type, keyType)
			}
		}
		return key, err
	}
	if key, err := x509.ParseECPrivateKey(pem.Bytes); err == nil {
		return key, err
	}
	return nil, fmt.Errorf("no valid private key found")
}

func extractPemBlocks(b []byte, onlyPrivateKeys bool) (ret []pem.Block) {
	for ok := true; ok; {
		pemBlock, rest := pem.Decode(b)
		b = rest
		if pemBlock != nil {
			if !onlyPrivateKeys || strings.Contains(pemBlock.Type, "PRIVATE") {
				ret = append(ret, *pemBlock)
			}
		} else {
			ok = false
		}
	}
	return
}
