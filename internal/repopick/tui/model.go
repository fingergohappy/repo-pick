package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

type focusPane int

const (
	focusRegistry focusPane = iota
	focusTree
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeAddName
	modeAddURL
	modeAddBranch
	modeSearch
	modeTargetDir
	modeConfirmDelete
	modeConfirmOverwrite
)

type confirmKind int

const (
	confirmDelete confirmKind = iota
	confirmOverwrite
)

type operationKind int

const (
	operationNone operationKind = iota
	operationOpen
	operationUpdate
	operationDownload
)

// model 保存 TUI 会话的交互状态。
type model struct {
	// ctx 是 TUI 生命周期内传递给 app 用例的上下文。
	ctx context.Context
	// svc 是 registry、cache、tree 和下载动作的 app 用例入口。
	svc app.Service
	// sessionCWD 是用户启动 repo-pick 时所在目录。
	sessionCWD string
	// focus 表示当前接收快捷键的栏目。
	focus focusPane
	// mode 表示当前是否处于输入或确认状态。
	mode inputMode
	// repositories 是左栏展示的 registry 仓库列表。
	repositories []config.Repository
	// selectedRepo 是左栏当前光标位置。
	selectedRepo int
	// openedRepo 是右侧目录树当前打开的仓库。
	openedRepo config.Repository
	// repoOpened 表示右侧目录树是否已有打开的仓库。
	repoOpened bool
	// currentPath 是目录树当前所在的仓库内路径。
	currentPath string
	// entries 是当前目录下的直接子级条目。
	entries []app.EntryResult
	// treeChildren 是目录路径到直接子级条目的缓存。
	treeChildren map[string][]app.EntryResult
	// expandedPaths 记录当前树 root 下已展开的目录路径。
	expandedPaths map[string]bool
	// selectedEntry 是右栏当前光标位置。
	selectedEntry int
	// searchQuery 是最近一次路径搜索关键词。
	searchQuery string
	// searchResults 是当前搜索结果列表。
	searchResults []app.EntryResult
	// showingSearch 表示右栏当前是否展示搜索结果。
	showingSearch bool
	// pendingName 是 registry 表单中当前待提交的名称。
	pendingName string
	// pendingURL 是 registry 表单中当前待提交的 Git 仓库地址。
	pendingURL string
	// pendingBranch 是 registry 表单中当前待提交的分支。
	pendingBranch string
	// pendingBranches 是 registry 表单从远端读取到的分支列表。
	pendingBranches []string
	// pendingDefaultBranch 是 registry 表单中远端 HEAD 指向的默认分支。
	pendingDefaultBranch string
	// branchQuery 是 registry 表单中用于过滤远端分支的搜索文本。
	branchQuery string
	// selectedBranch 是 registry 表单分支选择列表的当前光标位置。
	selectedBranch int
	// branchLoading 表示 registry 表单正在读取远端分支。
	branchLoading bool
	// branchErr 是 registry 表单最近一次读取分支的错误。
	branchErr error
	// editingRepository 是当前正在编辑的原 registry 配置。
	editingRepository config.Repository
	// editingRepositoryActive 表示 registry 表单正在编辑已有条目。
	editingRepositoryActive bool
	// pendingDownload 是覆盖确认中的下载请求。
	pendingDownload *downloadRequest
	// pendingConfirm 是当前确认框状态。
	pendingConfirm *confirmState
	// input 是新增、搜索和目标目录共用文本输入。
	input textinput.Model
	// status 是底部状态栏展示的最近动作结果。
	status string
	// err 是最近一次业务动作错误。
	err error
	// width 是当前终端宽度。
	width int
	// height 是当前终端高度。
	height int
	// showHelp 控制是否展示快捷键帮助视图。
	showHelp bool
	// pendingWindowCommand 表示已按下 ctrl-w，等待 h/l 选择窗口。
	pendingWindowCommand bool
	// registrySelectionFrame 是 registry 选中变化提示动画帧索引。
	registrySelectionFrame int
	// registrySelectionID 是当前 registry 选中变化提示编号。
	registrySelectionID uint64
	// selectionCursorFrame 是选中行光标动画帧索引。
	selectionCursorFrame int
	// selectionCursorTicking 表示选中行光标动画计时器已启动。
	selectionCursorTicking bool
	// operationKind 表示当前正在运行的长耗时操作。
	operationKind operationKind
	// operationLabel 是当前长耗时操作的展示文本。
	operationLabel string
	// operationPercent 是当前长耗时操作的百分比；-1 表示暂未解析到百分比。
	operationPercent int
	// operationFrame 是当前进度动画帧索引。
	operationFrame int
	// operationMessages 是当前长耗时操作的消息流。
	operationMessages <-chan tea.Msg
	// requestSeq 是异步请求编号的单调计数器。
	requestSeq uint64
	// entriesRequestID 是当前目录读取请求编号。
	entriesRequestID uint64
	// searchRequestID 是当前路径搜索请求编号。
	searchRequestID uint64
	// operationID 是当前长耗时操作编号。
	operationID uint64
}

// downloadRequest 保存一次待确认或待执行的下载请求。
type downloadRequest struct {
	// repository 是发起下载时右侧打开的仓库。
	repository config.Repository
	// entry 是要下载的仓库条目。
	entry app.EntryResult
	// targetDir 是下载目标目录。
	targetDir string
}

// treeRow 表示右侧树形视图中的一行。
type treeRow struct {
	// entry 是该行对应的仓库条目。
	entry app.EntryResult
	// prefix 是该行在树中的连接线前缀。
	prefix string
	// expanded 表示该目录是否已展开；文件固定为 false。
	expanded bool
	// root 表示该行是当前 tree root 自身。
	root bool
}

