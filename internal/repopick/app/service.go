// Package app 编排 registry、cache、tree 和 install 等底层模块。
package app

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/finger/repo-pick/internal/repopick/cache"
	"github.com/finger/repo-pick/internal/repopick/config"
	"github.com/finger/repo-pick/internal/repopick/install"
	"github.com/finger/repo-pick/internal/repopick/tree"
)

var (
	// ErrRepositoryNotFound 表示 registry 中不存在指定仓库。
	ErrRepositoryNotFound = errors.New("repository not found")
	// ErrTargetExists 表示下载目标已经存在且未允许覆盖。
	ErrTargetExists = install.ErrTargetExists
)

// RegistryService 表示 app 依赖的 registry 用例能力。
type RegistryService interface {
	// Add 添加一个已注册仓库。
	Add(config.Repository) error
	// List 返回已注册仓库列表。
	List() ([]config.Repository, error)
	// Update 更新指定名称的已注册仓库。
	Update(name string, repo config.Repository) error
	// Remove 删除指定名称的已注册仓库。
	Remove(name string) error
}

// CacheService 表示 app 依赖的 repo cache 能力。
type CacheService interface {
	// Ensure 确保仓库 cache 存在。
	Ensure(context.Context, config.Repository) (cache.Worktree, error)
	// EnsureWithProgress 确保仓库 cache 存在，并在 clone 时回传进度。
	EnsureWithProgress(context.Context, config.Repository, cache.ProgressFunc) (cache.Worktree, error)
	// Update 删除旧 cache 并重新下载。
	Update(context.Context, config.Repository) (cache.Worktree, error)
	// UpdateWithProgress 删除旧 cache 并重新下载，并回传 clone 进度。
	UpdateWithProgress(context.Context, config.Repository, cache.ProgressFunc) (cache.Worktree, error)
	// Delete 删除仓库 cache。
	Delete(config.Repository) error
	// ListRemoteBranches 查询远端仓库分支。
	ListRemoteBranches(context.Context, string) (cache.RemoteBranches, error)
}

// EntryInstaller 表示 app 依赖的文件或目录复制能力。
type EntryInstaller interface {
	// CopyEntry 复制单个文件或目录。
	CopyEntry(context.Context, string, string, bool) install.Result
	// CopyEntryWithProgress 复制单个文件或目录，并回传复制进度。
	CopyEntryWithProgress(context.Context, string, string, bool, install.ProgressFunc) install.Result
}

// Service 负责执行应用用例编排。
type Service struct {
	// Registry 提供 registry 添加、查看、编辑和删除能力。
	Registry RegistryService
	// Cache 提供远程仓库 cache 生命周期能力。
	Cache CacheService
	// Installer 提供文件和目录复制能力。
	Installer EntryInstaller
}

// AddRepositoryRequest 表示添加 registry 仓库的结构化请求。
type AddRepositoryRequest struct {
	// Name 是本地 registry 名称。
	Name string
	// URL 是 Git 仓库地址。
	URL string
	// Branch 是可选 Git 分支；为空时使用远端默认分支。
	Branch string
}

// EditRepositoryRequest 表示编辑 registry 仓库的结构化请求。
type EditRepositoryRequest struct {
	// Name 是原 registry 名称。
	Name string
	// NewName 是编辑后的本地 registry 名称。
	NewName string
	// URL 是编辑后的 Git 仓库地址。
	URL string
	// Branch 是编辑后的可选 Git 分支；为空时使用远端默认分支。
	Branch string
}

// ListRemoteBranchesRequest 表示查询远端分支的结构化请求。
type ListRemoteBranchesRequest struct {
	// URL 是 Git 仓库地址。
	URL string
}

// ListRemoteBranchesResult 表示远端分支查询结果。
type ListRemoteBranchesResult struct {
	// Default 是远端 HEAD 指向的默认分支名称。
	Default string
	// Branches 是远端 refs/heads 下的全部分支名称。
	Branches []string
}

// ProgressEvent 表示长耗时操作的一次进度事件。
type ProgressEvent struct {
	// Text 是操作进度的展示描述。
	Text string
	// Percent 是解析到的百分比；没有百分比时为 -1。
	Percent int
}

// ProgressFunc 接收长耗时操作的进度事件。
type ProgressFunc func(ProgressEvent)

// RemoveRepositoryRequest 表示删除 registry 仓库的结构化请求。
type RemoveRepositoryRequest struct {
	// Name 是要删除的 registry 名称。
	Name string
}

// RepositoryState 表示当前仓库 cache 的可浏览状态。
type RepositoryState struct {
	// Repository 是本次操作的仓库配置。
	Repository config.Repository
	// WorktreeDir 是本地 cache 工作区目录。
	WorktreeDir string
}

