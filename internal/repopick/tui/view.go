package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/finger/repo-pick/internal/repopick/app"
)

const (
	paneGapAllowance = 3
	minTerminalWidth = 64
)

// selectedLineStyle 是列表选中项的高亮样式。
var selectedLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Background(lipgloss.Color("238")).
	Bold(true)

// treeMetaStyle 是右侧目录上下文信息的样式。
var treeMetaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

// treeMetaLabelStyle 是右侧目录上下文标签的样式。
var treeMetaLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)

// treeSeparatorStyle 是右侧目录区域分隔线的样式。
var treeSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// treeHeaderStyle 是右侧内容表头的样式。
var treeHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)

// treeLoadingTitleStyle 是右侧加载态标题样式。
var treeLoadingTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

// treeLoadingTextStyle 是右侧加载态进度文本样式。
var treeLoadingTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))

// treeLoadingHintStyle 是右侧加载态说明样式。
var treeLoadingHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

// dirEntryStyle 是目录条目的样式。
var dirEntryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)

// fileEntryStyle 是文件条目的样式。
var fileEntryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

// emptyEntryStyle 是空列表提示的样式。
var emptyEntryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)

// registryEmptyTitleStyle 是 registry 空状态标题样式。
var registryEmptyTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)

// registryEmptyHintStyle 是 registry 空状态说明样式。
var registryEmptyHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

// registryEmptyKeyStyle 是 registry 空状态快捷键样式。
var registryEmptyKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Background(lipgloss.Color("24")).
	Bold(true).
	Padding(0, 1)

// registryEmptyActionStyle 是 registry 空状态动作文本样式。
var registryEmptyActionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

// modalTitleStyle 是弹框标题样式。
var modalTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

// modalDescStyle 是弹框说明文本样式。
var modalDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

// modalFieldLabelStyle 是弹框字段标签样式。
var modalFieldLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

// modalFieldValueStyle 是弹框字段值样式。
var modalFieldValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

// modalHintStyle 是弹框底部快捷键提示样式。
var modalHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

// modalDividerStyle 是弹框内部分隔线样式。
var modalDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

// statusTextStyle 是底部状态消息的样式。
var statusTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

// statusHelpStyle 是底部快捷键提示的弱化样式。
var statusHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

// View 渲染双栏主界面和底部状态。
func (m model) View() string {
	if m.showHelp {
		return m.helpView()
	}
	if m.width > 0 && m.width < minTerminalWidth {
		return m.narrowView()
	}

	leftWidth, rightWidth := paneWidths(m.width)
	left := m.paneView("Registry", m.registryLines(), leftWidth, m.focus == focusRegistry)
	right := m.paneView("Repository Tree", m.treeLines(), rightWidth, m.focus == focusTree)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	if m.isAddMode() {
		bodyHeight := max(8, m.height-3)
		body = lipgloss.Place(m.width, bodyHeight, lipgloss.Center, lipgloss.Center, m.addRepositoryModalView())
	}

	return body + "\n" + m.statusLine()
}

// paneView 渲染单个带边框的栏目。
func (m model) paneView(title string, lines []string, width int, focused bool) string {
	style := lipgloss.NewStyle().Width(width).Height(m.paneHeight()).Padding(0, 1).Border(lipgloss.NormalBorder())
	if focused {
		style = style.BorderForeground(lipgloss.Color("39"))
	}
	contentWidth := paneContentWidth(width)
	contentRows := max(1, m.paneBodyRows())
	allLines := append([]string{truncateVisible(title, contentWidth)}, lines...)
	if len(allLines) > contentRows {
		allLines = allLines[:contentRows]
	}
	for i, line := range allLines {
		allLines[i] = truncateVisible(firstLine(line), contentWidth)
	}
	content := strings.Join(allLines, "\n")
	return style.Render(content)
}

// registryLines 生成左栏 registry 文本行。
func (m model) registryLines() []string {
	leftWidth, _ := paneWidths(m.width)
	contentWidth := paneContentWidth(leftWidth)
	lineLimit := max(1, m.paneItemRows())
	if len(m.repositories) == 0 {
		return m.emptyRegistryLines()
	}
	start, end := visibleWindow(len(m.repositories), m.selectedRepo, lineLimit)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		repository := m.repositories[i]
		cursor := " "
		if i == m.selectedRepo {
			cursor = ">"
		}
		name := repository.Name
		if strings.TrimSpace(repository.Branch) != "" {
			name = fmt.Sprintf("%s [%s]", name, repository.Branch)
		}
		line := fmt.Sprintf("%s %s", cursor, name)
		line = truncateVisible(line, contentWidth)
		lines = append(lines, selectedLine(line, i == m.selectedRepo))
	}
	return lines
}

