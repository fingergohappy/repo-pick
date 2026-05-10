package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// registryLinesView 生成左栏 registry 面板正文。
func (m model) registryLinesView() string {
	leftWidth, _ := paneWidths(m.width)
	contentWidth := paneContentWidth(leftWidth)
	lineLimit := max(1, m.paneItemRows())
	if contentWidth <= 0 {
		return ""
	}
	if len(m.repositories) == 0 {
		return m.emptyRegistryLinesView(contentWidth)
	}

	start, end := visibleWindow(len(m.repositories), m.selectedRepo, lineLimit)
	if start >= end {
		return ""
	}
	rows := make([][]string, 0, end-start)
	for i := start; i < end; i++ {
		repository := m.repositories[i]
		cursor := " "
		if i == m.selectedRepo {
			cursor = m.selectionCursor()
		}
		rows = append(rows, []string{
			fmt.Sprintf("%s %s", cursor, registryListName(repository.Name)),
			shortRepositoryUpdatedAt(repository.LastUpdatedAt),
		})
	}

	selected := m.selectedRepo - start
	if selected < 0 || selected >= len(rows) {
		selected = -1
	}
	return renderTableRows(rows, contentWidth, func(row int, col int) lipgloss.Style {
		isSelected := row == selected
		if col == 1 {
			return registryUpdatedAtStyle(isSelected)
		}
		return registryNameStyle(isSelected)
	})
}

// registryListName 返回左侧 registry 列表中的名称展示，不包含分支。
func registryListName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "-"
	}
	return name
}

// shortRepositoryUpdatedAt 将 RFC3339 更新时间压缩成适合左栏展示的短文本。
func shortRepositoryUpdatedAt(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	updatedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "?"
	}
	localUpdatedAt := updatedAt.Local()
	now := time.Now()
	if sameCalendarDay(localUpdatedAt, now) {
		return localUpdatedAt.Format("15:04")
	}
	if localUpdatedAt.Year() == now.Year() {
		return localUpdatedAt.Format("01-02")
	}
	return localUpdatedAt.Format("2006")
}

// sameCalendarDay 判断两个时间是否属于同一个本地日历日。
func sameCalendarDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// emptyRegistryLinesView 生成 registry 为空时的占位内容。
func (m model) emptyRegistryLinesView(contentWidth int) string {
	if contentWidth <= 0 {
		return ""
	}
	cardWidth := max(12, min(contentWidth, 28))
	innerWidth := max(8, cardWidth-2)
	cta := fmt.Sprintf("%s %s", registryEmptyKeyStyle.Render("a"), registryEmptyActionStyle.Render("添加 registry"))
	card := emptyCardContainerStyle(cardWidth).Render(strings.Join([]string{
		registryEmptyTitleStyle.Render("暂无 registry"),
		registryEmptyHintStyle.Render("添加后显示在这里"),
		truncateVisible(cta, innerWidth),
	}, "\n"))

	contentHeight := max(1, m.paneItemRows())
	return renderCenteredBlock(contentWidth, contentHeight, card)
}
