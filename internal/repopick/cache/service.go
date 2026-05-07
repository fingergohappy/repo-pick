// Package cache 管理远程 Git 仓库在本地的 shallow clone 工作区。
package cache

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/finger/repo-pick/internal/repopick/config"
)

// Worktree 表示 cache 中可读取的仓库工作区。
type Worktree struct {
	// Dir 是本地 cache 工作区的完整路径。
	Dir string
	// Created 表示本次调用新生成了本地 cache 工作区。
	Created bool
}

// RemoteBranches 表示远端仓库可选择的分支信息。
type RemoteBranches struct {
	// Default 是远端 HEAD 指向的默认分支名称。
	Default string
	// Branches 是远端 refs/heads 下的全部分支名称。
	Branches []string
}

// Progress 表示 Git clone 输出的一次进度更新。
type Progress struct {
	// Text 是 Git 输出中的进度描述。
	Text string
	// Percent 是解析到的百分比；没有百分比时为 -1。
	Percent int
}

// ProgressFunc 接收 Git clone 进度更新。
type ProgressFunc func(Progress)

// Service 负责 repo cache 的生命周期。
type Service struct {
	// RootDir 是所有 repo cache 的父目录。
	RootDir string
	// GitPath 是 git 可执行文件路径；为空时使用 PATH 中的 git。
	GitPath string
}

// Ensure 确保指定仓库已经 shallow clone 到 cache。
func (s Service) Ensure(ctx context.Context, repo config.Repository) (Worktree, error) {
	return s.EnsureWithProgress(ctx, repo, nil)
}

// EnsureWithProgress 确保指定仓库 cache 存在，并在 clone 时回传进度。
func (s Service) EnsureWithProgress(ctx context.Context, repo config.Repository, progress ProgressFunc) (Worktree, error) {
	if err := ctx.Err(); err != nil {
		return Worktree{}, err
	}
	dir, err := s.repoDir(repo)
	if err != nil {
		return Worktree{}, err
	}

	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return Worktree{}, fmt.Errorf("cache path %q is not a directory", dir)
		}
		return Worktree{Dir: dir}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Worktree{}, fmt.Errorf("stat cache %q: %w", dir, err)
	}

	if err := s.clone(ctx, repo.URL, repo.Branch, dir, progress); err != nil {
		return Worktree{}, err
	}
	return Worktree{Dir: dir, Created: true}, nil
}

// Update 删除旧 cache 并重新 shallow clone 指定仓库。
func (s Service) Update(ctx context.Context, repo config.Repository) (Worktree, error) {
	return s.UpdateWithProgress(ctx, repo, nil)
}

// UpdateWithProgress 删除旧 cache 并重新 shallow clone 指定仓库，同时回传 clone 进度。
func (s Service) UpdateWithProgress(ctx context.Context, repo config.Repository, progress ProgressFunc) (Worktree, error) {
	if err := ctx.Err(); err != nil {
		return Worktree{}, err
	}
	dir, err := s.repoDir(repo)
	if err != nil {
		return Worktree{}, err
	}

	// 更新失败不恢复旧 cache；用户后续可以重新打开或再次更新。
	if err := os.RemoveAll(dir); err != nil {
		return Worktree{}, fmt.Errorf("remove cache %q: %w", dir, err)
	}
	if err := s.clone(ctx, repo.URL, repo.Branch, dir, progress); err != nil {
		return Worktree{}, err
	}
	return Worktree{Dir: dir, Created: true}, nil
}

// Delete 删除指定仓库 URL 和分支对应的 cache 目录。
func (s Service) Delete(repo config.Repository) error {
	dir, err := s.repoDir(repo)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// ListRemoteBranches 查询远端仓库可选分支和默认分支。
func (s Service) ListRemoteBranches(ctx context.Context, repoURL string) (RemoteBranches, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return RemoteBranches{}, errors.New("repo URL is required")
	}

	defaultBranch, err := s.defaultBranch(ctx, repoURL)
	if err != nil {
		return RemoteBranches{}, err
	}
	branches, err := s.remoteBranches(ctx, repoURL)
	if err != nil {
		return RemoteBranches{}, err
	}
	return RemoteBranches{Default: defaultBranch, Branches: branches}, nil
}

// clone 将远程仓库 shallow clone 到指定 cache 目录。
func (s Service) clone(ctx context.Context, repoURL string, branch string, dir string, progress ProgressFunc) error {
	if strings.TrimSpace(repoURL) == "" {
		return errors.New("repo URL is required")
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("create cache root: %w", err)
	}

	args := []string{"clone", "--progress", "--depth", "1", "--single-branch"}
	if branch = strings.TrimSpace(branch); branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL, dir)
	if err := s.runProgress(ctx, "", args, progress); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	return nil
}

// repoDir 返回指定仓库 URL 和可选分支对应的稳定 cache 目录。
func (s Service) repoDir(repo config.Repository) (string, error) {
	repo.URL = strings.TrimSpace(repo.URL)
	if repo.URL == "" {
		return "", errors.New("repo URL is required")
	}
	repo.Branch = strings.TrimSpace(repo.Branch)
	return filepath.Join(s.rootDir(), hashRepository(repo)), nil
}

// rootDir 返回 cache 父目录，未显式配置时使用系统用户 cache 目录。
func (s Service) rootDir() string {
	if strings.TrimSpace(s.RootDir) != "" {
		return s.RootDir
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "repo-pick", "repos")
	}
	return filepath.Join(cacheDir, "repo-pick", "repos")
}

