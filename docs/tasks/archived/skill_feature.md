---
title: skill
type: feature
date: 2026-04-27
status: archived
version: "1.0"
summary: 设计 `skill` 模块的发现与描述能力，从 repo 模块返回的本地 worktree 中识别指定路径下的 `<name>/SKILL.md`，并为 search、install 和 preview 提供稳定的 skill 列表与完整 `SKILL.md` 内容。默认扫描 `skills` 目录，但允许调用方传入自定义目录路径。
scope:
  - internal/skilldown/skill
  - internal/skilldown/app
  - internal/skilldown/repo
  - internal/skilldown/install
  - internal/skilldown/output
  - test/testdata
---

# skill

## 概述

`skill` 功能负责在本地 worktree 中发现可安装的 skill，并输出给 `search`、`install`、preview 和后续 TUI 流程复用的结构化结果。

## 背景

README 将第一版目标限定为从远程仓库搜索 skill 目录，并下载指定或全部 skill。默认目录是 `skills/<name>`，但调用方可以传入自定义目录路径，例如 `a/b/<name>`。当前 `repo` 模块已经负责把远程仓库临时 clone 到本机；`install` 设计依赖 `skill` 模块提供 `Name` 和 `Path`。因此 `skill` 模块需要成为仓库读取和本地安装之间的窄接口：只判断“什么是 skill”，不做 clone、不写目标目录，也不处理覆盖策略。

发现规则保持简单明确：先扫描 worktree 根目录下指定目录路径对应的目录，默认是 `skills`；只有包含 `SKILL.md` 的一级子目录才视为可安装 skill。如果指定目录路径下没有发现任何 skill，再尝试扫描该路径下的 `skills` 子目录。名称默认来自目录名，可在不增加复杂性的前提下读取 `SKILL.md` front matter 中的 `name` 和 `description` 用于展示，并返回完整 `SKILL.md` 内容供 preview 使用。

## 核心概念

### Skill 目录

- 职责: 表示仓库中一个可安装的 skill 源目录。
- 粒度: per `<dirPath>/<name>` directory。
- 边界: 不识别任意深度目录，不兼容旧布局，不从普通文档目录推断 skill。

### 搜索目录路径

- 职责: 表示 worktree 根目录下用于承载 skill 子目录的相对目录路径。
- 粒度: per discover call。
- 边界: 允许 `a/b` 这种多级相对路径；前导 `/` 会被去掉；不允许空片段、`.` 或 `..`。

### Skill 描述

- 职责: 保存发现结果中 search、install、preview 和输出层需要的最小字段。
- 粒度: per discovered skill。
- 边界: 不表达版本、来源锁定、安装状态或更新状态。

### Skill 内容

- 职责: 保存完整 `SKILL.md` 内容，供 preview 或后续完整详情场景使用。
- 粒度: per `SKILL.md`。
- 边界: 不加载 skill 目录中的其它资源文件，不在 `skill` 模块中拆分 preview 正文、截断或渲染 markdown。

### 发现结果

- 职责: 给上层返回稳定排序后的 skill 列表。
- 粒度: per worktree scan。
- 边界: 不直接打印，不筛选 CLI 参数，不执行安装。

## 核心流程

```text
dirPath = app resolves configured skills directory path, default "skills"
worktree = repo.Clone(ctx, repoURL, CloneOptions{SparsePaths: []string{dirPath}})
defer repo.Cleanup(worktree)

skills = skill.Discover(worktree.Dir, dirPath)

if command == "search":
    output.PrintSkills(skills)

if command == "install":
    selected = app selects by --skill when provided, otherwise all skills
    install selected skills
```

## 关键机制

### 发现规则

优先识别以下结构：

```text
<worktree>/<dirPath>/<skill-name>/SKILL.md
```

如果 `<dirPath>` 下没有发现任何 skill，再识别：

```text
<worktree>/<dirPath>/skills/<skill-name>/SKILL.md
```

发现流程：

```text
if dirPath is empty:
    dirPath = "skills"
dirPath = trim leading "/"

skillsDir = filepath.Join(root, dirPath)
entries = os.ReadDir(skillsDir)

for each entry in entries:
    if entry is not dir:
        continue
    if <dirPath>/<entry.Name>/SKILL.md does not exist:
        continue
    append Skill{Name: entry.Name, Path: "<dirPath>/<entry.Name>"}

if no skills were discovered and dirPath != "skills":
    repeat the same scan under filepath.Join(root, dirPath, "skills")

sort by Skill.Name
```

如果搜索目录不存在，返回空列表和 nil error。这样 `app` 可以统一处理“未发现 skill”的用户提示。

### 目录路径约束

`dirPath` 需要满足路径安全要求：

