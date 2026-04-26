// Package repo 负责把远程 Git 仓库临时 clone 到本机，并管理临时目录生命周期。
package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const tempDirPrefix = "skill-down-"

// Worktree 表示一次临时 clone 的本地仓库副本。
type Worktree struct {
	Dir string
}

// CloneOptions 表示 clone 阶段的 I/O 约束，不表达业务搜索规则。
type CloneOptions struct {
	SparsePaths []string
}

// Cloner 负责把远程仓库 clone 到本机临时目录，并管理该临时目录的清理。
type Cloner interface {
	Clone(ctx context.Context, repoURL string, options CloneOptions) (Worktree, error)
	Cleanup(worktree Worktree) error
}

// GitCloner 使用本机 git 命令实现临时浅 clone。
type GitCloner struct {
	GitPath string
}

// Clone 创建临时目录，执行 shallow clone，并强制开启 sparse checkout。
func (c GitCloner) Clone(ctx context.Context, repoURL string, options CloneOptions) (Worktree, error) {
	if strings.TrimSpace(repoURL) == "" {
		return Worktree{}, errors.New("repo URL is required")
	}

	dir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		return Worktree{}, fmt.Errorf("create temp worktree: %w", err)
	}

	worktree := Worktree{Dir: dir}
	if err := c.cloneInto(ctx, repoURL, dir, true); err != nil {
		if isPartialCloneUnsupported(err) {
			if cleanupErr := os.RemoveAll(dir); cleanupErr != nil {
				return Worktree{}, fmt.Errorf("clone failed and cleanup temp worktree: %v: %w", cleanupErr, err)
			}
			dir, err = os.MkdirTemp("", tempDirPrefix)
			if err != nil {
				return Worktree{}, fmt.Errorf("recreate temp worktree after partial clone fallback: %w", err)
			}
			worktree = Worktree{Dir: dir}
			if err := c.cloneInto(ctx, repoURL, dir, false); err != nil {
				return Worktree{}, cleanupAfterCloneError(dir, err)
			}
		} else {
			return Worktree{}, cleanupAfterCloneError(dir, err)
		}
	}

	if err := c.setSparseCheckout(ctx, worktree.Dir, options.SparsePaths); err != nil {
		return Worktree{}, cleanupAfterCloneError(worktree.Dir, err)
	}

	return worktree, nil
}

// Cleanup 删除 repo 模块创建的临时 worktree，避免误删调用方传入的普通路径。
func (c GitCloner) Cleanup(worktree Worktree) error {
	if err := validateWorktreeDir(worktree.Dir); err != nil {
		return err
	}
	return os.RemoveAll(worktree.Dir)
}

func (c GitCloner) cloneInto(ctx context.Context, repoURL, dir string, usePartialClone bool) error {
	args := []string{"clone", "--depth=1"}
	if usePartialClone {
		args = append(args, "--filter=blob:none")
	}
	args = append(args, "--sparse", repoURL, dir)
	return c.run(ctx, "", args...)
}

func (c GitCloner) setSparseCheckout(ctx context.Context, dir string, sparsePaths []string) error {
	args := append([]string{"sparse-checkout", "set"}, sparsePaths...)
	return c.run(ctx, dir, args...)
}

func (c GitCloner) run(ctx context.Context, dir string, args ...string) error {
	gitPath := c.GitPath
	if gitPath == "" {
		gitPath = "git"
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return commandError{args: append([]string{gitPath}, args...), output: string(out), err: err}
	}
	return nil
}

type commandError struct {
	args   []string
	output string
	err    error
}

func (e commandError) Error() string {
	output := strings.TrimSpace(e.output)
	if output == "" {
		return fmt.Sprintf("%s: %v", strings.Join(e.args, " "), e.err)
	}
	return fmt.Sprintf("%s: %v\n%s", strings.Join(e.args, " "), e.err, output)
}

func (e commandError) Unwrap() error {
	return e.err
}

func isPartialCloneUnsupported(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unknown option") && strings.Contains(msg, "filter")
}

func cleanupAfterCloneError(dir string, cloneErr error) error {
	if cleanupErr := os.RemoveAll(dir); cleanupErr != nil {
		return fmt.Errorf("cleanup temp worktree %q after clone failure: %v: %w", dir, cleanupErr, cloneErr)
	}
	return cloneErr
}

func validateWorktreeDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("worktree dir is required")
	}

	cleanDir := filepath.Clean(dir)
	if filepath.Base(cleanDir) == cleanDir {
		return fmt.Errorf("refuse to cleanup non-absolute worktree dir %q", dir)
	}
	if !strings.HasPrefix(filepath.Base(cleanDir), tempDirPrefix) {
		return fmt.Errorf("refuse to cleanup non skill-down worktree dir %q", dir)
	}
	if filepath.Clean(filepath.Dir(cleanDir)) != filepath.Clean(os.TempDir()) {
		return fmt.Errorf("refuse to cleanup worktree outside temp dir %q", dir)
	}
	return nil
}
