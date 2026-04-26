// Package install 负责把一个本地源目录完整复制到指定目标目录。
package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ResultStatus 表示单次目录复制的结果状态。
type ResultStatus string

const (
	// ResultInstalled 表示源目录已经复制到目标目录。
	ResultInstalled ResultStatus = "installed"
	// ResultFailed 表示复制失败，Err 包含失败原因。
	ResultFailed ResultStatus = "failed"
)

// Result 表示单次目录复制的结果。
type Result struct {
	// SourceDir 是调用方传入的源目录。
	SourceDir string
	// TargetDir 是调用方传入的最终目标目录。
	TargetDir string
	// Status 是本次复制的结果状态。
	Status ResultStatus
	// Err 是复制失败时的错误原因。
	Err error
}

// Installer 负责把本地源目录复制到最终目标目录。
type Installer struct{}

// CopyDir 校验源目录和目标目录，并按 force 策略复制整个源目录。
func (i Installer) CopyDir(ctx context.Context, sourceDir string, targetDir string, force bool) Result {
	result := Result{
		SourceDir: sourceDir,
		TargetDir: targetDir,
		Status:    ResultFailed,
	}

	if err := ctx.Err(); err != nil {
		result.Err = err
		return result
	}
	if err := validateSourceDir(sourceDir); err != nil {
		result.Err = err
		return result
	}
	if err := validateTargetDir(targetDir); err != nil {
		result.Err = err
		return result
	}
	if err := prepareTargetDir(targetDir, force); err != nil {
		result.Err = err
		return result
	}
	if err := copyDir(ctx, sourceDir, targetDir); err != nil {
		result.Err = err
		return result
	}

	result.Status = ResultInstalled
	return result
}

// validateSourceDir 校验源目录存在且确实是目录。
func validateSourceDir(sourceDir string) error {
	if strings.TrimSpace(sourceDir) == "" {
		return errors.New("source dir is required")
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("stat source dir %q: %w", sourceDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source dir %q is not a directory", sourceDir)
	}
	return nil
}

// validateTargetDir 校验目标目录参数可用于复制。
func validateTargetDir(targetDir string) error {
	if strings.TrimSpace(targetDir) == "" {
		return errors.New("target dir is required")
	}
	return nil
}

// prepareTargetDir 按 force 策略处理已存在的目标目录。
func prepareTargetDir(targetDir string, force bool) error {
	_, err := os.Stat(targetDir)
	if err == nil {
		if !force {
			return fmt.Errorf("target dir %q already exists", targetDir)
		}
		return os.RemoveAll(targetDir)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("stat target dir %q: %w", targetDir, err)
}

// copyDir 递归复制源目录下的子目录和普通文件。
func copyDir(ctx context.Context, sourceDir string, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("copy %q: only regular files are supported", path)
		}

		// 普通文件复制前确保父目录存在，避免源目录遍历顺序变化导致写入失败。
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode().Perm())
	})
}

// copyFile 复制单个普通文件内容并保留权限位。
func copyFile(sourcePath string, targetPath string, perm os.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create target file %q: %w", targetPath, err)
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("copy %q to %q: %w", sourcePath, targetPath, err)
	}
	return nil
}
