package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/finger/repo-pick/internal/repopick/app"
)

// treeLinesView 生成右栏目录树面板正文。
func (m model) treeLinesView() string {
	_, rightWidth := paneWidths(m.width)
	contentWidth := paneContentWidth(rightWidth)
	lineLimit := max(1, m.paneItemRows())
	if m.treeOperationInProgress() {
		return renderCenteredBlock(contentWidth, lineLimit, m.treeLoadingLines()...)
	}
	if m.showRegistrySelectionPreview() {
		return renderCenteredBlock(contentWidth, lineLimit, m.registrySelectionPreviewLines()...)
	}
	if !m.repoOpened {
		return emptyEntryStyle.Render("未打开 repository")
	}

	context := m.treeContextLinesView()
	entryLines := []string{}
	entryLines = append(entryLines, context)
	if strings.TrimSpace(context) != "" {
		entryLines = append(entryLines, m.treeSeparatorLine(rightWidth))
	}
	if m.showingSearch {
		table := m.searchResultRowsView(contentWidth, lineLimit)
		entryLines = append(entryLines, table)
	} else {
		entryLines = append(entryLines, m.treeRowsView(contentWidth, lineLimit)...)
	}
	return strings.Join(entryLines, "\n")
}

// treeContextLinesView 生成右侧内容列表上方的仓库上下文区域。
func (m model) treeContextLinesView() string {
	rows := [][2]string{
		{"registry", m.openedRepositoryName()},
		{"url", m.openedRepo.URL},
		{"branch", m.openedRepositoryBranch()},
		{"path", displayPath(m.currentPath)},
	}
	if m.mode == modeSearch {
		rows = append(rows, [2]string{"search", m.input.View()})
	} else if m.showingSearch {
		rows = append(rows, [2]string{"search", m.searchQuery})
	}
	_, rightWidth := paneWidths(m.width)
	return renderKeyValueTable(rows, paneContentWidth(rightWidth), func(row int, col int) lipgloss.Style {
		if col == 0 {
			return treeMetaLabelStyle
		}
		return treeMetaStyle
	})
}

// openedRepositoryName 返回当前打开 registry 的名称。
func (m model) openedRepositoryName() string {
	return strings.TrimSpace(m.openedRepo.Name)
}

// openedRepositoryBranch 返回当前打开 registry 的分支展示。
func (m model) openedRepositoryBranch() string {
	branch := strings.TrimSpace(m.openedRepo.Branch)
	if branch == "" {
		return "远端默认分支"
	}
	return branch
}

// treeSeparatorLine 返回搜索区域和文件列表之间的分隔线。
func (m model) treeSeparatorLine(width int) string {
	return treeSeparatorStyle.Render(strings.Repeat("-", max(12, width-6)))
}

// treeLoadingLines 生成右侧仓库加载模板态。
func (m model) treeLoadingLines() []string {
	_, rightWidth := paneWidths(m.width)
	contentWidth := max(24, rightWidth-6)
	title := "正在加载 repository"
	if m.operationKind == operationUpdate {
		title = "正在更新 repository"
	}
	status := m.operationStatus()
	if status == "" {
		status = "working"
	}

	return []string{
		loadingTextLine(title, contentWidth, treeLoadingTitleStyle),
		treeLoadingTextStyle.Render(status),
		treeLoadingHintStyle.Render("cache 准备完成后会自动展示目录树"),
		"",
		centerLine(m.treeProgressBarView(contentWidth), contentWidth),
		centerLine(m.treeProgressPercentView(), contentWidth),
	}
}

// registrySelectionPreviewLines 生成右侧 registry 选中变化提示。
func (m model) registrySelectionPreviewLines() []string {
	repo, ok := m.activeRepository()
	if !ok {
		return []string{emptyEntryStyle.Render("未打开 repository")}
	}
	_, rightWidth := paneWidths(m.width)
	contentWidth := max(24, rightWidth-6)
	branch := strings.TrimSpace(repo.Branch)
	if branch == "" {
		branch = "远端默认分支"
	}

	return []string{
		loadingTextLine("已选择 registry", contentWidth, treeLoadingTitleStyle),
		loadingTextLine(m.registrySelectionStatus(), contentWidth, treeLoadingTextStyle),
		"",
		treeMetaStyle.Render("url: " + strings.TrimSpace(repo.URL)),
		treeMetaStyle.Render("branch: " + branch),
		"",
		centerLine(treeLoadingHintStyle.Render("按 l 打开该 repository"), contentWidth),
	}
}

