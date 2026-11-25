package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	vMu sync.RWMutex
	v   *viper.Viper
)

// InitViper initializes and loads configuration using Viper (defaulting to TOML)
func InitViper(configPath string) (*Config, error) {
	vMu.Lock()
	defer vMu.Unlock()

	v = viper.New()

	resolvedPath := resolveConfigPath(configPath)
	v.SetConfigFile(resolvedPath)

	ext := strings.ToLower(filepath.Ext(resolvedPath))
	switch ext {
	case ".toml":
		v.SetConfigType("toml")
	case ".json":
		v.SetConfigType("json")
	case ".yaml", ".yml":
		v.SetConfigType("yaml")
	default:
		v.SetConfigType("toml")
	}

	// Set defaults
	v.SetDefault("spool_dir", `C:\ProgramData\DosPrintGo\spool`)
	v.SetDefault("http_port", 8989)
	v.SetDefault("mappings", []MappingConfig{
		{
			Port:         "LPT1",
			TargetType:   TargetTCP,
			Target:       "192.168.1.150:9100",
			Description:  "Zebra ZPL - Etiquetas de Gondola",
			TimeoutMs:    5000,
			InactivityMs: 300,
		},
		{
			Port:         "LPT2",
			TargetType:   TargetTCP,
			Target:       "192.168.1.151:9100",
			Description:  "Argox PPLA/PPLB - Etiquetas Recebimento/Expedição",
			TimeoutMs:    5000,
			InactivityMs: 300,
		},
	})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			configDir := filepath.Dir(resolvedPath)
			_ = os.MkdirAll(configDir, 0755)
			if writeErr := v.WriteConfigAs(resolvedPath); writeErr != nil {
				// Fallback to local config.toml if directory is not writable
				localPath := "config.toml"
				_ = v.WriteConfigAs(localPath)
			}
		} else {
			return nil, fmt.Errorf("erro ao ler arquivo de configuração '%s': %w", resolvedPath, err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("erro ao mapear configuração: %w", err)
	}

	normalizeConfig(cfg)
	return cfg, nil
}

// GetConfig returns the current configuration parsed from Viper
func GetConfig() (*Config, error) {
	vMu.RLock()
	defer vMu.RUnlock()

	if v == nil {
		return nil, fmt.Errorf("viper não foi inicializado")
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	normalizeConfig(cfg)
	return cfg, nil
}

// SaveConfig saves an updated configuration to disk via Viper
func SaveConfig(cfg *Config) error {
	vMu.Lock()
	defer vMu.Unlock()

	if v == nil {
		return fmt.Errorf("viper não foi inicializado")
	}

	normalizeConfig(cfg)
	v.Set("spool_dir", cfg.SpoolDir)
	v.Set("http_port", cfg.HTTPPort)
	v.Set("mappings", cfg.Mappings)

	if err := v.WriteConfig(); err != nil {
		// If WriteConfig fails, try WriteConfigAs
		return v.WriteConfigAs(v.ConfigFileUsed())
	}

	return nil
}

// ConfigFileUsed returns the file path of the active config
func ConfigFileUsed() string {
	vMu.RLock()
	defer vMu.RUnlock()
	if v != nil {
		return v.ConfigFileUsed()
	}
	return ""
}

func resolveConfigPath(path string) string {
	if path != "" {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs
		}
		return path
	}

	// 1. Check current working directory for config.toml
	if _, err := os.Stat("config.toml"); err == nil {
		abs, err := filepath.Abs("config.toml")
		if err == nil {
			return abs
		}
		return "config.toml"
	}

	// 2. Check current working directory for config.json (backward compatibility)
	if _, err := os.Stat("config.json"); err == nil {
		abs, err := filepath.Abs("config.json")
		if err == nil {
			return abs
		}
		return "config.json"
	}

	// 3. Check executable directory for config.toml
	exeDir := "."
	exePath, err := os.Executable()
	if err == nil {
		exeDir = filepath.Dir(exePath)
	}

	exeToml := filepath.Join(exeDir, "config.toml")
	if _, err := os.Stat(exeToml); err == nil {
		abs, err := filepath.Abs(exeToml)
		if err == nil {
			return abs
		}
		return exeToml
	}

	exeJson := filepath.Join(exeDir, "config.json")
	if _, err := os.Stat(exeJson); err == nil {
		abs, err := filepath.Abs(exeJson)
		if err == nil {
			return abs
		}
		return exeJson
	}

	// 4. Default target: config.toml in current execution folder
	absToml, err := filepath.Abs(exeToml)
	if err == nil {
		return absToml
	}
	return exeToml
}

func normalizeConfig(cfg *Config) {
	if cfg.SpoolDir == "" {
		cfg.SpoolDir = `C:\ProgramData\DosPrintGo\spool`
	}
	if cfg.HTTPPort <= 0 {
		cfg.HTTPPort = 8989
	}
	for i := range cfg.Mappings {
		cfg.Mappings[i].Port = strings.ToUpper(strings.TrimSpace(cfg.Mappings[i].Port))
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
}
