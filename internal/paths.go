package internal

import (
	"os"
	"path/filepath"
)

var (
	ConfigDir string

	StateDir string

	DefaultConfigFile string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	ConfigDir = filepath.Join(homeDir, ".config")
	StateDir = filepath.Join(homeDir, ".local/state")
	DefaultConfigFile = filepath.Join(ConfigDir, "netclip/netclip.json")
}