// emptyRegistryLines 生成 registry 为空时的占位内容。
func (m model) emptyRegistryLines() []string {
	leftWidth, _ := paneWidths(m.width)
	contentWidth := max(12, leftWidth-4)

	return []string{
		"",
		centerLine(registryEmptyTitleStyle.Render("暂无 registry"), contentWidth),
		centerLine(registryEmptyHintStyle.Render("添加 registry 后会显示在这里"), contentWidth),
		"",
		centerLine(fmt.Sprintf("%s %s", registryEmptyKeyStyle.Render("a"), registryEmptyActionStyle.Render("添加 registry")), contentWidth),
	}
}

// treeLines 生成右栏目录树文本行。
func (m model) treeLines() []string {
	_, rightWidth := paneWidths(m.width)
	contentWidth := paneContentWidth(rightWidth)
	lineLimit := max(1, m.paneItemRows())
	if m.treeOperationInProgress() {
		return limitLines(m.treeLoadingLines(), lineLimit)
	}
	if m.showRegistrySelectionPreview() {
		return limitLines(m.registrySelectionPreviewLines(), lineLimit)
	}
	if !m.repoOpened {
		return []string{emptyEntryStyle.Render("未打开 repository")}
	}

	lines := m.treeContextLines()
	lines = append(lines, m.treeSeparatorLine(), m.treeContentHeaderLine())
	entryLimit := max(0, lineLimit-len(lines))

	if m.showingSearch {
		entries := m.visibleEntries()
		if len(entries) == 0 {
			return append(lines, emptyEntryStyle.Render("暂无条目"))
		}
		start, end := visibleWindow(len(entries), m.selectedEntry, entryLimit)
		for i := start; i < end; i++ {
			lines = append(lines, m.searchEntryLine(entries[i], i == m.selectedEntry, contentWidth))
		}
		return lines
	}

	rows := m.visibleTreeRows()
	if len(rows) == 0 {
		return append(lines, emptyEntryStyle.Render("暂无条目"))
	}
	start, end := visibleWindow(len(rows), m.selectedEntry, entryLimit)
	for i := start; i < end; i++ {
		lines = append(lines, m.treeEntryLine(rows[i], i == m.selectedEntry, contentWidth))
	}
	return lines
}

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
	if operationStatus := m.statusOperationLine(); operationStatus != "" {
		status = operationStatus
	}
	return renderStatusLine(status, m.focusHelpLine(), m.width)
}

// focusHelpLine 返回当前焦点对应的底部快捷键提示。
func (m model) focusHelpLine() string {
	if m.focus == focusRegistry {
		return "ctrl-w h/l focus | j/k move | l open repo | a add registry | e edit registry | r reload list | d delete repo+cache | u update repo cache | ? help"
	}
	return "ctrl-w h/l focus | j/k move | l expand/collapse | o enter root | h parent root | i download | / search | ? help"
}

