// Package tui 提供 repo registry 浏览、仓库目录树选择和下载入口。
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/app"
)

// Run 启动 Bubble Tea 交互式 TUI。
func Run(ctx context.Context, svc app.Service, sessionCWD string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	program := tea.NewProgram(newModel(ctx, svc, sessionCWD), tea.WithContext(ctx), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
