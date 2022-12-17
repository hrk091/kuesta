/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
)

type CredCfg struct {
	// Disable TLS
	NoTLS bool

	// ClientAuth specifies how to handle client cert
	ClientAuth tls.ClientAuthType

	// Skip verifying server sert
	SkipVerifyServerCert bool

	// Path to the cert file
	CrtPath string

	// Path to the private key file
	KeyPath string

	// Path to the CA cert file
	CACrtPath string
}

func (o *CredCfg) loadKeyPair() (*tls.Certificate, error) {
	if o.CrtPath == "" || o.KeyPath == "" {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(o.CrtPath, o.KeyPath)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("load cert key-pair: %w", err))
	}
	return &certificate, nil
}

func (o *CredCfg) caCertPool() (*x509.CertPool, error) {
	if o.CACrtPath == "" {
		return nil, nil
	}
	caCrtFile, err := os.ReadFile(o.CACrtPath)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("read CA cert file: %w", err))
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCrtFile)
	return certPool, nil
}

// ServerTLSConfig returns tls.Config for server by loading certificates from files.
func (o *CredCfg) ServerTLSConfig() (*tls.Config, error) {
	if o.NoTLS {
		return nil, nil
	}
	if o.CrtPath == "" || o.KeyPath == "" {
		return nil, fmt.Errorf("TLS key-pair must be provided to enable TLS")
	}
	cert, err := o.loadKeyPair()
	if err != nil {
		return nil, err
	}
	certPool, err := o.caCertPool()
	if err != nil {
		return nil, err
	}

	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{*cert},
		ClientCAs:    certPool,
		ClientAuth:   o.ClientAuth,
	}

	return &tlsCfg, nil
}

func (o *CredCfg) ClientTLSConfig() (*tls.Config, error) {
	if o.NoTLS {
		return nil, nil
	}

	tlsCfg := tls.Config{}
	if o.SkipVerifyServerCert {
		tlsCfg.InsecureSkipVerify = true
		return &tlsCfg, nil
	}

	if cert, err := o.loadKeyPair(); err != nil {
		return nil, err
	} else if cert != nil {
		tlsCfg.Certificates = []tls.Certificate{*cert}
	}

	if certPool, err := o.caCertPool(); err != nil {
		return nil, err
	} else if certPool != nil {
		tlsCfg.RootCAs = certPool
	}

	return &tlsCfg, nil
}

// GRPCServerCredentials returns grpc.ServerOption according to the given credential config.
func GRPCServerCredentials(o *CredCfg) ([]grpc.ServerOption, error) {
	if o.NoTLS {
		return []grpc.ServerOption{}, nil
	}

	tlsCfg, err := o.ServerTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("generate tls config: %w", err)
	}

	tCred := credentials.NewTLS(tlsCfg)
	return []grpc.ServerOption{grpc.Creds(tCred)}, nil
}
