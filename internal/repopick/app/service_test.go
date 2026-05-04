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

	"github.com/finger/repo-pick/internal/repopick/cache"
	"github.com/finger/repo-pick/internal/repopick/config"
	"github.com/finger/repo-pick/internal/repopick/install"
)

func TestServiceRepositoryRegistryOperations(t *testing.T) {
	registrySvc := &fakeRegistry{listed: []config.Repository{{Name: "official", URL: "repo"}}}
	cacheSvc := &fakeCache{}
	service := Service{Registry: registrySvc, Cache: cacheSvc, Installer: install.Installer{}}

	if err := service.AddRepository(context.Background(), AddRepositoryRequest{Name: "personal", URL: "repo2", Branch: "dev"}); err != nil {
		t.Fatalf("AddRepository() error = %v", err)
	}
	repos, err := service.ListRepositories(context.Background())
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if err := service.RemoveRepository(context.Background(), RemoveRepositoryRequest{Name: "official"}); err != nil {
		t.Fatalf("RemoveRepository() error = %v", err)
	}

	if !reflect.DeepEqual(registrySvc.added, []config.Repository{{Name: "personal", URL: "repo2", Branch: "dev"}}) {
		t.Fatalf("added repositories = %#v", registrySvc.added)
	}
	if !reflect.DeepEqual(repos, registrySvc.listed) {
		t.Fatalf("ListRepositories() = %#v, want %#v", repos, registrySvc.listed)
	}
	if !reflect.DeepEqual(registrySvc.removed, []string{"official"}) {
		t.Fatalf("removed repositories = %#v, want [official]", registrySvc.removed)
	}
	if !reflect.DeepEqual(cacheSvc.deleted, []config.Repository{{Name: "official", URL: "repo"}}) {
		t.Fatalf("deleted caches = %#v, want official cache", cacheSvc.deleted)
	}
}

func TestServiceListRemoteBranchesUsesCache(t *testing.T) {
	cacheSvc := &fakeCache{branches: cache.RemoteBranches{
		Default:  "main",
		Branches: []string{"dev", "main"},
	}}
	service := Service{Registry: &fakeRegistry{}, Cache: cacheSvc, Installer: install.Installer{}}

	result, err := service.ListRemoteBranches(context.Background(), ListRemoteBranchesRequest{URL: "repo"})
	if err != nil {
		t.Fatalf("ListRemoteBranches() error = %v", err)
	}

	if result.Default != "main" || !reflect.DeepEqual(result.Branches, []string{"dev", "main"}) {
		t.Fatalf("ListRemoteBranches() = %#v, want main/dev+main", result)
	}
	if !reflect.DeepEqual(cacheSvc.branchURLs, []string{"repo"}) {
		t.Fatalf("branchURLs = %#v, want repo", cacheSvc.branchURLs)
	}
}

func TestServiceEnsureAndUpdateRepositoryUseCache(t *testing.T) {
	cacheSvc := &fakeCache{worktree: cache.Worktree{Dir: "/tmp/repo"}}
	repo := config.Repository{Name: "official", URL: "repo"}
	service := Service{Registry: &fakeRegistry{}, Cache: cacheSvc, Installer: install.Installer{}}

	ensured, err := service.EnsureRepository(context.Background(), repo)
	if err != nil {
		t.Fatalf("EnsureRepository() error = %v", err)
	}
	updated, err := service.UpdateRepository(context.Background(), repo)
	if err != nil {
		t.Fatalf("UpdateRepository() error = %v", err)
	}

	if ensured.WorktreeDir != "/tmp/repo" || updated.WorktreeDir != "/tmp/repo" {
		t.Fatalf("states = %#v %#v, want cache worktree dir", ensured, updated)
	}
	if len(cacheSvc.ensured) != 1 || len(cacheSvc.updated) != 1 {
		t.Fatalf("cache calls ensure=%#v update=%#v", cacheSvc.ensured, cacheSvc.updated)
	}
}

