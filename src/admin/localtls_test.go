package admin

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Uqda/Core/src/config"
)

func TestLocalTLSRequiresMatchingNodeIdentity(t *testing.T) {
	local := config.GenerateConfig()
	other := config.GenerateConfig()
	for _, tc := range []struct {
		name string
		cert *tls.Certificate
		wantOK bool
	}{
		{"same node", local.Certificate, true},
		{"different node", other.Certificate, false},
		{"no client certificate", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := net.Pipe()
			defer left.Close()
			defer right.Close()
			_ = left.SetDeadline(time.Now().Add(2 * time.Second))
			_ = right.SetDeadline(time.Now().Add(2 * time.Second))
			server := tls.Server(left, PinnedTLSConfig(local.Certificate))
			clientConfig := PinnedTLSConfig(local.Certificate)
			if tc.cert == nil {
				clientConfig.Certificates = nil
			} else {
				clientConfig.Certificates = []tls.Certificate{*tc.cert}
			}
			client := tls.Client(right, clientConfig)
			result := make(chan error, 1)
			go func() {
				err := server.Handshake()
				_ = left.Close()
				result <- err
			}()
			_ = client.Handshake()
			if ok := <-result == nil; ok != tc.wantOK {
				t.Fatalf("server authentication success=%v, want %v", ok, tc.wantOK)
			}
		})
	}
	if PinnedTLSConfig(nil) != nil {
		t.Fatal("missing identity must not produce a TLS configuration")
	}
}

func TestLocalTLSPinsServerIdentity(t *testing.T) {
	local := config.GenerateConfig()
	other := config.GenerateConfig()
	leaf, err := x509.ParseCertificate(other.Certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := PinnedTLSConfig(local.Certificate).VerifyConnection(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{leaf},
	}); err == nil {
		t.Fatal("different server identity accepted")
	}
}

func TestHandlerRegistrationConcurrentWithRequests(t *testing.T) {
	a := &AdminSocket{handlers: make(map[string]handler)}
	answer := func(json.RawMessage) (interface{}, error) { return "ok", nil }
	if err := a.AddHandler("probe", "test", nil, answer); err != nil {
		t.Fatal(err)
	}
	left, right := net.Pipe()
	defer right.Close()
	go a.handleRequest(left)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_ = a.AddHandler(fmt.Sprintf("handler%d", i), "test", nil, answer)
		}
	}()
	encoder, decoder := json.NewEncoder(right), json.NewDecoder(right)
	for i := 0; i < 100; i++ {
		if err := encoder.Encode(AdminSocketRequest{Name: "probe", KeepAlive: i < 99}); err != nil {
			t.Fatal(err)
		}
		var response AdminSocketResponse
		if err := decoder.Decode(&response); err != nil || response.Status != "success" {
			t.Fatalf("request failed: %v", err)
		}
	}
	<-done
}
