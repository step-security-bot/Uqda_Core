package core

import (
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/Uqda/Core/src/config"
)

func TestMeshCertificateValidation(t *testing.T) {
	cfg := config.GenerateConfig()
	cert, err := x509.ParseCertificate(cfg.Certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	c := &Core{}
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
	if err := c.verifyTLSConnection(state); err != nil {
		t.Fatalf("valid self-issued mesh certificate rejected: %v", err)
	}
	if err := c.verifyTLSConnection(tls.ConnectionState{}); err == nil {
		t.Fatal("missing certificate accepted")
	}
	key := cert.PublicKey.(ed25519.PublicKey)
	if err := verifyTLSStateIdentity(state, key); err != nil {
		t.Fatal(err)
	}
	other := config.GenerateConfig()
	if err := verifyTLSStateIdentity(state, ed25519.PrivateKey(other.PrivateKey).Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("mismatched overlay identity accepted")
	}
	bad := *cert
	bad.Signature = append([]byte(nil), cert.Signature...)
	bad.Signature[0] ^= 1
	if err := c.verifyTLSConnection(tls.ConnectionState{PeerCertificates: []*x509.Certificate{&bad}}); err == nil {
		t.Fatal("invalid self-signature accepted")
	}
	bad = *cert
	bad.NotAfter = time.Now().Add(-time.Hour)
	if err := c.verifyTLSConnection(tls.ConnectionState{PeerCertificates: []*x509.Certificate{&bad}}); err == nil {
		t.Fatal("expired certificate accepted")
	}
}
