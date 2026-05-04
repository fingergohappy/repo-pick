package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

const operationTickInterval = 120 * time.Millisecond

// listRepositoriesCommand 创建读取 registry 列表的异步命令。
func (m model) listRepositoriesCommand() tea.Cmd {
	return func() tea.Msg {
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoriesLoadedMsg{repositories: repositories, err: err}
	}
}

// openRepositoryCommand 创建打开仓库并读取目录树根目录的异步命令。
func (m model) openRepositoryCommand(repo config.Repository) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.svc.EnsureRepository(m.ctx, repo); err != nil {
			return entriesLoadedMsg{repository: repo, err: err}
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo})
		return entriesLoadedMsg{repository: repo, path: result.DirPath, entries: result.Entries, err: err}
	}
}

// openRepositoryProgressCommand 创建带 Git 进度的打开仓库命令。
func (m model) openRepositoryProgressCommand(repo config.Repository, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		_, err := m.svc.EnsureRepositoryWithProgress(m.ctx, repo, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{kind: operationOpen, baseLabel: baseLabel, event: event})
		})
		if err != nil {
			messages <- entriesLoadedMsg{repository: repo, err: err}
			return nil
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo})
		messages <- entriesLoadedMsg{repository: repo, path: result.DirPath, entries: result.Entries, err: err}
		return nil
	}
}

// listEntriesCommand 创建读取指定目录条目的异步命令。
func (m model) listEntriesCommand(repo config.Repository, dirPath string, selectPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		return entriesLoadedMsg{
			repository: repo,
			path:       result.DirPath,
			selectPath: selectPath,
			entries:    result.Entries,
			err:        err,
		}
	}
}

// loadTreeChildrenCommand 创建读取树节点子级条目的异步命令。
func (m model) loadTreeChildrenCommand(repo config.Repository, dirPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		return treeChildrenLoadedMsg{
			repository: repo,
			path:       result.DirPath,
			entries:    result.Entries,
			err:        err,
		}
	}
}

// searchEntriesCommand 创建路径搜索异步命令。
func (m model) searchEntriesCommand(repo config.Repository, query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.SearchEntries(m.ctx, app.SearchEntriesRequest{Repository: repo, Query: query})
		return searchResultMsg{query: result.Query, entries: result.Entries, err: err}
	}
}

// updateRepositoryCommand 创建删除旧 cache 并重新下载仓库的异步命令。
func (m model) updateRepositoryCommand(repo config.Repository, dirPath string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.svc.UpdateRepository(m.ctx, repo); err != nil {
			return repositoryUpdatedMsg{repository: repo, err: err}
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		return repositoryUpdatedMsg{repository: repo, path: result.DirPath, entries: result.Entries, err: err}
	}
}

// updateRepositoryProgressCommand 创建带 Git 进度的更新仓库命令。
func (m model) updateRepositoryProgressCommand(repo config.Repository, dirPath string, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		if _, err := m.svc.UpdateRepositoryWithProgress(m.ctx, repo, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{kind: operationUpdate, baseLabel: baseLabel, event: event})
		}); err != nil {
			messages <- repositoryUpdatedMsg{repository: repo, err: err}
			return nil
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		messages <- repositoryUpdatedMsg{repository: repo, path: result.DirPath, entries: result.Entries, err: err}
		return nil
	}
}

// removeRepositoryCommand 创建删除 registry 和 cache 的异步命令。
func (m model) removeRepositoryCommand(repo config.Repository) tea.Cmd {
	return func() tea.Msg {
		err := m.svc.RemoveRepository(m.ctx, app.RemoveRepositoryRequest{Name: repo.Name})
		if err != nil {
			return repositoryRemovedMsg{repository: repo, err: err}
		}
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoryRemovedMsg{repository: repo, repositories: repositories, err: err}
	}
}

// addRepositoryCommand 创建新增 registry 并重新加载列表的异步命令。
func (m model) addRepositoryCommand(name string, repoURL string, branch string) tea.Cmd {
	return func() tea.Msg {
		err := m.svc.AddRepository(m.ctx, app.AddRepositoryRequest{Name: name, URL: repoURL, Branch: branch})
		if err != nil {
			return repositoryAddedMsg{err: err}
		}
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoryAddedMsg{repositories: repositories, err: err}
	}
}

// listBranchesCommand 创建读取远端分支列表的异步命令。
func (m model) listBranchesCommand(repoURL string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.ListRemoteBranches(m.ctx, app.ListRemoteBranchesRequest{URL: repoURL})
		return branchesLoadedMsg{
			url:           repoURL,
			defaultBranch: result.Default,
			branches:      result.Branches,
			err:           err,
		}
	}
}

// downloadEntryProgressCommand 创建带复制进度的下载当前条目命令。
func (m model) downloadEntryProgressCommand(repo config.Repository, request downloadRequest, force bool, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		result, err := m.svc.DownloadEntryWithProgress(m.ctx, app.DownloadEntryRequest{
			Repository: repo,
			Entry:      request.entry,
			TargetDir:  request.targetDir,
			Force:      force,
		}, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{kind: operationDownload, baseLabel: baseLabel, event: event})
		})
		messages <- downloadResultMsg{request: request, result: result, err: err}
		return nil
	}
}

// operationTickCommand 创建长耗时操作进度动画的下一帧命令。
func operationTickCommand() tea.Cmd {
	return tea.Tick(operationTickInterval, func(time.Time) tea.Msg {
		return operationTickMsg{}
	})
}

// listenOperationCommand 等待当前长耗时操作的下一条消息。
func (m model) listenOperationCommand() tea.Cmd {
	messages := m.operationMessages
	return func() tea.Msg {
		if messages == nil {
			return operationChannelClosedMsg{}
		}
		msg, ok := <-messages
		if !ok {
			return operationChannelClosedMsg{}
		}
		return msg
	}
}

// sendOperationProgress 尽量发送进度消息；UI 忙时允许丢弃中间进度。
func sendOperationProgress(messages chan tea.Msg, msg operationProgressMsg) {
	select {
	case messages <- msg:
	default:
	}
}