// treeContextLines 生成右侧内容列表上方的仓库上下文区域。
func (m model) treeContextLines() []string {
	lines := []string{
		m.treeMetaLine("registry", m.openedRepositoryName()),
		m.treeMetaLine("url", m.openedRepo.URL),
		m.treeMetaLine("branch", m.openedRepositoryBranch()),
		m.treeMetaLine("path", displayPath(m.currentPath)),
	}
	if m.mode == modeSearch {
		lines = append(lines, m.treeMetaLine("search", m.input.View()))
	} else if m.showingSearch {
		lines = append(lines, m.treeMetaLine("search", m.searchQuery))
	}
	return lines
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

// treeMetaLine 渲染右侧上下文信息行。
func (m model) treeMetaLine(label string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	_, rightWidth := paneWidths(m.width)
	contentWidth := paneContentWidth(rightWidth)
	labelText := label + "  "
	value = truncateVisible(value, max(1, contentWidth-lipgloss.Width(labelText)))
	return fmt.Sprintf("%s  %s", treeMetaLabelStyle.Render(label), treeMetaStyle.Render(value))
}

// treeSeparatorLine 返回搜索区域和文件列表之间的分隔线。
func (m model) treeSeparatorLine() string {
	_, rightWidth := paneWidths(m.width)
	return treeSeparatorStyle.Render(strings.Repeat("-", max(12, rightWidth-6)))
}

// treeContentHeaderLine 渲染右侧文件内容表头。
func (m model) treeContentHeaderLine() string {
	if m.showingSearch {
		_, rightWidth := paneWidths(m.width)
		header := truncateVisible(fmt.Sprintf("  %-4s %-8s %s", "type", "size", "path"), paneContentWidth(rightWidth))
		return treeHeaderStyle.Render(header)
	}
	return treeHeaderStyle.Render("  tree")
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

	return m.centerTreeLoadingBlock([]string{
		loadingTextLine(title, contentWidth, treeLoadingTitleStyle),
		loadingTextLine(status, contentWidth, treeLoadingTextStyle),
		"",
		centerLine(m.treeProgressBarView(contentWidth), contentWidth),
		centerLine(m.treeProgressPercentView(), contentWidth),
		"",
		centerLine(treeLoadingHintStyle.Render("cache 准备完成后会自动展示目录树"), contentWidth),
	})
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

	return m.centerTreeLoadingBlock([]string{
		loadingTextLine("已选择 registry", contentWidth, treeLoadingTitleStyle),
		loadingTextLine(m.registrySelectionStatus(), contentWidth, treeLoadingTextStyle),
		"",
		loadingTextLine("url: "+strings.TrimSpace(repo.URL), contentWidth, treeMetaStyle),
		loadingTextLine("branch: "+branch, contentWidth, treeMetaStyle),
		"",
		centerLine(treeLoadingHintStyle.Render("按 l 打开该 repository"), contentWidth),
	})
}

// centerTreeLoadingBlock 将右侧加载块放在树面板内容区靠上三分之一处。
func (m model) centerTreeLoadingBlock(lines []string) []string {
	paneHeight := max(8, m.height-3)
	contentHeight := max(1, paneHeight-1)
	topPadding := max(0, (contentHeight-len(lines))/3)
	centered := make([]string, 0, topPadding+len(lines))
	for i := 0; i < topPadding; i++ {
		centered = append(centered, "")
	}
	return append(centered, lines...)
}

// loadingTextLine 将加载态文本限制在面板宽度内并居中展示。
func loadingTextLine(text string, width int, style lipgloss.Style) string {
	text = style.Inline(true).MaxWidth(width).Render(text)
	return centerLine(text, width)
}

// treeProgressBarView 渲染居中的右侧仓库加载进度条。
func (m model) treeProgressBarView(width int) string {
	barWidth := max(16, min(48, width-8))
	percent := 0.0
	options := []progress.Option{
		progress.WithWidth(barWidth),
		progress.WithSolidFill("#00A6A6"),
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
	cursor := " "
	if selected {
		cursor = ">"
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
	line = truncateVisible(line, width)
	if selected {
		return selectedLine(line, true)
	}
	if entry.Type == app.EntryDir {
		return dirEntryStyle.Render(line)
	}
	return fileEntryStyle.Render(line)
}

// treeEntryMeta 返回文件节点的弱化补充信息。
func treeEntryMeta(entry app.EntryResult) string {
	if entry.Type == app.EntryDir {
		return ""
	}
	return fmt.Sprintf("  %s", entrySize(entry))
}

// searchEntryLine 渲染右侧一个搜索结果条目。
func (m model) searchEntryLine(entry app.EntryResult, selected bool, width int) string {
	cursor := " "
	if selected {
		cursor = ">"
	}
	name := entry.Path
	if entry.Type == app.EntryDir {
		name += "/"
	}
	line := fmt.Sprintf("%s %-4s %-8s %s", cursor, entryIcon(entry), entrySize(entry), name)
	line = truncateVisible(line, width)
	if selected {
		return selectedLine(line, true)
	}
	if entry.Type == app.EntryDir {
		return dirEntryStyle.Render(line)
	}
	return fileEntryStyle.Render(line)
}

// isAddMode 判断当前是否处于 registry 表单流程。
func (m model) isAddMode() bool {
	return m.mode == modeAddName || m.mode == modeAddURL || m.mode == modeAddBranch
}

// addRepositoryModalView 渲染 registry 新增或编辑弹框。
func (m model) addRepositoryModalView() string {
	modalWidth := max(32, min(76, m.width-4))
	innerWidth := max(24, modalWidth-6)
	title := "添加 Registry"
	if m.editingRepositoryActive {
		title = "编辑 Registry"
	}
	lines := []string{
		centerLine(modalTitleStyle.Render(title), innerWidth),
		modalDescStyle.Render("同一 URL 可添加多个分支；URL 和 branch 组合不能重复"),
		modalDividerLine(innerWidth),
	}

	name := m.pendingName
	if m.mode == modeAddName {
		name = m.input.View()
	}
	lines = append(lines, addModalFieldLine("name", name, m.mode == modeAddName))

	repoURL := m.pendingURL
	if m.mode == modeAddURL {
		repoURL = m.input.View()
	}
	lines = append(lines, addModalFieldLine("url", repoURL, m.mode == modeAddURL), "")

	if m.mode == modeAddBranch {
		lines = append(lines, m.branchSelectLines()...)
	} else {
		branch := m.pendingBranch
		if strings.TrimSpace(branch) == "" {
			branch = "使用远端默认分支"
		}
		lines = append(lines, addModalFieldLine("branch", branch, false))
	}
	lines = append(lines, modalDividerLine(innerWidth), modalHintStyle.Render("Tab/Shift+Tab 切焦点   输入搜索分支   Enter 确认   Esc 取消"))

	style := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))
	return style.Render(strings.Join(lines, "\n"))
}

// addModalFieldLine 渲染新增弹框中的单个字段行。
func addModalFieldLine(label string, value string, active bool) string {
	text := fmt.Sprintf("  %-7s %s", label, emptyPlaceholder(value, "-"))
	if active {
		return selectedLineStyle.Render(text)
	}
	return fmt.Sprintf("  %s %s", modalFieldLabelStyle.Render(fmt.Sprintf("%-7s", label)), modalFieldValueStyle.Render(emptyPlaceholder(value, "-")))
}

// modalDividerLine 渲染新增弹框中的横向分隔线。
func modalDividerLine(width int) string {
	return modalDividerStyle.Render(strings.Repeat("-", max(12, width)))
}

// branchSelectLines 生成 registry 表单里的分支选择行。
func (m model) branchSelectLines() []string {
	if m.branchLoading {
		return []string{
			modalFieldLabelStyle.Render("branch"),
			modalDescStyle.Render("  正在获取远端分支..."),
		}
	}

	query := m.branchQuery
	if m.mode == modeAddBranch {
		query = m.input.View()
	}
	lines := []string{
		modalFieldLabelStyle.Render("branch"),
		fmt.Sprintf("  %s %s", modalFieldLabelStyle.Render("search"), modalFieldValueStyle.Render(emptyPlaceholder(query, "-"))),
	}
	if m.branchErr != nil {
		message := "  获取失败，Enter 使用远端默认分支"
		if m.editingRepositoryActive && strings.TrimSpace(m.pendingBranch) != "" {
			message = "  获取失败，Enter 保留当前分支"
		}
		lines = append(lines, modalDescStyle.Render(message))
		return lines
	}

	choices := m.branchChoiceLabels()
	if strings.TrimSpace(m.branchQuery) != "" && len(m.filteredBranchNames()) == 0 {
		lines = append(lines, modalDescStyle.Render("  无匹配分支"))
	}
	headerLines := len(lines)
	start, end := m.branchWindow(len(choices))
	for i := start; i < end; i++ {
		cursor := " "
		if i == m.selectedBranch {
			cursor = ">"
		}
		line := fmt.Sprintf("  %s %s", cursor, choices[i])
		lines = append(lines, selectedLine(line, i == m.selectedBranch))
	}
	if start > 0 {
		head := append([]string{}, lines[:headerLines]...)
		lines = append(head, append([]string{"    ..."}, lines[headerLines:]...)...)
	}
	if end < len(choices) {
		lines = append(lines, "    ...")
	}
	return lines
}

// selectedLine 按需渲染选中行高亮。
func selectedLine(line string, selected bool) string {
	if !selected {
		return line
	}
	return selectedLineStyle.Render(line)
}

// centerLine 按可见宽度居中渲染单行文本。
func centerLine(line string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(line)
}

// branchChoiceLabels 返回分支选择列表的展示文本。
func (m model) branchChoiceLabels() []string {
	defaultLabel := "使用远端默认分支"
	if strings.TrimSpace(m.pendingDefaultBranch) != "" {
		defaultLabel = fmt.Sprintf("%s (%s)", defaultLabel, m.pendingDefaultBranch)
	}
	choices := []string{defaultLabel}
	choices = append(choices, m.filteredBranchNames()...)
	return choices
}

// branchWindow 返回当前终端高度下可见的分支列表区间。
func (m model) branchWindow(total int) (int, int) {
	visible := max(3, min(8, m.height-13))
	if total <= visible {
		return 0, total
	}
	start := m.selectedBranch - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}

// emptyPlaceholder 在展示字段为空时返回占位文本。
func emptyPlaceholder(value string, placeholder string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return placeholder
	}
	return value
}

