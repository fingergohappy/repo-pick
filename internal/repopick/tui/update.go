package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/config"
)

// Update 根据 Bubble Tea 消息更新 TUI 状态。
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case repositoriesLoadedMsg:
		return m.handleRepositoriesLoaded(msg)
	case entriesLoadedMsg:
		return m.handleEntriesLoaded(msg)
	case treeChildrenLoadedMsg:
		return m.handleTreeChildrenLoaded(msg)
	case searchResultMsg:
		return m.handleSearchResult(msg)
	case repositoryUpdatedMsg:
		return m.handleRepositoryUpdated(msg)
	case repositoryRemovedMsg:
		return m.handleRepositoryRemoved(msg)
	case repositoryAddedMsg:
		return m.handleRepositoryAdded(msg)
	case repositoryEditedMsg:
		return m.handleRepositoryEdited(msg)
	case branchesLoadedMsg:
		return m.handleBranchesLoaded(msg)
	case downloadResultMsg:
		return m.handleDownloadResult(msg)
	case editorFinishedMsg:
		return m.handleEditorFinished(msg)
	case operationTickMsg:
		return m.handleOperationTick(msg)
	case registrySelectionTickMsg:
		return m.handleRegistrySelectionTick(msg)
	case selectionCursorTickMsg:
		return m.handleSelectionCursorTick()
	case operationProgressMsg:
		return m.handleOperationProgress(msg)
	case operationChannelClosedMsg:
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

// handleRepositoriesLoaded 将 registry 加载结果写入左栏状态。
func (m model) handleRepositoriesLoaded(msg repositoriesLoadedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "加载 registry 失败"
		return m, nil
	}
	m.err = nil
	m.repositories = msg.repositories
	m.selectedRepo = clampCursor(m.selectedRepo, len(m.repositories))
	m.status = fmt.Sprintf("已加载 %d 个 registry", len(m.repositories))
	if !m.selectionCursorTicking {
		m.selectionCursorTicking = true
		return m, selectionCursorTickCommand()
	}
	return m, nil
}

// handleEntriesLoaded 将仓库目录读取结果写入右栏状态。
func (m model) handleEntriesLoaded(msg entriesLoadedMsg) (model, tea.Cmd) {
	if msg.operationKind != operationNone {
		if !m.currentOperation(msg.operationKind, msg.requestID) {
			return m, nil
		}
		m.clearOperation(msg.operationKind, msg.requestID)
	} else if msg.requestID != 0 && msg.requestID != m.entriesRequestID {
		return m, nil
	}
	if msg.err != nil {
		m.err = msg.err
		m.status = "加载目录失败"
		return m, nil
	}
	m.err = nil
	m.repositories = replaceRepository(m.repositories, msg.repository)
	m.openedRepo = msg.repository
	m.repoOpened = true
	m.currentPath = msg.path
	m.resetTreeRoot(msg.path, msg.entries)
	m.showingSearch = false
	m.searchResults = nil
	m.registrySelectionID = 0
	m.registrySelectionFrame = 0
	m.selectedEntry = indexForPath(m.visibleEntries(), msg.selectPath)
	m.focus = focusTree
	m.status = fmt.Sprintf("%s: %s", msg.repository.Name, displayPath(msg.path))
	return m, nil
}

// handleTreeChildrenLoaded 将目录展开结果写入树缓存。
func (m model) handleTreeChildrenLoaded(msg treeChildrenLoadedMsg) (model, tea.Cmd) {
	if !m.repoOpened || !sameRepository(m.openedRepo, msg.repository) {
		return m, nil
	}
	if msg.err != nil {
		m.err = msg.err
		m.status = "展开目录失败"
		return m, nil
	}
	m.err = nil
	m.ensureTreeMaps()
	m.treeChildren[msg.path] = msg.entries
	m.expandedPaths[msg.path] = true
	m.selectedEntry = indexForPath(m.visibleEntries(), msg.path)
	m.status = fmt.Sprintf("已展开 %s", displayPath(msg.path))
	return m, nil
}

