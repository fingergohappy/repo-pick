// Package config 负责解析 skill-down 的用户级配置文件。
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config 表示用户级 config.yaml 配置。
type Config struct {
	// Repositories 是已注册的 skill 仓库列表。
	Repositories []Repository `yaml:"repositories"`
	// Repo 是 repo 模块相关配置。
	Repo RepoConfig `yaml:"repo"`
}

// Repository 表示一个可用的 skill 仓库配置。
type Repository struct {
	// Name 是本地 registry 名称。
	Name string `yaml:"name"`
	// URL 是 Git 仓库地址。
	URL string `yaml:"url"`
	// SkillDir 是仓库中承载 skill 子目录的相对路径。
	SkillDir string `yaml:"skillDir"`
}

// RepoConfig 表示 repo 模块相关配置。
type RepoConfig struct {
	// DownloadDir 是远程仓库 clone 的父目录，为空时使用系统临时目录。
	DownloadDir string `yaml:"downloadDir"`
}

// Store 负责读取和写入完整用户配置。
type Store interface {
	Load() (Config, error)
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

// FileStore 使用 Viper 读写单个 config.yaml。
type FileStore struct {
	path string
}

// NewFileStore 创建基于文件路径的配置 Store。
func NewFileStore(path string) FileStore {
	return FileStore{path: path}
}

// Load 读取配置文件；文件不存在时返回空配置。
func (s FileStore) Load() (Config, error) {
	v := s.newViper()
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return Config{}, nil
		}
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save 将完整配置写回文件。
func (s FileStore) Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	v := s.newViper()
	v.Set("repositories", cfg.Repositories)
	v.Set("repo", cfg.Repo)
	return v.WriteConfigAs(s.path)
}

func (s FileStore) newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigFile(s.path)
	v.SetConfigType("yaml")
	return v
}
