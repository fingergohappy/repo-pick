package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// statusLine 生成底部状态栏文本。
func (m model) statusLine() string {
	if m.mode != modeNormal {
		if m.isAddMode() {
			return fitPlainLine(m.status, m.width)
		}
		if m.mode == modeSearch {
			return fitPlainLine(m.status, m.width)
		}
		if m.mode == modeConfirmDelete || m.mode == modeConfirmOverwrite {
			return fitPlainLine(m.status, m.width)
		}
		return fitPlainLine(m.prompt()+m.input.View(), m.width)
	}
	if m.err != nil {
		return renderStatusLine("error: "+firstLine(m.err.Error()), "esc clear | ? help", m.width)
	}
	status := m.status
	if m.operationKind == operationOpen || m.operationKind == operationUpdate {
		status = ""
	}
	if operationStatus := m.statusOperationLine(); operationStatus != "" {
		status = operationStatus
	}
	return renderStatusLine(status, m.focusHelpLine(), m.width)
}

// focusHelpLine 返回当前焦点对应的底部快捷键提示。
func (m model) focusHelpLine() string {
	if m.focus == focusRegistry {
		return keyHelp("j/k", "move", "l", "open", "a", "add", "e", "edit", "r", "reload", "d", "delete", "u", "update", "Tab", "tree", "?", "help")
	}
	return keyHelp("j/k", "move", "l", "expand", "C", "collapse", "h", "parent", "o", "root", "e", "editor", "i", "download", "I", "target", "/", "search", "Tab", "registry", "?", "help")
}

// prompt 返回当前输入模式的提示文本。
func (m model) prompt() string {
	switch m.mode {
	case modeAddName:
		return "repository name "
	case modeAddURL:
		return "repo url "
	case modeSearch:
		return "search "
	case modeTargetDir:
		return "target dir "
	default:
		return ""
	}
}

// keyHelp 生成底栏快捷键提示文本。
func keyHelp(parts ...string) string {
	chunks := make([]string, 0, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		chunks = append(chunks, fmt.Sprintf("[%s]%s", parts[i], parts[i+1]))
	}
	return strings.Join(chunks, " ")
}

// renderStatusLine 将状态和快捷键压缩到终端宽度内。
func renderStatusLine(status string, help string, width int) string {
	if width <= 0 {
		return ""
	}
	status = firstLine(status)
	help = firstLine(help)
	if help == "" {
		return statusLineContainerStyle(width).Render(statusTextStyle.Render(truncateVisible(status, width)))
	}

	fullHelp := " | " + help
	if lipgloss.Width(status)+lipgloss.Width(fullHelp) <= width {
		return renderStatusBlocks(status, fullHelp, width)
	}

	compactHelp := " | ? help"
	if lipgloss.Width(compactHelp) < width {
		return renderStatusBlocks(status, compactHelp, width)
	}
	return statusLineContainerStyle(width).Render(statusTextStyle.Render(truncateVisible(status, width)))
}

// renderStatusBlocks 渲染状态栏左右区域。
func renderStatusBlocks(status string, help string, width int) string {
	helpWidth := lipgloss.Width(help)
	statusWidth := max(1, width-helpWidth)
	left := statusTextStyle.Width(statusWidth).Render(truncateVisible(status, statusWidth))
	right := statusHelpStyle.Render(help)
	return statusLineContainerStyle(width).Render(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
}
