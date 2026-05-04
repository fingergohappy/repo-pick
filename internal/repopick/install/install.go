// Package install 负责把本地文件或目录复制到指定目标路径。
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

// ResultStatus 表示单次复制的结果状态。
type ResultStatus string

const (
	// ResultInstalled 表示源条目已经复制到目标路径。
	ResultInstalled ResultStatus = "installed"
	// ResultFailed 表示复制失败，Err 包含失败原因。
	ResultFailed ResultStatus = "failed"
)

var (
	// ErrTargetExists 表示目标路径已经存在且未允许覆盖。
	ErrTargetExists = errors.New("target already exists")
)

// Result 表示单次复制的结果。
type Result struct {
	// SourcePath 是调用方传入的源文件或目录。
	SourcePath string
	// TargetPath 是调用方传入的最终目标路径。
	TargetPath string
	// Status 是本次复制的结果状态。
	Status ResultStatus
	// Err 是复制失败时的错误原因。
	Err error
}

// Progress 表示本地复制过程中的一次进度更新。
type Progress struct {
	// CurrentPath 是当前正在复制的源文件路径。
	CurrentPath string
	// BytesCopied 是已经复制的字节数。
	BytesCopied int64
	// TotalBytes 是本次复制需要处理的总字节数。
	TotalBytes int64
	// Percent 是当前复制百分比。
	Percent int
}

// ProgressFunc 接收本地复制过程中的进度更新。
type ProgressFunc func(Progress)

// Installer 负责把本地源条目复制到最终目标路径。
type Installer struct{}

// CopyEntry 校验源条目和目标路径，并按 force 策略复制文件或目录。
func (i Installer) CopyEntry(ctx context.Context, sourcePath string, targetPath string, force bool) Result {
	return i.CopyEntryWithProgress(ctx, sourcePath, targetPath, force, nil)
}

// CopyEntryWithProgress 校验源条目和目标路径，复制时按字节回传进度。
func (i Installer) CopyEntryWithProgress(ctx context.Context, sourcePath string, targetPath string, force bool, progress ProgressFunc) Result {
	result := Result{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Status:     ResultFailed,
	}

	if err := ctx.Err(); err != nil {
		result.Err = err
		return result
	}
	info, err := validateSourcePath(sourcePath)
	if err != nil {
		result.Err = err
		return result
	}
	if err := validateTargetPath(targetPath); err != nil {
		result.Err = err
		return result
	}
	if err := prepareTargetPath(targetPath, force); err != nil {
		result.Err = err
		return result
	}

	// 先统计总字节数，后续复制回调用它计算百分比。
	totalBytes, err := totalCopyBytes(ctx, sourcePath, info)
	if err != nil {
		result.Err = err
		return result
	}
	copiedBytes := int64(0)
	reportProgress := func(currentPath string, copiedDelta int64) {
		copiedBytes += copiedDelta
		if progress == nil {
			return
		}
		progress(Progress{
			CurrentPath: currentPath,
			BytesCopied: copiedBytes,
			TotalBytes:  totalBytes,
			Percent:     copyPercent(copiedBytes, totalBytes),
		})
	}
	reportProgress(sourcePath, 0)

	if info.IsDir() {
		err = copyDir(ctx, sourcePath, targetPath, reportProgress)
	} else {
		err = copyFileWithParent(ctx, sourcePath, targetPath, info.Mode().Perm(), reportProgress)
	}
	if err != nil {
		result.Err = err
		return result
	}

	remainingBytes := totalBytes - copiedBytes
	if remainingBytes < 0 {
		remainingBytes = 0
	}
	reportProgress(sourcePath, remainingBytes)
	result.Status = ResultInstalled
	return result
}

// validateSourcePath 校验源路径存在且是普通文件或目录。
func validateSourcePath(sourcePath string) (os.FileInfo, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return nil, errors.New("source path is required")
	}

	info, err := os.Lstat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("stat source path %q: %w", sourcePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("source path %q is a symlink", sourcePath)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source path %q is not a regular file or directory", sourcePath)
	}
	return info, nil
}

// validateTargetPath 校验目标路径参数可用于复制。
func validateTargetPath(targetPath string) error {
	if strings.TrimSpace(targetPath) == "" {
		return errors.New("target path is required")
	}
	return nil
}

// prepareTargetPath 按 force 策略处理已存在的目标文件或目录。
func prepareTargetPath(targetPath string, force bool) error {
	_, err := os.Lstat(targetPath)
	if err == nil {
		if !force {
			return fmt.Errorf("%w: %q", ErrTargetExists, targetPath)
		}
		return os.RemoveAll(targetPath)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("stat target path %q: %w", targetPath, err)
}

// totalCopyBytes 统计本次复制需要处理的普通文件总字节数。
func totalCopyBytes(ctx context.Context, sourcePath string, info os.FileInfo) (int64, error) {
	if !info.IsDir() {
		return info.Size(), nil
	}

	var total int64
	err := filepath.WalkDir(sourcePath, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy %q: symlinks are not supported", current)
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("copy %q: only regular files are supported", current)
		}
		total += info.Size()
		return nil
	})
	return total, err
}

// copyDir 递归复制源目录下的子目录和普通文件。
func copyDir(ctx context.Context, sourceDir string, targetDir string, progress func(string, int64)) error {
	return filepath.WalkDir(sourceDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		rel, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy %q: symlinks are not supported", current)
		}
		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("copy %q: only regular files are supported", current)
		}

		return copyFileWithParent(ctx, current, targetPath, info.Mode().Perm(), progress)
	})
}

// copyFileWithParent 确保父目录存在后复制单个文件。
func copyFileWithParent(ctx context.Context, sourcePath string, targetPath string, perm os.FileMode, progress func(string, int64)) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return copyFile(ctx, sourcePath, targetPath, perm, progress)
}

// copyFile 复制单个普通文件内容并保留权限位。
func copyFile(ctx context.Context, sourcePath string, targetPath string, perm os.FileMode, progress func(string, int64)) error {
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

	// 手动按固定 buffer 复制，确保大文件可以持续回传进度。
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := sourceFile.Read(buffer)
		if n > 0 {
			written, writeErr := targetFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("copy %q to %q: %w", sourcePath, targetPath, writeErr)
			}
			if written != n {
				return fmt.Errorf("copy %q to %q: %w", sourcePath, targetPath, io.ErrShortWrite)
			}
			if progress != nil {
				progress(sourcePath, int64(written))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read source file %q: %w", sourcePath, readErr)
		}
	}
	return nil
}

// copyPercent 根据已复制字节和总字节数计算展示百分比。
func copyPercent(copiedBytes int64, totalBytes int64) int {
	if totalBytes <= 0 {
		return 100
	}
	percent := int(copiedBytes * 100 / totalBytes)
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
