---
title: tui-interface
type: feature
date: 2026-04-27
status: draft
version: "1.0"
summary: 设计并实现 `skilldown browse` 的三栏 TUI 界面，用于交互式浏览 registry、搜索 skill、预览 `SKILL.md` 并安装选中的 skill。TUI 只负责交互状态和渲染，业务流程复用现有 app 用例层。
scope:
  - cmd/skilldown
  - internal/skilldown/cli
  - internal/skilldown/tui
  - internal/skilldown/app
  - internal/skilldown/config
  - internal/skilldown/registry
  - internal/skilldown/output
---

# tui-interface

## 概述

新增 `skilldown browse [repo]` 交互式 TUI 入口，采用三栏布局展示 registry、skill 列表和 `SKILL.md` 预览，并通过 Vim 风格快捷键完成搜索、选择、刷新和安装。

## 背景

当前项目已经具备 `search`、`install`、`registry` 等 CLI 能力，也已经预留 `internal/skilldown/tui.Run`，但 TUI 尚未实现。普通 CLI 适合脚本调用，交互式选择多个 skill、查看完整 `SKILL.md`、确认安装目录等流程则需要更直观的终端界面。

TUI 设计应继续复用 `internal/skilldown/app.Service`，避免在 TUI 中重复实现仓库解析、clone、skill 发现和安装规则。界面层只处理用户输入、状态切换、列表筛选、预览渲染和确认弹窗。

## 核心概念

### 三栏主界面

- 职责: 将浏览流程拆成 registry 来源、skill 列表、skill 预览三个稳定区域。
- 粒度: per TUI session。
- 边界: 不在主界面承载复杂命令面板，不把安装表单常驻在 preview 区域。

### Registry 栏

- 职责: 展示已注册仓库，支持搜索 registry、添加 registry，并在光标停留后自动载入当前仓库。
- 粒度: per repository entry。
- 边界: 不直接读写配置文件；添加 registry 时仍调用 app 层方法。

### Skill 列表栏

- 职责: 展示当前仓库发现到的 skill，支持搜索、上下移动、多选、刷新当前仓库和设置当前搜索目录。
- 粒度: per loaded repository。
- 边界: 不直接 clone 仓库，不直接扫描目录；刷新仍通过 app 搜索能力完成。

### Preview 栏

- 职责: 展示当前光标所在 skill 的名称、描述和完整 `SKILL.md` 内容。
- 粒度: per selected skill cursor。
- 边界: 只负责阅读预览，不放安装按钮，不改变安装状态。

### 安装确认框

- 职责: 在用户按 `i` 后确认安装目录和覆盖策略，并触发安装。
- 粒度: per install action。
- 边界: 只暴露 `targetRoot` 和 `force` 两个第一版必需选项，不增加更新、删除或 marketplace 行为。

## 核心流程

```text
skilldown browse [repo]
    cli 创建 browse 命令
    cli 调用 tui.Run(ctx, service, initialRepo)

tui.Run
    初始化三栏 model
    如果 initialRepo 非空，作为显式仓库来源
    否则加载 registry 列表
    渲染 registry / skills / preview 三栏

registry 光标移动
    j/k 更新当前 registry 光标
    启动短延迟防抖
    防抖结束后调用 app.Search
    中间栏展示搜索结果
    右侧 preview 展示当前 skill 的 SKILL.md

skill 列表操作
    / 过滤当前 skill 列表
    j/k 移动 skill 光标
    space 切换选中状态
    r 重新搜索当前仓库
    a 设置当前 skillDirPath 后重新搜索

安装操作
    i 打开确认框
    用户确认 targetRoot 和 force
    tui 调用 app.Install
    tui 展示每个 skill 的安装结果
```

## 关键机制

### Vim 风格快捷键

主界面使用固定快捷键，按当前聚焦栏目解释上下文动作：

```text
h/l     左右切换栏目
j/k     当前栏目上下移动
H       快速回到左侧 registry 栏
L       快速跳到右侧 preview 栏
/       当前栏目搜索：左栏搜索 registry，中栏搜索 skill
a       左栏添加 registry；中栏设置 skillDirPath
space   中栏选择/取消选择 skill
r       中栏刷新当前仓库 skill 列表
i       打开安装确认框
?       显示快捷键帮助
q       退出
```

### Registry 自动载入防抖

左栏 `j/k` 只立即更新光标，不立即 clone。光标在某个 registry 上停留短暂延迟后，TUI 再触发当前仓库搜索。这样可以避免用户快速上下移动时连续 clone 多个远程仓库。

### 复用 app 用例层

TUI 所有业务动作都应转换为 app 请求：

```go
svc.Search(ctx, app.SearchRequest{
    RepoURL:      repoURL,
    SkillDirPath: skillDirPath,
})

svc.Install(ctx, app.InstallRequest{
    RepoURL:      repoURL,
    SkillName:    skillName,
    SkillDirPath: skillDirPath,
    TargetRoot:   targetRoot,
    Force:        force,
})
```

TUI 不直接调用 `repo.Clone`、`skill.Discover` 或 `install.CopyDir`。

### 安装确认

按 `i` 打开确认框。确认框只允许修改：

- `targetRoot`: 安装根目录；默认沿用 app 当前逻辑，即命令启动目录下 `.codex/skills`。
- `force`: 是否覆盖已存在目标目录；默认关闭。

确认后对已选中的 skill 执行安装。未选中任何 skill 时，TUI 应提示先选择 skill，不隐式安装全部。

## 实现规格

### `internal/skilldown/cli/cli.go`

```go
func newBrowseCommand(ctx context.Context, svc app.Service) *cobra.Command
```

- `newBrowseCommand`: 新增 `browse [repo]` 命令，接收可选显式仓库地址，并调用 `tui.Run(ctx, svc, initialRepo)`。
- 实现状态 [todo]