// confirmState 保存当前确认框上下文。
type confirmState struct {
	// kind 是确认框的业务类型。
	kind confirmKind
	// repository 是删除确认对应的仓库。
	repository config.Repository
}

type repositoriesLoadedMsg struct {
	// repositories 是从配置中读取到的 registry 仓库列表。
	repositories []config.Repository
	// err 是加载 registry 时产生的错误。
	err error
}

type entriesLoadedMsg struct {
	// requestID 是本次目录读取请求编号。
	requestID uint64
	// operationKind 表示该目录结果是否来自长耗时操作。
	operationKind operationKind
	// repository 是本次加载的仓库。
	repository config.Repository
	// path 是本次加载的目录路径。
	path string
	// selectPath 是加载后需要定位的条目路径。
	selectPath string
	// entries 是目录下的直接子级条目。
	entries []app.EntryResult
	// err 是加载目录树时产生的错误。
	err error
}

type treeChildrenLoadedMsg struct {
	// repository 是本次加载子树的仓库。
	repository config.Repository
	// path 是被展开的目录路径。
	path string
	// entries 是目录下的直接子级条目。
	entries []app.EntryResult
	// err 是加载目录子级时产生的错误。
	err error
}

type searchResultMsg struct {
	// requestID 是本次搜索请求编号。
	requestID uint64
	// repository 是本次搜索所属的仓库。
	repository config.Repository
	// query 是本次路径搜索关键词。
	query string
	// entries 是匹配到的文件或目录条目。
	entries []app.EntryResult
	// err 是搜索过程中产生的错误。
	err error
}

type repositoryUpdatedMsg struct {
	// operationID 是本次更新操作编号。
	operationID uint64
	// repository 是被更新的仓库。
	repository config.Repository
	// entries 是更新后当前路径下的条目。
	entries []app.EntryResult
	// path 是更新后展示的目录路径。
	path string
	// err 是更新过程中产生的错误。
	err error
}

type repositoryRemovedMsg struct {
	// repository 是被删除的仓库。
	repository config.Repository
	// repositories 是删除后重新加载的 registry 列表。
	repositories []config.Repository
	// err 是删除过程中产生的错误。
	err error
}

type repositoryAddedMsg struct {
	// repositories 是新增后重新加载的 registry 列表。
	repositories []config.Repository
	// err 是新增过程中产生的错误。
	err error
}

type repositoryEditedMsg struct {
	// oldRepository 是编辑前的 registry 配置。
	oldRepository config.Repository
	// repository 是编辑后的 registry 配置。
	repository config.Repository
	// repositories 是编辑后重新加载的 registry 列表。
	repositories []config.Repository
	// err 是编辑过程中产生的错误。
	err error
}

type branchesLoadedMsg struct {
	// url 是本次查询分支的 Git 仓库地址。
	url string
	// defaultBranch 是远端 HEAD 指向的默认分支。
	defaultBranch string
	// branches 是远端 refs/heads 下的全部分支名称。
	branches []string
	// err 是查询远端分支时产生的错误。
	err error
}

type downloadResultMsg struct {
	// operationID 是本次下载操作编号。
	operationID uint64
	// request 是本次下载请求。
	request downloadRequest
	// result 是 app 下载返回的结构化结果。
	result app.DownloadEntryResult
	// err 是下载过程中产生的错误。
	err error
}

type editorFinishedMsg struct {
	// entry 是本次用 editor 打开的仓库文件。
	entry app.EntryResult
	// err 是 editor 退出时返回的错误。
	err error
}

type operationTickMsg struct {
	// operationID 是本次进度动画所属的操作编号。
	operationID uint64
}

type registrySelectionTickMsg struct {
	// selectionID 是本次 registry 选中提示动画编号。
	selectionID uint64
}

type selectionCursorTickMsg struct{}

type operationProgressMsg struct {
	// operationID 是本次进度所属的操作编号。
	operationID uint64
	// kind 是进度所属的操作类型。
	kind operationKind
	// baseLabel 是操作的基础展示文本。
	baseLabel string
	// event 是 app 层传回的进度事件。
	event app.ProgressEvent
}

type operationChannelClosedMsg struct {
	// operationID 是已关闭消息流所属的操作编号。
	operationID uint64
}

// newModel 创建 TUI 初始状态。
func newModel(ctx context.Context, svc app.Service, sessionCWD string) model {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 512
	input.Width = 56

	return model{
		ctx:        ctx,
		svc:        svc,
		sessionCWD: strings.TrimSpace(sessionCWD),
		focus:      focusRegistry,
		mode:       modeNormal,
		input:      input,
		width:      100,
		height:     30,
		status:     "ready",
		// 初始没有长耗时操作，也就没有可展示百分比。
		operationPercent: -1,
	}
}

// Init 初始化 registry 列表。
func (m model) Init() tea.Cmd {
	return m.listRepositoriesCommand()
}

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
	m.status = fmt.Sprintf("已获取 %d 个分支", len(msg.branches))
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

// activeRepository 返回左栏当前选中的仓库。
func (m model) activeRepository() (config.Repository, bool) {
	if len(m.repositories) == 0 {
		return config.Repository{}, false
	}
	return m.repositories[clampCursor(m.selectedRepo, len(m.repositories))], true
}

// visibleEntries 返回右栏当前实际显示的条目列表。
func (m model) visibleEntries() []app.EntryResult {
	if m.showingSearch {
		return m.searchResults
	}
	rows := m.visibleTreeRows()
	entries := make([]app.EntryResult, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, row.entry)
	}
	return entries
}
