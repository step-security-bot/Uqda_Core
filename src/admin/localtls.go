package admin

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{*cert},
		ClientAuth:   tls.RequireAnyClientCert,
		// The self-signed node certificate is verified by exact public-key pin
		// below, not by a public CA, hostname or the certificate's validity dates.
		InsecureSkipVerify: true, // #nosec G402 -- mandatory VerifyConnection pin
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
