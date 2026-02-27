package plugin

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// DefaultCertsDir is the default directory for mTLS certificates.
const DefaultCertsDir = "~/.orchestra/certs"

// ResolveCertsDir expands ~ in the certs directory path.
func ResolveCertsDir(certsDir string) string {
	if certsDir == "" {
		certsDir = DefaultCertsDir
	}
	if len(certsDir) > 0 && certsDir[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			certsDir = filepath.Join(home, certsDir[1:])
		}
	}
	return certsDir
}

// EnsureCA loads or generates a CA certificate and private key at the given
// certs directory. The CA cert is stored at certsDir/ca.crt and the key at
// certsDir/ca.key. Uses ed25519 for fast, compact keys.
func EnsureCA(certsDir string) (*x509.Certificate, crypto.PrivateKey, error) {
	certsDir = ResolveCertsDir(certsDir)

	caCertPath := filepath.Join(certsDir, "ca.crt")
	caKeyPath := filepath.Join(certsDir, "ca.key")

	// Try to load existing CA.
	if caCert, caKey, err := loadCertAndKey(caCertPath, caKeyPath); err == nil {
		return caCert, caKey, nil
	}

	// Generate new CA.
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return nil, nil, fmt.Errorf("create certs dir: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	serialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Orchestra"},
			CommonName:   "Orchestra CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("create CA certificate: %w", err)
	}

	if err := writePEMFile(caCertPath, "CERTIFICATE", certDER); err != nil {
		return nil, nil, err
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	if err := writePEMFile(caKeyPath, "PRIVATE KEY", keyDER); err != nil {
		return nil, nil, err
	}

	caCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	return caCert, priv, nil
}

// GenerateCert creates a new certificate signed by the given CA for the named
// entity. Certificates are stored at certsDir/{name}.crt and certsDir/{name}.key.
func GenerateCert(certsDir string, name string, caCert *x509.Certificate, caKey crypto.PrivateKey) (tls.Certificate, error) {
	certsDir = ResolveCertsDir(certsDir)

	certPath := filepath.Join(certsDir, name+".crt")
	keyPath := filepath.Join(certsDir, name+".key")

	// Try to load existing cert.
	if tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		return tlsCert, nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	serialNumber, err := randomSerialNumber()
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Orchestra"},
			CommonName:   name,
		},
		NotBefore: time.Now().Add(-1 * time.Minute),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames:    []string{name, "localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, pub, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	if err := writePEMFile(certPath, "CERTIFICATE", certDER); err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal private key: %w", err)
	}
	if err := writePEMFile(keyPath, "PRIVATE KEY", keyDER); err != nil {
		return tls.Certificate{}, err
	}

	return tls.LoadX509KeyPair(certPath, keyPath)
}

// ServerTLSConfig returns a TLS configuration for a QUIC server that uses mTLS.
// It loads or generates the named certificate and requires client certificates
// signed by the same CA.
func ServerTLSConfig(certsDir string, name string) (*tls.Config, error) {
	caCert, caKey, err := EnsureCA(certsDir)
	if err != nil {
		return nil, fmt.Errorf("ensure CA: %w", err)
	}

	cert, err := GenerateCert(certsDir, name, caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("generate server cert: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		NextProtos:   []string{"orchestra-plugin"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ClientTLSConfig returns a TLS configuration for a QUIC client that uses mTLS.
// It loads or generates the named certificate and trusts the CA.
func ClientTLSConfig(certsDir string, name string) (*tls.Config, error) {
	caCert, caKey, err := EnsureCA(certsDir)
	if err != nil {
		return nil, fmt.Errorf("ensure CA: %w", err)
	}

	cert, err := GenerateCert(certsDir, name, caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("generate client cert: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		NextProtos:   []string{"orchestra-plugin"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// loadCertAndKey loads a PEM-encoded certificate and private key from disk.
func loadCertAndKey(certPath, keyPath string) (*x509.Certificate, crypto.PrivateKey, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("no PEM block in %s", certPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("no PEM block in %s", keyPath)
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// writePEMFile writes a PEM-encoded block to the given file path with 0600 perms.
func writePEMFile(path string, blockType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

// randomSerialNumber generates a random serial number for X.509 certificates.
func randomSerialNumber() (*big.Int, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	return n, nil
}
