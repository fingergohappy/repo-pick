package tui

import (
	"context"
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
