// Package config 负责解析 repo-pick 的用户级配置文件。
package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 表示用户级 config.yaml 配置。
type Config struct {
	// Repositories 是用户登记的远程仓库书签列表。
	Repositories []Repository `yaml:"repositories"`
}

// Repository 表示一个远程 Git 仓库书签。
type Repository struct {
	// Name 是本地 registry 名称。
	Name string `yaml:"name"`
	// URL 是 Git 仓库地址。
	URL string `yaml:"url"`
	// Branch 是可选 Git 分支；为空时使用远端默认分支。
	Branch string `yaml:"branch,omitempty"`
	// LastUpdatedAt 是本地 cache 最近一次成功生成或刷新的时间。
	LastUpdatedAt string `yaml:"last_updated_at,omitempty"`
}

// Store 负责读取和写入完整用户配置。
type Store interface {
	// Load 读取完整配置。
	Load() (Config, error)
	// Save 保存完整配置。
	Save(Config) error
}

// Parse 将 YAML 配置内容解析为 Config。
func Parse(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadFile 从指定路径读取并解析 config.yaml。
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(data)
}

// FileStore 使用单个 YAML 文件读写配置。
type FileStore struct {
	// path 是用户级配置文件的完整路径。
	path string
}

// NewFileStore 创建基于文件路径的配置 Store。
func NewFileStore(path string) FileStore {
	return FileStore{path: path}
}

// Load 读取配置文件；文件不存在时返回空配置。
func (s FileStore) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	return Parse(data)
}

// Save 将完整配置写回文件。
func (s FileStore) Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
