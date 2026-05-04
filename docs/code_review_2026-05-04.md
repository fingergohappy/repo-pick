# Code Review - 2026-05-04

## 范围

本次 review 基于当前工作区状态，重点检查 repo-pick 重命名迁移后的代码结构、TUI 布局和容易影响终端展示的实现细节。

当前代码主线是：

- `cmd/repo-pick`: 程序入口和依赖组装。
- `internal/repopick/config`: 用户配置读写。
- `internal/repopick/registry`: 仓库书签业务规则。
- `internal/repopick/cache`: Git shallow clone cache 生命周期。
- `internal/repopick/tree`: cache 工作区目录树读取与搜索。
- `internal/repopick/install`: 文件或目录复制。
- `internal/repopick/app`: 应用用例编排。
- `internal/repopick/tui`: Bubble Tea 交互和渲染。

整体分层方向合理，没有看到核心包依赖 TUI 或 CLI 的倒挂；主要问题集中在 TUI 的尺寸适配、状态栏展示和文档残留。

## Findings

### P2: TUI 主列表没有可视窗口裁剪

位置：

- `internal/repopick/tui/view.go:97`
- `internal/repopick/tui/view.go:98`
- `internal/repopick/tui/view.go:109`
- `internal/repopick/tui/view.go:119`
- `internal/repopick/tui/view.go:153`

`paneView` 只设置了固定高度，但 `registryLines` 和 `treeLines` 会把全部仓库、目录树节点或搜索结果都渲染出来。Lipgloss 的 `Height` 会补齐高度，但不会裁掉更长内容；条目数量超过终端高度时，内容会把底部状态栏挤出屏幕，当前选中项也可能不可见。

建议：

- 根据 `m.height` 计算每个 pane 的内容可用行数。
- 对 registry、tree 和 search result 做围绕当前光标的窗口切片。
- 如果后续需要滚动体验，可以引入 Bubble Tea 的 `viewport` 或 `list`，但当前直接实现切片即可。

### P2: 窄终端和长文本会破坏横向布局

位置：

- `internal/repopick/tui/view.go:207`
- `internal/repopick/tui/view.go:213`
- `internal/repopick/tui/view.go:244`
- `internal/repopick/tui/view.go:334`
- `internal/repopick/tui/view.go:371`
- `internal/repopick/tui/view.go:569`

`paneWidths` 固定最小宽度为左栏 20、右栏 40。终端宽度小于约 63 列时，双栏必然超出屏幕。长 repo 名、长路径、长搜索结果和底部快捷键 help 没有截断，容易换行并打乱双栏结构。

建议：

- 为 pane 内容统一做单行 truncate。
- 底部状态栏限制到 `m.width` 内，长 help 可以压缩或只显示 `? help`。
- 对极窄终端给出最小宽度提示，或者降级为单栏布局。

### P2: 错误状态可能把 Git 输出整段渲染到底部

位置：

- `internal/repopick/tui/view.go:186`
- `internal/repopick/cache/service.go:398`

`statusLine` 在 `m.err != nil` 时直接返回 `m.err.Error()`。`commandError.Error()` 会包含 git 命令、底层错误和完整 stderr/stdout，失败时可能产生多行输出，导致底部状态区变成大块文本并破坏布局。

建议：

- 状态栏只展示首行或摘要。
- 完整错误可以放在独立错误弹框，或者后续提供详细错误视图。

### P3: TUI 命令层保留了未使用的双轨实现

位置：

- `internal/repopick/tui/commands.go:21`
- `internal/repopick/tui/commands.go:84`
- `internal/repopick/tui/keys.go:257`

当前打开和更新仓库都走带进度的 `openRepositoryProgressCommand`、`updateRepositoryProgressCommand`。非进度版本 `openRepositoryCommand`、`updateRepositoryCommand` 没有调用方，保留会让后续维护者以为存在两套路径。

建议：

- 删除未使用的非进度命令，保留单一路径。
- 对应测试继续覆盖进度路径即可。

### P3: docs 中仍有旧 skilldown 设计残留

位置：

