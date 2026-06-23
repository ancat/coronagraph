package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"sync"
	"time"
)

// CA holds a root certificate authority used to sign per-host TLS certificates.
type CA struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey

	mu    sync.RWMutex
	cache map[string]*tls.Certificate
}

// LoadCA loads a root CA certificate and private key from PEM files.
func LoadCA(certPath, keyPath string) (*CA, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read CA key: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("decode CA cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("decode CA key PEM")
	}
	key, err := parsePrivateKey(keyBlock)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA key must be RSA")
	}

	return &CA{
		cert:  cert,
		key:   rsaKey,
		cache: make(map[string]*tls.Certificate),
	}, nil
}

func parsePrivateKey(block *pem.Block) (any, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported key type %q", block.Type)
	}
}

// CertificateForHost returns a TLS certificate valid for the given hostname.
func (ca *CA) CertificateForHost(host string) (*tls.Certificate, error) {
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}

	ca.mu.RLock()
	if cert, ok := ca.cache[hostname]; ok {
		ca.mu.RUnlock()
		return cert, nil
	}
	ca.mu.RUnlock()

	ca.mu.Lock()
	defer ca.mu.Unlock()

	if cert, ok := ca.cache[hostname]; ok {
		return cert, nil
	}

	cert, err := ca.generateCertificate(hostname)
	if err != nil {
		return nil, err
	}
	ca.cache[hostname] = cert
	return cert, nil
}

func (ca *CA) generateCertificate(hostname string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		DNSNames:  []string{hostname},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, ca.cert, &priv.PublicKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("sign leaf cert: %w", err)
	}

	log.Printf("generated certificate for %s\n", hostname)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("build TLS cert: %w", err)
	}
	return &tlsCert, nil
}
