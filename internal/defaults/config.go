package defaults

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CLIConfig stores persistent configuration for the CLI.
type CLIConfig struct {
	Workspace     string `json:"workspace,omitempty"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	ReadOnly      bool   `json:"readonly,omitempty"`
	BasicAuthUser string `json:"basic_auth_user,omitempty"`
	BasicAuthPass string `json:"basic_auth_pass,omitempty"`
}

func cliConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "liveboard", "config.json")
}

// LoadCLIConfig reads the CLI config from ~/.config/liveboard/config.json.
func LoadCLIConfig() *CLIConfig {
	path := cliConfigPath()
	if path == "" {
		return &CLIConfig{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &CLIConfig{}
	}
	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &CLIConfig{}
	}
	return &cfg
}