// helpView 渲染快捷键帮助。
func (m model) helpView() string {
	return strings.Join([]string{
		"ctrl-w h/l 切换 registry/tree   j/k 移动   q 退出",
		"registry: l 打开仓库   a 添加 registry/选分支   e 编辑 registry   r 重载列表   d 删除仓库+cache   u 更新仓库 cache",
		"tree: l 展开/收起目录   o 进入目录作为 root   h 返回上级 root",
		"tree: i 下载到启动目录   I 输入目标目录",
		"/ 搜索当前仓库路径   Esc 关闭搜索/错误/确认   ? 关闭帮助",
	}, "\n")
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

// narrowView 渲染终端过窄时的提示。
func (m model) narrowView() string {
	message := fmt.Sprintf("terminal too narrow: need at least %d columns", minTerminalWidth)
	return fitPlainLine(message, m.width) + "\n" + renderStatusLine(m.status, "? help", m.width)
}

// renderStatusLine 将状态和快捷键压缩到终端宽度内。
func renderStatusLine(status string, help string, width int) string {
	if width <= 0 {
		return ""
	}
	status = firstLine(status)
	help = firstLine(help)
	if help == "" {
		return statusTextStyle.Render(truncateVisible(status, width))
	}

	helpText := " | " + help
	if lipgloss.Width(status)+lipgloss.Width(helpText) <= width {
		return statusTextStyle.Render(status) + statusHelpStyle.Render(helpText)
	}

	compactHelp := " | ? help"
	if lipgloss.Width(compactHelp) < width {
		statusWidth := max(1, width-lipgloss.Width(compactHelp))
		return statusTextStyle.Render(truncateVisible(status, statusWidth)) + statusHelpStyle.Render(compactHelp)
	}
	return statusTextStyle.Render(truncateVisible(status, width))
}

// fitPlainLine 将普通文本限制为单行并按宽度截断。
func fitPlainLine(line string, width int) string {
	return truncateVisible(firstLine(line), width)
}

// firstLine 取首行文本，避免多行错误或输入破坏状态栏布局。
func firstLine(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return text[:index]
	}
	return text
}