// EntryType 表示仓库条目的类型。
type EntryType = tree.EntryType

const (
	// EntryFile 表示普通文件。
	EntryFile = tree.EntryFile
	// EntryDir 表示目录。
	EntryDir = tree.EntryDir
)

// EntryResult 表示一个可展示或下载的仓库条目。
type EntryResult = tree.Entry

// ListEntriesRequest 表示列出目录条目的结构化请求。
type ListEntriesRequest struct {
	// Repository 是要读取的仓库。
	Repository config.Repository
	// DirPath 是相对仓库根目录的目录路径。
	DirPath string
}

// ListEntriesResult 表示目录条目列表。
type ListEntriesResult struct {
	// Repository 是条目所属仓库。
	Repository config.Repository
	// DirPath 是本次读取的目录路径。
	DirPath string
	// Entries 是目录下的直接子级条目。
	Entries []EntryResult
}

// SearchEntriesRequest 表示搜索仓库路径的结构化请求。
type SearchEntriesRequest struct {
	// Repository 是要搜索的仓库。
	Repository config.Repository
	// Query 是路径搜索关键词。
	Query string
}

// SearchEntriesResult 表示路径搜索结果。
type SearchEntriesResult struct {
	// Repository 是条目所属仓库。
	Repository config.Repository
	// Query 是路径搜索关键词。
	Query string
	// Entries 是匹配到的文件或目录条目。
	Entries []EntryResult
}

// ResolveEntryPathRequest 表示解析仓库条目本地路径的请求。
type ResolveEntryPathRequest struct {
	// Repository 是条目所属仓库。
	Repository config.Repository
	// Entry 是要解析本地路径的文件或目录。
	Entry EntryResult
}

// ResolveEntryPathResult 表示仓库条目在本地 cache 中的路径。
type ResolveEntryPathResult struct {
	// Path 是条目在本地 cache 工作区中的绝对路径。
	Path string
}

// DownloadEntryRequest 表示下载文件或目录的结构化请求。
type DownloadEntryRequest struct {
	// Repository 是条目所属仓库。
	Repository config.Repository
	// Entry 是要下载的文件或目录。
	Entry EntryResult
	// TargetDir 是目标目录，不是最终文件名。
	TargetDir string
	// Force 表示是否覆盖已存在的最终目标。
	Force bool
}

// DownloadEntryResult 表示下载动作结果。
type DownloadEntryResult struct {
	// Entry 是被下载的仓库条目。
	Entry EntryResult
	// Copy 是底层复制结果。
	Copy install.Result
}

// AddRepository 添加一个 registry 仓库。
func (s Service) AddRepository(ctx context.Context, req AddRepositoryRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.Registry.Add(config.Repository{Name: req.Name, URL: req.URL, Branch: req.Branch})
}

// EditRepository 编辑一个 registry 仓库；来源变化时同步删除旧 cache。
func (s Service) EditRepository(ctx context.Context, req EditRepositoryRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	oldRepo, err := s.findRepository(req.Name)
	if err != nil {
		return err
	}
	newRepo := config.Repository{Name: req.NewName, URL: req.URL, Branch: req.Branch}
	if err := s.Registry.Update(req.Name, newRepo); err != nil {
		return err
	}
	if sameRepositorySource(oldRepo, newRepo) {
		return nil
	}
	return s.Cache.Delete(oldRepo)
}

// ListRemoteBranches 返回远端仓库可选择的分支列表。
func (s Service) ListRemoteBranches(ctx context.Context, req ListRemoteBranchesRequest) (ListRemoteBranchesResult, error) {
	if err := ctx.Err(); err != nil {
		return ListRemoteBranchesResult{}, err
	}
	branches, err := s.Cache.ListRemoteBranches(ctx, req.URL)
	if err != nil {
		return ListRemoteBranchesResult{}, err
	}
	return ListRemoteBranchesResult{Default: branches.Default, Branches: branches.Branches}, nil
}

// ListRepositories 返回已注册仓库列表。
func (s Service) ListRepositories(ctx context.Context) ([]config.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Registry.List()
}

// RemoveRepository 删除 registry 记录，并同步删除该仓库 cache。
func (s Service) RemoveRepository(ctx context.Context, req RemoveRepositoryRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo, err := s.findRepository(req.Name)
	if err != nil {
		return err
	}
	if err := s.Registry.Remove(req.Name); err != nil {
		return err
	}
	return s.Cache.Delete(repo)
}

