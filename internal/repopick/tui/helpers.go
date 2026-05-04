package tui

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

var operationFrames = []string{"-", "\\", "|", "/"}

// clampCursor 将光标限制在可见列表范围内。
func clampCursor(cursor int, length int) int {
	if length == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

// indexForPath 返回指定路径在条目列表中的位置，未找到时返回 0。
func indexForPath(entries []app.EntryResult, entryPath string) int {
	entryPath = strings.TrimSpace(entryPath)
	if entryPath == "" {
		return clampCursor(0, len(entries))
	}
	for i, entry := range entries {
		if entry.Path == entryPath {
			return i
		}
	}
	return clampCursor(0, len(entries))
}

// resetTreeRoot 重置右侧树 root 和已展开节点缓存。
func (m *model) resetTreeRoot(rootPath string, entries []app.EntryResult) {
	m.entries = entries
	m.treeChildren = map[string][]app.EntryResult{rootPath: entries}
	m.expandedPaths = map[string]bool{}
}

// ensureTreeMaps 确保树形视图缓存 map 已初始化。
func (m *model) ensureTreeMaps() {
	if m.treeChildren == nil {
		m.treeChildren = map[string][]app.EntryResult{m.currentPath: m.entries}
	}
	if m.expandedPaths == nil {
		m.expandedPaths = map[string]bool{}
	}
}

// visibleTreeRows 返回当前 root 下展开后的可见树行。
func (m model) visibleTreeRows() []treeRow {
	rows := []treeRow{}
	m.appendTreeRows(&rows, m.currentPath, nil)
	return rows
}

// appendTreeRows 递归追加指定目录下的可见树行。
func (m model) appendTreeRows(rows *[]treeRow, dirPath string, ancestorsLast []bool) {
	children := m.treeChildren[dirPath]
	if children == nil && dirPath == m.currentPath {
		children = m.entries
	}
	for i, entry := range children {
		last := i == len(children)-1
		expanded := entry.Type == app.EntryDir && m.expandedPaths[entry.Path]
		*rows = append(*rows, treeRow{entry: entry, prefix: treePrefix(ancestorsLast, last), expanded: expanded})
		if expanded {
			m.appendTreeRows(rows, entry.Path, append(ancestorsLast, last))
		}
	}
}

// treePrefix 返回一行树节点的连接线前缀。
func treePrefix(ancestorsLast []bool, currentLast bool) string {
	var builder strings.Builder
	for _, last := range ancestorsLast {
		if last {
			builder.WriteString("    ")
			continue
		}
		builder.WriteString("│   ")
	}
	if currentLast {
		builder.WriteString("└── ")
	} else {
		builder.WriteString("├── ")
	}
	return builder.String()
}

// selectedVisibleEntry 返回右侧当前选中的可见条目。
func (m model) selectedVisibleEntry() (app.EntryResult, bool) {
	entries := m.visibleEntries()
	if len(entries) == 0 {
		return app.EntryResult{}, false
	}
	return entries[clampCursor(m.selectedEntry, len(entries))], true
}

// sameRepository 判断两个仓库配置是否指向同一个 registry 条目。
func sameRepository(a config.Repository, b config.Repository) bool {
	return strings.TrimSpace(a.Name) == strings.TrimSpace(b.Name) &&
		strings.TrimSpace(a.URL) == strings.TrimSpace(b.URL) &&
		strings.TrimSpace(a.Branch) == strings.TrimSpace(b.Branch)
}

// displayPath 将空仓库路径展示为根目录。
func displayPath(repoPath string) string {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return "/"
	}
	return "/" + repoPath
}

// parentPath 返回仓库内路径的父目录。
func parentPath(entryPath string) string {
	entryPath = strings.TrimSpace(entryPath)
	if entryPath == "" {
		return ""
	}
	parent := path.Dir(entryPath)
	if parent == "." {
		return ""
	}
	return parent
}

// entryIcon 返回条目类型对应的简短展示符号。
func entryIcon(entry app.EntryResult) string {
	if entry.Type == app.EntryDir {
		return "d"
	}
	return "f"
}

// entrySize 返回条目大小展示文本。
func entrySize(entry app.EntryResult) string {
	if entry.Type == app.EntryDir {
		return "-"
	}
	return fmt.Sprintf("%d", entry.Size)
}