// truncateVisible 按终端可见宽度截断单行文本。
func truncateVisible(text string, width int) string {
	text = firstLine(text)
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	suffix := "..."
	limit := width - lipgloss.Width(suffix)
	var builder strings.Builder
	for _, char := range text {
		next := builder.String() + string(char)
		if lipgloss.Width(next) > limit {
			break
		}
		builder.WriteRune(char)
	}
	return builder.String() + suffix
}

// limitLines 截取最多 limit 行。
func limitLines(lines []string, limit int) []string {
	if limit < 0 {
		limit = 0
	}
	if len(lines) <= limit {
		return lines
	}
	return lines[:limit]
}

// visibleWindow 返回围绕当前光标的可见列表区间。
func visibleWindow(total int, selected int, limit int) (int, int) {
	if total <= 0 || limit <= 0 {
		return 0, 0
	}
	if total <= limit {
		return 0, total
	}
	selected = clampCursor(selected, total)
	start := selected - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > total {
		start = total - limit
	}
	return start, start + limit
}

// paneHeight 返回主栏目可用高度。
func (m model) paneHeight() int {
	return max(8, m.height-3)
}

// paneBodyRows 返回栏目边框内的可用文本行数。
func (m model) paneBodyRows() int {
	return max(1, m.paneHeight()-2)
}

// paneItemRows 返回扣除栏目标题后的可用列表行数。
func (m model) paneItemRows() int {
	return max(1, m.paneBodyRows()-1)
}

// paneContentWidth 返回栏目边框和左右 padding 内的文本宽度。
func paneContentWidth(width int) int {
	return max(1, width-4)
}

// paneWidths 返回左右两栏宽度，registry 保持较窄导航区域。
func paneWidths(totalWidth int) (int, int) {
	leftWidth := max(20, totalWidth*25/100)
	rightWidth := max(40, totalWidth-leftWidth-paneGapAllowance)
	return leftWidth, rightWidth
}
