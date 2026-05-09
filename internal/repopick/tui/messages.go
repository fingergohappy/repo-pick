package tui

import (
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

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
