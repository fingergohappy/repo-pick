---
title: tui-repo-downloader
type: feature
date: 2026-05-02
status: draft
version: "1.0"
summary: 将 repo-pick 从 skill 专用下载器调整为 TUI-only 的远程 Git 仓库文件/目录下载器。程序启动后进入 TUI，registry 作为仓库书签，cache 保存仓库的 shallow clone 完整工作区，用户在目录树中选择文件或目录并下载到目标目录。
scope:
  - cmd/repo-pick
  - internal/repopick/app
  - internal/repopick/cache
  - internal/repopick/config
  - internal/repopick/install
  - internal/repopick/registry
  - internal/repopick/tui
---

# tui-repo-downloader

## 概述

`repo-pick` 后续不再限制为下载 `skills/<name>/SKILL.md` 结构，也不再提供 `search`、`install`、`registry` 等命令式 CLI 子命令。程序启动后直接进入 TUI，用户在 TUI 中管理 registry、浏览当前仓库目录、搜索路径，并把选中的文件或目录下载到当前目录或指定目录。

这个设计取代旧的 `repo-pick browse` + skill 列表模型。核心对象从 `Skill` 调整为普通文件系统 `Entry`，也就是仓库中的文件或目录。

## 目标

- 启动 `repo-pick` 后直接进入 TUI。
- registry 保存用户登记的仓库名称和 URL。
- cache 保存每个仓库的 shallow clone 完整工作区，避免每次打开都重新下载。
- TUI 默认聚焦左侧 registry，通过 `ctrl-w h/l` 在 registry 和目录树之间切换焦点。
- 用户可以用 `h/j/k/o` 在目录树中返回、移动和进入目录。
- 用户可以下载当前选中的文件或目录。
- `/` 搜索当前仓库的全部路径。
- 删除 registry 时同步删除该仓库 cache。
- 下载目标同名时弹确认，用户选择覆盖或取消。

## 设计原则

- 只保留当前最优方案，不做旧接口、旧配置、旧命令或旧目录规则的兼容性改动。
- 删除不再服务新模型的 skill 专用逻辑、Cobra 子命令、sparse checkout 流程和 `skillDir` 配置字段。
- 不保留 legacy alias、deprecated wrapper 或双轨实现，避免维护者误以为旧路径仍被支持。
- 修改时直接收敛到 TUI-only、registry + cache、Entry 下载模型。

## 非目标

- 不再识别 skill 或 command 的特殊结构。
- 不再要求 `SKILL.md`、`COMMAND.md` 或 front matter。
- 不保留 Cobra 子命令交互面，例如 `repo-pick search`、`repo-pick install`。
- 不做 sparse clone，因为 TUI 需要自由进入仓库任意目录。
- 不下载完整 Git 历史，只下载当前分支浅历史。
- `/` 第一版只做路径搜索，不做文件全文搜索。
- 同名目标第一版不做自动改名。

## 核心概念

### Registry

Registry 是用户维护的仓库书签列表，持久化在用户配置文件中。

```text
~/.config/repo-pick/config.yaml
```

Registry 只保存仓库名称和 URL，不保存 clone 内容，不表达 skill、command 或默认下载目录。

第一版要求 registry name 唯一，并强制 repo URL 唯一。URL 唯一可以避免两个 registry 共用同一份 cache 时删除行为变复杂。

### Cache

Cache 是 registry 仓库在本地的 shallow clone 工作区。

```text
~/.cache/repo-pick/repos/<url-hash>/
```

Cache 被 `repo-pick` 独占管理，用户不应在 cache 中编辑文件。更新仓库时可以直接删除旧 cache 后重新下载；如果重新下载失败，该仓库本次不能浏览是正常结果，用户可以稍后再次更新或重新打开。

### Entry

Entry 表示仓库中的一个文件或目录。

```go
type Entry struct {
    Name string
    Path string
    Type EntryType
    Size int64
}
```

- `Name`: 文件或目录名，用于展示和目标路径拼接。
- `Path`: 相对仓库根目录的 slash 风格路径。
- `Type`: 文件或目录。
- `Size`: 文件大小；目录大小第一版可以为空或显示为 `-`。

