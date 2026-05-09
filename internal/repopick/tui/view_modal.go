package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/finger/repo-pick/internal/repopick/config"
)

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
	repoURL := m.pendingURL
	if m.mode == modeAddURL {
		repoURL = m.input.View()
	}
	branch := m.pendingBranch
	if strings.TrimSpace(branch) == "" {
		branch = "使用远端默认分支"
	}
	activeRow := -1
	if m.mode == modeAddName {
		activeRow = 0
	}
	if m.mode == modeAddURL {
		activeRow = 1
	}
	if m.mode == modeAddBranch {
		activeRow = 2
	}
	// 输入框自带 "> " 光标提示，非聚焦值对齐到提示后的内容起点，避免切换焦点时字段内容左右跳动。
	if m.mode == modeAddName && strings.TrimSpace(repoURL) != "" {
		repoURL = "  " + repoURL
	}
	if m.mode == modeAddURL && strings.TrimSpace(name) != "" {
		name = "  " + name
	}
	fieldRows := [][2]string{
		{"name", name},
		{"url", repoURL},
	}
	fieldResolver := func(row int, col int) lipgloss.Style {
		if col == 0 {
			return modalFieldLabelStyle
		}
		if row == activeRow {
			return selectedLineStyle
		}
		return modalFieldValueStyle
	}
	lines = append(lines, renderKeyValueTable(fieldRows, innerWidth, fieldResolver))

	if m.mode == modeAddBranch {
		lines = append(lines, m.branchSelectView())
	} else {
		lines = append(lines, renderKeyValueTable([][2]string{{"branch", branch}}, innerWidth, func(row int, col int) lipgloss.Style {
			if col == 0 {
				return modalFieldLabelStyle
			}
			if activeRow == 2 {
				return selectedLineStyle
			}
			return modalFieldValueStyle
		}))
	}
	lines = append(lines, modalDividerLine(innerWidth), modalHintStyle.Render("Tab/Shift+Tab 切焦点   输入搜索分支   Enter 确认   Esc 取消"))

	return renderModal(strings.Join(lines, "\n"), modalWidth)
}

// deleteRepositoryConfirmModalView 渲染 registry 删除确认弹框。
func (m model) deleteRepositoryConfirmModalView() string {
	modalWidth := max(32, min(76, m.width-4))
	innerWidth := max(24, modalWidth-6)
	var repo config.Repository
	if m.pendingConfirm != nil {
		repo = m.pendingConfirm.repository
	}
	branch := strings.TrimSpace(repo.Branch)
	if branch == "" {
		branch = "远端默认分支"
	}

	lines := []string{
		centerLine(modalTitleStyle.Render("删除 Registry"), innerWidth),
		modalDescStyle.Render("删除后会同步移除本地 cache。"),
		modalDividerLine(innerWidth),
		renderKeyValueTable([][2]string{
			{"name", repo.Name},
			{"url", repo.URL},
			{"branch", branch},
		}, innerWidth, func(row int, col int) lipgloss.Style {
			if col == 0 {
				return modalFieldLabelStyle
			}
			return modalFieldValueStyle
		}),
		modalDividerLine(innerWidth),
		modalHintStyle.Render("y 确认删除   n/Esc 取消"),
	}
	return renderModal(strings.Join(lines, "\n"), modalWidth)
}

// modalDividerLine 渲染新增弹框中的横向分隔线。
func modalDividerLine(width int) string {
	return modalDividerStyle.Render(strings.Repeat("-", max(12, width)))
}

