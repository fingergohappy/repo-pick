package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// paneRenderOptions 描述 pane 基础参数。
type paneRenderOptions struct {
	width   int
	height  int
	focused bool
	title   string
}

// renderScreen 组装上下结构：主屏 + status。
func renderScreen(main string, status string) string {
	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

// renderPane 组装单个面板：边框、标题和内容。
func renderPane(title string, body string, options paneRenderOptions) string {
	style := paneStyle(options.width, options.height, options.focused)
	contentHeight := max(1, options.height-2)
	contentWidth := paneContentWidth(options.width)
	bodyLines := splitRenderedLines(body)
	contentLines := make([]string, 0, contentHeight)
	contentLines = append(contentLines, renderPaneTitleLine(title, contentWidth, options.focused))
	contentLines = append(contentLines, renderPaneSeparatorLine(contentWidth))
	for _, line := range bodyLines {
		contentLines = append(contentLines, truncateVisible(firstLine(line), contentWidth))
	}
	if len(contentLines) < contentHeight {
		for i := len(contentLines); i < contentHeight; i++ {
			contentLines = append(contentLines, "")
		}
	}
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	return style.Render(strings.Join(contentLines, "\n"))
}

// renderPaneTitleLine 将标题按可见宽度居中并根据焦点上色。
func renderPaneTitleLine(title string, width int, focused bool) string {
	label := truncateVisible(firstLine(title), width)
	return paneHeaderStyle(focused).Render(centerLine(label, width))
}

// renderPaneSeparatorLine 返回栏目分隔线。
func renderPaneSeparatorLine(width int) string {
	return paneDividerStyle().Render(strings.Repeat("─", max(12, width)))
}

// renderCenteredBlock 将文本块竖向偏上三分之一放置。
func renderCenteredBlock(width int, height int, lines ...string) string {
	if width <= 0 || height < 0 {
		return ""
	}
	cleanLines := make([]string, 0, len(lines))
	for _, line := range lines {
		split := splitRenderedLines(line)
		if len(split) == 0 {
			cleanLines = append(cleanLines, "")
			continue
		}
		for _, item := range split {
			cleanLines = append(cleanLines, centerLine(item, width))
		}
	}
	if len(cleanLines) > height {
		cleanLines = cleanLines[:height]
	}
	block := lipgloss.JoinVertical(lipgloss.Center, cleanLines...)
	if height <= lipgloss.Height(block) {
		return block
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Position(0.65), block)
}

// renderModal 构造统一弹框。
func renderModal(body string, width int) string {
	return modalStyle(width).Render(body)
}

// placeOverlay 将 overlay 居中覆盖到底图。
func placeOverlay(base string, overlay string, width int, height int) string {
	if width <= 0 || height <= 0 {
		return base
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}
