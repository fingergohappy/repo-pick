package tui

import (
	"fmt"
	"os"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

// handleKey 处理 TUI 快捷键。
func (m model) handleKey(msg tea.KeyMsg) (model, tea.Cmd) {
	if m.mode != modeNormal {
		return m.handleModeKey(msg)
	}
	if m.pendingWindowCommand {
		return m.handleWindowCommandKey(msg)
	}
	if m.showHelp && msg.String() != "?" {
		m.showHelp = false
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "esc":
		return m.closeTransientState()
	case "tab":
		return m.focusNextPane()
	case "shift+tab":
		return m.focusPreviousPane()
	case "ctrl+w":
		m.pendingWindowCommand = true
		m.status = "ctrl-w: h/l 切换窗口"
		return m, nil
	case "/":
		return m.startSearch()
	}

	if m.focus == focusRegistry {
		return m.handleRegistryKey(msg)
	}
	return m.handleTreeKey(msg)
}

// handleWindowCommandKey 处理 ctrl-w 后的窗口切换按键。
func (m model) handleWindowCommandKey(msg tea.KeyMsg) (model, tea.Cmd) {
	m.pendingWindowCommand = false
	switch msg.String() {
	case "h":
		m.focus = focusRegistry
		m.status = "已切到 registry"
		return m, nil
	case "l":
		if !m.repoOpened {
			return m.openSelectedRepository()
		}
		m.focus = focusTree
		m.status = "已切到 repository tree"
		return m, nil
	default:
		m.status = "已取消窗口切换"
		return m, nil
	}
}

// focusNextPane 将焦点切到下一个主栏目。
func (m model) focusNextPane() (model, tea.Cmd) {
	if m.focus == focusRegistry {
		if !m.repoOpened {
			return m.openSelectedRepository()
		}
		m.focus = focusTree
		m.status = "已切到 repository tree"
		return m, nil
	}
	m.focus = focusRegistry
	m.status = "已切到 registry"
	return m, nil
}

// focusPreviousPane 将焦点切到上一个主栏目。
func (m model) focusPreviousPane() (model, tea.Cmd) {
	return m.focusNextPane()
}

// handleRegistryKey 处理 registry 焦点下的快捷键。
func (m model) handleRegistryKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		return m.moveRegistrySelection(1)
	case "k", "up":
		return m.moveRegistrySelection(-1)
	case "l", "right", "enter":
		return m.openSelectedRepository()
	case "a":
		return m.startAddName()
	case "e":
		return m.startEditName()
	case "r":
		m.status = "正在刷新 registry"
		return m, m.listRepositoriesCommand()
	case "d":
		return m.startDeleteConfirm()
	case "u":
		return m.updateSelectedRepository()
	default:
		return m, nil
	}
}

// moveRegistrySelection 移动左侧 registry 光标并启动右侧选中提示动画。
func (m model) moveRegistrySelection(delta int) (model, tea.Cmd) {
	previous := m.selectedRepo
	m.selectedRepo = clampCursor(m.selectedRepo+delta, len(m.repositories))
	if m.selectedRepo == previous {
		return m, nil
	}
	return m.startRegistrySelectionPreview()
}

// startRegistrySelectionPreview 标记当前 registry 选中提示动画开始。
func (m model) startRegistrySelectionPreview() (model, tea.Cmd) {
	repo, ok := m.activeRepository()
	if !ok {
		return m, nil
	}
	m.registrySelectionFrame = 0
	m.registrySelectionID = m.nextRequestID()
	m.status = fmt.Sprintf("已选择 registry: %s", repositoryLabel(repo))
	return m, registrySelectionTickCommand(m.registrySelectionID)
}

// handleTreeKey 处理目录树焦点下的快捷键。
func (m model) handleTreeKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "h", "left":
		return m.openParentDirectory()
	case "j", "down":
		m.selectedEntry = clampCursor(m.selectedEntry+1, len(m.visibleEntries()))
		return m, nil
	case "k", "up":
		m.selectedEntry = clampCursor(m.selectedEntry-1, len(m.visibleEntries()))
		return m, nil
	case "l", "right", "enter":
		return m.toggleSelectedTreeEntry()
	case "o":
		return m.openSelectedEntry()
	case "e":
		return m.editSelectedFile()
	case "i":
		return m.downloadSelectedEntry(m.sessionCWD)
	case "I":
		return m.startTargetDirInput()
	default:
		return m, nil
	}
}