// branchSelectView 生成 registry 表单里的分支选择块。
func (m model) branchSelectView() string {
	if m.branchLoading {
		return renderKeyValueTable([][2]string{{"branch", m.branchLoadingStatus() + "..."}}, max(24, min(72, m.width-10)), func(row int, col int) lipgloss.Style {
			if col == 0 {
				return modalFieldLabelStyle
			}
			return modalDescStyle
		})
	}

	query := m.branchQuery
	if m.mode == modeAddBranch {
		query = m.input.View()
	}
	width := max(24, min(72, m.width-10))
	parts := []string{
		renderKeyValueTable([][2]string{
			{"branch", ""},
			{"search", emptyPlaceholder(query, "-")},
		}, width, func(row int, col int) lipgloss.Style {
			if col == 0 {
				return modalFieldLabelStyle
			}
			return modalFieldValueStyle
		}),
	}
	if m.branchErr != nil {
		message := "  获取失败，Enter 使用远端默认分支"
		if m.editingRepositoryActive && strings.TrimSpace(m.pendingBranch) != "" {
			message = "  获取失败，Enter 保留当前分支"
		}
		parts = append(parts, modalDescStyle.Render(message))
		return strings.Join(parts, "\n")
	}
	parts = append(parts, modalDescStyle.Render(fmt.Sprintf("  已获取 %d 个分支", len(m.pendingBranches))))

	choices := m.branchChoiceLabels()
	if strings.TrimSpace(m.branchQuery) != "" && len(m.filteredBranchNames()) == 0 {
		parts = append(parts, modalDescStyle.Render("  无匹配分支"))
	}
	start, end := m.branchWindow(len(choices))
	rows := make([][]string, 0, end-start)
	for i := start; i < end; i++ {
		cursor := " "
		if i == m.selectedBranch {
			cursor = m.selectionCursor()
		}
		rows = append(rows, []string{cursor, choices[i]})
	}
	if start > 0 {
		parts = append(parts, modalDescStyle.Render("..."))
	}
	selected := m.selectedBranch - start
	if selected < 0 || selected >= len(rows) {
		selected = -1
	}
	parts = append(parts, renderSelectableTable(rows, selected, width, func(row int, col int) lipgloss.Style {
		if col == 0 {
			return modalFieldLabelStyle
		}
		return modalFieldValueStyle
	}))
	if end < len(choices) {
		parts = append(parts, modalDescStyle.Render("..."))
	}
	return strings.Join(parts, "\n")
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
	width := max(36, min(84, m.width-4))
	innerWidth := max(28, width-6)
	lines := []string{
		centerLine(modalTitleStyle.Render("快捷键帮助"), innerWidth),
		modalDescStyle.Render("当前界面支持键盘优先操作；底栏只展示最常用入口。"),
		modalDividerLine(innerWidth),
		helperSectionTable([]helpSection{{name: "通用", rows: [][2]string{
			{"Tab", "切换 registry/tree 焦点"},
			{"j/k 或 ↑/↓", "移动光标"},
			{"Enter 或 l", "选择、打开或展开"},
			{"Esc", "关闭搜索、错误或确认状态"},
			{"q", "退出"},
		}}}, innerWidth),
		helperSectionTable([]helpSection{{name: "Registry", rows: [][2]string{
			{"a", "添加 registry 并选择分支"},
			{"e", "编辑当前 registry"},
			{"r", "重新加载 registry 列表"},
			{"d", "删除当前 registry 和 cache"},
			{"u", "更新当前 repository cache"},
		}}}, innerWidth),
		helperSectionTable([]helpSection{{name: "Repository Tree", rows: [][2]string{
			{"h 或 ←", "返回上级 root"},
			{"o", "进入目录作为 root"},
			{"e", "用 EDITOR 打开当前文件"},
			{"i", "下载到启动目录"},
			{"I", "输入目标目录后下载"},
			{"/", "搜索当前仓库路径"},
			{"?", "关闭帮助"},
		}}}, innerWidth),
	}

	if maxLines := max(1, m.height-4); len(lines) > maxLines {
		lines = append(lines[:maxLines-1], modalDescStyle.Render("..."))
	}
	content := strings.Join(lines, "\n")
	return renderModal(content, width)
}

// helpSection 是帮助分组定义。
type helpSection struct {
	name string
	rows [][2]string
}

// helperSectionTable 渲染一组帮助文本。
func helperSectionTable(sections []helpSection, width int) string {
	parts := make([]string, 0, len(sections)*2)
	for _, section := range sections {
		parts = append(parts, helpSectionStyle.Render(section.name))
		parts = append(parts, renderHelpTable(section.rows, width))
	}
	return strings.Join(parts, "\n")
}