// handleSearchResult 将路径搜索结果写入右栏状态。
func (m model) handleSearchResult(msg searchResultMsg) (model, tea.Cmd) {
	if msg.requestID != 0 && msg.requestID != m.searchRequestID {
		return m, nil
	}
	if !m.repoOpened || !sameRepository(m.openedRepo, msg.repository) {
		return m, nil
	}
	if msg.err != nil {
		m.err = msg.err
		m.status = "搜索失败"
		return m, nil
	}
	m.err = nil
	m.searchQuery = msg.query
	m.searchResults = msg.entries
	m.showingSearch = true
	m.selectedEntry = clampCursor(0, len(m.searchResults))
	m.status = fmt.Sprintf("找到 %d 个路径", len(m.searchResults))
	return m, nil
}

// handleRepositoryUpdated 将 cache 更新后的目录结果写入右栏。
func (m model) handleRepositoryUpdated(msg repositoryUpdatedMsg) (model, tea.Cmd) {
	if !m.currentOperation(operationUpdate, msg.operationID) {
		return m, nil
	}
	m.clearOperation(operationUpdate, msg.operationID)
	if msg.err != nil {
		m.err = msg.err
		m.status = "更新仓库失败"
		if sameRepository(m.openedRepo, msg.repository) {
			m.repoOpened = false
			m.entries = nil
			m.treeChildren = nil
			m.expandedPaths = nil
			m.searchResults = nil
			m.showingSearch = false
		}
		return m, nil
	}
	m.err = nil
	m.repositories = replaceRepository(m.repositories, msg.repository)
	m.openedRepo = msg.repository
	m.repoOpened = true
	m.currentPath = msg.path
	m.resetTreeRoot(msg.path, msg.entries)
	m.searchResults = nil
	m.showingSearch = false
	m.registrySelectionID = 0
	m.registrySelectionFrame = 0
	m.selectedEntry = clampCursor(0, len(m.visibleEntries()))
	m.status = fmt.Sprintf("%s 已更新", msg.repository.Name)
	return m, nil
}

// handleRepositoryRemoved 将 registry 删除结果写入左栏状态。
func (m model) handleRepositoryRemoved(msg repositoryRemovedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "删除 repository 失败"
		return m, nil
	}
	m.err = nil
	m.repositories = msg.repositories
	m.selectedRepo = clampCursor(m.selectedRepo, len(m.repositories))
	if m.repoOpened && m.openedRepo.Name == msg.repository.Name {
		m.repoOpened = false
		m.openedRepo = config.Repository{}
		m.currentPath = ""
		m.entries = nil
		m.treeChildren = nil
		m.expandedPaths = nil
		m.searchResults = nil
		m.showingSearch = false
	}
	m.status = fmt.Sprintf("%s 已删除", msg.repository.Name)
	return m, nil
}

// handleRepositoryAdded 将 registry 新增结果写入左栏状态。
func (m model) handleRepositoryAdded(msg repositoryAddedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "添加 registry 失败"
		return m, nil
	}
	m.err = nil
	m.clearAddState()
	m.repositories = msg.repositories
	m.selectedRepo = clampCursor(len(m.repositories)-1, len(m.repositories))
	m.status = "registry 已添加"
	return m, nil
}

// handleRepositoryEdited 将 registry 编辑结果写入左栏状态。
func (m model) handleRepositoryEdited(msg repositoryEditedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "编辑 registry 失败"
		return m, nil
	}
	m.err = nil
	m.clearAddState()
	m.repositories = msg.repositories
	m.selectedRepo = indexForRepositoryName(m.repositories, msg.repository.Name)
	if m.repoOpened && sameRepository(m.openedRepo, msg.oldRepository) {
		if sameRepositorySource(msg.oldRepository, msg.repository) {
			m.openedRepo = msg.repository
		} else {
			m.repoOpened = false
			m.openedRepo = config.Repository{}
			m.currentPath = ""
			m.entries = nil
			m.treeChildren = nil
			m.expandedPaths = nil
			m.searchResults = nil
			m.showingSearch = false
		}
	}
	m.status = fmt.Sprintf("%s 已编辑", msg.repository.Name)
	return m, nil
}

