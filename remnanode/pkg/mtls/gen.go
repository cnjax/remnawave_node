package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ServerName is the DNS name used for xray's internal API mTLS.
// Must appear in the server cert's SANs so Go's strict hostname verification passes.
const ServerName = "internal.remnawave.local"

// Certs holds the ephemeral mTLS certificate set for node↔xray gRPC communication.
// Generated once at startup; completely separate from the SECRET_KEY node certificates.
type Certs struct {
	CACertPEM     string
	ServerCertPEM string
	ServerKeyPEM  string
	ClientCertPEM string
	ClientKeyPEM  string
}

var (
	once  sync.Once
	store *Certs
)

// Get returns the singleton ephemeral mTLS cert set, generating it on first call.
func Get() *Certs {
	once.Do(func() {
		var err error
		store, err = generate()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate internal mTLS certs")
		}
		log.Info().Msg("Generated ephemeral mTLS certificates for internal xray API")
	})
	return store
}

func generate() (*Certs, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Remnawave Internal CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, err
	}

	// Server cert — SAN required because Go 1.15+ ignores CN for hostname verification.
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: ServerName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{ServerName},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: ServerName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	return &Certs{
		CACertPEM:     certPEM(caCertDER),
		ServerCertPEM: certPEM(serverCertDER),
		ServerKeyPEM:  ecKeyPEM(serverKey),
		ClientCertPEM: certPEM(clientCertDER),
		ClientKeyPEM:  ecKeyPEM(clientKey),
	}, nil
}

func certPEM(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func ecKeyPEM(key *ecdsa.PrivateKey) string {
	der, _ := x509.MarshalECPrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}
