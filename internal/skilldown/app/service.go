// Package app 编排 registry、repo、skill 和 install 等底层模块。
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/finger/skill-down/internal/skilldown/config"
	"github.com/finger/skill-down/internal/skilldown/install"
	"github.com/finger/skill-down/internal/skilldown/repo"
	"github.com/finger/skill-down/internal/skilldown/skill"
)

const defaultSkillDir = "skills"

var (
	// ErrSkillNotFound 表示安装请求没有匹配到目标 skill。
	ErrSkillNotFound = errors.New("skill not found")
	// ErrAmbiguousSkill 表示隐式 registry 安装时多个仓库包含同名 skill。
	ErrAmbiguousSkill = errors.New("ambiguous skill name")
	// ErrInstallFailed 表示至少一个目录复制操作失败。
	ErrInstallFailed = errors.New("install failed")
)

// RegistryService 表示 app 依赖的 registry 用例能力。
type RegistryService interface {
	// Add 添加一个已注册仓库。
	Add(config.Repository) error
	// List 返回已注册仓库列表。
	List() ([]config.Repository, error)
	// Remove 删除指定名称的已注册仓库。
	Remove(name string) error
	// Resolve 解析显式仓库或 registry 仓库列表。
	Resolve(explicitRepo string) ([]config.Repository, error)
}

// Service 负责执行应用用例编排。
type Service struct {
	// Registry 提供 registry 添加、查看、删除和解析能力。
	Registry RegistryService
	// Cloner 提供远程仓库临时 clone 和 cleanup 能力。
	Cloner repo.Cloner
	// Installer 提供 skill 目录复制能力。
	Installer install.Installer
}

// AddRepositoryRequest 表示添加 registry 仓库的结构化请求。
type AddRepositoryRequest struct {
	// Name 是本地 registry 名称。
	Name string
	// URL 是 Git 仓库地址。
	URL string
	// SkillDirPath 是仓库内承载 skill 子目录的相对路径。
	SkillDirPath string
}

// RemoveRepositoryRequest 表示删除 registry 仓库的结构化请求。
type RemoveRepositoryRequest struct {
	// Name 是要删除的 registry 名称。
	Name string
}

// SearchRequest 表示搜索 skill 的结构化请求。
type SearchRequest struct {
	// RepoURL 是显式指定的 Git 仓库地址；为空时使用 registry。
	RepoURL string
	// SkillDirPath 是本次搜索覆盖 registry 配置的 skill 目录路径。
	SkillDirPath string
}

// InstallRequest 表示安装 skill 的结构化请求。
type InstallRequest struct {
	// RepoURL 是显式指定的 Git 仓库地址；为空时使用 registry。
	RepoURL string
	// SkillName 是要安装的 skill 名称；为空时安装所有发现的 skill。
	SkillName string
	// SkillDirPath 是本次安装覆盖 registry 配置的 skill 目录路径。
	SkillDirPath string
	// TargetRoot 是安装根目录；为空时使用当前目录下的 .codex/skills。
	TargetRoot string
	// Force 表示是否覆盖已存在的目标目录。
	Force bool
}

// SkillResult 表示一个可展示或安装的 skill。
type SkillResult struct {
	// Name 是 skill 的安全名称。
	Name string
	// Path 是相对 worktree 根目录的 slash 风格 skill 目录路径。
	Path string
	// Description 是 SKILL.md front matter 中的描述。
	Description string
	// Content 是完整 SKILL.md 文件内容。
	Content string
}

// RepositorySearchResult 表示单个仓库的搜索结果。
type RepositorySearchResult struct {
	// Repository 是本次搜索的仓库配置。
	Repository config.Repository
	// Skills 是该仓库中发现的 skill 列表。
	Skills []SkillResult
}

// SearchResult 表示一次搜索请求的结构化结果。
type SearchResult struct {
	// Repositories 是按仓库分组的搜索结果。
	Repositories []RepositorySearchResult
}

// InstallItemResult 表示一个 skill 的安装结果。
type InstallItemResult struct {
	// Repository 是 skill 来源仓库。
	Repository config.Repository
	// Skill 是被安装的 skill。
	Skill SkillResult
	// Copy 是底层目录复制结果。
	Copy install.Result
}

// InstallResult 表示一次安装请求的结构化结果。
type InstallResult struct {
	// Results 是每个已尝试安装的 skill 结果。
	Results []InstallItemResult
}

// AddRepository 添加一个 registry 仓库。
func (s Service) AddRepository(ctx context.Context, req AddRepositoryRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.Registry.Add(config.Repository{
		Name:     req.Name,
		URL:      req.URL,
		SkillDir: req.SkillDirPath,
	})
}

// ListRepositories 返回已注册仓库列表。
func (s Service) ListRepositories(ctx context.Context) ([]config.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Registry.List()
}

// RemoveRepository 删除指定 registry 仓库。
func (s Service) RemoveRepository(ctx context.Context, req RemoveRepositoryRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.Registry.Remove(req.Name)
}

// Search 搜索显式仓库或 registry 仓库中的 skill。
func (s Service) Search(ctx context.Context, req SearchRequest) (SearchResult, error) {
	repositories, err := s.Registry.Resolve(req.RepoURL)
	if err != nil {
		return SearchResult{}, err
	}

	result := SearchResult{Repositories: make([]RepositorySearchResult, 0, len(repositories))}
	for _, repository := range repositories {
		group, err := s.searchRepository(ctx, repository, req.SkillDirPath)
		if err != nil {
			return SearchResult{}, err
		}
		result.Repositories = append(result.Repositories, group)
	}
	return result, nil
}