func TestServiceListAndSearchEntriesUseCachedWorktree(t *testing.T) {
	worktree := createWorktree(t)
	repo := config.Repository{Name: "official", URL: "repo"}
	service := Service{
		Registry:  &fakeRegistry{},
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	listResult, err := service.ListEntries(context.Background(), ListEntriesRequest{Repository: repo})
	if err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
	searchResult, err := service.SearchEntries(context.Background(), SearchEntriesRequest{Repository: repo, Query: "guide"})
	if err != nil {
		t.Fatalf("SearchEntries() error = %v", err)
	}

	if got := listResult.Entries[0].Path; got != "docs" {
		t.Fatalf("ListEntries()[0].Path = %q, want docs", got)
	}
	if len(searchResult.Entries) != 1 || searchResult.Entries[0].Path != "docs/guide.md" {
		t.Fatalf("SearchEntries() = %#v, want docs/guide.md", searchResult.Entries)
	}
}

func TestServiceDownloadEntryCopiesSelectedFile(t *testing.T) {
	worktree := createWorktree(t)
	targetDir := t.TempDir()
	repo := config.Repository{Name: "official", URL: "repo"}
	service := Service{
		Registry:  &fakeRegistry{},
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	result, err := service.DownloadEntry(context.Background(), DownloadEntryRequest{
		Repository: repo,
		Entry:      EntryResult{Name: "guide.md", Path: "docs/guide.md", Type: EntryFile, Size: 6},
		TargetDir:  targetDir,
	})
	if err != nil {
		t.Fatalf("DownloadEntry() error = %v", err)
	}

	if result.Copy.Status != install.ResultInstalled {
		t.Fatalf("Copy status = %q, want installed", result.Copy.Status)
	}
	assertFileContent(t, filepath.Join(targetDir, "guide.md"), "guide\n")
}

// TestServiceDownloadEntryWithProgressReportsCopyProgress 验证下载用例会透出复制进度。
func TestServiceDownloadEntryWithProgressReportsCopyProgress(t *testing.T) {
	worktree := createWorktree(t)
	targetDir := t.TempDir()
	repo := config.Repository{Name: "official", URL: "repo"}
	service := Service{
		Registry:  &fakeRegistry{},
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}
	var events []ProgressEvent

	_, err := service.DownloadEntryWithProgress(context.Background(), DownloadEntryRequest{
		Repository: repo,
		Entry:      EntryResult{Name: "guide.md", Path: "docs/guide.md", Type: EntryFile, Size: 6},
		TargetDir:  targetDir,
	}, func(event ProgressEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("DownloadEntryWithProgress() error = %v", err)
	}

	if len(events) == 0 {
		t.Fatal("progress events empty")
	}
	last := events[len(events)-1]
	if last.Percent != 100 || !strings.Contains(last.Text, "copying guide.md: 100%") {
		t.Fatalf("last progress = %#v, want guide.md 100%% progress", last)
	}
}

func TestServiceDownloadRootUsesRepositoryName(t *testing.T) {
	worktree := createWorktree(t)
	targetDir := t.TempDir()
	repo := config.Repository{Name: "official", URL: "repo"}
	service := Service{
		Registry:  &fakeRegistry{},
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	_, err := service.DownloadEntry(context.Background(), DownloadEntryRequest{
		Repository: repo,
		Entry:      EntryResult{Type: EntryDir},
		TargetDir:  targetDir,
	})
	if err != nil {
		t.Fatalf("DownloadEntry() error = %v", err)
	}

	assertFileContent(t, filepath.Join(targetDir, "official", "README.md"), "readme\n")
}

func TestServiceDownloadEntryRejectsEscapingPath(t *testing.T) {
	service := Service{
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: t.TempDir()}},
		Installer: install.Installer{},
	}

	_, err := service.DownloadEntry(context.Background(), DownloadEntryRequest{
		Repository: config.Repository{Name: "official", URL: "repo"},
		Entry:      EntryResult{Name: "bad", Path: "../bad", Type: EntryFile},
		TargetDir:  t.TempDir(),
	})
	if err == nil {
		t.Fatal("DownloadEntry() error = nil, want escaping path error")
	}
}

func TestServiceDownloadEntryReturnsTargetExists(t *testing.T) {
	worktree := createWorktree(t)
	targetDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write existing: %v", err)
	}
	service := Service{
		Cache:     &fakeCache{worktree: cache.Worktree{Dir: worktree}},
		Installer: install.Installer{},
	}

	_, err := service.DownloadEntry(context.Background(), DownloadEntryRequest{
		Repository: config.Repository{Name: "official", URL: "repo"},
		Entry:      EntryResult{Name: "README.md", Path: "README.md", Type: EntryFile},
		TargetDir:  targetDir,
	})
	if !errors.Is(err, ErrTargetExists) {
		t.Fatalf("DownloadEntry() error = %v, want ErrTargetExists", err)
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
	// added 记录添加过的仓库。
	added []config.Repository
	// removed 记录删除过的仓库名称。
	removed []string
	// listed 是 List 返回的仓库列表。
	listed []config.Repository
	// err 是测试注入的 registry 错误。
	err error
}

// Add 记录添加请求。
func (r *fakeRegistry) Add(repo config.Repository) error {
	if r.err != nil {
		return r.err
	}
	r.added = append(r.added, repo)
	return nil
}

// List 返回测试仓库列表。
func (r *fakeRegistry) List() ([]config.Repository, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

// Remove 记录删除请求。
func (r *fakeRegistry) Remove(name string) error {
	if r.err != nil {
		return r.err
	}
	r.removed = append(r.removed, name)
	return nil
}

type fakeCache struct {
	// worktree 是 Ensure 和 Update 返回的工作区。
	worktree cache.Worktree
	// ensured 记录 Ensure 请求。
	ensured []config.Repository
	// updated 记录 Update 请求。
	updated []config.Repository
	// deleted 记录 Delete 请求。
	deleted []config.Repository
	// branches 是 ListRemoteBranches 返回的分支信息。
	branches cache.RemoteBranches
	// branchURLs 记录 ListRemoteBranches 请求。
	branchURLs []string
	// err 是测试注入的 cache 错误。
	err error
}

// Ensure 记录 ensure 请求并返回测试工作区。
func (c *fakeCache) Ensure(ctx context.Context, repo config.Repository) (cache.Worktree, error) {
	if c.err != nil {
		return cache.Worktree{}, c.err
	}
	c.ensured = append(c.ensured, repo)
	return c.worktree, nil
}

// EnsureWithProgress 记录 ensure 请求并返回测试工作区。
func (c *fakeCache) EnsureWithProgress(ctx context.Context, repo config.Repository, progress cache.ProgressFunc) (cache.Worktree, error) {
	if progress != nil {
		progress(cache.Progress{Text: "Receiving objects: 50%", Percent: 50})
	}
	return c.Ensure(ctx, repo)
}

// Update 记录 update 请求并返回测试工作区。
func (c *fakeCache) Update(ctx context.Context, repo config.Repository) (cache.Worktree, error) {
	if c.err != nil {
		return cache.Worktree{}, c.err
	}
	c.updated = append(c.updated, repo)
	return c.worktree, nil
}

// UpdateWithProgress 记录 update 请求并返回测试工作区。
func (c *fakeCache) UpdateWithProgress(ctx context.Context, repo config.Repository, progress cache.ProgressFunc) (cache.Worktree, error) {
	if progress != nil {
		progress(cache.Progress{Text: "Receiving objects: 50%", Percent: 50})
	}
	return c.Update(ctx, repo)
}

// Delete 记录 cache 删除请求。
func (c *fakeCache) Delete(repo config.Repository) error {
	if c.err != nil {
		return c.err
	}
	c.deleted = append(c.deleted, repo)
	return nil
}

// ListRemoteBranches 记录远端分支查询请求并返回测试分支。
func (c *fakeCache) ListRemoteBranches(ctx context.Context, repoURL string) (cache.RemoteBranches, error) {
	if c.err != nil {
		return cache.RemoteBranches{}, c.err
	}
	c.branchURLs = append(c.branchURLs, repoURL)
	return c.branches, nil
}

func createWorktree(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatalf("write guide: %v", err)
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
