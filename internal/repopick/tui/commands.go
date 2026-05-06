package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finger/repo-pick/internal/repopick/app"
	"github.com/finger/repo-pick/internal/repopick/config"
)

const operationTickInterval = 120 * time.Millisecond
const registrySelectionTickInterval = 90 * time.Millisecond

// listRepositoriesCommand 创建读取 registry 列表的异步命令。
func (m model) listRepositoriesCommand() tea.Cmd {
	return func() tea.Msg {
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoriesLoadedMsg{repositories: repositories, err: err}
	}
}

// openRepositoryProgressCommand 创建带 Git 进度的打开仓库命令。
func (m model) openRepositoryProgressCommand(operationID uint64, repo config.Repository, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		_, err := m.svc.EnsureRepositoryWithProgress(m.ctx, repo, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{operationID: operationID, kind: operationOpen, baseLabel: baseLabel, event: event})
		})
		if err != nil {
			messages <- entriesLoadedMsg{requestID: operationID, operationKind: operationOpen, repository: repo, err: err}
			return nil
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo})
		messages <- entriesLoadedMsg{requestID: operationID, operationKind: operationOpen, repository: repo, path: result.DirPath, entries: result.Entries, err: err}
		return nil
	}
}

// listEntriesCommand 创建读取指定目录条目的异步命令。
func (m model) listEntriesCommand(requestID uint64, repo config.Repository, dirPath string, selectPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		return entriesLoadedMsg{
			requestID:  requestID,
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
func (m model) searchEntriesCommand(requestID uint64, repo config.Repository, query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.svc.SearchEntries(m.ctx, app.SearchEntriesRequest{Repository: repo, Query: query})
		return searchResultMsg{requestID: requestID, repository: repo, query: result.Query, entries: result.Entries, err: err}
	}
}

// updateRepositoryProgressCommand 创建带 Git 进度的更新仓库命令。
func (m model) updateRepositoryProgressCommand(operationID uint64, repo config.Repository, dirPath string, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		if _, err := m.svc.UpdateRepositoryWithProgress(m.ctx, repo, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{operationID: operationID, kind: operationUpdate, baseLabel: baseLabel, event: event})
		}); err != nil {
			messages <- repositoryUpdatedMsg{operationID: operationID, repository: repo, err: err}
			return nil
		}
		result, err := m.svc.ListEntries(m.ctx, app.ListEntriesRequest{Repository: repo, DirPath: dirPath})
		messages <- repositoryUpdatedMsg{operationID: operationID, repository: repo, path: result.DirPath, entries: result.Entries, err: err}
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

// editRepositoryCommand 创建编辑 registry 并重新加载列表的异步命令。
func (m model) editRepositoryCommand(oldRepo config.Repository, name string, repoURL string, branch string) tea.Cmd {
	return func() tea.Msg {
		repo := config.Repository{Name: name, URL: repoURL, Branch: branch}
		err := m.svc.EditRepository(m.ctx, app.EditRepositoryRequest{Name: oldRepo.Name, NewName: name, URL: repoURL, Branch: branch})
		if err != nil {
			return repositoryEditedMsg{oldRepository: oldRepo, repository: repo, err: err}
		}
		repositories, err := m.svc.ListRepositories(m.ctx)
		return repositoryEditedMsg{oldRepository: oldRepo, repository: repo, repositories: repositories, err: err}
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
func (m model) downloadEntryProgressCommand(operationID uint64, request downloadRequest, force bool, messages chan tea.Msg, baseLabel string) tea.Cmd {
	return func() tea.Msg {
		defer close(messages)
		result, err := m.svc.DownloadEntryWithProgress(m.ctx, app.DownloadEntryRequest{
			Repository: request.repository,
			Entry:      request.entry,
			TargetDir:  request.targetDir,
			Force:      force,
		}, func(event app.ProgressEvent) {
			sendOperationProgress(messages, operationProgressMsg{operationID: operationID, kind: operationDownload, baseLabel: baseLabel, event: event})
		})
		messages <- downloadResultMsg{operationID: operationID, request: request, result: result, err: err}
		return nil
	}
}

// operationTickCommand 创建长耗时操作进度动画的下一帧命令。
func operationTickCommand(operationID uint64) tea.Cmd {
	return tea.Tick(operationTickInterval, func(time.Time) tea.Msg {
		return operationTickMsg{operationID: operationID}
	})
}

// registrySelectionTickCommand 创建 registry 选中提示动画的下一帧命令。
func registrySelectionTickCommand(selectionID uint64) tea.Cmd {
	return tea.Tick(registrySelectionTickInterval, func(time.Time) tea.Msg {
		return registrySelectionTickMsg{selectionID: selectionID}
	})
}

// listenOperationCommand 等待当前长耗时操作的下一条消息。
func (m model) listenOperationCommand(operationID uint64) tea.Cmd {
	messages := m.operationMessages
	return func() tea.Msg {
		if messages == nil {
			return operationChannelClosedMsg{operationID: operationID}
		}
		msg, ok := <-messages
		if !ok {
			return operationChannelClosedMsg{operationID: operationID}
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
