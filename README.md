# repo-pick

`repo-pick` 是一个 TUI-only 的远程 Git 仓库文件/目录下载工具。

启动后进入终端界面：左侧管理仓库书签，右侧浏览仓库目录树。仓库会 shallow clone 到本地 cache，用户选中文件或目录后下载到启动目录或指定目录。

## 当前能力

- 启动 `repo-pick` 直接进入 TUI。
- registry 保存仓库 `name`、`url` 和可选 `branch`；同一 `url` 可存在多个分支。
- cache 保存每个仓库的完整浅克隆工作区。
- 支持目录树浏览、路径搜索、手动更新 cache。
- 打开或更新仓库 cache 时展示 Git clone 进度。
- 支持下载文件或目录，并展示本地复制进度。
- 目标同名时提示覆盖或取消。

## 安装

Homebrew 安装：

```bash
brew tap fingergohappy/tap
brew install repo-pick
```

发布新版本时推送 `vX.Y.Z` tag；GitHub Actions 会编译 release 二进制，并更新 `fingergohappy/homebrew-tap` 中的 formula。

## 配置

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

## Cache

仓库 cache 路径：

```text
~/.cache/repo-pick/repos/<url-or-url+branch-hash>/
```

`Ensure` 语义：

- 有 cache：直接读取本地工作区，不联网。
- 无 cache：执行 `git clone --depth 1 --single-branch`；配置了 `branch` 时额外传入 `--branch <branch>`。

`Update` 语义：

- 删除旧 cache。
- 重新执行 shallow clone。
- 如果重新下载失败，不恢复旧 cache；该仓库本次不能浏览是正常结果。

## 快捷键

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

Repository Tree：

```text
h       返回上级 root
j/k     移动
l       展开或收起选中目录
o       进入目录，并把该目录作为新的 root；文件会定位到所在目录
i       下载当前条目到启动目录
I       输入目标目录后下载当前条目
```

## 项目结构

```text
cmd/repo-pick/             # 程序入口，直接启动 TUI
internal/repopick/app/     # 应用用例编排
internal/repopick/cache/   # Git 仓库 cache 生命周期
internal/repopick/config/  # 用户配置读写
internal/repopick/install/ # 文件和目录复制
internal/repopick/registry/# 仓库书签管理
internal/repopick/tree/    # cache 工作区目录树读取与搜索
internal/repopick/tui/     # Bubble Tea 终端界面
configs/                   # 配置样例
docs/                      # 设计文档和任务文档
test/testdata/             # 测试数据
```

## 开发

```bash
go mod download
go test ./...
```