// EnsureRepository 确保仓库 cache 存在并返回本地工作区状态。
func (s Service) EnsureRepository(ctx context.Context, repo config.Repository) (RepositoryState, error) {
	worktree, err := s.Cache.Ensure(ctx, repo)
	if err != nil {
		return RepositoryState{}, err
	}
	if worktree.Created {
		repo, err = s.recordRepositoryUpdatedAt(repo)
		if err != nil {
			return RepositoryState{}, err
		}
	}
	return RepositoryState{Repository: repo, WorktreeDir: worktree.Dir}, nil
}

// EnsureRepositoryWithProgress 确保仓库 cache 存在并回传 clone 进度。
func (s Service) EnsureRepositoryWithProgress(ctx context.Context, repo config.Repository, progress ProgressFunc) (RepositoryState, error) {
	worktree, err := s.Cache.EnsureWithProgress(ctx, repo, cacheProgressFunc(progress))
	if err != nil {
		return RepositoryState{}, err
	}
	if worktree.Created {
		repo, err = s.recordRepositoryUpdatedAt(repo)
		if err != nil {
			return RepositoryState{}, err
		}
	}
	return RepositoryState{Repository: repo, WorktreeDir: worktree.Dir}, nil
}

// UpdateRepository 删除旧 cache 并重新 shallow clone 仓库。
func (s Service) UpdateRepository(ctx context.Context, repo config.Repository) (RepositoryState, error) {
	worktree, err := s.Cache.Update(ctx, repo)
	if err != nil {
		return RepositoryState{}, err
	}
	repo, err = s.recordRepositoryUpdatedAt(repo)
	if err != nil {
		return RepositoryState{}, err
	}
	return RepositoryState{Repository: repo, WorktreeDir: worktree.Dir}, nil
}

// UpdateRepositoryWithProgress 删除旧 cache 并重新 shallow clone 仓库，同时回传 clone 进度。
func (s Service) UpdateRepositoryWithProgress(ctx context.Context, repo config.Repository, progress ProgressFunc) (RepositoryState, error) {
	worktree, err := s.Cache.UpdateWithProgress(ctx, repo, cacheProgressFunc(progress))
	if err != nil {
		return RepositoryState{}, err
	}
	repo, err = s.recordRepositoryUpdatedAt(repo)
	if err != nil {
		return RepositoryState{}, err
	}
	return RepositoryState{Repository: repo, WorktreeDir: worktree.Dir}, nil
}

// ListEntries 列出仓库中指定目录的直接子级条目。
func (s Service) ListEntries(ctx context.Context, req ListEntriesRequest) (ListEntriesResult, error) {
	worktree, err := s.Cache.Ensure(ctx, req.Repository)
	if err != nil {
		return ListEntriesResult{}, err
	}
	entries, err := tree.List(worktree.Dir, req.DirPath)
	if err != nil {
		return ListEntriesResult{}, err
	}
	return ListEntriesResult{Repository: req.Repository, DirPath: cleanRepoPath(req.DirPath), Entries: entries}, nil
}

// SearchEntries 搜索当前仓库的全部路径。
func (s Service) SearchEntries(ctx context.Context, req SearchEntriesRequest) (SearchEntriesResult, error) {
	worktree, err := s.Cache.Ensure(ctx, req.Repository)
	if err != nil {
		return SearchEntriesResult{}, err
	}
	entries, err := tree.Search(worktree.Dir, req.Query)
	if err != nil {
		return SearchEntriesResult{}, err
	}
	return SearchEntriesResult{Repository: req.Repository, Query: strings.TrimSpace(req.Query), Entries: entries}, nil
}

// ResolveEntryPath 返回仓库条目在本地 cache 工作区中的安全路径。
func (s Service) ResolveEntryPath(ctx context.Context, req ResolveEntryPathRequest) (ResolveEntryPathResult, error) {
	worktree, err := s.Cache.Ensure(ctx, req.Repository)
	if err != nil {
		return ResolveEntryPathResult{}, err
	}
	entryPath, err := sourcePathForEntry(worktree.Dir, req.Entry)
	if err != nil {
		return ResolveEntryPathResult{}, err
	}
	return ResolveEntryPathResult{Path: entryPath}, nil
}

// DownloadEntry 将指定文件或目录下载到目标目录下。
func (s Service) DownloadEntry(ctx context.Context, req DownloadEntryRequest) (DownloadEntryResult, error) {
	return s.DownloadEntryWithProgress(ctx, req, nil)
}

