package repo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGitClonerCloneCreatesReadableSparseShallowWorktree(t *testing.T) {
	sourceRepo := createSourceRepo(t)

	cloner := GitCloner{}
	worktree, err := cloner.Clone(context.Background(), sourceRepo, CloneOptions{
		SparsePaths: []string{"skills/alpha"},
	})
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}

	if info, err := os.Stat(worktree.Dir); err != nil || !info.IsDir() {
		t.Fatalf("Worktree.Dir is not a readable directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktree.Dir, "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("sparse path was not checked out: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktree.Dir, "docs", "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("path outside sparse checkout exists or stat failed unexpectedly: %v", err)
	}

	out := runGit(t, worktree.Dir, "rev-list", "--count", "HEAD")
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("clone is not shallow, commit count = %q", strings.TrimSpace(out))
	}

	if err := cloner.Cleanup(worktree); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(worktree.Dir); !os.IsNotExist(err) {
		t.Fatalf("Cleanup() did not remove worktree, stat error = %v", err)
	}
}

func TestGitClonerCloneCleansTempDirOnFailure(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	if runtime.GOOS == "windows" {
		t.Setenv("TEMP", tempRoot)
		t.Setenv("TMP", tempRoot)
	}

	cloner := GitCloner{}
	_, err := cloner.Clone(context.Background(), filepath.Join(tempRoot, "missing-repo"), CloneOptions{
		SparsePaths: []string{"skills"},
	})
	if err == nil {
		t.Fatal("Clone() error = nil, want error")
	}

	matches, globErr := filepath.Glob(filepath.Join(tempRoot, "skill-down-*"))
	if globErr != nil {
		t.Fatalf("glob temp dirs: %v", globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("Clone() left temp dirs after failure: %v", matches)
	}
}

func createSourceRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	copyTestdata(t, filepath.Join("..", "..", "..", "test", "testdata", "source_repo"), repoDir)
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "initial")

	return "file://" + filepath.ToSlash(repoDir)
}

func copyTestdata(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy testdata: %v", err)
	}
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
