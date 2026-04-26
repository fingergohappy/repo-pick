---
title: repo
type: feature
date: 2026-04-27
status: archived
version: "1.0"
summary: 设计 repo 模块的远程仓库 clone 能力，支持把仓库临时浅 clone 到本机，并在默认路径下尽量减少磁盘占用。第一版不做长期缓存，命令结束后清理临时 clone。
scope:
  - internal/skilldown/repo
  - internal/skilldown/app
  - test/testdata
---

# repo

## 概述

repo 功能负责把远程 Git 仓库临时浅 clone 到本机，向上层返回可读取的 worktree 路径，并在命令结束后清理临时数据。

## 背景

`skill-down` 需要从远程仓库搜索并下载 skill。相比直接调用特定托管平台 API，第一版使用本机 `git` 命令可以保持实现简单，并天然支持常见 Git URL。为了避免下载整个仓库和长期占用磁盘，repo 功能必须使用 shallow clone，并默认放在系统临时目录中。

第一版 repo 只解决“把远程仓库临时 clone 到本机并管理生命周期”的需求，不实现 skill 搜索、`SKILL.md` 解析、安装策略、持久缓存、仓库镜像、增量更新、认证适配或多平台 API 封装。

## 核心概念

### 临时 Worktree

- 职责: 表示一次命令执行期间可读取的本地仓库副本。
- 粒度: per repo clone。
- 边界: 不作为长期缓存保存；命令结束后默认删除。

### 浅 Clone

- 职责: 只获取远程仓库最新提交所需内容，降低网络和磁盘成本。
- 粒度: per repo URL。
- 边界: 不支持历史版本搜索，不实现版本锁定。

## 核心流程

```text
parse CLI args
worktree = repo.Clone(ctx, repoURL, cloneOptions)
defer repo.Cleanup(worktree)

app passes worktree.Dir to skill or install modules
repo only owns clone and cleanup
```

## 关键机制

### Clone 位置

默认使用系统临时目录创建 clone 目录:

```go
dir, err := os.MkdirTemp("", "skill-down-*")
```

这样不会污染当前项目，也不需要维护 cache 失效、并发锁和清理策略。第一版不提供默认持久缓存；如后续确实需要，可新增显式参数，例如 `--keep-worktree` 或 `--cache-dir`，但默认仍应清理。

### Git Clone 策略

必须使用 shallow clone 和 sparse checkout 来限制下载内容。repo 模块只接收上层传入的 sparse paths，不内置 `skills` 目录规则:

```bash
git clone --depth=1 --sparse <repo> <tmpdir>
git -C <tmpdir> sparse-checkout set <paths...>
```

这里的 sparse checkout 只是 clone 阶段的 I/O 优化，不表示 repo 模块负责识别 skill。是否传入 `skills`、是否扫描 `skills/<name>`、如何解析 `SKILL.md` 都属于上层流程和 `skill` 模块。

如果当前 Git 版本和远程服务支持 partial clone，可以额外增加 `--filter=blob:none`:

```bash
git clone --depth=1 --filter=blob:none --sparse <repo> <tmpdir>
```

partial clone 只作为进一步减少 blob 下载的优化；无论是否启用 partial clone，都不能退回完整 clone。

### 清理策略

`Clone` 成功后，调用方必须在命令结束时执行 `Cleanup`。`Cleanup` 只删除 repo 模块创建的临时目录，避免误删用户指定路径。

## 实现规格

### `internal/skilldown/repo`

```go
// Worktree 表示一次临时 clone 的本地仓库副本。
type Worktree struct {
    Dir string
}

// CloneOptions 表示 clone 阶段的 I/O 约束，不表达业务搜索规则。
type CloneOptions struct {
    SparsePaths []string
}

// Cloner 负责把远程仓库 clone 到本机临时目录，并管理该临时目录的清理。
type Cloner interface {
    Clone(ctx context.Context, repoURL string, options CloneOptions) (Worktree, error)
    Cleanup(worktree Worktree) error
}

// GitCloner 使用本机 git 命令实现浅 clone。
type GitCloner struct {
    GitPath string
}

func (c GitCloner) Clone(ctx context.Context, repoURL string, options CloneOptions) (Worktree, error)
func (c GitCloner) Cleanup(worktree Worktree) error
```