// errorsIsTargetExists 判断错误是否是目标已存在。
func errorsIsTargetExists(err error) bool {
	return errors.Is(err, app.ErrTargetExists)
}

// clearAddState 清理新增 registry 弹框的临时状态。
func (m *model) clearAddState() {
	m.mode = modeNormal
	m.pendingName = ""
	m.pendingURL = ""
	m.pendingBranches = nil
	m.pendingDefaultBranch = ""
	m.branchQuery = ""
	m.selectedBranch = 0
	m.branchLoading = false
	m.branchErr = nil
	m.input.Blur()
}

// branchChoiceCount 返回分支选择列表的可选项数量。
func (m model) branchChoiceCount() int {
	return len(m.branchChoiceNames())
}

// selectedBranchName 返回当前选择的显式分支；默认分支项返回空字符串。
func (m model) selectedBranchName() string {
	names := m.branchChoiceNames()
	if m.selectedBranch <= 0 || m.selectedBranch >= len(names) {
		return ""
	}
	return strings.TrimSpace(names[m.selectedBranch])
}

// branchChoiceNames 返回当前搜索条件下的分支原始名称；第一项为空表示使用默认分支。
func (m model) branchChoiceNames() []string {
	choices := []string{""}
	return append(choices, m.filteredBranchNames()...)
}

// filteredBranchNames 返回匹配分支搜索文本的远端分支名称。
func (m model) filteredBranchNames() []string {
	query := strings.ToLower(strings.TrimSpace(m.branchQuery))
	if query == "" {
		return m.pendingBranches
	}
	branches := make([]string, 0, len(m.pendingBranches))
	for _, branch := range m.pendingBranches {
		if fuzzyRuneMatch(strings.ToLower(branch), query) {
			branches = append(branches, branch)
		}
	}
	return branches
}

// defaultBranchSelection 返回当前搜索条件下默认选中的分支行。
func (m model) defaultBranchSelection() int {
	if strings.TrimSpace(m.branchQuery) != "" && len(m.filteredBranchNames()) > 0 {
		return 1
	}
	return 0
}

// fuzzyRuneMatch 判断 query 字符是否按顺序出现在 target 中。
func fuzzyRuneMatch(target string, query string) bool {
	if query == "" {
		return true
	}
	targetRunes := []rune(target)
	cursor := 0
	for _, queryRune := range query {
		found := false
		for cursor < len(targetRunes) {
			if targetRunes[cursor] == queryRune {
				cursor++
				found = true
				break
			}
			cursor++
		}
		if !found {
			return false
		}
	}
	return true
}

// startOperation 标记一个长耗时操作开始运行。
func (m *model) startOperation(kind operationKind, label string) {
	m.operationKind = kind
	m.operationLabel = strings.TrimSpace(label)
	m.operationPercent = -1
	m.operationFrame = 0
}

// clearOperation 按类型清理长耗时操作状态。
func (m *model) clearOperation(kind operationKind) {
	if m.operationKind != kind {
		return
	}
	m.operationKind = operationNone
	m.operationLabel = ""
	m.operationPercent = -1
	m.operationFrame = 0
	m.operationMessages = nil
}

// operationStatus 返回当前长耗时操作的状态栏文本。
func (m model) operationStatus() string {
	if m.operationKind == operationNone {
		return ""
	}
	frame := operationFrames[m.operationFrame%len(operationFrames)]
	if m.operationLabel == "" {
		return frame + " working"
	}
	return frame + " " + m.operationLabel
}

// treeOperationInProgress 判断当前长耗时操作是否应在右侧树面板展示。
func (m model) treeOperationInProgress() bool {
	return m.operationKind == operationOpen || m.operationKind == operationUpdate
}

// statusOperationLine 返回仍应放在底部状态栏的长耗时操作文本。
func (m model) statusOperationLine() string {
	if m.operationKind != operationDownload {
		return ""
	}
	return m.operationStatus()
}

// formatOperationProgress 合成状态栏展示的 Git 进度文本。
func formatOperationProgress(baseLabel string, event app.ProgressEvent) string {
	baseLabel = strings.TrimSpace(baseLabel)
	text := strings.TrimSpace(event.Text)
	if text == "" {
		return baseLabel
	}
	if baseLabel == "" {
		return text
	}
	return fmt.Sprintf("%s - %s", baseLabel, text)
}

// min 返回两个整数中的较小值。
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的较大值。
func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