// handleModeKey 处理输入框和确认框状态下的按键。
func (m model) handleModeKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch m.mode {
	case modeConfirmDelete, modeConfirmOverwrite:
		return m.handleConfirmKey(msg)
	case modeAddBranch:
		return m.handleAddBranchKey(msg)
	}

	if m.mode == modeAddName || m.mode == modeAddURL {
		switch msg.String() {
		case "up", "shift+tab":
			return m.moveAddFocus(-1)
		case "down", "tab":
			return m.moveAddFocus(1)
		}
	}

	switch msg.String() {
	case "esc":
		if m.isAddMode() {
			m.clearAddState()
		} else {
			m.mode = modeNormal
			m.input.Blur()
		}
		m.status = "已取消"
		return m, nil
	case "enter":
		return m.commitInput()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleAddBranchKey 处理 registry 表单中的分支选择按键。
func (m model) handleAddBranchKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.clearAddState()
		m.status = "已取消"
		return m, nil
	case "tab":
		return m.moveAddFocus(1)
	case "shift+tab":
		return m.moveAddFocus(-1)
	case "down":
		if !m.branchLoading {
			m.selectedBranch = clampCursor(m.selectedBranch+1, m.branchChoiceCount())
		}
		return m, nil
	case "up":
		if m.branchLoading || m.selectedBranch == 0 {
			return m.moveAddFocus(-1)
		}
		if !m.branchLoading {
			m.selectedBranch = clampCursor(m.selectedBranch-1, m.branchChoiceCount())
		}
		return m, nil
	case "ctrl+u":
		m.input.SetValue("")
		m.branchQuery = ""
		m.selectedBranch = m.defaultBranchSelection()
		return m, nil
	case "enter":
		if m.branchLoading {
			m.status = "正在获取分支"
			return m, nil
		}
		name := strings.TrimSpace(m.pendingName)
		if name == "" {
			m.status = "registry name 不能为空"
			return m.focusAddName(name)
		}
		repoURL := strings.TrimSpace(m.pendingURL)
		if repoURL == "" {
			m.status = "repo URL 不能为空"
			return m.focusAddURL(repoURL)
		}
		branch := m.selectedBranchName()
		if m.editingRepositoryActive && m.branchErr != nil {
			branch = strings.TrimSpace(m.pendingBranch)
		}
		m.mode = modeNormal
		if m.editingRepositoryActive {
			return m, m.editRepositoryCommand(m.editingRepository, name, repoURL, branch)
		}
		return m, m.addRepositoryCommand(name, repoURL, branch)
	default:
		return m.updateBranchSearch(msg)
	}
}

// handleConfirmKey 处理删除和覆盖确认框按键。
func (m model) handleConfirmKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.mode = modeNormal
		m.pendingConfirm = nil
		m.pendingDownload = nil
		m.status = "已取消"
		return m, nil
	case "y":
		confirm := m.pendingConfirm
		m.mode = modeNormal
		m.pendingConfirm = nil
		if confirm == nil {
			return m, nil
		}
		if confirm.kind == confirmDelete {
			return m, m.removeRepositoryCommand(confirm.repository)
		}
		if m.pendingDownload == nil {
			m.pendingDownload = nil
			return m, nil
		}
		request := *m.pendingDownload
		m.pendingDownload = nil
		baseLabel := fmt.Sprintf("downloading %s", request.entry.Name)
		messages := make(chan tea.Msg, 64)
		m.operationMessages = messages
		operationID := m.startOperation(operationDownload, baseLabel)
		return m, tea.Batch(operationTickCommand(operationID), m.listenOperationCommand(operationID), m.downloadEntryProgressCommand(operationID, request, true, messages, baseLabel))
	default:
		return m, nil
	}
}

// closeTransientState 关闭搜索结果或清理错误提示。
func (m model) closeTransientState() (model, tea.Cmd) {
	if m.showingSearch {
		m.showingSearch = false
		m.searchResults = nil
		m.selectedEntry = clampCursor(0, len(m.visibleEntries()))
		m.status = "已关闭搜索"
		return m, nil
	}
	if m.err != nil {
		m.err = nil
		m.status = "ready"
	}
	return m, nil
}