### Session CWD

Session CWD 是用户启动 `repo-pick` 时所在的目录。按 `i` 下载时默认使用 Session CWD，而不是 TUI 运行过程中可能变化的其它路径。

## Git 缓存策略

### 首次打开仓库

如果当前 registry 没有 cache，TUI 打开该仓库时执行 shallow clone:

```bash
git clone --depth 1 --single-branch <repo-url> <cache-dir>
```

这里的“完整仓库”指当前分支最新工作区的全部文件，不包括完整提交历史，也不包括其它分支历史。

### 后续打开仓库

如果 cache 已存在，默认直接读取本地 cache，不联网。

### 更新仓库

用户在 TUI 中按 `u` 更新当前 repo cache。更新直接删除旧 cache 后重新 shallow clone：

```bash
rm -rf <cache-dir>
git clone --depth 1 --single-branch <repo-url> <cache-dir>
```

因为 cache 是工具独占目录，删除只作用于 cache，不影响用户项目目录。更新失败时不恢复旧 cache。

### 删除仓库

用户在 TUI 左侧按 `d` 删除当前 registry。确认后执行：

1. 从 config 删除 registry 记录。
2. 删除该 URL 对应的 cache 目录。
3. 从 TUI 列表移除该仓库。

如果 cache 删除失败，TUI 应展示错误，并且不要静默吞掉失败原因。

## TUI 布局

第一版使用双主栏加底部状态区。

```text
+----------------------+-----------------------------------------+
| Registry             | Repository Tree                         |
| > personal           | /                                       |
|   official           |   commands/                             |
|   templates          |   skills/                               |
|                      |   README.md                             |
+----------------------+-----------------------------------------+
| status: personal cached | i download | I choose target | / search |
+----------------------------------------------------------------+
```

右侧 preview 不是第一版必要能力。后续如果需要预览文本文件，可以在不改变主流程的情况下增加第三栏。

## 交互规则

### 全局快捷键

```text
ctrl-w h/l 在左侧 registry 和中间目录树之间切换焦点
/       搜索当前 repo 的全部路径
Esc     关闭搜索框、确认框或错误提示
q       退出
?       显示快捷键帮助
```

### 左侧 Registry 焦点

```text
j       下移 registry 光标
k       上移 registry 光标
l       打开当前 registry 对应的 repo tree
ctrl-w l 切到中间目录树；如果仓库未打开，先打开当前 registry
a       新增 registry，输入 name 和 url
d       删除当前 registry，确认后同步删除 cache
u       更新当前 registry 的 cache
```

TUI 启动后默认焦点在左侧 registry。

### 中间目录树焦点

```text
h       返回上级目录
j       下移条目光标
k       上移条目光标
l       展开或收起选中目录
o       当前条目是目录时进入目录；当前条目是文件时不进入
i       下载当前文件或目录到 Session CWD
I       输入目标目录后下载当前文件或目录
/       搜索当前 repo 的全部路径
ctrl-w h 切回左侧 registry
```

目录树中的 `Path` 始终是相对 repo 根目录的路径。进入和返回只改变 TUI 当前路径，不触发网络请求。

### 搜索

`/` 打开搜索输入框，搜索范围是当前 repo 的全部路径。第一版只匹配路径名，不读取文件内容。

```text
query: review

commands/review.md
skills/code-review/
docs/review-workflow.md
```

搜索结果仍然是 Entry。用户可以在结果中移动光标，按 `l` 跳转到目录或定位文件，按 `i` / `I` 下载结果项。

## 下载规则

下载动作支持文件和目录。

```text
选中 commands/review.md
按 i
=> <session-cwd>/review.md

选中 skills/code-review/
按 i
=> <session-cwd>/code-review/

选中 templates/go-api/
按 I，输入 /tmp/demo
=> /tmp/demo/go-api/
```

目标参数始终表示“目标目录”，不是完整目标文件名。最终路径统一是：

