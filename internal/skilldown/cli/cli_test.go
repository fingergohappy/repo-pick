package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/finger/skill-down/internal/skilldown/app"
	"github.com/finger/skill-down/internal/skilldown/config"
	"github.com/finger/skill-down/internal/skilldown/install"
	"github.com/finger/skill-down/internal/skilldown/registry"
	"github.com/finger/skill-down/internal/skilldown/repo"
)

func TestBrowseCommandRunsTUI(t *testing.T) {
	store := &memoryStore{}
	service := app.Service{Registry: registry.NewService(store), Cloner: &emptyCloner{}, Installer: install.Installer{}}
	var gotRepo string
	runTUIForTest := runTUI
	runTUI = func(ctx context.Context, svc app.Service, initialRepo string) error {
		gotRepo = initialRepo
		return nil
	}
	t.Cleanup(func() { runTUI = runTUIForTest })

	code, _, stderr := executeForTest(context.Background(), service, "browse", "https://github.com/org/skills")

	if code != 0 {
		t.Fatalf("Execute() code = %d, want 0, stderr = %q", code, stderr)
	}
	if gotRepo != "https://github.com/org/skills" {
		t.Fatalf("initialRepo = %q, want explicit repo", gotRepo)
	}
}

func TestRegistryAddCommandCallsApp(t *testing.T) {
	store := &memoryStore{}
	service := app.Service{Registry: registry.NewService(store), Cloner: &emptyCloner{}, Installer: install.Installer{}}

	code, _, stderr := executeForTest(context.Background(), service, "registry", "add", "https://github.com/org/skills", "--name", "official", "--skill-dir", "custom")

	if code != 0 {
		t.Fatalf("Execute() code = %d, want 0, stderr = %q", code, stderr)
	}
	want := []config.Repository{{Name: "official", URL: "https://github.com/org/skills", SkillDir: "custom"}}
	if got := store.cfg.Repositories; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("repositories = %#v, want %#v", got, want)
	}
}

func TestRegistryListCommandPrintsRepositories(t *testing.T) {
	store := &memoryStore{cfg: config.Config{Repositories: []config.Repository{{
		Name:     "official",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}}}
	service := app.Service{Registry: registry.NewService(store), Cloner: &emptyCloner{}, Installer: install.Installer{}}

	code, stdout, stderr := executeForTest(context.Background(), service, "registry", "list")

	if code != 0 {
		t.Fatalf("Execute() code = %d, want 0, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "official") || !strings.Contains(stdout, "https://github.com/org/skills") {
		t.Fatalf("stdout = %q, want repository details", stdout)
	}
}

func TestSearchCommandPrintsDiscoveredSkills(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{"alpha": "# Alpha\n"})
	service := app.Service{
		Registry:  registry.NewService(&memoryStore{}),
		Cloner:    &singleWorktreeCloner{worktree: repo.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	code, stdout, stderr := executeForTest(context.Background(), service, "search", "https://github.com/org/skills")

	if code != 0 {
		t.Fatalf("Execute() code = %d, want 0, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "alpha") || !strings.Contains(stdout, "skills/alpha") {
		t.Fatalf("stdout = %q, want discovered skill", stdout)
	}
}

func TestInstallCommandCopiesSkill(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{"alpha": "# Alpha\n"})
	targetRoot := t.TempDir()
	service := app.Service{
		Registry:  registry.NewService(&memoryStore{}),
		Cloner:    &singleWorktreeCloner{worktree: repo.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	code, stdout, stderr := executeForTest(context.Background(), service, "install", "https://github.com/org/skills", "--skill", "alpha", "--to", targetRoot)

	if code != 0 {
		t.Fatalf("Execute() code = %d, want 0, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "installed") || !strings.Contains(stdout, "alpha") {
		t.Fatalf("stdout = %q, want install result", stdout)
	}
	assertFileContent(t, filepath.Join(targetRoot, "alpha", "SKILL.md"), "# Alpha\n")
}

func executeForTest(ctx context.Context, svc app.Service, args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := execute(ctx, args, svc, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

type memoryStore struct {
	cfg config.Config
}

func (s *memoryStore) Load() (config.Config, error) {
	return s.cfg, nil
}

func (s *memoryStore) Save(cfg config.Config) error {
	s.cfg = cfg
	return nil
}

type emptyCloner struct{}

func (c *emptyCloner) Clone(ctx context.Context, repoURL string, options repo.CloneOptions) (repo.Worktree, error) {
	return repo.Worktree{}, nil
}

func (c *emptyCloner) Cleanup(worktree repo.Worktree) error {
	return nil
}

type singleWorktreeCloner struct {
	worktree repo.Worktree
}

func (c *singleWorktreeCloner) Clone(ctx context.Context, repoURL string, options repo.CloneOptions) (repo.Worktree, error) {
	return c.worktree, nil
}

func (c *singleWorktreeCloner) Cleanup(worktree repo.Worktree) error {
	return nil
}

func createWorktree(t *testing.T, dirPath string, skills map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, content := range skills {
		skillDir := filepath.Join(root, filepath.FromSlash(dirPath), name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}
	return root
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s content = %q, want %q", path, string(data), want)
	}
}