// Install 安装显式仓库或 registry 仓库中的 skill。
func (s Service) Install(ctx context.Context, req InstallRequest) (InstallResult, error) {
	repositories, err := s.Registry.Resolve(req.RepoURL)
	if err != nil {
		return InstallResult{}, err
	}

	targetRoot, err := defaultTargetRoot(req.TargetRoot)
	if err != nil {
		return InstallResult{}, err
	}

	candidates, cleanup, err := s.installCandidates(ctx, repositories, req)
	if err != nil {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return InstallResult{}, cleanupErr
		}
		return InstallResult{}, err
	}

	selected := selectCandidates(candidates, req.SkillName)
	if len(selected) == 0 {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return InstallResult{}, cleanupErr
		}
		return InstallResult{}, fmt.Errorf("%w: %s", ErrSkillNotFound, req.SkillName)
	}
	if req.RepoURL == "" && hasDuplicateSkillName(selected) {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return InstallResult{}, cleanupErr
		}
		return InstallResult{}, ErrAmbiguousSkill
	}

	result := InstallResult{Results: make([]InstallItemResult, 0, len(selected))}
	var installErr error
	for _, candidate := range selected {
		copyResult := s.Installer.CopyDir(
			ctx,
			candidate.sourceDir,
			filepath.Join(targetRoot, candidate.skill.Name),
			req.Force,
		)
		item := InstallItemResult{
			Repository: candidate.repository,
			Skill:      candidate.skill,
			Copy:       copyResult,
		}
		result.Results = append(result.Results, item)
		if copyResult.Err != nil && installErr == nil {
			installErr = fmt.Errorf("%w: %s: %v", ErrInstallFailed, candidate.skill.Name, copyResult.Err)
		}
	}
	if cleanupErr := cleanup(); cleanupErr != nil && installErr == nil {
		installErr = cleanupErr
	}
	return result, installErr
}

func (s Service) searchRepository(ctx context.Context, repository config.Repository, skillDirPath string) (RepositorySearchResult, error) {
	if err := ctx.Err(); err != nil {
		return RepositorySearchResult{}, err
	}

	dirPath := effectiveSkillDir(repository, skillDirPath)
	worktree, err := s.Cloner.Clone(ctx, repository.URL, repo.CloneOptions{
		SparsePaths: []string{dirPath},
	})
	if err != nil {
		return RepositorySearchResult{}, err
	}

	skills, discoverErr := skill.Discover(worktree.Dir, dirPath)
	cleanupErr := s.Cloner.Cleanup(worktree)
	if discoverErr != nil {
		return RepositorySearchResult{}, discoverErr
	}
	if cleanupErr != nil {
		return RepositorySearchResult{}, cleanupErr
	}

	return RepositorySearchResult{
		Repository: repository,
		Skills:     skillResults(skills),
	}, nil
}

type installCandidate struct {
	repository config.Repository
	skill      SkillResult
	sourceDir  string
}

func (s Service) installCandidates(ctx context.Context, repositories []config.Repository, req InstallRequest) ([]installCandidate, func() error, error) {
	worktrees := make([]repo.Worktree, 0, len(repositories))
	candidates := []installCandidate{}

	cleanup := func() error {
		var cleanupErr error
		for _, worktree := range worktrees {
			if err := s.Cloner.Cleanup(worktree); err != nil && cleanupErr == nil {
				cleanupErr = err
			}
		}
		return cleanupErr
	}

	for _, repository := range repositories {
		if err := ctx.Err(); err != nil {
			return nil, cleanup, err
		}

		dirPath := effectiveSkillDir(repository, req.SkillDirPath)
		worktree, err := s.Cloner.Clone(ctx, repository.URL, repo.CloneOptions{
			SparsePaths: []string{dirPath},
		})
		if err != nil {
			return nil, cleanup, err
		}
		worktrees = append(worktrees, worktree)

		discovered, err := skill.Discover(worktree.Dir, dirPath)
		if err != nil {
			return nil, cleanup, err
		}
		for _, found := range skillResults(discovered) {
			candidates = append(candidates, installCandidate{
				repository: repository,
				skill:      found,
				sourceDir:  filepath.Join(worktree.Dir, filepath.FromSlash(found.Path)),
			})
		}
	}
	return candidates, cleanup, nil
}

func skillResults(skills []skill.Skill) []SkillResult {
	results := make([]SkillResult, 0, len(skills))
	for _, found := range skills {
		results = append(results, SkillResult{
			Name:        found.Name,
			Path:        found.Path,
			Description: found.Description,
			Content:     found.Content,
		})
	}
	return results
}

func selectCandidates(candidates []installCandidate, skillName string) []installCandidate {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return candidates
	}

	selected := []installCandidate{}
	for _, candidate := range candidates {
		if candidate.skill.Name == skillName {
			selected = append(selected, candidate)
		}
	}
	return selected
}

func hasDuplicateSkillName(candidates []installCandidate) bool {
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if seen[candidate.skill.Name] {
			return true
		}
		seen[candidate.skill.Name] = true
	}
	return false
}

func effectiveSkillDir(repository config.Repository, override string) string {
	dirPath := strings.TrimSpace(override)
	if dirPath == "" {
		dirPath = strings.TrimSpace(repository.SkillDir)
	}
	if dirPath == "" {
		return defaultSkillDir
	}

	dirPath = filepath.ToSlash(dirPath)
	dirPath = strings.TrimLeft(dirPath, "/")
	if dirPath == "" {
		return defaultSkillDir
	}
	return path.Clean(dirPath)
}

func defaultTargetRoot(targetRoot string) (string, error) {
	targetRoot = strings.TrimSpace(targetRoot)
	if targetRoot != "" {
		return targetRoot, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".codex", "skills"), nil
}
