package core

import (
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func (c *Core) generateTLSConfig(cert *tls.Certificate) (*tls.Config, error) {
	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		ClientAuth:   tls.RequireAnyClientCert,
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return cert, nil
		},
		// Mesh identities are self-issued, not DNS/CA identities. Validate the
		// certificate here and bind its key to the overlay handshake below.
		VerifyConnection:   c.verifyTLSConnection,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}
	return config, nil
}

func (c *Core) verifyTLSConnection(state tls.ConnectionState) error {
	if len(state.PeerCertificates) != 1 {
		return fmt.Errorf("mesh TLS requires exactly one identity certificate")
	}
	cert := state.PeerCertificates[0]
	if _, ok := cert.PublicKey.(ed25519.PublicKey); !ok {
		return fmt.Errorf("mesh TLS requires an Ed25519 identity")
	}
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("mesh TLS identity certificate is outside its validity period")
	}
	return cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature)
}

// verifyTLSPeerIdentity binds the certificate used by a direct TLS transport
// to the Ed25519 identity authenticated by the UQDA handshake. Certificate
// chains are intentionally not used because UQDA identities are self-issued.
func verifyTLSPeerIdentity(conn net.Conn, expected ed25519.PublicKey) error {
	if tracked, ok := conn.(*linkConn); ok {
		conn = tracked.Conn
	}
	var state tls.ConnectionState
	switch transport := conn.(type) {
	case *tls.Conn:
		state = transport.ConnectionState()
	case *linkQUICStream:
		state = transport.Conn.ConnectionState().TLS
	default:
		return nil
	}
	return verifyTLSStateIdentity(state, expected)
}

func verifyTLSStateIdentity(state tls.ConnectionState, expected ed25519.PublicKey) error {
	if len(state.PeerCertificates) != 1 {
		return fmt.Errorf("TLS peer presented %d certificates, want exactly one", len(state.PeerCertificates))
	}
	peerKey, ok := state.PeerCertificates[0].PublicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("TLS peer certificate does not contain an Ed25519 key")
	}
	if !peerKey.Equal(expected) {
		return fmt.Errorf("TLS peer certificate does not match UQDA identity")
	}
	return nil
}
