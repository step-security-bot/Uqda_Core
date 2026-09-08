//go:build windows

package config

import (
	"path/filepath"
	"testing"
)

func TestWindowsAdminEndpointIsIndependent(t *testing.T) {
	if got := getDefaults().DefaultAdminListen; got != "tcp://localhost:19001" {
		t.Fatalf("unexpected UQDA Windows admin endpoint: %s", got)
	}
}

func TestWindowsConfigFileUsesProgramData(t *testing.T) {
	t.Setenv("ProgramData", `D:\SharedData`)
	want := filepath.Join(`D:\SharedData`, "UQDA", "uqda.conf")
	if got := windowsConfigFile(); got != want {
		t.Fatalf("windowsConfigFile() = %q, want %q", got, want)
	}
}

func TestWindowsConfigFileFallback(t *testing.T) {
	t.Setenv("ProgramData", "")
	if got, want := windowsConfigFile(), `C:\ProgramData\UQDA\uqda.conf`; got != want {
		t.Fatalf("windowsConfigFile() = %q, want %q", got, want)
	}
}
