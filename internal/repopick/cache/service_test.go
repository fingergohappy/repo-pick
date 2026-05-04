package cache

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/finger/repo-pick/internal/repopick/config"
)

func TestServiceEnsureClonesFullShallowWorktree(t *testing.T) {
	sourceRepo := createSourceRepo(t)
	service := Service{RootDir: t.TempDir()}
	repo := config.Repository{Name: "fixture", URL: sourceRepo}

	worktree, err := service.Ensure(context.Background(), repo)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	assertFileContent(t, filepath.Join(worktree.Dir, "README.md"), "root readme\n")
	assertFileContent(t, filepath.Join(worktree.Dir, "docs", "guide.md"), "guide\n")
	if count := strings.TrimSpace(runGit(t, worktree.Dir, "rev-list", "--count", "HEAD")); count != "1" {
		t.Fatalf("commit count = %q, want shallow clone with one commit", count)
	}
}

func TestServiceEnsureClonesSelectedBranch(t *testing.T) {
	sourceRepo, _ := createSourceRepoWithBranch(t)
	service := Service{RootDir: t.TempDir()}
	repo := config.Repository{Name: "fixture", URL: sourceRepo, Branch: "feature"}

	worktree, err := service.Ensure(context.Background(), repo)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	assertFileContent(t, filepath.Join(worktree.Dir, "FEATURE.md"), "feature\n")
}

func TestServiceListRemoteBranchesReturnsDefaultAndHeads(t *testing.T) {
	sourceRepo, defaultBranch := createSourceRepoWithBranch(t)
	service := Service{RootDir: t.TempDir()}

	result, err := service.ListRemoteBranches(context.Background(), sourceRepo)
	if err != nil {
		t.Fatalf("ListRemoteBranches() error = %v", err)
	}

	if result.Default != defaultBranch {
		t.Fatalf("Default = %q, want %q", result.Default, defaultBranch)
	}
	if !containsString(result.Branches, defaultBranch) || !containsString(result.Branches, "feature") {
		t.Fatalf("Branches = %#v, want default and feature", result.Branches)
	}
}

// TestParseProgressExtractsGitPercent 验证 Git 进度文本中的百分比可以被解析。
func TestParseProgressExtractsGitPercent(t *testing.T) {
	progress := parseProgress("Receiving objects:  42% (42/100), 1.23 MiB | 2.00 MiB/s")

	if progress.Percent != 42 {
		t.Fatalf("Percent = %d, want 42", progress.Percent)
	}
	if progress.Text != "Receiving objects:  42% (42/100), 1.23 MiB | 2.00 MiB/s" {
		t.Fatalf("Text = %q, want git progress text", progress.Text)
	}
}

func TestServiceEnsureUsesExistingCacheWithoutGit(t *testing.T) {
	root := t.TempDir()
	service := Service{RootDir: root}
	repo := config.Repository{Name: "fixture", URL: "https://example.com/repo.git"}
	dir, err := service.repoDir(repo)
	if err != nil {
		t.Fatalf("repoDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	worktree, err := Service{RootDir: root, GitPath: "missing-git"}.Ensure(context.Background(), repo)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if worktree.Dir != dir {
		t.Fatalf("Worktree.Dir = %q, want %q", worktree.Dir, dir)
	}
}

func TestServiceUpdateDeletesAndClonesAgain(t *testing.T) {
	sourceRepo := createSourceRepo(t)
	service := Service{RootDir: t.TempDir()}
	repo := config.Repository{Name: "fixture", URL: sourceRepo}

	worktree, err := service.Ensure(context.Background(), repo)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Dir, "local.txt"), []byte("remove me\n"), 0o644); err != nil {
		t.Fatalf("write local cache file: %v", err)
	}

	updated, err := service.Update(context.Background(), repo)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Dir != worktree.Dir {
		t.Fatalf("Update() dir = %q, want same cache dir %q", updated.Dir, worktree.Dir)
	}
	if _, err := os.Stat(filepath.Join(updated.Dir, "local.txt")); !os.IsNotExist(err) {
		t.Fatalf("Update() kept old cache content, stat error = %v", err)
	}
}

func TestServiceUpdateFailureLeavesMissingCache(t *testing.T) {
	service := Service{RootDir: t.TempDir(), GitPath: "missing-git"}
	repo := config.Repository{Name: "fixture", URL: "https://example.com/repo.git"}
	dir, err := service.repoDir(repo)
	if err != nil {
		t.Fatalf("repoDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	_, err = service.Update(context.Background(), repo)
	if err == nil {
		t.Fatal("Update() error = nil, want git error")
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("Update() left cache dir after failure, stat error = %v", statErr)
	}
}

func TestServiceDeleteRemovesCache(t *testing.T) {
	service := Service{RootDir: t.TempDir()}
	repo := config.Repository{Name: "fixture", URL: "https://example.com/repo.git"}
	dir, err := service.repoDir(repo)
	if err != nil {
		t.Fatalf("repoDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	if err := service.Delete(repo); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("Delete() did not remove cache, stat error = %v", err)
	}
}

func createSourceRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("root readme\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatalf("write guide: %v", err)
	}
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(repoDir, "docs", "second.md"), []byte("second\n"), 0o644); err != nil {
		t.Fatalf("write second: %v", err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "second")

	return "file://" + filepath.ToSlash(repoDir)
}

func createSourceRepoWithBranch(t *testing.T) (string, string) {
	t.Helper()

	repoURL := createSourceRepo(t)
	repoDir := strings.TrimPrefix(repoURL, "file://")
	defaultBranch := strings.TrimSpace(runGit(t, repoDir, "branch", "--show-current"))
	runGit(t, repoDir, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repoDir, "FEATURE.md"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "feature")
	runGit(t, repoDir, "checkout", defaultBranch)
	return repoURL, defaultBranch
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
