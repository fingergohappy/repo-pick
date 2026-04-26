package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/finger/skill-down/internal/skilldown/config"
	"github.com/finger/skill-down/internal/skilldown/install"
	"github.com/finger/skill-down/internal/skilldown/registry"
	"github.com/finger/skill-down/internal/skilldown/repo"
)

func TestServiceRegistryDelegatesToRegistry(t *testing.T) {
	registrySvc := &fakeRegistry{}
	service := Service{Registry: registrySvc}

	if err := service.AddRepository(context.Background(), AddRepositoryRequest{
		Name:         "official",
		URL:          "https://github.com/org/skills",
		SkillDirPath: "custom",
	}); err != nil {
		t.Fatalf("AddRepository() error = %v", err)
	}
	if err := service.RemoveRepository(context.Background(), RemoveRepositoryRequest{Name: "official"}); err != nil {
		t.Fatalf("RemoveRepository() error = %v", err)
	}

	wantAdded := config.Repository{Name: "official", URL: "https://github.com/org/skills", SkillDir: "custom"}
	if !reflect.DeepEqual(registrySvc.added, []config.Repository{wantAdded}) {
		t.Fatalf("added repositories = %#v, want %#v", registrySvc.added, []config.Repository{wantAdded})
	}
	if !reflect.DeepEqual(registrySvc.removed, []string{"official"}) {
		t.Fatalf("removed repositories = %#v, want [official]", registrySvc.removed)
	}
}

func TestServiceSearchClonesResolvedReposAndReturnsGroupedSkills(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{
		"alpha": "---\nname: alpha\ndescription: first\n---\n# Alpha\n",
	})
	registrySvc := &fakeRegistry{resolved: []config.Repository{{
		Name:     "official",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}}
	cloner := &fakeCloner{worktrees: []repo.Worktree{{Dir: worktree}}}
	service := Service{Registry: registrySvc, Cloner: cloner, Installer: install.Installer{}}

	result, err := service.Search(context.Background(), SearchRequest{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if !reflect.DeepEqual(registrySvc.resolvedWith, []string{""}) {
		t.Fatalf("Resolve() args = %#v, want empty repo arg", registrySvc.resolvedWith)
	}
	if got := cloner.cloneOptions[0].SparsePaths; !reflect.DeepEqual(got, []string{"skills"}) {
		t.Fatalf("Clone() sparse paths = %#v, want [skills]", got)
	}
	if !reflect.DeepEqual(cloner.cleaned, []repo.Worktree{{Dir: worktree}}) {
		t.Fatalf("Cleanup() worktrees = %#v, want worktree cleanup", cloner.cleaned)
	}

	want := SearchResult{Repositories: []RepositorySearchResult{{
		Repository: config.Repository{Name: "official", URL: "https://github.com/org/skills", SkillDir: "skills"},
		Skills: []SkillResult{{
			Name:        "alpha",
			Path:        "skills/alpha",
			Description: "first",
			Content:     "---\nname: alpha\ndescription: first\n---\n# Alpha\n",
		}},
	}}}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("Search() = %#v, want %#v", result, want)
	}
}

func TestServiceSearchRequestSkillDirOverridesRegistrySkillDir(t *testing.T) {
	worktree := createWorktree(t, "custom", map[string]string{
		"beta": "# Beta\n",
	})
	registrySvc := &fakeRegistry{resolved: []config.Repository{{
		Name:     "official",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}}
	cloner := &fakeCloner{worktrees: []repo.Worktree{{Dir: worktree}}}
	service := Service{Registry: registrySvc, Cloner: cloner, Installer: install.Installer{}}

	result, err := service.Search(context.Background(), SearchRequest{SkillDirPath: "custom"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got := cloner.cloneOptions[0].SparsePaths; !reflect.DeepEqual(got, []string{"custom"}) {
		t.Fatalf("Clone() sparse paths = %#v, want [custom]", got)
	}
	if result.Repositories[0].Skills[0].Path != "custom/beta" {
		t.Fatalf("skill path = %q, want custom/beta", result.Repositories[0].Skills[0].Path)
	}
}

func TestServiceInstallInstallsMatchedSkill(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{
		"alpha": "# Alpha\n",
		"beta":  "# Beta\n",
	})
	targetRoot := t.TempDir()
	registrySvc := &fakeRegistry{resolved: []config.Repository{{
		Name:     "official",
		URL:      "https://github.com/org/skills",
		SkillDir: "skills",
	}}}
	cloner := &fakeCloner{worktrees: []repo.Worktree{{Dir: worktree}}}
	service := Service{Registry: registrySvc, Cloner: cloner, Installer: install.Installer{}}

	result, err := service.Install(context.Background(), InstallRequest{
		SkillName:  "alpha",
		TargetRoot: targetRoot,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("Install() result count = %d, want 1", len(result.Results))
	}
	if result.Results[0].Skill.Name != "alpha" || result.Results[0].Copy.Status != install.ResultInstalled {
		t.Fatalf("Install() result = %#v, want installed alpha", result.Results[0])
	}
	assertFileContent(t, filepath.Join(targetRoot, "alpha", "SKILL.md"), "# Alpha\n")
	if _, err := os.Stat(filepath.Join(targetRoot, "beta")); !os.IsNotExist(err) {
		t.Fatalf("Install() installed unselected beta, stat error = %v", err)
	}
	if !reflect.DeepEqual(cloner.cleaned, []repo.Worktree{{Dir: worktree}}) {
		t.Fatalf("Cleanup() worktrees = %#v, want worktree cleanup", cloner.cleaned)
	}
}

func TestServiceInstallRejectsImplicitDuplicateSkillName(t *testing.T) {
	first := createWorktree(t, "skills", map[string]string{"alpha": "# One\n"})
	second := createWorktree(t, "skills", map[string]string{"alpha": "# Two\n"})
	service := Service{
		Registry: &fakeRegistry{resolved: []config.Repository{
			{Name: "one", URL: "https://github.com/org/one", SkillDir: "skills"},
			{Name: "two", URL: "https://github.com/org/two", SkillDir: "skills"},
		}},
		Cloner:    &fakeCloner{worktrees: []repo.Worktree{{Dir: first}, {Dir: second}}},
		Installer: install.Installer{},
	}

	_, err := service.Install(context.Background(), InstallRequest{SkillName: "alpha", TargetRoot: t.TempDir()})
	if !errors.Is(err, ErrAmbiguousSkill) {
		t.Fatalf("Install() error = %v, want ErrAmbiguousSkill", err)
	}
}

func TestServiceInstallUsesCurrentDirectoryDefaultTargetRoot(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{"alpha": "# Alpha\n"})
	cwd := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	service := Service{
		Registry:  &fakeRegistry{resolved: []config.Repository{{Name: "official", URL: "repo", SkillDir: "skills"}}},
		Cloner:    &fakeCloner{worktrees: []repo.Worktree{{Dir: worktree}}},
		Installer: install.Installer{},
	}

	if _, err := service.Install(context.Background(), InstallRequest{SkillName: "alpha"}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	assertFileContent(t, filepath.Join(cwd, ".codex", "skills", "alpha", "SKILL.md"), "# Alpha\n")
}

func TestServiceInstallReturnsNotFoundForMissingSkill(t *testing.T) {
	worktree := createWorktree(t, "skills", map[string]string{"alpha": "# Alpha\n"})
	service := Service{
		Registry:  &fakeRegistry{resolved: []config.Repository{{Name: "official", URL: "repo", SkillDir: "skills"}}},
		Cloner:    &fakeCloner{worktrees: []repo.Worktree{{Dir: worktree}}},
		Installer: install.Installer{},
	}

	_, err := service.Install(context.Background(), InstallRequest{SkillName: "missing", TargetRoot: t.TempDir()})
	if !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("Install() error = %v, want ErrSkillNotFound", err)
	}
}

func TestAppPackageDoesNotImportUIFrameworks(t *testing.T) {
	forbidden := map[string]bool{
		"github.com/spf13/cobra":             true,
		"github.com/charmbracelet/bubbles":   true,
		"github.com/charmbracelet/bubbletea": true,
		"github.com/charmbracelet/lipgloss":  true,
		"github.com/charmbracelet/huh":       true,
	}
	assertForbiddenImports(t, ".", forbidden)
}

func assertForbiddenImports(t *testing.T, packagePath string, forbidden map[string]bool) {
	t.Helper()

	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", packagePath)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list imports failed: %v\n%s", err, out)
	}

	for _, importPath := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if forbidden[importPath] {
			t.Fatalf("app package imports forbidden package %q", importPath)
		}
	}
}

