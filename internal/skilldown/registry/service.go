// Package registry 管理配置中的已注册 skill 仓库列表。
package registry

import (
	"errors"
	"strings"

	"github.com/finger/skill-down/internal/skilldown/config"
)

const defaultSkillDir = "skills"

var (
	// ErrDuplicateName 表示 registry 中已存在同名仓库。
	ErrDuplicateName = errors.New("registry repository name already exists")
	// ErrNotFound 表示 registry 中不存在指定仓库。
	ErrNotFound = errors.New("registry repository not found")
	// ErrEmptyRegistry 表示未显式传入 repo 且 registry 为空。
	ErrEmptyRegistry = errors.New("registry is empty")
)

// Service 在 config.Store 之上提供 registry 业务规则。
type Service struct {
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
	for _, existing := range cfg.Repositories {
		if existing.Name == repo.Name {
			return ErrDuplicateName
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
	return cfg.Repositories, nil
}

// Remove 删除指定名称的已注册仓库，并持久化到配置文件。
func (s Service) Remove(name string) error {
	cfg, err := s.store.Load()
	if err != nil {
		return err
	}

	next := cfg.Repositories[:0]
	found := false
	for _, repo := range cfg.Repositories {
		if repo.Name == name {
			found = true
			continue
		}
		next = append(next, repo)
	}
	if !found {
		return ErrNotFound
	}

	cfg.Repositories = next
	return s.store.Save(cfg)
}

// Resolve 解析本次命令要使用的仓库。显式 repo 优先，不读取配置。
func (s Service) Resolve(explicitRepo string) ([]config.Repository, error) {
	explicitRepo = strings.TrimSpace(explicitRepo)
	if explicitRepo != "" {
		return []config.Repository{normalizeRepository(config.Repository{
			Name: explicitRepo,
			URL:  explicitRepo,
		})}, nil
	}

	repos, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, ErrEmptyRegistry
	}
	return repos, nil
}

func normalizeRepository(repo config.Repository) config.Repository {
	repo.Name = strings.TrimSpace(repo.Name)
	repo.URL = strings.TrimSpace(repo.URL)
	repo.SkillDir = strings.TrimSpace(repo.SkillDir)
	if repo.SkillDir == "" {
		repo.SkillDir = defaultSkillDir
	}
	return repo
}
