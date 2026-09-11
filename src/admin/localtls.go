package admin

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

// PinnedTLSConfig grants administration only to a client holding this node's
// private key. Both sides pin the same public key; no network-peer keys are
// accepted. Never copy a node private key to another machine for remote admin.
// Existing Unix sockets retain their OS access controls and do not use TLS.
func PinnedTLSConfig(cert *tls.Certificate) *tls.Config {
	if cert == nil || len(cert.Certificate) == 0 {
		return nil
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil
	}
	key, ok := leaf.PublicKey.(ed25519.PublicKey)
	if !ok {
		return nil
	}
	pinned := append(ed25519.PublicKey(nil), key...)
	privateKey, ok := cert.PrivateKey.(ed25519.PrivateKey)
	if !ok || len(privateKey) != ed25519.PrivateKeySize || !privateKey.Public().(ed25519.PublicKey).Equal(key) {
		return nil
	}
	// Use a separate, reproducible admin certificate. The mesh certificate
	// remains unchanged. Only this node's identity is in the local trust store.
	template := &x509.Certificate{}
	template.SerialNumber = big.NewInt(1)
	template.Subject = pkix.Name{CommonName: "UQDA local administration"}
	template.DNSNames = []string{"uqda-admin.local"}
	template.NotBefore = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	template.NotAfter = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
	template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	template.BasicConstraintsValid = true
	template.IsCA = true
	der, err := x509.CreateCertificate(rand.Reader, template, template, key, privateKey)
	if err != nil {
		return nil
	}
	adminLeaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil
	}
	roots := x509.NewCertPool()
	roots.AddCert(adminLeaf)
	adminCert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: privateKey, Leaf: adminLeaf}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{adminCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		RootCAs:      roots,
		ClientCAs:    roots,
		ServerName:   "uqda-admin.local",

		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) != 1 {
				return fmt.Errorf("administration requires the local node certificate")
			}
			peer, ok := state.PeerCertificates[0].PublicKey.(ed25519.PublicKey)
			if !ok || !bytes.Equal(peer, pinned) {
				return fmt.Errorf("administration certificate does not match this node")
			}
			return nil
		},
	}
}
