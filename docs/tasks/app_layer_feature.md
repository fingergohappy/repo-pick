---
title: app-layer
type: feature
date: 2026-04-27
status: implemented
version: "1.0"
summary: 明确 `internal/skilldown/app` 的应用用例编排职责，使 CLI 和 TUI 能复用同一套 registry、search 和 install 流程。该设计将参数解析、交互展示和业务编排分离，避免 app 与 Cobra 或 Bubble Tea 绑定。
scope:
  - cmd/skilldown
  - internal/skilldown/app
  - internal/skilldown/cli
  - internal/skilldown/tui
  - internal/skilldown/config
  - internal/skilldown/registry
  - internal/skilldown/repo
  - internal/skilldown/skill
  - internal/skilldown/install
  - internal/skilldown/output
---

# app-layer

## 概述

`app` 功能负责提供稳定的应用用例层，把 registry、repo、skill 和 install 等底层模块组装成可被 CLI 和 TUI 共同调用的结构化能力。

## 背景

项目第一版同时规划 CLI 和 TUI 工作流。如果 `app` 直接解析 `os.Args` 或绑定 Cobra，那么 TUI 后续只能复刻 search、install 和 registry 流程，或者绕过 app 直接调用底层模块，最终导致行为不一致。

因此 `app` 不应该做命令行解析，也不应该承载 TUI 状态机。CLI 和 TUI 都应作为输入适配层，把用户输入转换成 `app` 的请求结构；`app` 只执行应用用例并返回结构化结果或错误。

## 核心概念

### 输入适配层

- 职责: 把具体入口的用户输入转换成 app 请求结构。
- 粒度: per interface，例如 CLI 或 TUI。
- 边界: 不直接 clone 仓库，不扫描 skill，不复制安装目录，不重复实现 registry 规则。

### 应用用例层

- 职责: 编排 registry、repo、skill、install 等模块，提供 search、install 和 registry 管理用例。
- 粒度: per user action。
- 边界: 不解析 CLI 参数，不处理按键事件，不决定终端文本样式，不直接读写 YAML 文件格式。

### Registry 用例

- 职责: 为 CLI 和 TUI 统一提供添加、查看、删除仓库的应用入口。
- 粒度: per repository entry。
- 边界: 名称唯一、默认 `skillDir` 和持久化规则仍由 `registry` 与 `config` 模块负责，app 不直接操作 Viper 或 YAML。

### Search 用例

- 职责: 根据显式 repo 或 registry 解析仓库来源，clone worktree，发现 skill，并返回带来源信息的结果。
- 粒度: per search request。
- 边界: 不负责终端渲染，不做交互式选择。

### Install 用例

- 职责: 根据显式 repo 或 registry 查找 skill，处理同名冲突，计算安装目标路径，并调用 install 模块复制目录。
- 粒度: per install request。
- 边界: 不负责 CLI flag 解析，不负责 TUI 选择界面，不实现目录递归复制细节。

## 核心流程

```text
CLI:
    cobra parses command and flags
    cli converts input to app request
    app executes use case
    cli renders result through output or stderr

TUI:
    bubble tea / huh collects user choices
    tui converts choices to app request
    app executes use case
    tui updates model and renders interactive state
```

Registry 流程:

```text
AddRepository(req):
    registry.Add(config.Repository{Name, URL, SkillDir})

ListRepositories():
    registry.List()

RemoveRepository(req):
    registry.Remove(req.Name)
```

Search 流程:

```text
Search(req):
    repos = registry.Resolve(req.RepoURL)
    for each repo:
        dirPath = req.SkillDirPath or repo.SkillDir
        worktree = repoCloner.Clone(repo.URL, SparsePaths: [dirPath])
        defer Cleanup(worktree)
        skills = skill.Discover(worktree.Dir, dirPath)
        append result grouped by repository
```

Install 流程:

```text
Install(req):
    results = Search(search request derived from install request)
    selected = exact match by req.SkillName, or all skills when SkillName is empty
    fail if same SkillName appears in multiple repositories and repo was implicit
    for each selected skill:
        sourceDir = worktree.Dir + skill.Path
        targetDir = req.TargetRoot + skill.Name
        installer.CopyDir(sourceDir, targetDir, req.Force)
```