// hashURL 返回 repo URL 的稳定哈希字符串。
func hashURL(repoURL string) string {
	sum := sha256.Sum256([]byte(repoURL))
	return hex.EncodeToString(sum[:])
}

// hashRepository 返回 repo URL 和可选分支对应的稳定哈希字符串。
func hashRepository(repo config.Repository) string {
	if repo.Branch == "" {
		return hashURL(repo.URL)
	}
	sum := sha256.Sum256([]byte(repo.URL + "\n" + repo.Branch))
	return hex.EncodeToString(sum[:])
}

// defaultBranch 查询远端 HEAD 指向的默认分支名称。
func (s Service) defaultBranch(ctx context.Context, repoURL string) (string, error) {
	out, err := s.output(ctx, "", "ls-remote", "--symref", repoURL, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ref:") || !strings.HasSuffix(line, "\tHEAD") {
			continue
		}
		ref := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "ref:"), "\tHEAD"))
		return strings.TrimPrefix(ref, "refs/heads/"), nil
	}
	return "", nil
}

// remoteBranches 查询远端 refs/heads 下的全部分支名称。
func (s Service) remoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	out, err := s.output(ctx, "", "ls-remote", "--heads", repoURL)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		branch, ok := parseHeadRef(line)
		if !ok || seen[branch] {
			continue
		}
		seen[branch] = true
		branches = append(branches, branch)
	}
	sort.Strings(branches)
	return branches, nil
}

// parseHeadRef 从 git ls-remote --heads 的单行输出中解析分支名称。
func parseHeadRef(line string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
	if len(parts) != 2 {
		return "", false
	}
	ref := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(ref, "refs/heads/") {
		return "", false
	}
	branch := strings.TrimPrefix(ref, "refs/heads/")
	return branch, branch != ""
}

// run 执行 git 命令，并在失败时保留 git 输出。
func (s Service) run(ctx context.Context, dir string, args ...string) error {
	_, err := s.output(ctx, dir, args...)
	return err
}

// output 执行 git 命令，并返回 stderr/stdout 合并输出。
func (s Service) output(ctx context.Context, dir string, args ...string) (string, error) {
	gitPath := s.GitPath
	if gitPath == "" {
		gitPath = "git"
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", commandError{args: append([]string{gitPath}, args...), output: string(out), err: err}
	}
	return string(out), nil
}

// runProgress 执行 git 命令，并从 stderr/stdout 流式解析进度。
func (s Service) runProgress(ctx context.Context, dir string, args []string, progress ProgressFunc) error {
	gitPath := s.GitPath
	if gitPath == "" {
		gitPath = "git"
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
	cmd.Dir = dir
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	var output strings.Builder
	var outputMu sync.Mutex
	appendOutput := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		outputMu.Lock()
		defer outputMu.Unlock()
		output.WriteString(line)
		output.WriteByte('\n')
	}

	if err := cmd.Start(); err != nil {
		return commandError{args: append([]string{gitPath}, args...), err: err}
	}

	var wg sync.WaitGroup
	var readErr error
	var readErrMu sync.Mutex
	recordReadErr := func(err error) {
		if err == nil {
			return
		}
		readErrMu.Lock()
		defer readErrMu.Unlock()
		if readErr == nil {
			readErr = err
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		recordReadErr(scanProgressOutput(stderr, appendOutput, progress))
	}()
	go func() {
		defer wg.Done()
		recordReadErr(scanProgressOutput(stdout, appendOutput, progress))
	}()

	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		outputMu.Lock()
		text := output.String()
		outputMu.Unlock()
		return commandError{args: append([]string{gitPath}, args...), output: text, err: err}
	}
	if readErr != nil {
		return readErr
	}
	return nil
}

var progressPercentRegexp = regexp.MustCompile(`(\d{1,3})%`)

// scanProgressOutput 按 Git 进度输出中的换行或回车拆分并回调。
func scanProgressOutput(reader io.Reader, appendOutput func(string), progress ProgressFunc) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(splitGitProgress)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		appendOutput(line)
		if progress != nil {
			progress(parseProgress(line))
		}
	}
	return scanner.Err()
}

// splitGitProgress 按 '\r' 或 '\n' 拆分 Git 进度输出。
func splitGitProgress(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// parseProgress 从 Git 输出行中提取进度文本和百分比。
func parseProgress(line string) Progress {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "remote:"))
	percent := -1
	if match := progressPercentRegexp.FindStringSubmatch(line); len(match) == 2 {
		if _, err := fmt.Sscanf(match[1], "%d", &percent); err != nil {
			percent = -1
		}
	}
	return Progress{Text: line, Percent: percent}
}

type commandError struct {
	// args 是本次执行的 git 命令参数。
	args []string
	// output 是 git 输出的 stderr/stdout 合并内容。
	output string
	// err 是 exec 返回的底层错误。
	err error
}

// Error 返回包含 git 命令和输出的错误文本。
func (e commandError) Error() string {
	output := strings.TrimSpace(e.output)
	if output == "" {
		return fmt.Sprintf("%s: %v", strings.Join(e.args, " "), e.err)
	}
	return fmt.Sprintf("%s: %v\n%s", strings.Join(e.args, " "), e.err, output)
}

// Unwrap 返回底层 exec 错误。
func (e commandError) Unwrap() error {
	return e.err
}
