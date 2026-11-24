package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitViperDefaultAndPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dosprintgo_test_*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// 1. First initialization when file does not exist -> creates default file
	cfg, err := InitViper(configPath)
	if err != nil {
		t.Fatalf("InitViper falhou: %v", err)
	}

	if len(cfg.Mappings) == 0 {
		t.Errorf("esperava mapeamentos padrão, obteve 0")
	}

	if cfg.HTTPPort != 8989 {
		t.Errorf("esperava porta HTTP padrão 8989, obteve %d", cfg.HTTPPort)
	}

	// 2. Add a new custom mapping and save
	cfg.Mappings = append(cfg.Mappings, MappingConfig{
		Port:         "lpt3", // lowercase to test uppercase normalization
		TargetType:   TargetTCP,
		Target:       "192.168.1.199:9100",
		Description:  "Zebra Almoxarifado",
		TimeoutMs:    3000,
		InactivityMs: 250,
	})

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig falhou: %v", err)
	}

	// 3. Read config back using GetConfig
	updatedCfg, err := GetConfig()
	if err != nil {
		t.Fatalf("GetConfig falhou: %v", err)
	}

	foundLPT3 := false
	for _, m := range updatedCfg.Mappings {
		if m.Port == "LPT3" {
			foundLPT3 = true
			if m.Target != "192.168.1.199:9100" {
				t.Errorf("target do LPT3 incorreto: %s", m.Target)
			}
			break
		}
	}

	if !foundLPT3 {
		t.Errorf("porta LPT3 recém-adicionada não foi encontrada após SaveConfig")
	}
}

func TestInitViperTOMLPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dosprintgo_toml_*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.toml")

	// 1. Initialize TOML config
	cfg, err := InitViper(configPath)
	if err != nil {
		t.Fatalf("InitViper falhou para TOML: %v", err)
	}

	if len(cfg.Mappings) == 0 {
		t.Errorf("esperava mapeamentos padrão no TOML, obteve 0")
	}

	// 2. Verify that config.toml was created on disk
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("arquivo config.toml não foi gravado no disco: %v", err)
	}

	// TOML should contain keys like spool_dir and mappings
	strContent := string(content)
	if !strings.Contains(strContent, "spool_dir") {
		t.Errorf("conteúdo do config.toml não contém 'spool_dir':\n%s", strContent)
	}

	// 3. Add and save a new mapping in TOML
	cfg.Mappings = append(cfg.Mappings, MappingConfig{
		Port:         "LPT4",
		TargetType:   TargetTCP,
		Target:       "10.0.0.88:9100",
		Description:  "Argox Expedição",
		TimeoutMs:    4000,
		InactivityMs: 300,
	})

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig TOML falhou: %v", err)
	}

	// 4. Verify updated file content
	updatedContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("falha ao ler config.toml atualizado: %v", err)
	}

	if !strings.Contains(string(updatedContent), "10.0.0.88:9100") {
		t.Errorf("novo IP não encontrado no config.toml:\n%s", string(updatedContent))
	}
}