// treeRowsView 生成树内容行。
func (m model) treeRowsView(contentWidth int, lineLimit int) []string {
	rows := m.visibleTreeRows()
	if len(rows) == 0 {
		return []string{emptyEntryStyle.Render("暂无条目")}
	}
	entryLimit := max(0, lineLimit-1)
	if entryLimit <= 0 {
		return []string{}
	}
	start, end := visibleWindow(len(rows), m.selectedEntry, entryLimit)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		selected := i == m.selectedEntry
		lines = append(lines, m.treeEntryLine(rows[i], selected, contentWidth))
	}
	return lines
}

// searchResultRowsView 生成搜索结果为 table。
func (m model) searchResultRowsView(contentWidth int, lineLimit int) string {
	entries := m.visibleEntries()
	if len(entries) == 0 {
		return emptyEntryStyle.Render("暂无条目")
	}
	entryLimit := max(0, lineLimit-1)
	if entryLimit <= 0 {
		return ""
	}
	start, end := visibleWindow(len(entries), m.selectedEntry, entryLimit)
	rows := make([][]string, 0, end-start+1)
	rows = append(rows, []string{"", "type", "size", "path"})
	for i := start; i < end; i++ {
		entry := entries[i]
		cursor := " "
		if i == m.selectedEntry {
			cursor = m.selectionCursor()
		}
		name := entry.Path
		if entry.Type == app.EntryDir {
			name += "/"
		}
		rows = append(rows, []string{cursor, entryIcon(entry), entrySize(entry), name})
	}
	selected := m.selectedEntry - start + 1
	if selected < 0 || selected >= len(rows) {
		selected = -1
	}
	resolver := func(row int, col int) lipgloss.Style {
		if row == 0 {
			return treeHeaderStyle
		}
		if selected >= 0 && row == selected {
			return selectedLineStyle
		}
		entry := m.visibleEntries()[start+row-1]
		if entry.Type == app.EntryDir {
			if entry.Path == "" {
				return treeRootEntryStyle
			}
			return dirEntryStyle
		}
		return fileEntryStyle
	}
	return renderTableRows(rows, contentWidth, resolver)
}

// loadingTextLine 将加载态文本限制在面板宽度内并居中展示。
func loadingTextLine(text string, width int, style lipgloss.Style) string {
	text = style.Inline(true).MaxWidth(width).Render(text)
	return centerLine(text, width)
}

// treeProgressBarView 渲染右侧仓库加载进度条。
func (m model) treeProgressBarView(width int) string {
	barWidth := max(16, min(48, width-8))
	percent := 0.0
	options := []progress.Option{
		progress.WithWidth(barWidth),
		progress.WithSolidFill("6"),
		progress.WithFillCharacters('█', '░'),
		progress.WithoutPercentage(),
	}
	if m.operationPercent >= 0 {
		percent = float64(m.operationPercent) / 100
	}
	bar := progress.New(options...)
	return bar.ViewAs(percent)
}

// treeProgressPercentView 渲染右侧仓库加载百分比。
func (m model) treeProgressPercentView() string {
	if m.operationPercent < 0 {
		return treeLoadingHintStyle.Render("waiting for git progress")
	}
	return treeLoadingTextStyle.Render(fmt.Sprintf("%d%%", m.operationPercent))
}

// treeEntryLine 渲染右侧一个文件或目录条目。
func (m model) treeEntryLine(row treeRow, selected bool, width int) string {
	if row.root {
		return m.treeRootEntryLine(row, selected, width)
	}
	cursor := " "
	if selected {
		cursor = m.selectionCursor()
	}
	entry := row.entry
	marker := "•"
	if entry.Type == app.EntryDir {
		if row.expanded {
			marker = "▾"
		} else {
			marker = "▸"
		}
	}
	name := entry.Name
	if entry.Type == app.EntryDir {
		name += "/"
	}
	line := fmt.Sprintf("%s %s%s %s%s", cursor, row.prefix, marker, name, treeEntryMeta(entry))
	if selected {
		return selectedLineStyle.Render(truncateVisible(line, width))
	}
	if entry.Type == app.EntryDir {
		return dirEntryStyle.Render(truncateVisible(line, width))
	}
	return fileEntryStyle.Render(truncateVisible(line, width))
}

// treeRootEntryLine 渲染右侧当前 tree root 行。
func (m model) treeRootEntryLine(row treeRow, selected bool, width int) string {
	cursor := " "
	if selected {
		cursor = m.selectionCursor()
	}
	line := truncateVisible(fmt.Sprintf("%s %s", cursor, displayPath(row.entry.Path)), width)
	if selected {
		return selectedLineStyle.Render(line)
	}
	return treeRootEntryStyle.Render(line)
}

// treeEntryMeta 返回文件节点的弱化补充信息。
func treeEntryMeta(entry app.EntryResult) string {
	if entry.Type == app.EntryDir {
		return ""
	}
	return fmt.Sprintf("  %s", entrySize(entry))
}