- 为空时使用默认值 `skills`。
- 允许 `a/b` 这种仓库内相对路径。
- 如果以 `/` 开头，去掉前导 `/` 后按仓库内相对路径处理。
- 不允许空路径片段、`.` 或 `..`。

`skill` 模块只接收仓库内路径，不接收文件系统绝对路径。这样可以支持 monorepo 中的嵌套 skill 目录，同时避免扫描范围逃逸到 worktree 外。

### 元数据解析

`SKILL.md` 的 front matter 可选读取以下字段：

```yaml
---
name: alpha
description: Example skill
---
```

- `name`: 用于展示和匹配时的 skill 名称；为空时使用目录名。
- `description`: 用于 `search` 输出；为空时保持空字符串。

为避免第一版复杂化，元数据解析只支持 YAML front matter。front matter 缺失或字段缺失不应导致发现失败；但 YAML 格式损坏应返回错误，因为这表示 skill 定义文件不可信。

### 内容返回

`Discover` 需要读取并返回 `SKILL.md` 的完整内容：

```text
Skill.Content = full SKILL.md content
```

`Content` 用于 CLI/TUI preview 和后续需要完整 skill 定义的场景。第一版不在 `skill` 模块里做 front matter 去除、markdown 渲染、截断、高亮或终端宽度适配，这些属于 `output` 或 TUI 层。

### 名称约束

`Skill.Name` 需要满足本地安装的路径安全要求：

- 不为空。
- 不包含路径分隔符。
- 不等于 `.` 或 `..`。
- 第一版不做大小写归一化，按仓库中的实际名称返回。

如果 `SKILL.md` 中的 `name` 不合法，返回带路径上下文的错误。这样可以避免后续 `install` 使用不安全名称拼接目标路径。

### 搜索过滤

第一版 `skill` 模块只提供发现列表，不实现模糊搜索。`search <repo>` 默认列出全部发现结果；`install --skill <name>` 的精确匹配由 `app` 完成。

如后续需要 `skilldown search <repo> <query>`，可以在 `skill` 模块新增简单的 `Filter(skills []Skill, query string) []Skill`，但当前不预留复杂查询语法。

## 实现规格

### `internal/skilldown/skill`

```go
// Skill 表示 worktree 中一个可安装的 skill。
type Skill struct {
    Name        string
    Path        string
    Description string
    Content     string
}

// Discover 从 repo worktree 中指定目录路径下发现 skill。
func Discover(root string, dirPath string) ([]Skill, error)
```

- `Skill`: 保存发现、安装和 preview 所需字段；`Path` 使用相对 worktree 的 slash 路径，例如 `skills/alpha`、`agents/alpha` 或 `a/b/skills/alpha`，`Content` 保存完整 `SKILL.md`。
- `Discover`: 读取 `<root>/<dirPath>`，识别包含 `SKILL.md` 的一级子目录；如果没有发现任何 skill，再读取 `<root>/<dirPath>/skills`；最终返回按名称排序的结果。

#### 实现状态 [done]

已实现。当前实现使用 `os.ReadDir`、`os.ReadFile`、`path/filepath` 和 `gopkg.in/yaml.v3`；front matter 边界用简单明确的 markdown 头部规则拆分，YAML 内容交给 `yaml.v3` 解析。

### `internal/skilldown/app`

```go
func Run(ctx context.Context, args []string) int
```

- `Run`: 在 search 和 install 流程中解析搜索目录路径，调用 `repo.Clone` 时传入相同 sparse path，把 `Worktree.Dir` 和目录路径传给 `skill.Discover`，并处理空结果、指定 skill 不存在和错误输出。

#### 实现状态 [deferred]

尚未实现，已由 `docs/tasks/app_layer_feature.md` 继续跟踪。app 层完成 CLI 参数筛选，不把命令行语义放进 `skill` 模块。

### `internal/skilldown/repo`

```go
type Worktree struct {
    Dir string
}

type CloneOptions struct {
    SparsePaths []string
}
```

- `Worktree`: 为 `skill.Discover` 提供可读取的本地仓库根目录。
- `CloneOptions`: search 和 install 流程应传入 app 解析出的搜索目录路径，例如默认 `[]string{"skills"}`，让 repo 只 checkout 发现所需目录。

#### 实现状态 [done]

已存在 `GitCloner` 的浅 clone、sparse checkout、partial clone fallback 和临时目录清理实现。`skill` 模块应复用该 worktree，不再直接调用 git。

### `internal/skilldown/install`

```go
func (i Installer) Install(ctx context.Context, sourceDir string, skill skill.Skill, targetRoot string, force bool) Result
```

- `Install`: 使用 `Skill.Name` 拼接目标目录，使用 `Skill.Path` 定位 worktree 内的源目录。

