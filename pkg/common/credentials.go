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
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
)

type CredCfg struct {
	NoTLS     bool
	Insecure  bool
	CrtPath   string
	KeyPath   string
	CACrtPath string
}

func (o *CredCfg) loadKeyPair() (*tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(o.CrtPath, o.KeyPath)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("load cert key pair: %w", err))
	}
	return &certificate, nil
}

func (o *CredCfg) parseCACert() (*x509.Certificate, error) {
	caCrtFile, err := os.ReadFile(o.CACrtPath)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("read CA cert file: %w", err))
	}
	block, _ := pem.Decode(caCrtFile)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("parse ca certificate: %w", err))
	}
	return caCert, nil
}

// LoadCertificates loads certificates from files.
func (o *CredCfg) LoadCertificates() ([]tls.Certificate, *x509.CertPool, error) {
	cert, err := o.loadKeyPair()
	if err != nil {
		return nil, nil, err
	}
	caBundle, err := o.parseCACert()
	if err != nil {
		return nil, nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(caBundle)

	return []tls.Certificate{*cert}, certPool, nil
}

// GRPCServerCredentials returns grpc.ServerOption according to the given credential config.
func GRPCServerCredentials(o *CredCfg) ([]grpc.ServerOption, error) {
	if o.NoTLS {
		return []grpc.ServerOption{}, nil
	}

	certs, certPool, err := o.LoadCertificates()
	if err != nil {
		return nil, fmt.Errorf("load certificates: %w", err)
	}

	var tlsCfg tls.Config
	if o.Insecure {
		tlsCfg = tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			Certificates: certs,
			ClientCAs:    certPool,
		}
	} else {
		tlsCfg = tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: certs,
			ClientCAs:    certPool,
		}
	}

	tCred := credentials.NewTLS(&tlsCfg)
	return []grpc.ServerOption{grpc.Creds(tCred)}, nil
}