// openSelectedRepository 打开左栏当前选中的仓库。
func (m model) openSelectedRepository() (model, tea.Cmd) {
	repo, ok := m.activeRepository()
	if !ok {
		m.status = "没有 registry"
		return m, nil
	}
	baseLabel := fmt.Sprintf("loading repo cache: %s", repo.Name)
	m.status = fmt.Sprintf("正在打开 %s", repo.Name)
	messages := make(chan tea.Msg, 64)
	m.operationMessages = messages
	operationID := m.startOperation(operationOpen, baseLabel)
	m.entriesRequestID = operationID
	m.searchRequestID = operationID
	return m, tea.Batch(operationTickCommand(operationID), m.listenOperationCommand(operationID), m.openRepositoryProgressCommand(operationID, repo, messages, baseLabel))
}

// updateSelectedRepository 删除并重新下载左栏当前选中仓库的 cache。
func (m model) updateSelectedRepository() (model, tea.Cmd) {
	repo, ok := m.activeRepository()
	if !ok {
		m.status = "没有 registry"
		return m, nil
	}
	dirPath := ""
	if m.repoOpened && m.openedRepo.Name == repo.Name {
		dirPath = m.currentPath
	}
	m.status = fmt.Sprintf("正在更新 %s", repo.Name)
	baseLabel := fmt.Sprintf("updating repo cache: %s", repo.Name)
	messages := make(chan tea.Msg, 64)
	m.operationMessages = messages
	operationID := m.startOperation(operationUpdate, baseLabel)
	m.entriesRequestID = operationID
	m.searchRequestID = operationID
	return m, tea.Batch(operationTickCommand(operationID), m.listenOperationCommand(operationID), m.updateRepositoryProgressCommand(operationID, repo, dirPath, messages, baseLabel))
}

// startAddName 进入新增 registry 名称输入模式。
func (m model) startAddName() (model, tea.Cmd) {
	m.clearAddState()
	return m.focusAddName("")
}

// startEditName 进入编辑当前 registry 的名称输入模式。
func (m model) startEditName() (model, tea.Cmd) {
	repo, ok := m.activeRepository()
	if !ok {
		m.status = "没有可编辑的 registry"
		return m, nil
	}
	m.clearAddState()
	m.editingRepository = repo
	m.editingRepositoryActive = true
	m.pendingName = repo.Name
	m.pendingURL = repo.URL
	m.pendingBranch = repo.Branch
	return m.focusAddName(repo.Name)
}

// focusAddName 将 registry 表单焦点切到 name 输入框。
func (m model) focusAddName(value string) (model, tea.Cmd) {
	m.mode = modeAddName
	m.input.Placeholder = "name"
	m.input.SetValue(value)
	return m.focusInput()
}

// focusAddURL 将 registry 表单焦点切到 URL 输入框。
func (m model) focusAddURL(value string) (model, tea.Cmd) {
	m.mode = modeAddURL
	m.input.Placeholder = "repo url"
	m.input.SetValue(value)
	return m.focusInput()
}

// moveAddFocus 在 registry 表单的 name、URL 和 branch 区域之间移动焦点。
func (m model) moveAddFocus(delta int) (model, tea.Cmd) {
	switch m.mode {
	case modeAddName:
		m.pendingName = strings.TrimSpace(m.input.Value())
		if delta > 0 {
			return m.focusAddURL(m.pendingURL)
		}
	case modeAddURL:
		m.pendingURL = strings.TrimSpace(m.input.Value())
		if delta < 0 {
			return m.focusAddName(m.pendingName)
		}
		if delta > 0 {
			return m.startAddBranchSelection(m.pendingURL)
		}
	case modeAddBranch:
		m.branchQuery = strings.TrimSpace(m.input.Value())
		if delta < 0 {
			return m.focusAddURL(m.pendingURL)
		}
	}
	return m, nil
}

