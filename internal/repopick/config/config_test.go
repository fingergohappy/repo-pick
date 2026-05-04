package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileParsesExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "..", "configs", "config.example.yaml")

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if len(cfg.Repositories) != 2 {
		t.Fatalf("len(Repositories) = %d, want 2", len(cfg.Repositories))
	}

	official := cfg.Repositories[0]
	if official.Name != "official" {
		t.Errorf("official.Name = %q, want %q", official.Name, "official")
	}
	if official.URL != "https://github.com/org/tools" {
		t.Errorf("official.URL = %q, want %q", official.URL, "https://github.com/org/tools")
	}
	personal := cfg.Repositories[1]
	if personal.Branch != "main" {
		t.Errorf("personal.Branch = %q, want %q", personal.Branch, "main")
	}
}

func TestParseReadsConfigYAML(t *testing.T) {
	data := []byte(`
repositories:
  - name: official
    url: https://github.com/org/tools
    branch: dev
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := cfg.Repositories[0].Name; got != "official" {
		t.Errorf("Repositories[0].Name = %q, want %q", got, "official")
	}
	if got := cfg.Repositories[0].URL; got != "https://github.com/org/tools" {
		t.Errorf("Repositories[0].URL = %q, want %q", got, "https://github.com/org/tools")
	}
	if got := cfg.Repositories[0].Branch; got != "dev" {
		t.Errorf("Repositories[0].Branch = %q, want %q", got, "dev")
	}
}

func TestLoadFileReturnsReadError(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("LoadFile() error = nil, want error")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("LoadFile() error = %v, want not exist error", err)
	}
}

func TestFileStoreLoadMissingConfigReturnsEmptyConfig(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Repositories) != 0 {
		t.Fatalf("len(Repositories) = %d, want 0", len(cfg.Repositories))
	}
}

func TestFileStoreSaveCreatesParentAndWritesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo-pick", "config.yaml")
	store := NewFileStore(path)

	cfg := Config{
		Repositories: []Repository{{
			Name: "official",
			URL:  "https://github.com/org/tools",
		}},
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Repositories[0].Name; got != "official" {
		t.Fatalf("Repositories[0].Name = %q, want %q", got, "official")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "skillDir") || strings.Contains(string(data), "downloadDir") {
		t.Fatalf("config contains old fields:\n%s", data)
	}
}