- `docs/tasks/app_layer_feature.md:7`
- `docs/tasks/app_layer_feature.md:117`
- `docs/tasks/install_feature.md:7`
- `docs/tasks/install_feature.md:61`
- `docs/tasks/tui_interface_feature.md:7`
- `docs/tasks/tui_interface_feature.md:139`
- `docs/tasks/tui_repo_downloader_feature.md:161`
- `docs/tasks/tui_repo_downloader_feature.md:332`

部分非归档任务文档仍描述 `cmd/skilldown`、`internal/skilldown/*`、`CopyDir` 或 Tab 切换焦点。当前实现已经迁移到 `cmd/repo-pick` 和 `internal/repopick/*`，复制接口也是 `CopyEntry`，README 和实现使用 `ctrl-w h/l` 切换焦点。

建议：

- 将仍然有效的任务文档更新到 repo-pick 语义，或明确移动到 `docs/tasks/archived/`。
- 已废弃的 skilldown 设计移动到 archived，避免误导后续实现。

## 补充 Findings

### P1: force 覆盖会先删除目标，再校验源目录

位置：

- `internal/repopick/install/install.go:85`
- `internal/repopick/install/install.go:91`

`CopyEntryWithProgress` 在 `prepareTargetPath` 阶段遇到 `force=true` 会直接 `RemoveAll(targetPath)`，随后才递归统计和校验源目录。若源目录中存在 symlink、非普通文件、权限错误或 context 被取消，目标已经被删除，但新内容不会写入。

建议：

- 先完整校验源目录和可复制内容，再删除目标。
- 更稳妥的实现是先复制到临时路径，全部成功后再替换目标。

### P2: TUI 长耗时操作缺少过期消息隔离

位置：

- `internal/repopick/tui/keys.go:257`
- `internal/repopick/tui/keys.go:272`
- `internal/repopick/tui/keys.go:526`
- `internal/repopick/tui/model.go:339`
- `internal/repopick/tui/model.go:379`
- `internal/repopick/tui/model.go:395`
- `internal/repopick/tui/model.go:482`

当前可以在已有 open/update/download 未完成时继续启动新的操作，并覆盖 `operationMessages`。open、update、download 和 search 的结果消息本身没有 request id；`entriesLoadedMsg`、`repositoryUpdatedMsg`、`searchResultMsg` 和 `downloadResultMsg` 也没有完整校验当前 repo/path。目录展开结果已有 `sameRepository` 校验，新增 registry 的分支加载也会按 URL 丢弃过期消息，但打开仓库、切换目录、搜索和下载结果仍可能接受过期消息。典型风险是先打开 A、再打开 B，如果 A 后完成，A 的结果可能覆盖 B；搜索结果也可能来自旧仓库状态。

建议：

- 为异步操作增加 `operationID` 或 request token，handler 只接受当前 token 的消息。
- 或者在 `operationKind != operationNone` 时禁止启动新的长耗时操作。
- 搜索、切换目录和下载等消息也应带上 repo/path/token，并忽略过期结果；已有上下文校验的目录展开和分支加载可以保留当前策略或统一迁移到 token。

### P2: 更新未打开仓库失败会清空当前右侧视图

位置：

- `internal/repopick/tui/model.go:396`

`handleRepositoryUpdated` 在更新失败时会无条件把 `repoOpened`、`entries`、`treeChildren`、`searchResults` 清空。若当前右侧打开的是 B，左侧选中 A 并更新失败，B 的目录树也会被清掉。

建议：

- 仅当 `sameRepository(m.openedRepo, msg.repository)` 时清空右侧状态。
- 更新其他仓库失败时只更新 `err/status`，保留当前打开视图。

### P3: model 中保留了未使用字段

位置：

- `internal/repopick/tui/model.go:106`

`pendingSelectPath` 没有读写调用方。它和未使用的非进度命令一样属于迁移后的残留状态，会增加后续维护成本。

建议：

- 删除该字段。
- 如果未来需要延迟定位路径，再随具体需求重新引入。

## 验证结果

已执行：

```bash
go test ./...
go vet ./...
go test -race ./...
```

结果：

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `go test -race ./...` 通过。

## 结论

当前代码结构总体可以继续推进。下一步优先处理 TUI 的纵向滚动、横向截断和 force 覆盖安全性，因为这些问题最容易在真实仓库、小终端或失败路径里直接影响使用。未使用命令和旧文档残留可以作为低风险清理项跟进。
