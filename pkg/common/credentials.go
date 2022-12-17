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

func NewTLSConfig(opts ...TLSConfigOpts) (*tls.Config, error) {
	cfg := &tls.Config{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("new tls config: %w", err)
		}
	}
	return cfg, nil
}

type TLSConfigOpts func(c *tls.Config) error

type TLSParams struct {
	// Disable TLS
	NoTLS bool

	// ClientAuth specifies how to handle client cert
	ClientAuth tls.ClientAuthType

	// Skip verifying server cert
	SkipVerifyServer bool

	// Path to the cert file
	CrtPath string

	// Path to the private key file
	KeyPath string

	// Path to the CA cert file
	CACrtPath string

	// To verify the server hostname
	ServerName string
}

// Certificates sets certificate to tls.Config by loading cert key-pairs from files.
func (o *TLSParams) Certificates() TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		if o.CrtPath == "" || o.KeyPath == "" {
			return fmt.Errorf("TLS key-pair must be provided to enable TLS")
		}
		cert, err := o.loadKeyPair()
		if err != nil {
			return err
		}
		cfg.Certificates = []tls.Certificate{*cert}
		return nil
	}
}

// VerifyClient applies client certificate verification settings on tls.Config.
func (o *TLSParams) VerifyClient() TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		cfg.ClientAuth = o.ClientAuth
		if certPool, err := o.caCertPool(); err != nil {
			return err
		} else {
			cfg.ClientCAs = certPool
		}
		return nil
	}
}

// VerifyServer applies server verification settings on tls.Config.
func (o *TLSParams) VerifyServer() TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		if o.SkipVerifyServer {
			cfg.InsecureSkipVerify = true
			return nil
		}
		if o.ServerName != "" {
			cfg.ServerName = o.ServerName
		}
		if certPool, err := o.caCertPool(); err != nil {
			return err
		} else if certPool != nil {
			cfg.RootCAs = certPool
		}
		return nil
	}
}

func (o *TLSParams) loadKeyPair() (*tls.Certificate, error) {
	if o.CrtPath == "" || o.KeyPath == "" {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(o.CrtPath, o.KeyPath)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("load cert key-pair: %w", err))
	}
	return &certificate, nil
}

func (o *TLSParams) caCertPool() (*x509.CertPool, error) {
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

// GRPCServerCredentials returns grpc.ServerOption according to the given credential config.
func GRPCServerCredentials(o *TLSParams) ([]grpc.ServerOption, error) {
	if o.NoTLS {
		return []grpc.ServerOption{}, nil
	}

	tlsCfg, err := NewTLSConfig(o.Certificates(), o.VerifyClient())
	if err != nil {
		return nil, fmt.Errorf("new tls config: %w", err)
	}

	tCred := credentials.NewTLS(tlsCfg)
	return []grpc.ServerOption{grpc.Creds(tCred)}, nil
}
