//go:build windows

package config

import (
	"os"
	"path/filepath"
)

func windowsConfigFile() string {
	if programData := os.Getenv("ProgramData"); programData != "" {
		return filepath.Join(programData, "UQDA", "uqda.conf")
	}
	return `C:\ProgramData\UQDA\uqda.conf`
}

// Sane defaults for the Windows platform. The "default" options may be
// may be replaced by the running configuration.
func getDefaults() platformDefaultParameters {
	return platformDefaultParameters{
		// Admin
		DefaultAdminListen: "tcp://localhost:19001",

		// Configuration (used for uqdactl)
		DefaultConfigFile: windowsConfigFile(),

		// Multicast interfaces
		DefaultMulticastInterfaces: []MulticastInterfaceConfig{
			{Regex: ".*", Beacon: true, Listen: true},
		},

		// TUN
		MaximumIfMTU:  65535,
		DefaultIfMTU:  65535,
		DefaultIfName: "UQDA",
	}
}
