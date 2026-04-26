package install

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCopyDirCopiesSourceDirectoryToMissingTarget 验证目标不存在时会完整复制源目录。
func TestCopyDirCopiesSourceDirectoryToMissingTarget(t *testing.T) {
	sourceDir := createSourceDir(t)
	targetDir := filepath.Join(t.TempDir(), "nested", "target")

	result := Installer{}.CopyDir(context.Background(), sourceDir, targetDir, false)

	if result.Status != ResultInstalled {
		t.Fatalf("CopyDir() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	if result.SourceDir != sourceDir || result.TargetDir != targetDir {
		t.Fatalf("CopyDir() result paths = (%q, %q), want (%q, %q)", result.SourceDir, result.TargetDir, sourceDir, targetDir)
	}
	assertFileContent(t, filepath.Join(targetDir, "SKILL.md"), "skill body\n")
	assertFileContent(t, filepath.Join(targetDir, "assets", "note.txt"), "asset body\n")
}

// TestCopyDirRejectsExistingTargetWithoutForce 验证未开启 force 时不会覆盖已有目标目录。
func TestCopyDirRejectsExistingTargetWithoutForce(t *testing.T) {
	sourceDir := createSourceDir(t)
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	existingPath := filepath.Join(targetDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	result := Installer{}.CopyDir(context.Background(), sourceDir, targetDir, false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyDir() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil {
		t.Fatal("CopyDir() err = nil, want target exists error")
	}
	assertFileContent(t, existingPath, "keep me\n")
	if _, err := os.Stat(filepath.Join(targetDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("CopyDir() wrote source content despite force=false, stat error = %v", err)
	}
}

// TestCopyDirForceOnlyReplacesTargetDirectory 验证 force 只替换目标目录本身。
func TestCopyDirForceOnlyReplacesTargetDirectory(t *testing.T) {
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

	result := Installer{}.CopyDir(context.Background(), sourceDir, targetDir, true)

	if result.Status != ResultInstalled {
		t.Fatalf("CopyDir() status = %q, want %q, err = %v", result.Status, ResultInstalled, result.Err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("CopyDir() kept old target content, stat error = %v", err)
	}
	assertFileContent(t, filepath.Join(targetDir, "SKILL.md"), "skill body\n")
	assertFileContent(t, siblingPath, "sibling\n")
}

// TestCopyDirRejectsMissingOrFileSource 验证源路径不存在或不是目录时复制失败。
func TestCopyDirRejectsMissingOrFileSource(t *testing.T) {
	targetRoot := t.TempDir()
	fileSource := filepath.Join(targetRoot, "file-source")
	if err := os.WriteFile(fileSource, []byte("not a dir\n"), 0o644); err != nil {
		t.Fatalf("write file source: %v", err)
	}

	tests := []struct {
		name      string
		sourceDir string
	}{
		{name: "missing", sourceDir: filepath.Join(targetRoot, "missing")},
		{name: "file", sourceDir: fileSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetDir := filepath.Join(targetRoot, "target-"+tt.name)

			result := Installer{}.CopyDir(context.Background(), tt.sourceDir, targetDir, false)

			if result.Status != ResultFailed {
				t.Fatalf("CopyDir() status = %q, want %q", result.Status, ResultFailed)
			}
			if result.Err == nil {
				t.Fatal("CopyDir() err = nil, want source validation error")
			}
			if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
				t.Fatalf("CopyDir() created target for invalid source, stat error = %v", err)
			}
		})
	}
}

// TestCopyDirRejectsInvalidTargetDir 验证空白目标目录参数会失败。
func TestCopyDirRejectsInvalidTargetDir(t *testing.T) {
	result := Installer{}.CopyDir(context.Background(), createSourceDir(t), " \t\n", false)

	if result.Status != ResultFailed {
		t.Fatalf("CopyDir() status = %q, want %q", result.Status, ResultFailed)
	}
	if result.Err == nil {
		t.Fatal("CopyDir() err = nil, want target validation error")
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
		case "github.com/finger/skill-down/internal/skilldown/repo",
			"github.com/finger/skill-down/internal/skilldown/skill",
			"github.com/finger/skill-down/internal/skilldown/app",
			"github.com/finger/skill-down/internal/skilldown/output":
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
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("skill body\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "assets", "note.txt"), []byte("asset body\n"), 0o644); err != nil {
		t.Fatalf("write nested source file: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(sourceDir, "SKILL.md"), 0o600); err != nil {
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
