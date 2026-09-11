package main

import (
	"bytes"
	"testing"

	"github.com/Uqda/Core/src/config"
	"github.com/hjson/hjson-go/v4"
)

func TestNormalisePreservesAdminAndExternalKey(t *testing.T) {
	for _, endpoint := range []string{"none", "tcp://localhost:19099", "unix:///tmp/uqda-test.sock"} {
		for _, asJSON := range []bool{false, true} {
			cfg := config.GenerateConfig()
			cfg.AdminListen = endpoint
			cfg.PrivateKeyPath = "external-test-key.pem"
			before := append([]byte(nil), cfg.PrivateKey...)
			data, err := normaliseConfig(cfg, asJSON)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := hjson.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["AdminListen"] != endpoint {
				t.Fatalf("normalization changed the administration endpoint")
			}
			if _, exists := fields["PrivateKey"]; exists {
				t.Fatal("normalization exported an externally loaded private key")
			}
			if fields["PrivateKeyPath"] != cfg.PrivateKeyPath || !bytes.Equal(before, cfg.PrivateKey) {
				t.Fatal("normalization changed key configuration")
			}
		}
	}
}
