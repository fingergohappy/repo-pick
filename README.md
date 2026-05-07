# repo-pick

[English](readme-en.md) | 简体中文

`repo-pick` 是一个 TUI-only 的远程 Git 仓库文件/目录下载工具。它把远程仓库 shallow clone 到本地 cache，让你在终端里浏览 work tree，并把选中的文件、目录或整个仓库下载到本地目录。

## Demo

<!-- TODO: 添加演示视频或 GIF -->

## Quick Start

安装：

```bash
brew tap fingergohappy/tap
brew install repo-pick
```

启动：

```bash
repo-pick
```

第一次使用：

```text
a       添加 registry
l       打开当前 registry
j/k     在 work tree 中移动
i       下载当前条目到启动目录
```

## Core Workflow

1. 在左侧 `Registry` 面板按 `a` 添加一个远程 Git 仓库。
2. 按 `l` 打开当前 registry。首次打开会 shallow clone 到本地 cache，后续优先复用 cache。
3. 在右侧 `Repository Tree` 面板浏览目录、搜索路径或展开目录。
4. 选中文件、目录或根目录 `/` 后按 `i` 下载到启动目录，或按 `I` 输入目标目录后下载。
5. 选中文件后按 `e` 用 `EDITOR` 打开 cache 中的文件。

`Repository Tree` 中的 `/` 是当前 root。光标停在仓库根 `/` 上按 `i` 或 `I` 会下载整个仓库，目标目录名使用 registry 名称。

按 `e` 打开文件时会执行 `EDITOR` 环境变量中的命令，例如 `EDITOR=vim` 或 `EDITOR="code -w"`。未设置 `EDITOR` 时只会在状态栏提示，不会启动外部程序。

删除 registry、覆盖已存在目标路径等高风险动作会先提示确认。底部状态栏会根据当前焦点区域展示可用快捷键。

## Keybindings

全局：

```text
ctrl-w h 切换到 registry
ctrl-w l 切换到 repository tree；未打开仓库时会打开当前 registry
/       搜索当前仓库路径
Esc     关闭搜索、确认或错误
?       显示/关闭帮助
q       退出
```

Registry：

```text
j/k     移动
l       打开当前仓库
a       新增 registry；弹框中输入 name/url，并可选择远端分支
e       编辑当前 registry；弹框中修改 name/url/branch
r       重载 registry 列表；只重新读取配置，不更新仓库内容
d       删除 registry，并同步删除 cache
u       更新当前仓库 cache；删除旧 cache 并重新下载仓库内容
```

删除 registry 会弹出确认框；按 `y` 确认，按 `n` 或 `Esc` 取消。

Repository Tree：

```text
h       返回上级 root
j/k     移动
l       展开或收起选中目录
o       进入目录，并把该目录作为新的 root；文件会定位到所在目录
e       用 EDITOR 打开当前文件
i       下载当前条目到启动目录
I       输入目标目录后下载当前条目
```

## Configuration

用户配置文件：

```text
~/.config/repo-pick/config.yaml
```

示例：

```yaml
repositories:
  - name: official
    url: https://github.com/org/tools
  - name: personal
    url: git@github.com:finger/my-tools.git
    branch: main
```

字段说明：

- `repositories[].name`: 本地 registry 名称，必须唯一。
- `repositories[].url`: Git 仓库地址；允许重复。
- `repositories[].branch`: 可选 Git 分支；同一 URL 下分支不能重复，为空或不配置时使用远端默认分支。
- `repositories[].last_updated_at`: 本地 cache 最近一次成功生成或刷新的时间；由程序自动维护。

## Cache Behavior

仓库 cache 路径：

```text
~/.cache/repo-pick/repos/<url-or-url+branch-hash>/
```

`Ensure` 语义：

- 有 cache：直接读取本地工作区，不联网。
- 无 cache：执行 `git clone --depth 1 --single-branch`；配置了 `branch` 时额外传入 `--branch <branch>`。
- 首次成功生成 cache 后更新配置中的 `last_updated_at`。

`Update` 语义：

- 删除旧 cache。
- 重新执行 shallow clone。
- 成功后更新配置中的 `last_updated_at`。
- 如果重新下载失败，不恢复旧 cache；该仓库本次不能浏览是正常结果。

## Development

```bash
go mod download
go test ./...
```

主要目录：

```text
cmd/repo-pick/             # 程序入口，直接启动 TUI
internal/repopick/app/     # 应用用例编排
internal/repopick/cache/   # Git 仓库 cache 生命周期
internal/repopick/config/  # 用户配置读写
internal/repopick/install/ # 文件和目录复制
internal/repopick/registry/# 仓库书签管理
internal/repopick/tree/    # cache 工作区目录树读取与搜索
internal/repopick/tui/     # Bubble Tea 终端界面
```
