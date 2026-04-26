package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/finger/skill-down/internal/skilldown/app"
	"github.com/finger/skill-down/internal/skilldown/cli"
	"github.com/finger/skill-down/internal/skilldown/config"
	"github.com/finger/skill-down/internal/skilldown/install"
	"github.com/finger/skill-down/internal/skilldown/registry"
	"github.com/finger/skill-down/internal/skilldown/repo"
)

// main 启动 CLI 程序。
func main() {
	configPath, err := userConfigPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	store := config.NewFileStore(configPath)
	service := app.Service{
		Registry:  registry.NewService(store),
		Cloner:    repo.GitCloner{},
		Installer: install.Installer{},
	}

	os.Exit(cli.Execute(context.Background(), os.Args[1:], service))
}

// userConfigPath 返回 skill-down 的用户级配置文件路径。
func userConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "skill-down", "config.yaml"), nil
}