#### 实现状态 [out-of-scope]

安装模块由独立任务跟踪。`install` 不应重新扫描 `skills` 目录，也不应解析 `SKILL.md`。

### `internal/skilldown/output`

```go
func PrintSkills(skills []skill.Skill)
```

- `PrintSkills`: 输出 `skill.Discover` 返回的名称、路径和可选描述。
- preview 输出: 使用 `Skill.Content` 展示完整 `SKILL.md` 内容，必要时由 output 层负责截断或格式化。

#### 实现状态 [out-of-scope]

输出模块由独立任务跟踪。第一版输出保持脚本友好，例如每行一个 skill。

## 依赖关系

### 外部依赖

- `os`: 读取 `skills` 目录和判断 `SKILL.md` 是否存在。
- `path/filepath`: 处理本地 worktree 路径。
- `path`: 生成 slash 风格的相对路径。
- `sort`: 保证发现结果稳定。
- `strings`: 处理 front matter 边界和名称校验。
- `gopkg.in/yaml.v3`: 解析 `SKILL.md` YAML front matter。

### 内部调用

- `internal/skilldown/app` → `internal/skilldown/repo.GitCloner.Clone()`: 获取包含搜索目录 sparse path 的临时 worktree。
- `internal/skilldown/app` → `internal/skilldown/skill.Discover()`: 发现可搜索和可安装的 skill。
- `internal/skilldown/app` → `internal/skilldown/install.Installer.Install()`: 将已选中的 `skill.Skill` 安装到目标目录。
- `internal/skilldown/output` → `internal/skilldown/skill.Skill`: 展示搜索结果。

## 任务清单

- [x] 新增 `internal/skilldown/skill` 包和 `Skill` 类型。
- [x] 实现 `Discover(root string, dirPath string) ([]Skill, error)`，先扫描指定目录路径下的一级子目录。
- [x] `dirPath` 为空时默认使用 `skills`。
- [x] 指定目录路径下没有发现 skill 时，fallback 扫描 `<dirPath>/skills`。
- [x] 校验 `dirPath`，允许 `a/b`，去掉前导 `/`，拒绝空片段、`.` 和 `..`。
- [x] 只把包含 `SKILL.md` 的目录纳入发现结果。
- [x] 解析可选 YAML front matter 中的 `name` 和 `description`。
- [x] 读取完整 `SKILL.md` 内容并保存到 `Skill.Content`。
- [x] 校验 `Skill.Name`，拒绝空名称、路径分隔符、`.` 和 `..`。
- [x] 保证发现结果按 `Skill.Name` 稳定排序。
- [x] 增加 `test/testdata` 中多个 skill、缺失 `SKILL.md`、非法名称和损坏 front matter 的测试数据。
- [x] 增加 `internal/skilldown/skill` 单元测试。
- [ ] 在后续 app 实现中接入 search 和 install 流程，由 `docs/tasks/app_layer_feature.md` 跟踪。
- [x] 运行 `go test ./...` 验证相关逻辑。

## 验收标准

- [x] `skill.Discover(root, "skills")` 能从 `<root>/skills/<name>/SKILL.md` 发现 skill。
- [x] `skill.Discover(root, "agents")` 能从 `<root>/agents/<name>/SKILL.md` 发现 skill。
- [x] `skill.Discover(root, "/a/b")` 能从 `<root>/a/b/<name>/SKILL.md` 发现 skill。
- [x] 当 `<root>/a/b` 下没有发现 skill 时，`skill.Discover(root, "a/b")` 能从 `<root>/a/b/skills/<name>/SKILL.md` 发现 skill。
- [x] 非目录条目、缺失 `SKILL.md` 的目录和搜索目录外部内容不会出现在结果中。
- [x] 搜索目录不存在时返回空列表，不直接报错退出。
- [x] 返回的 `Skill.Path` 是相对 worktree 的 slash 路径，例如 `skills/alpha`、`agents/alpha` 或 `a/b/skills/alpha`。
- [x] 返回结果排序稳定，测试不依赖文件系统遍历顺序。
- [x] `SKILL.md` front matter 中合法的 `name` 和 `description` 能被读取。
- [x] `Skill.Content` 返回完整 `SKILL.md` 内容，可用于 preview 和后续完整详情场景。
- [x] 非法 skill 名称会返回错误，避免后续安装写入不安全路径。
- [x] `skill` 模块不调用 git、不复制文件、不处理 `--force`，边界保持在发现和描述。

## 变更历史

| 版本 | 日期 | 状态 | 说明 |
|------|------|------|------|
| 1.0 | 2026-04-27 | archived | skill 发现与描述能力已实现；app 接入由 app-layer 任务继续跟踪 |
