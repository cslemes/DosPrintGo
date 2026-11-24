package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TargetType defines the destination print type
type TargetType string

const (
	TargetTCP     TargetType = "tcp"
	TargetSpooler TargetType = "windows_printer"
)

// MappingConfig defines one LPT redirection rule
type MappingConfig struct {
	Port         string     `json:"port" toml:"port" mapstructure:"port"`
	TargetType   TargetType `json:"target_type" toml:"target_type" mapstructure:"target_type"`
	Target       string     `json:"target" toml:"target" mapstructure:"target"`
	Description  string     `json:"description" toml:"description" mapstructure:"description"`
	TimeoutMs    int        `json:"timeout_ms" toml:"timeout_ms" mapstructure:"timeout_ms"`
	InactivityMs int        `json:"inactivity_ms" toml:"inactivity_ms" mapstructure:"inactivity_ms"`
}

// Config represents the application configuration
type Config struct {
	SpoolDir string          `json:"spool_dir" toml:"spool_dir" mapstructure:"spool_dir"`
	HTTPPort int             `json:"http_port" toml:"http_port" mapstructure:"http_port"` // Port for local Web GUI, default 8989
	Mappings []MappingConfig `json:"mappings" toml:"mappings" mapstructure:"mappings"`
}

// DefaultConfig returns a sensible starter configuration
func DefaultConfig() *Config {
	return &Config{
		SpoolDir: `C:\ProgramData\DosPrintGo\spool`,
		Mappings: []MappingConfig{
			{
				Port:         "LPT1",
				TargetType:   TargetTCP,
				Target:       "192.168.1.150:9100",
				Description:  "Zebra ZPL (Etiquetas Gôndola)",
				TimeoutMs:    5000,
				InactivityMs: 300,
			},
			{
				Port:         "LPT2",
				TargetType:   TargetTCP,
				Target:       "192.168.1.151:9100",
				Description:  "Argox PPLA/PPLB (Etiquetas Recebimento)",
				TimeoutMs:    5000,
				InactivityMs: 300,
			},
		},
	}
}

// Load loads the configuration from a JSON file path
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		// Default path in current executable directory or ProgramData
		exePath, err := os.Executable()
		if err == nil {
			localConfig := filepath.Join(filepath.Dir(exePath), "config.json")
			if _, err := os.Stat(localConfig); err == nil {
				configPath = localConfig
			}
		}
		if configPath == "" {
			configPath = `C:\ProgramData\DosPrintGo\config.json`
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler o arquivo de configuração '%s': %w", configPath, err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("formato inválido no arquivo de configuração '%s': %w", configPath, err)
	}

	// Apply default values if needed
	if cfg.SpoolDir == "" {
		cfg.SpoolDir = `C:\ProgramData\DosPrintGo\spool`
	}

	for i := range cfg.Mappings {
		if cfg.Mappings[i].TimeoutMs <= 0 {
			cfg.Mappings[i].TimeoutMs = 5000
		}
		if cfg.Mappings[i].InactivityMs <= 0 {
			cfg.Mappings[i].InactivityMs = 300
		}
		if cfg.Mappings[i].TargetType == "" {
			cfg.Mappings[i].TargetType = TargetTCP
		}
	}

	return cfg, nil
}

// GetTimeoutDuration returns the TCP timeout as time.Duration
func (m *MappingConfig) GetTimeoutDuration() time.Duration {
	return time.Duration(m.TimeoutMs) * time.Millisecond
}

// GetInactivityDuration returns the file quiet period as time.Duration
func (m *MappingConfig) GetInactivityDuration() time.Duration {
	return time.Duration(m.InactivityMs) * time.Millisecond
}