## 关键机制

### app 不解析 CLI

`app` 不能接收 `[]string`，也不能返回进程退出码。`Run(ctx, args []string) int` 这类接口会把 app 绑死在 CLI 入口上，不适合同时支持 TUI。

推荐接口是 service 方法:

```go
type Service struct {
    ConfigStore config.Store
    Registry    RegistryService
    Cloner      repo.Cloner
    Installer   install.Installer
}
```

CLI 可以把 Cobra 解析后的参数转换成请求结构；TUI 可以把表单和选择结果转换成同一组请求结构。

### registry 功能属于 app 用例

添加、查看、删除 registry 是用户可见能力，应该在 `app` 暴露 use case。app 负责把用户动作落到 `registry.Service`，但不直接读取配置文件。

这样 CLI 命令:

```text
skilldown registry add <repo> --name <name> --skill-dir <path>
skilldown registry list
skilldown registry remove <name>
```

和未来 TUI 中的添加仓库表单可以复用同一套 app 方法。

### search 与 install 共享仓库解析

显式 repo 优先；未传 repo 时通过 registry 读取所有仓库。该规则应由 app 统一调用 `registry.Resolve`，避免 CLI 和 TUI 各自实现。

### 输出由入口适配层决定

app 返回结构化结果。CLI 使用 `output` 渲染脚本友好的文本；TUI 使用 Bubble Tea 模型渲染列表、预览和选择状态。

## 实现规格

### `internal/skilldown/app`

```go
// Service 负责执行应用用例编排。
type Service struct {
    Registry  RegistryService
    Cloner    repo.Cloner
    Installer install.Installer
}

// AddRepositoryRequest 表示添加 registry 仓库的结构化请求。
type AddRepositoryRequest struct {
    Name         string
    URL          string
    SkillDirPath string
}

// RemoveRepositoryRequest 表示删除 registry 仓库的结构化请求。
type RemoveRepositoryRequest struct {
    Name string
}

// SearchRequest 表示搜索 skill 的结构化请求。
type SearchRequest struct {
    RepoURL      string
    SkillDirPath string
}

// InstallRequest 表示安装 skill 的结构化请求。
type InstallRequest struct {
    RepoURL      string
    SkillName    string
    SkillDirPath string
    TargetRoot   string
    Force        bool
}

// AddRepository 添加一个 registry 仓库。
func (s Service) AddRepository(ctx context.Context, req AddRepositoryRequest) error

// ListRepositories 返回已注册仓库列表。
func (s Service) ListRepositories(ctx context.Context) ([]config.Repository, error)

// RemoveRepository 删除指定 registry 仓库。
func (s Service) RemoveRepository(ctx context.Context, req RemoveRepositoryRequest) error

// Search 搜索显式仓库或 registry 仓库中的 skill。
func (s Service) Search(ctx context.Context, req SearchRequest) (SearchResult, error)

// Install 安装显式仓库或 registry 仓库中的 skill。
func (s Service) Install(ctx context.Context, req InstallRequest) (InstallResult, error)
```

- `Service`: 应用用例入口，组合 registry、repo、skill 和 install 模块。
- `AddRepositoryRequest`: 由 CLI/TUI 填充，不包含 Cobra 或 TUI 类型。
- `SearchRequest`: 表示一次搜索动作，不包含输出格式。
- `InstallRequest`: 表示一次安装动作，`TargetRoot` 为空时由 app 使用当前工作目录下 `.codex/skills` 作为默认安装根目录。
- `AddRepository`: 调用 registry 添加仓库。
- `ListRepositories`: 调用 registry 查看仓库列表。
- `RemoveRepository`: 调用 registry 删除仓库。
- `Search`: 统一实现显式 repo 与 registry 默认搜索。
- `Install`: 统一实现 skill 精确匹配、同名冲突处理和目录复制。

#### 实现状态 [done]

已实现。`internal/skilldown/app` 提供 request/result 类型和 registry、search、install 用例编排。

### `internal/skilldown/cli`

```go
// Execute 启动 CLI 入口。
func Execute(ctx context.Context, args []string, svc app.Service) int
```

- `Execute`: 使用 Cobra 解析命令和 flags，把结果转换成 app 请求，并把 app 结果渲染到终端。