### `internal/skilldown/tui/tui.go`

```go
func Run(ctx context.Context, svc app.Service, initialRepo string) error
```

- `Run`: 启动 Bubble Tea 程序，初始化三栏模型，绑定 app 服务和初始仓库。
- 实现状态 [todo]

### `internal/skilldown/tui`

```go
type model struct {
    svc app.Service
    initialRepo string
    focus pane
    repositories []config.Repository
    skills []app.SkillResult
    selected map[string]bool
    skillDirPath string
    targetRoot string
    force bool
    err error
}
```

- `model`: TUI 的核心状态，保存聚焦栏目、registry 列表、当前 skill 列表、选中项、安装选项和错误状态。
- `selected`: 第一版可以用 skill 名称作为 key；如果后续支持跨仓库同时选择，再升级为 repository + skill 组合 key。
- 实现状态 [todo]

### `internal/skilldown/tui` 消息与命令

```go
type searchResultMsg struct {
    result app.SearchResult
    err error
}

type installResultMsg struct {
    result app.InstallResult
    err error
}
```

- `searchResultMsg`: 承载异步搜索结果，更新 skill 列表和 preview。
- `installResultMsg`: 承载安装结果，更新底部状态或结果视图。
- 实现状态 [todo]

### `internal/skilldown/tui` 测试

```go
func TestModelMovesFocusWithVimKeys(t *testing.T)
func TestModelSelectsSkillsWithSpace(t *testing.T)
func TestModelDebouncesRegistryLoad(t *testing.T)
func TestModelOpensInstallConfirm(t *testing.T)
```

- TUI 测试优先覆盖状态转移和命令触发，不要求验证完整 ANSI 渲染。
- 实现状态 [todo]

## 任务清单

- [ ] 在 CLI 中新增 `browse [repo]` 命令，并接入 `tui.Run`。
- [ ] 将 `internal/skilldown/tui.Run` 从占位实现改为 Bubble Tea 程序入口。
- [ ] 实现三栏 TUI 状态模型：registry、skills、preview。
- [ ] 实现 Vim 风格快捷键：`h`、`l`、`j`、`k`、`H`、`L`、`/`、`a`、`space`、`r`、`i`、`?`、`q`。
- [ ] 实现 registry 光标停留后的防抖自动载入。
- [ ] 实现左栏 registry 搜索和中栏 skill 搜索。
- [ ] 实现中栏 `a` 设置 `skillDirPath` 并重新搜索。
- [ ] 实现中栏 `r` 刷新当前仓库 skill 列表。
- [ ] 实现 `space` 多选 skill 和选中数量展示。
- [ ] 实现右栏 `SKILL.md` preview 渲染。
- [ ] 实现 `i` 安装确认框，支持 `targetRoot` 和 `force`。
- [ ] 调用 `app.Service.Install` 安装已选 skill，并展示每项结果。
- [ ] 增加聚焦移动、选择、刷新、安装确认等状态测试。
- [ ] 运行 `go test ./...` 验证现有 CLI/app 行为不回退。

## 验收标准

- [ ] `skilldown browse` 可以从 registry 列表进入三栏 TUI。
- [ ] `skilldown browse <repo>` 可以直接使用显式仓库作为来源。
- [ ] 左栏移动到 registry 后不会立即连续 clone，停留后才触发搜索。
- [ ] 中间栏可以搜索、移动、刷新、设置 `skillDirPath` 并多选 skill。
- [ ] 右侧 preview 能展示当前 skill 的名称、描述和完整 `SKILL.md` 内容。
- [ ] 按 `i` 会打开安装确认框，确认后只安装已选 skill。
- [ ] 安装流程复用 `app.Service.Install`，TUI 不重复实现 clone、discover、copy 逻辑。
- [ ] `?` 能展示完整快捷键说明。
- [ ] 退出 TUI 后不留下未清理的临时仓库 worktree。
- [ ] `go test ./...` 通过。

## 依赖关系

### 外部依赖

- `github.com/charmbracelet/bubbletea`: TUI 状态循环和消息处理。
- `github.com/charmbracelet/bubbles`: 列表、文本输入、viewport 等组件。
- `github.com/charmbracelet/lipgloss`: 三栏布局、聚焦样式和状态栏样式。
- `github.com/charmbracelet/huh`: 可用于添加 registry、设置目录和安装确认等简单表单。

### 内部调用

- `internal/skilldown/cli` → `internal/skilldown/tui.Run()`: `browse` 命令启动 TUI。
- `internal/skilldown/tui` → `internal/skilldown/app.Service.ListRepositories()`: 初始化 registry 列表。
- `internal/skilldown/tui` → `internal/skilldown/app.Service.AddRepository()`: 左栏添加 registry。
- `internal/skilldown/tui` → `internal/skilldown/app.Service.Search()`: 自动载入、刷新和目录变更后搜索 skill。
- `internal/skilldown/tui` → `internal/skilldown/app.Service.Install()`: 安装确认后安装已选 skill。

## 非目标

- [skip] 更新已安装 skill — 第一版 README 已明确暂不包含。
- [skip] 删除已安装 skill — 第一版 README 已明确暂不包含。
- [skip] marketplace 能力 — 第一版 README 已明确暂不包含。
- [skip] 版本锁定 — 第一版 README 已明确暂不包含。
- [skip] 私有仓库认证专门适配 — 第一版 README 已明确暂不包含。
- [skip] 命令面板模式 — 本次已确定采用三栏 Vim 浏览界面。

## 变更历史

| 版本 | 日期 | 状态 | 说明 |
|------|------|------|------|
| 1.0 | 2026-04-27 | draft | 初始版本 |
