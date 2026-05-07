// Package registry 管理配置中的远程仓库书签列表。
package registry

import (
	"errors"
	"strings"

	"github.com/finger/repo-pick/internal/repopick/config"
)

var (
	// ErrEmptyName 表示 registry name 为空。
	ErrEmptyName = errors.New("registry repository name is required")
	// ErrEmptyURL 表示 repo URL 为空。
	ErrEmptyURL = errors.New("registry repository URL is required")
	// ErrDuplicateName 表示 registry 中已存在同名仓库。
	ErrDuplicateName = errors.New("registry repository name already exists")
	// ErrDuplicateURLBranch 表示 registry 中已存在相同 URL 和分支的仓库。
	ErrDuplicateURLBranch = errors.New("registry repository URL and branch already exists")
	// ErrNotFound 表示 registry 中不存在指定仓库。
	ErrNotFound = errors.New("registry repository not found")
)

// Service 在 config.Store 之上提供 registry 业务规则。
type Service struct {
	// store 是 registry 持久化依赖。
	store config.Store
}

// NewService 创建 registry 服务。
func NewService(store config.Store) Service {
	return Service{store: store}
}

// Add 添加一个已注册仓库，并持久化到配置文件。
func (s Service) Add(repo config.Repository) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}

	repo = normalizeRepository(repo)
	if repo.Name == "" {
		return ErrEmptyName
	}
	if repo.URL == "" {
		return ErrEmptyURL
	}

	for _, existing := range cfg.Repositories {
		existing = normalizeRepository(existing)
		if existing.Name == repo.Name {
			return ErrDuplicateName
		}
		if existing.URL == repo.URL && existing.Branch == repo.Branch {
			return ErrDuplicateURLBranch
		}
	}

	cfg.Repositories = append(cfg.Repositories, repo)
	return s.store.Save(cfg)
}

// List 返回已注册仓库列表。
func (s Service) List() ([]config.Repository, error) {
	cfg, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	repositories := make([]config.Repository, 0, len(cfg.Repositories))
	for _, repo := range cfg.Repositories {
		repositories = append(repositories, normalizeRepository(repo))
	}
	return repositories, nil
}

// Update 更新指定名称的已注册仓库，并持久化到配置文件。
func (s Service) Update(name string, repo config.Repository) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	repo = normalizeRepository(repo)
	if repo.Name == "" {
		return ErrEmptyName
	}
	if repo.URL == "" {
		return ErrEmptyURL
	}

	foundIndex := -1
	for i, existing := range cfg.Repositories {
		if strings.TrimSpace(existing.Name) == name {
			foundIndex = i
			break
		}
	}
	if foundIndex < 0 {
		return ErrNotFound
	}

	for i, existing := range cfg.Repositories {
		if i == foundIndex {
			continue
		}
		existing = normalizeRepository(existing)
		if existing.Name == repo.Name {
			return ErrDuplicateName
		}
		if existing.URL == repo.URL && existing.Branch == repo.Branch {
			return ErrDuplicateURLBranch
		}
	}

	cfg.Repositories[foundIndex] = repo
	return s.store.Save(cfg)
}

// Remove 删除指定名称的已注册仓库，并持久化到配置文件。
func (s Service) Remove(name string) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	next := cfg.Repositories[:0]
	found := false
	for _, repo := range cfg.Repositories {
		if strings.TrimSpace(repo.Name) == name {
			found = true
			continue
		}
		next = append(next, normalizeRepository(repo))
	}
	if !found {
		return ErrNotFound
	}

	cfg.Repositories = next
	return s.store.Save(cfg)
}

// normalizeRepository 去掉 registry 仓库配置中的首尾空白。
func normalizeRepository(repo config.Repository) config.Repository {
	repo.Name = strings.TrimSpace(repo.Name)
	repo.URL = strings.TrimSpace(repo.URL)
	repo.Branch = strings.TrimSpace(repo.Branch)
	repo.LastUpdatedAt = strings.TrimSpace(repo.LastUpdatedAt)
	return repo
}
