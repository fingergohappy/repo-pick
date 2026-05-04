package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/cache"
	"github.com/finger/repo-pick/internal/repopick/config"
	"github.com/finger/repo-pick/internal/repopick/install"
	"github.com/finger/repo-pick/internal/repopick/registry"
	"github.com/finger/repo-pick/internal/repopick/tui"
)

// main 启动 repo-pick TUI 程序。
func main() {
	configPath, err := userConfigPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cacheRoot, err := userCacheRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sessionCWD, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := config.NewFileStore(configPath)
	service := app.Service{
		Registry:  registry.NewService(store),
		Cache:     cache.Service{RootDir: cacheRoot},
		Installer: install.Installer{},
	}

	if err := tui.Run(context.Background(), service, sessionCWD); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// userConfigPath 返回 repo-pick 固定的 XDG 风格用户配置文件路径。
func userConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "repo-pick", "config.yaml"), nil
}

// userCacheRoot 返回 repo-pick 管理的 repo cache 父目录。
func userCacheRoot() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache dir: %w", err)
	}
	return filepath.Join(cacheDir, "repo-pick", "repos"), nil
}