// DownloadEntryWithProgress 将指定文件或目录下载到目标目录下，并回传复制进度。
func (s Service) DownloadEntryWithProgress(ctx context.Context, req DownloadEntryRequest, progress ProgressFunc) (DownloadEntryResult, error) {
	worktree, err := s.Cache.Ensure(ctx, req.Repository)
	if err != nil {
		return DownloadEntryResult{}, err
	}

	sourcePath, err := sourcePathForEntry(worktree.Dir, req.Entry)
	if err != nil {
		return DownloadEntryResult{}, err
	}
	targetName := req.Entry.Name
	if strings.TrimSpace(req.Entry.Path) == "" {
		targetName = strings.TrimSpace(req.Repository.Name)
	}
	if targetName == "" {
		return DownloadEntryResult{}, errors.New("target entry name is required")
	}

	targetDir := strings.TrimSpace(req.TargetDir)
	if targetDir == "" {
		return DownloadEntryResult{}, errors.New("target dir is required")
	}
	copyResult := s.Installer.CopyEntryWithProgress(ctx, sourcePath, filepath.Join(targetDir, targetName), req.Force, installProgressFunc(progress, sourcePath, targetName))
	result := DownloadEntryResult{Entry: req.Entry, Copy: copyResult}
	if copyResult.Err != nil {
		return result, copyResult.Err
	}
	return result, nil
}

// findRepository 按 registry 名称查找仓库配置。
func (s Service) findRepository(name string) (config.Repository, error) {
	repositories, err := s.Registry.List()
	if err != nil {
		return config.Repository{}, err
	}

	name = strings.TrimSpace(name)
	for _, repo := range repositories {
		if strings.TrimSpace(repo.Name) == name {
			return repo, nil
		}
	}
	return config.Repository{}, fmt.Errorf("%w: %s", ErrRepositoryNotFound, name)
}

// recordRepositoryUpdatedAt 将仓库 cache 成功更新时间写回 registry 配置。
func (s Service) recordRepositoryUpdatedAt(repo config.Repository) (config.Repository, error) {
	repo.LastUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if s.Registry == nil {
		return repo, nil
	}
	return repo, s.Registry.Update(repo.Name, repo)
}

// sourcePathForEntry 将仓库内 Entry 路径转换为安全的本地源路径。
func sourcePathForEntry(root string, entry EntryResult) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	entryPath := strings.TrimSpace(entry.Path)
	if entryPath == "" {
		return rootAbs, nil
	}

	entryPath = filepath.ToSlash(entryPath)
	for _, part := range strings.Split(entryPath, "/") {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("entry path cannot contain empty, . or .. segments")
		}
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(path.Clean(entryPath)))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("entry path %q escapes root", entry.Path)
	}
	return target, nil
}

// cleanRepoPath 将目录路径规范化为仓库内 slash 风格路径。
func cleanRepoPath(dirPath string) string {
	dirPath = strings.TrimSpace(filepath.ToSlash(dirPath))
	if dirPath == "" || dirPath == "/" || dirPath == "." {
		return ""
	}
	return path.Clean(strings.TrimLeft(dirPath, "/"))
}

// sameRepositorySource 判断两个 registry 是否指向相同的远端来源。
func sameRepositorySource(a config.Repository, b config.Repository) bool {
	return strings.TrimSpace(a.URL) == strings.TrimSpace(b.URL) &&
		strings.TrimSpace(a.Branch) == strings.TrimSpace(b.Branch)
}

// cacheProgressFunc 将 cache 层进度事件转换为 app 层进度事件。
func cacheProgressFunc(progress ProgressFunc) cache.ProgressFunc {
	if progress == nil {
		return nil
	}
	return func(event cache.Progress) {
		progress(ProgressEvent{Text: event.Text, Percent: event.Percent})
	}
}

// installProgressFunc 将 install 层复制进度转换为 app 层进度事件。
func installProgressFunc(progress ProgressFunc, sourceRoot string, rootLabel string) install.ProgressFunc {
	if progress == nil {
		return nil
	}
	return func(event install.Progress) {
		progress(ProgressEvent{
			Text:    downloadProgressText(sourceRoot, rootLabel, event),
			Percent: event.Percent,
		})
	}
}

// downloadProgressText 生成下载进度的展示文本。
func downloadProgressText(sourceRoot string, rootLabel string, event install.Progress) string {
	current := strings.TrimSpace(event.CurrentPath)
	label := filepath.Base(current)
	if rel, err := filepath.Rel(sourceRoot, current); err == nil && rel != "." {
		label = filepath.ToSlash(rel)
	}
	if label == "." || label == string(filepath.Separator) || label == "" {
		label = strings.TrimSpace(rootLabel)
	}
	if label == "" {
		label = filepath.Base(sourceRoot)
	}
	if event.TotalBytes > 0 {
		return fmt.Sprintf("copying %s: %d%% (%d/%d bytes)", label, event.Percent, event.BytesCopied, event.TotalBytes)
	}
	return fmt.Sprintf("copying %s: %d%%", label, event.Percent)
}