// startAddBranchSelection 进入 registry 表单分支选择区域，必要时先异步读取远端分支。
func (m model) startAddBranchSelection(repoURL string) (model, tea.Cmd) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		m.status = "repo URL 不能为空"
		return m.focusAddURL(repoURL)
	}

	urlChanged := repoURL != m.pendingURL
	if urlChanged {
		m.branchQuery = ""
	}
	needsLoad := urlChanged || (!m.branchLoading && len(m.pendingBranches) == 0 && m.branchErr == nil && strings.TrimSpace(m.pendingDefaultBranch) == "")
	m.pendingURL = repoURL
	m.mode = modeAddBranch
	m.input.Placeholder = "branch search"
	m.input.SetValue(m.branchQuery)
	_ = m.input.Focus()
	if !needsLoad {
		m.selectedBranch = clampCursor(m.selectedBranch, m.branchChoiceCount())
		return m, nil
	}

	m.pendingBranches = nil
	m.pendingDefaultBranch = ""
	m.selectedBranch = 0
	m.branchLoading = true
	m.branchErr = nil
	m.status = "正在获取远端分支"
	return m, m.listBranchesCommand(repoURL)
}

// updateBranchSearch 将普通输入写入分支搜索框并刷新分支选中项。
func (m model) updateBranchSearch(msg tea.KeyMsg) (model, tea.Cmd) {
	oldQuery := m.branchQuery
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.branchQuery = strings.TrimSpace(m.input.Value())
	if m.branchQuery != oldQuery {
		m.selectedBranch = m.defaultBranchSelection()
	}
	return m, cmd
}

// startDeleteConfirm 进入 registry 删除确认模式。
func (m model) startDeleteConfirm() (model, tea.Cmd) {
	repo, ok := m.activeRepository()
	if !ok {
		m.status = "没有可删除的 registry"
		return m, nil
	}
	m.mode = modeConfirmDelete
	m.pendingConfirm = &confirmState{kind: confirmDelete, repository: repo}
	m.status = fmt.Sprintf("Delete %s and cache? y/n", repo.Name)
	return m, nil
}

// startSearch 进入当前仓库路径搜索输入模式。
func (m model) startSearch() (model, tea.Cmd) {
	if !m.repoOpened {
		m.status = "请先打开 repository"
		return m, nil
	}
	m.mode = modeSearch
	m.input.Placeholder = "search paths"
	m.input.SetValue(m.searchQuery)
	return m.focusInput()
}

// startTargetDirInput 进入下载目标目录输入模式。
func (m model) startTargetDirInput() (model, tea.Cmd) {
	if !m.repoOpened || len(m.visibleEntries()) == 0 {
		m.status = "没有可下载条目"
		return m, nil
	}
	m.mode = modeTargetDir
	m.input.Placeholder = "target directory"
	m.input.SetValue(m.sessionCWD)
	return m.focusInput()
}

// commitInput 根据当前输入模式提交文本内容。
func (m model) commitInput() (model, tea.Cmd) {
	mode := m.mode
	value := strings.TrimSpace(m.input.Value())
	m.mode = modeNormal
	m.input.Blur()

	switch mode {
	case modeAddName:
		if value == "" {
			m.status = "registry name 不能为空"
			return m, nil
		}
		m.pendingName = value
		return m.focusAddURL(m.pendingURL)
	case modeAddURL:
		return m.startAddBranchSelection(value)
	case modeSearch:
		if value == "" {
			m.invalidateSearchRequest()
			m.showingSearch = false
			m.searchResults = nil
			m.status = "已清空搜索"
			return m, nil
		}
		requestID := m.nextRequestID()
		m.searchRequestID = requestID
		return m, m.searchEntriesCommand(requestID, m.openedRepo, value)
	case modeTargetDir:
		return m.downloadSelectedEntry(value)
	default:
		return m, nil
	}
}

// openParentDirectory 返回当前目录的上级目录。
func (m model) openParentDirectory() (model, tea.Cmd) {
	if !m.repoOpened {
		m.status = "请先打开 repository"
		return m, nil
	}
	if m.showingSearch {
		m.showingSearch = false
		m.searchResults = nil
		return m, nil
	}
	if m.currentPath == "" {
		m.status = "已经在根目录"
		return m, nil
	}
	parent := path.Dir(m.currentPath)
	if parent == "." {
		parent = ""
	}
	return m.startEntriesRequest(m.openedRepo, parent, m.currentPath)
}

// openSelectedEntry 进入目录或定位搜索结果中的文件。
func (m model) openSelectedEntry() (model, tea.Cmd) {
	entry, ok := m.selectedVisibleEntry()
	if !m.repoOpened || !ok {
		m.status = "没有可打开条目"
		return m, nil
	}
	if entry.Type != app.EntryDir {
		if m.showingSearch {
			return m.startEntriesRequest(m.openedRepo, parentPath(entry.Path), entry.Path)
		}
		m.status = "当前条目是文件"
		return m, nil
	}
	return m.startEntriesRequest(m.openedRepo, entry.Path, "")
}