```text
<target-dir>/<selected-entry-name>
```

如果选中的是仓库根目录，`selected-entry-name` 使用 registry name。这样下载根目录时目标路径仍然明确：

```text
选中 /
registry name = personal
按 i
=> <session-cwd>/personal/
```

### 同名确认

如果目标路径已存在，TUI 弹出确认：

```text
review.md already exists. Overwrite?
y overwrite    n cancel
```

用户确认后覆盖该目标文件或目录；用户取消后不做任何写入。第一版不自动生成 `review-1.md` 这类名称。

## 模块设计

### `cmd/repo-pick`

程序入口只做依赖组装并启动 TUI。

```go
func main() {
    // 创建 config store、registry service、cache service、installer 和 TUI。
}
```

不再通过 Cobra 注册子命令。

### `internal/repopick/config`

配置继续负责读写用户级 config。Repository 应收敛为通用仓库配置。

```go
type Repository struct {
    Name string `yaml:"name"`
    URL  string `yaml:"url"`
}
```

旧的 `SkillDir` 字段不再服务新模型，应直接删除，不做兼容读取或写出。

### `internal/repopick/registry`

Registry 继续负责 add/list/remove 规则。

新增规则：

- `Add` 校验 name 非空。
- `Add` 校验 URL 非空。
- `Add` 拒绝重复 name。
- `Add` 拒绝重复 URL。
- `Remove` 只删除 config 记录，cache 删除由更高层编排，避免 registry 直接依赖 cache。

### `internal/repopick/cache`

新增 cache 模块，负责 repo cache 生命周期。

```go
type Service struct {
    RootDir string
    GitPath string
}

func (s Service) Ensure(ctx context.Context, repo config.Repository) (Worktree, error)
func (s Service) Update(ctx context.Context, repo config.Repository) (Worktree, error)
func (s Service) Delete(repo config.Repository) error
```

- `Ensure`: cache 存在时直接返回；不存在时 shallow clone。
- `Update`: 删除旧 cache 并重新 shallow clone；不存在时等价于首次 clone。
- `Delete`: 删除该 repo URL 对应的 cache 目录。

Cache key 使用 repo URL 的稳定 hash，不使用 registry name。

### `internal/repopick/tree`

可以新增 tree 模块，负责读取 cache worktree 中的文件树。

```go
func List(root string, dirPath string) ([]Entry, error)
func Search(root string, query string) ([]Entry, error)
```

- `List`: 只列当前目录的直接子级。
- `Search`: 遍历当前 repo 全部路径并按路径名匹配。
- 两者都必须跳过 `.git`。
- 两者都不能返回逃逸到 repo 根目录外的路径。

### `internal/repopick/install`

复制能力统一收敛到 `CopyEntry`，不保留目录专用双轨接口。

```go
func (i Installer) CopyEntry(ctx context.Context, sourcePath string, targetPath string, force bool) Result
```

- source 是目录时递归复制目录。
- source 是普通文件时复制单个文件。
- target 已存在且 force=false 时返回失败。
- target 已存在且 force=true 时只删除 target 自身。
- 不跟随或复制符号链接特殊语义，第一版只支持普通文件和目录。

### `internal/repopick/app`

app 层可以保留为 TUI 复用的用例层，但类型需要从 skill 语义改为 repo entry 语义。

```go
func (s Service) ListRepositories(ctx context.Context) ([]config.Repository, error)
func (s Service) AddRepository(ctx context.Context, req AddRepositoryRequest) error
func (s Service) RemoveRepository(ctx context.Context, req RemoveRepositoryRequest) error
func (s Service) EnsureRepository(ctx context.Context, repo config.Repository) (RepositoryState, error)
func (s Service) UpdateRepository(ctx context.Context, repo config.Repository) (RepositoryState, error)
func (s Service) ListEntries(ctx context.Context, req ListEntriesRequest) (ListEntriesResult, error)
func (s Service) SearchEntries(ctx context.Context, req SearchEntriesRequest) (SearchEntriesResult, error)
func (s Service) DownloadEntry(ctx context.Context, req DownloadEntryRequest) (DownloadEntryResult, error)
```