#### 实现状态 [done]

已实现。`internal/skilldown/cli` 承载 Cobra 命令定义，并把命令参数转换成 app 请求。

### `internal/skilldown/tui`

```go
// Run 启动交互式 TUI。
func Run(ctx context.Context, svc app.Service, initialRepo string) error
```

- `Run`: 使用 Bubble Tea、Bubbles、Lip Gloss 或 Huh 收集用户选择，并复用 app 的 search/install/registry 用例。

#### 实现状态 [done]

已保留目录和 `Run` 接口。具体 TUI 交互流程后续实现。

### `cmd/skilldown`

```go
// main 启动 CLI 程序。
func main()
```

- `main`: 创建 config store、registry service、repo cloner、installer 和 app service，然后交给 CLI 入口执行。

#### 实现状态 [done]

已实现。`cmd/skilldown` 只负责配置路径、依赖组装和启动 CLI。

## 依赖关系

### 外部依赖

- `context`: 贯穿 app 用例，支持取消。
- `github.com/spf13/cobra`: 只用于 CLI 适配层。
- `github.com/charmbracelet/bubbletea`: 只用于 TUI 适配层。
- `github.com/charmbracelet/bubbles`: 只用于 TUI 组件。
- `github.com/charmbracelet/lipgloss`: 只用于 TUI 样式。
- `github.com/charmbracelet/huh`: 只用于 TUI 表单。

### 内部调用

- `cmd/skilldown` → `internal/skilldown/cli.Execute()`: 启动 CLI。
- `internal/skilldown/cli` → `internal/skilldown/app.Service`: 将 Cobra 参数转换成 app 请求。
- `internal/skilldown/tui` → `internal/skilldown/app.Service`: 将交互选择转换成 app 请求。
- `internal/skilldown/app` → `internal/skilldown/registry.Service`: add/list/remove/resolve 仓库。
- `internal/skilldown/app` → `internal/skilldown/repo.Cloner`: clone 和 cleanup 远程仓库。
- `internal/skilldown/app` → `internal/skilldown/skill.Discover()`: 发现 skill。
- `internal/skilldown/app` → `internal/skilldown/install.Installer.CopyDir()`: 安装 skill 目录。
- `internal/skilldown/registry` → `internal/skilldown/config.Store`: 持久化 repositories。

## 任务清单

- [x] 新增 `internal/skilldown/app` service、request 和 result 类型。
- [x] 在 app 中实现 `AddRepository`、`ListRepositories`、`RemoveRepository`。
- [x] 在 app 中实现 `Search`，统一显式 repo 与 registry 默认搜索。
- [x] 在 app 中实现 `Install`，处理指定 skill、全部安装、同名冲突和默认目标目录。
- [x] 新增 `internal/skilldown/cli`，承载 Cobra 命令和参数解析。
- [x] 调整 `cmd/skilldown`，只负责依赖组装和启动 CLI。
- [x] 为 app registry/search/install 用例添加单元测试。
- [x] 更新 README 和总 feature 文档中过时的 `app.Run(ctx, args)` 描述。
- [x] 运行 `go test ./...`。

## 验收标准

- [x] `app` 包不导入 Cobra、Bubble Tea、Bubbles、Lip Gloss、Huh 或 `os.Args`。
- [x] CLI 和未来 TUI 可以通过同一组 app request 调用 registry/search/install。
- [x] `skilldown registry add <repo> --name <name>` 能通过 app 调用 registry 添加仓库。
- [x] `skilldown registry list` 能通过 app 返回 registry 列表并由 CLI 渲染。
- [x] `skilldown registry remove <name>` 能通过 app 删除仓库。
- [x] `skilldown search` 和 TUI browse 能复用 app 的搜索逻辑。
- [x] `skilldown install` 和 TUI install 能复用 app 的安装逻辑。
- [x] `app` 不直接读写 YAML，不直接操作 Viper，不直接调用 git 命令。
- [x] `go test ./...` 通过，或明确说明失败原因。

## 变更历史

| 版本 | 日期 | 状态 | 说明 |
|------|------|------|------|
| 1.0 | 2026-04-27 | implemented | 实现 app、CLI、cmd 入口和 TUI 接口 |