// handleBranchesLoaded 将远端分支读取结果写入 registry 表单状态。
func (m model) handleBranchesLoaded(msg branchesLoadedMsg) (model, tea.Cmd) {
	if msg.url != m.pendingURL {
		return m, nil
	}
	m.branchLoading = false
	m.branchErr = msg.err
	if msg.err != nil {
		m.pendingBranches = nil
		m.pendingDefaultBranch = ""
		m.selectedBranch = 0
		m.status = "获取分支失败，Enter 使用默认分支添加"
		if m.editingRepositoryActive && strings.TrimSpace(m.pendingBranch) != "" {
			m.status = "获取分支失败，Enter 保留当前分支"
		}
		return m, nil
	}
	m.err = nil
	m.pendingDefaultBranch = msg.defaultBranch
	m.pendingBranches = msg.branches
	m.selectedBranch = m.defaultBranchSelection()
	m.status = ""
	return m, nil
}

// handleDownloadResult 根据下载结果展示状态或打开覆盖确认。
func (m model) handleDownloadResult(msg downloadResultMsg) (model, tea.Cmd) {
	if !m.currentOperation(operationDownload, msg.operationID) {
		return m, nil
	}
	m.clearOperation(operationDownload, msg.operationID)
	if msg.err != nil {
		if errorsIsTargetExists(msg.err) {
			m.pendingDownload = &msg.request
			m.pendingConfirm = &confirmState{kind: confirmOverwrite}
			m.mode = modeConfirmOverwrite
			m.status = fmt.Sprintf("%s already exists. Overwrite? y/n", downloadEntryLabel(msg.request.repository, msg.request.entry))
			return m, nil
		}
		m.err = msg.err
		m.status = "下载失败"
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("%s 下载完成", downloadEntryLabel(msg.request.repository, msg.request.entry))
	return m, nil
}

// handleEditorFinished 根据 editor 退出结果更新状态栏。
func (m model) handleEditorFinished(msg editorFinishedMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.status = "打开 editor 失败"
		return m, nil
	}
	m.err = nil
	m.status = fmt.Sprintf("%s 已关闭", msg.entry.Name)
	return m, nil
}

// handleOperationTick 推进长耗时操作的进度动画。
func (m model) handleOperationTick(msg operationTickMsg) (model, tea.Cmd) {
	if !m.currentOperation(m.operationKind, msg.operationID) {
		return m, nil
	}
	m.operationFrame++
	return m, operationTickCommand(msg.operationID)
}

// handleRegistrySelectionTick 推进 registry 选中变化提示动画。
func (m model) handleRegistrySelectionTick(msg registrySelectionTickMsg) (model, tea.Cmd) {
	if msg.selectionID == 0 || msg.selectionID != m.registrySelectionID || !m.showRegistrySelectionPreview() {
		return m, nil
	}
	m.registrySelectionFrame++
	return m, registrySelectionTickCommand(msg.selectionID)
}

// handleSelectionCursorTick 推进选中行光标动画。
func (m model) handleSelectionCursorTick() (model, tea.Cmd) {
	m.selectionCursorFrame++
	return m, selectionCursorTickCommand()
}

// handleOperationProgress 更新长耗时操作的进度文本。
func (m model) handleOperationProgress(msg operationProgressMsg) (model, tea.Cmd) {
	if !m.currentOperation(msg.kind, msg.operationID) {
		return m, nil
	}
	m.operationLabel = formatOperationProgress(msg.baseLabel, msg.event)
	m.operationPercent = msg.event.Percent
	return m, m.listenOperationCommand(msg.operationID)
}
