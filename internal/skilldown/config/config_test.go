package config

import (
	"os"
	"path/filepath"
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
	if official.URL != "https://github.com/org/skills" {
		t.Errorf("official.URL = %q, want %q", official.URL, "https://github.com/org/skills")
	}
	if official.SkillDir != "skills" {
		t.Errorf("official.SkillDir = %q, want %q", official.SkillDir, "skills")
	}

	personal := cfg.Repositories[1]
	if personal.Name != "personal" {
		t.Errorf("personal.Name = %q, want %q", personal.Name, "personal")
	}
	if personal.URL != "git@github.com:finger/my-skills.git" {
		t.Errorf("personal.URL = %q, want %q", personal.URL, "git@github.com:finger/my-skills.git")
	}
	if personal.SkillDir != "skills" {
		t.Errorf("personal.SkillDir = %q, want %q", personal.SkillDir, "skills")
	}

	if cfg.Repo.DownloadDir != "" {
		t.Errorf("Repo.DownloadDir = %q, want empty", cfg.Repo.DownloadDir)
	}
}

func TestParseReadsConfigYAML(t *testing.T) {
	data := []byte(`
repositories:
  - name: official
    url: https://github.com/org/skills
    skillDir: skills
repo:
  downloadDir: /tmp/skill-down
`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := cfg.Repositories[0].Name; got != "official" {
		t.Errorf("Repositories[0].Name = %q, want %q", got, "official")
	}
	if got := cfg.Repo.DownloadDir; got != "/tmp/skill-down" {
		t.Errorf("Repo.DownloadDir = %q, want %q", got, "/tmp/skill-down")
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
	if cfg.Repo.DownloadDir != "" {
		t.Fatalf("Repo.DownloadDir = %q, want empty", cfg.Repo.DownloadDir)
	}
}

func TestFileStoreSaveCreatesParentAndWritesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skill-down", "config.yaml")
	store := NewFileStore(path)

	cfg := Config{
		Repositories: []Repository{{
			Name:     "official",
			URL:      "https://github.com/org/skills",
			SkillDir: "skills",
		}},
		Repo: RepoConfig{DownloadDir: "/tmp/skill-down"},
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
	if got := loaded.Repo.DownloadDir; got != "/tmp/skill-down" {
		t.Fatalf("Repo.DownloadDir = %q, want %q", got, "/tmp/skill-down")
	}
}
