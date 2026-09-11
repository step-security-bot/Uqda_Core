package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPersistentConfigurationRequiresIdentity(t *testing.T) {
	for _, input := range []string{"{}", `{ "PrivateKey": "00" }`, `{ "PrivateKey": "" }`} {
		var cfg NodeConfig
		if _, err := cfg.ReadFrom(strings.NewReader(input)); err == nil {
			t.Fatal("accepted absent or malformed persistent identity")
		}
	}
	cfg := GenerateConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var restored NodeConfig
	if _, err := restored.ReadFrom(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cfg.PrivateKey, restored.PrivateKey) {
		t.Fatal("persistent identity changed")
	}
}