`RemoveRepository` 负责同时调用 registry remove 和 cache delete，保证 TUI 删除动作语义完整。

### `internal/repopick/tui`

TUI 只负责状态、输入、渲染和弹窗，不直接调用 git，也不直接复制文件。

核心状态建议：

```go
type focusPane int

const (
    focusRegistry focusPane = iota
    focusTree
)

type model struct {
    sessionCWD string
    focus focusPane
    repositories []config.Repository
    selectedRepo int
    currentPath string
    entries []app.EntryResult
    selectedEntry int
    searchQuery string
    searchResults []app.EntryResult
    pendingConfirm *confirmState
    status string
    err error
}
```

## 核心流程

### 启动

```text
repo-pick
    main 记录 Session CWD
    main 初始化 config、registry、cache、app、tui
    tui 加载 registry 列表
    焦点默认在左侧 registry
```

### 打开仓库

```text
用户在左侧按 l 或 ctrl-w l
    app.EnsureRepository(repo)
        cache 存在: 直接返回 worktree
        cache 不存在: shallow clone 完整工作区
    app.ListEntries(repo, "/")
    TUI 展示根目录条目
```

### 浏览目录

```text
中间栏按 l
    如果当前 Entry 是目录:
        currentPath = entry.Path
        app.ListEntries(repo, currentPath)
    如果当前 Entry 是文件:
        不进入，状态栏提示当前是文件

中间栏按 h
    currentPath = parent(currentPath)
    app.ListEntries(repo, currentPath)
```

### 新增 Registry

```text
左侧按 a
    弹出 name/url 输入
    app.AddRepository
    重新加载 registry 列表
```

新增 registry 不立即 clone。只有用户打开该 repo 或按 `u` 更新时才访问网络。

### 删除 Registry

```text
左侧按 d
    弹确认
    app.RemoveRepository
        registry.Remove(name)
        cache.Delete(repo)
    TUI 重新加载 registry 列表
```

### 下载 Entry

```text
中间栏按 i
    targetDir = Session CWD
    targetPath = targetDir + selectedEntry.Name
    如果 targetPath 存在:
        弹确认
    用户确认或 targetPath 不存在:
        app.DownloadEntry

中间栏按 I
    弹输入框读取 targetDir
    后续同 i
```

## 验收标准

- [ ] 运行 `repo-pick` 会直接进入 TUI，不需要 CLI 子命令。
- [ ] 启动后默认焦点在左侧 registry。
- [ ] `a` 可以新增 registry。
- [ ] `d` 可以删除 registry，并同步删除该 repo cache。
- [ ] 首次打开 repo 会 shallow clone 完整工作区到 cache。
- [ ] 已有 cache 时打开 repo 不联网。
- [ ] `u` 可以删除旧 cache 并重新下载；失败时不恢复旧 cache。
- [ ] `ctrl-w h/l` 可以在 registry 和目录树之间切换焦点。
- [ ] 目录树中 `h/j/k/o` 可以返回、移动和进入目录。
- [ ] `/` 可以搜索当前 repo 的路径。
- [ ] `i` 可以下载当前文件或目录到 Session CWD。
- [ ] `I` 可以输入目标目录并下载当前文件或目录。
- [ ] 目标同名时会询问是否覆盖。
- [ ] 取消覆盖不会修改目标文件或目录。
- [ ] `go test ./...` 通过，或明确说明失败原因。

## 迁移步骤

1. 新增 cache 和 tree 能力，先不动 TUI。
2. 扩展 install，使其支持文件和目录统一复制。
3. 调整 app 层，从 skill 用例迁移到 repo entry 用例。
4. 移除 Cobra CLI 子命令，让 `cmd/repo-pick` 直接启动 TUI。
5. 重写 TUI 状态模型和键位。
6. 更新 README，删除 skill 专用描述。
7. 运行测试并补充 registry/cache/tree/install/TUI 状态测试。

## 待确认

- 文本文件 preview 是否进入第一版。