// editSelectedFile 使用 EDITOR 打开右侧当前选中的文件。
func (m model) editSelectedFile() (model, tea.Cmd) {
	entry, ok := m.selectedVisibleEntry()
	if !m.repoOpened || !ok {
		m.status = "没有可打开条目"
		return m, nil
	}
	if entry.Type != app.EntryFile {
		m.status = "只能用 EDITOR 打开文件"
		return m, nil
	}
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		m.status = "EDITOR 未设置"
		return m, nil
	}
	result, err := m.svc.ResolveEntryPath(m.ctx, app.ResolveEntryPathRequest{Repository: m.openedRepo, Entry: entry})
	if err != nil {
		m.err = err
		m.status = "解析文件路径失败"
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("正在用 editor 打开 %s", entry.Name)
	return m, editorProcessCommand(editor, result.Path, entry)
}

// toggleSelectedTreeEntry 展开或收起右侧树中当前选中的目录。
func (m model) toggleSelectedTreeEntry() (model, tea.Cmd) {
	if !m.repoOpened {
		m.status = "请先打开 repository"
		return m, nil
	}
	if m.showingSearch {
		m.status = "搜索结果不支持展开"
		return m, nil
	}
	entry, ok := m.selectedVisibleEntry()
	if !ok {
		m.status = "没有可展开条目"
		return m, nil
	}
	if entry.Type != app.EntryDir {
		m.status = "文件不能展开"
		return m, nil
	}
	if entry.Path == m.currentPath {
		m.status = "已经在当前 root"
		return m, nil
	}

	m.ensureTreeMaps()
	if m.expandedPaths[entry.Path] {
		m.expandedPaths[entry.Path] = false
		m.selectedEntry = clampCursor(m.selectedEntry, len(m.visibleEntries()))
		m.status = fmt.Sprintf("已收起 %s", displayPath(entry.Path))
		return m, nil
	}
	if _, ok := m.treeChildren[entry.Path]; ok {
		m.expandedPaths[entry.Path] = true
		m.status = fmt.Sprintf("已展开 %s", displayPath(entry.Path))
		return m, nil
	}
	m.status = fmt.Sprintf("正在展开 %s", displayPath(entry.Path))
	return m, m.loadTreeChildrenCommand(m.openedRepo, entry.Path)
}

// downloadSelectedEntry 下载右栏当前选中的文件或目录。
func (m model) downloadSelectedEntry(targetDir string) (model, tea.Cmd) {
	entry, ok := m.selectedVisibleEntry()
	if !m.repoOpened || !ok {
		m.status = "没有可下载条目"
		return m, nil
	}
	targetDir = strings.TrimSpace(targetDir)
	if targetDir == "" {
		m.status = "目标目录不能为空"
		return m, nil
	}
	request := downloadRequest{repository: m.openedRepo, entry: entry, targetDir: targetDir}
	entryLabel := downloadEntryLabel(m.openedRepo, entry)
	m.status = fmt.Sprintf("正在下载 %s", entryLabel)
	baseLabel := fmt.Sprintf("downloading %s", entryLabel)
	messages := make(chan tea.Msg, 64)
	m.operationMessages = messages
	operationID := m.startOperation(operationDownload, baseLabel)
	return m, tea.Batch(operationTickCommand(operationID), m.listenOperationCommand(operationID), m.downloadEntryProgressCommand(operationID, request, false, messages, baseLabel))
}

// downloadEntryLabel 返回下载动作中展示的条目名称。
func downloadEntryLabel(repo config.Repository, entry app.EntryResult) string {
	if strings.TrimSpace(entry.Path) == "" {
		return repositoryLabel(repo)
	}
	return strings.TrimSpace(entry.Name)
}

// startEntriesRequest 发起一次带编号的目录读取请求。
func (m model) startEntriesRequest(repo config.Repository, dirPath string, selectPath string) (model, tea.Cmd) {
	requestID := m.nextRequestID()
	m.entriesRequestID = requestID
	return m, m.listEntriesCommand(requestID, repo, dirPath, selectPath)
}

// focusInput 聚焦共用文本输入框。
func (m model) focusInput() (model, tea.Cmd) {
	cmd := m.input.Focus()
	return m, cmd
}
