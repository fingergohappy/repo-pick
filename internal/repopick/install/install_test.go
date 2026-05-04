package install

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCopyEntryCopiesSourceDirectoryToMissingTarget 验证目标不存在时会完整复制源目录。
func TestCopyEntryCopiesSourceDirectoryToMissingTarget(t *testing.T) {
	sourceDir := createSourceDir(t)
	targetDir := filepath.Join(t.TempDir(), "nested", "target")

	result := Installer{}.CopyEntry(context.Background(), sourceDir, targetDir, false)

	if result.Status != ResultInstalled {
		t.Fatalf("CopyEntry() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	if result.SourcePath != sourceDir || result.TargetPath != targetDir {
		t.Fatalf("CopyEntry() result paths = (%q, %q), want (%q, %q)", result.SourcePath, result.TargetPath, sourceDir, targetDir)
	}
	assertFileContent(t, filepath.Join(targetDir, "README.md"), "readme body\n")
	assertFileContent(t, filepath.Join(targetDir, "assets", "note.txt"), "asset body\n")
}

// TestCopyEntryCopiesSourceFileToMissingTarget 验证普通文件可以复制到目标路径。
func TestCopyEntryCopiesSourceFileToMissingTarget(t *testing.T) {
	sourceDir := createSourceDir(t)
	sourcePath := filepath.Join(sourceDir, "README.md")
	targetPath := filepath.Join(t.TempDir(), "out", "README.md")

	result := Installer{}.CopyEntry(context.Background(), sourcePath, targetPath, false)

	if result.Status != ResultInstalled {
		t.Fatalf("CopyEntry() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	assertFileContent(t, targetPath, "readme body\n")
}

// TestCopyEntryWithProgressReportsBytes 验证复制进度会按字节回传并最终到达 100%。
func TestCopyEntryWithProgressReportsBytes(t *testing.T) {
	sourceDir := createSourceDir(t)
	targetDir := filepath.Join(t.TempDir(), "target")
	var events []Progress

	result := Installer{}.CopyEntryWithProgress(context.Background(), sourceDir, targetDir, false, func(event Progress) {
		events = append(events, event)
	})

	if result.Status != ResultInstalled {
		t.Fatalf("CopyEntryWithProgress() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	if len(events) == 0 {
		t.Fatal("progress events empty")
	}
	last := events[len(events)-1]
	if last.Percent != 100 || last.BytesCopied != last.TotalBytes || last.TotalBytes <= 0 {
		t.Fatalf("last progress = %#v, want completed byte progress", last)
	}
}

// TestCopyEntryRejectsExistingTargetWithoutForce 验证未开启 force 时不会覆盖已有目标。
func TestCopyEntryRejectsExistingTargetWithoutForce(t *testing.T) {
	sourceDir := createSourceDir(t)
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	existingPath := filepath.Join(targetDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	result := Installer{}.CopyEntry(context.Background(), sourceDir, targetDir, false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyEntry() status = %q, want %q", result.Status, ResultFailed)
	}
	if !errors.Is(result.Err, ErrTargetExists) {
		t.Fatalf("CopyEntry() err = %v, want ErrTargetExists", result.Err)
	}
	assertFileContent(t, existingPath, "keep me\n")
}

// TestCopyEntryForceOnlyReplacesTarget 验证 force 只替换目标本身。
func TestCopyEntryForceOnlyReplacesTarget(t *testing.T) {
	sourceDir := createSourceDir(t)
	parentDir := t.TempDir()
	targetDir := filepath.Join(parentDir, "target")
	siblingPath := filepath.Join(parentDir, "sibling.txt")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old target file: %v", err)
	}
	if err := os.WriteFile(siblingPath, []byte("sibling\n"), 0o644); err != nil {
		t.Fatalf("write sibling file: %v", err)
	}

	result := Installer{}.CopyEntry(context.Background(), sourceDir, targetDir, true)

	if result.Status != ResultInstalled {
		t.Fatalf("CopyEntry() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("CopyEntry() kept old target content, stat error = %v", err)
	}
	assertFileContent(t, filepath.Join(targetDir, "README.md"), "readme body\n")
	assertFileContent(t, siblingPath, "sibling\n")
}

// TestCopyEntryForceKeepsTargetWhenSourceValidationFails 验证 force 覆盖前会先完整校验源目录。
func TestCopyEntryForceKeepsTargetWhenSourceValidationFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows symlink permissions vary")
	}
	sourceDir := createSourceDir(t)
	if err := os.Symlink("missing", filepath.Join(sourceDir, "bad-link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	existingPath := filepath.Join(targetDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	result := Installer{}.CopyEntry(context.Background(), sourceDir, targetDir, true)

	if result.Status != ResultFailed {
		t.Fatalf("CopyEntry() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil || !strings.Contains(result.Err.Error(), "symlinks are not supported") {
		t.Fatalf("CopyEntry() err = %v, want symlink validation error", result.Err)
	}
	assertFileContent(t, existingPath, "keep me\n")
}

// TestCopyEntryRejectsInvalidSource 验证源路径不存在或不受支持时复制失败。
func TestCopyEntryRejectsInvalidSource(t *testing.T) {
	targetRoot := t.TempDir()
	missingSource := filepath.Join(targetRoot, "missing")

	result := Installer{}.CopyEntry(context.Background(), missingSource, filepath.Join(targetRoot, "target"), false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyEntry() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil {
		t.Fatal("CopyEntry() err = nil, want source validation error")
	}
}

// TestCopyEntryRejectsSymlink 验证符号链接不会被复制。
func TestCopyEntryRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows symlink permissions vary")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	source := filepath.Join(root, "link")
	if err := os.Symlink("missing", source); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	result := Installer{}.CopyEntry(context.Background(), source, target, false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyEntry() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil || !strings.Contains(result.Err.Error(), "symlink") {
		t.Fatalf("CopyEntry() err = %v, want symlink error", result.Err)
	}
}

// TestCopyEntryRejectsInvalidTargetPath 验证空白目标路径参数会失败。
func TestCopyEntryRejectsInvalidTargetPath(t *testing.T) {
	result := Installer{}.CopyEntry(context.Background(), createSourceDir(t), " \t\n", false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyEntry() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil {
		t.Fatal("CopyEntry() err = nil, want target validation error")
	}
}

// TestInstallPackageDoesNotImportAdjacentModules 验证 install 包不依赖相邻业务模块。
func TestInstallPackageDoesNotImportAdjacentModules(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list imports failed: %v\n%s", err, out)
	}

	for _, importPath := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		switch importPath {
		case "github.com/finger/repo-pick/internal/repopick/cache",
			"github.com/finger/repo-pick/internal/repopick/tree",
			"github.com/finger/repo-pick/internal/repopick/app",
			"github.com/finger/repo-pick/internal/repopick/config":
			t.Fatalf("install package imports forbidden package %q", importPath)
		}
	}
}

// createSourceDir 创建包含普通文件和子目录的测试源目录。
func createSourceDir(t *testing.T) string {
	t.Helper()

	sourceDir := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(filepath.Join(sourceDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("readme body\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "assets", "note.txt"), []byte("asset body\n"), 0o644); err != nil {
		t.Fatalf("write nested source file: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(sourceDir, "README.md"), 0o600); err != nil {
			t.Fatalf("chmod source file: %v", err)
		}
	}
	return sourceDir
}

// assertFileContent 校验指定文件内容。
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