- `Worktree`: 只保存 repo 模块创建的临时目录路径。
- `CloneOptions`: 只描述 clone 时需要减少读取的路径范围，不解析路径语义。
- `Cloner`: 隔离 clone 与清理契约，便于测试替换。
- `GitCloner`: 第一版唯一实现，内部通过 `exec.CommandContext` 调用本机 `git`。
- `Clone`: 创建临时目录并执行浅 clone；失败时清理已创建目录。
- `Cleanup`: 删除 `Worktree.Dir`，只接受 repo 模块返回的 worktree。

#### 实现状态 [done]

已实现。当前实现会创建系统临时目录，执行 shallow clone、sparse checkout 和 partial clone fallback，并在命令失败时返回包含 git 输出的错误信息。

### `internal/skilldown/app`

```go
func Run(ctx context.Context, args []string) int
```

- `Run`: 编排 CLI 参数、repo clone、后续模块调用和清理流程。

#### 实现状态 [todo]

尚未实现。实现时应确保 `Cleanup` 在 search 和 install 两条路径中都会执行；skill 搜索和安装规则由对应模块处理。

## 依赖关系

### 外部依赖

- `git`: 执行 shallow clone、可选 partial clone 与强制 sparse checkout。
- `os`: 创建和清理临时目录。
- `os/exec`: 调用本机 `git` 命令。
- `path/filepath`: 拼接和校验本地路径。

### 内部调用

- `internal/skilldown/app` → `internal/skilldown/repo.GitCloner.Clone()`: 在 search 和 install 开始时创建临时 worktree，并按流程需要传入 sparse paths。
- `internal/skilldown/app` → `internal/skilldown/repo.GitCloner.Cleanup()`: 在命令结束时清理临时 worktree。
- `internal/skilldown/app` → `internal/skilldown/skill`: 将 `Worktree.Dir` 交给 skill 模块做发现、解析和搜索过滤。
- `internal/skilldown/app` → `internal/skilldown/install`: 将 `Worktree.Dir` 和已选中的 skill 交给 install 模块执行安装。

## 任务清单

- [x] 定义 `internal/skilldown/repo.Worktree`、`CloneOptions`、`Cloner` 和 `GitCloner`。
- [x] 实现基于 `os.MkdirTemp("", "skill-down-*")` 的 clone 目录创建。
- [x] 实现强制使用 `git clone --depth=1 --sparse` 的浅 clone。
- [x] 实现基于 `CloneOptions.SparsePaths` 的 `git sparse-checkout set <paths...>`。
- [x] 在支持 partial clone 时增加 `--filter=blob:none`，不支持时仍保持 shallow clone 和 sparse checkout。
- [x] 实现 clone 失败时清理已创建临时目录。
- [x] 实现命令结束后的 worktree 清理。
- [x] 增加基于 `test/testdata` 的 repo clone 与清理测试。
- [x] 运行 `go test ./...` 验证相关逻辑。

## 验收标准

- [x] `repo.Clone(ctx, repoURL, options)` 能返回存在且可读的 `Worktree.Dir`。
- [x] repo 模块不解析 `SKILL.md`，不返回 skill 名称、描述或搜索结果。
- [x] 默认 clone 目录位于系统临时目录，不写入当前项目。
- [x] 命令结束后临时 clone 目录被删除。
- [x] clone 始终使用最新提交的浅 clone，不下载完整历史。
- [x] clone 始终执行 sparse checkout，不退回完整 worktree。
- [x] 目标 Git 不支持 partial clone 时仍可通过 shallow clone 和 sparse checkout 返回可读 worktree。

## 验收记录

- 2026-04-27: `go test ./internal/skilldown/repo` 通过。
- 2026-04-27: `go test ./...` 通过。

## 变更历史

| 版本 | 日期 | 状态 | 说明 |
|------|------|------|------|
| 1.0 | 2026-04-27 | accepted | 按现有 repo 实现完成验收 |