type fakeRegistry struct {
	added        []config.Repository
	removed      []string
	listed       []config.Repository
	resolved     []config.Repository
	resolvedWith []string
	err          error
}

func (r *fakeRegistry) Add(repo config.Repository) error {
	if r.err != nil {
		return r.err
	}
	r.added = append(r.added, repo)
	return nil
}

func (r *fakeRegistry) List() ([]config.Repository, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeRegistry) Remove(name string) error {
	if r.err != nil {
		return r.err
	}
	r.removed = append(r.removed, name)
	return nil
}

func (r *fakeRegistry) Resolve(explicitRepo string) ([]config.Repository, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.resolvedWith = append(r.resolvedWith, explicitRepo)
	if explicitRepo != "" && len(r.resolved) == 0 {
		return []config.Repository{{Name: explicitRepo, URL: explicitRepo, SkillDir: "skills"}}, nil
	}
	if len(r.resolved) == 0 {
		return nil, registry.ErrEmptyRegistry
	}
	return r.resolved, nil
}

type fakeCloner struct {
	worktrees    []repo.Worktree
	cloneOptions []repo.CloneOptions
	cleaned      []repo.Worktree
	cloneErr     error
	cleanupErr   error
}

func (c *fakeCloner) Clone(ctx context.Context, repoURL string, options repo.CloneOptions) (repo.Worktree, error) {
	if c.cloneErr != nil {
		return repo.Worktree{}, c.cloneErr
	}
	c.cloneOptions = append(c.cloneOptions, options)
	if len(c.worktrees) == 0 {
		return repo.Worktree{}, errors.New("missing fake worktree")
	}
	worktree := c.worktrees[0]
	c.worktrees = c.worktrees[1:]
	return worktree, nil
}

func (c *fakeCloner) Cleanup(worktree repo.Worktree) error {
	c.cleaned = append(c.cleaned, worktree)
	return c.cleanupErr
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
