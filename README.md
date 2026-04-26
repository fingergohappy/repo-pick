# skill-down

`skill-down` 是一个用于发现和下载远程仓库中 skills 的 Go 命令行工具。

项目目标是把“给一个仓库、搜索 skills 目录、下载指定 skill”收敛成稳定的 CLI/TUI 工作流，方便在不同项目中复用远程 skill。

## 当前状态

项目处于初始化阶段，已经完成：

- Go module 初始化
- CLI/TUI 依赖安装
- 按 Go project layout 初始化最小目录结构
- 第一版功能草案文档

业务代码尚未实现。

## 第一版目标

第一版聚焦最小可用能力：

- 从远程仓库搜索 skill
- 注册常用 skill 仓库，未显式传入仓库时默认从注册仓库搜索
- 识别 `skills/<name>` 形式的目录
- 支持调用方指定自定义 skill 搜索路径，默认路径为 `skills`
- 读取每个 skill 的完整 `SKILL.md` 内容，供搜索结果详情或 preview 使用
- 下载指定 skill 到目标仓库
- 支持下载全部发现到的 skill
- 目标目录已存在时默认拒绝覆盖，通过显式参数确认覆盖

暂不包含：

- 更新已安装 skill
- 删除已安装 skill
- marketplace 能力
- 版本锁定
- 私有仓库认证的专门适配

## 计划中的命令

```bash
# 注册常用 skill 仓库
skilldown registry add <repo> --name <name>

# 查看已注册仓库
skilldown registry list

# 移除已注册仓库
skilldown registry remove <name>

# 搜索远程仓库中的 skills
skilldown search <repo>

# 未传 repo 时，从已注册仓库搜索
skilldown search

# 安装指定 skill
skilldown install <repo> --skill <name> --to <dir>

# 未传 repo 时，从已注册仓库中查找指定 skill
skilldown install --skill <name> --to <dir>

# 安装全部发现到的 skills
skilldown install <repo> --to <dir>

# 覆盖已存在的目标目录
skilldown install <repo> --skill <name> --to <dir> --force
```

## 配置文件

配置文件保存到用户级配置目录，不写入当前项目仓库：

```text
os.UserConfigDir()/skill-down/config.yaml
```

常见路径：

```text
macOS:   ~/Library/Application Support/skill-down/config.yaml
Linux:   ~/.config/skill-down/config.yaml
Windows: %AppData%\skill-down\config.yaml
```

配置结构：

```yaml
repositories:
  - name: official
    url: https://github.com/org/skills
    skillDir: skills
  - name: personal
    url: git@github.com:finger/my-skills.git
    skillDir: skills

repo:
  downloadDir: ""
```

仓库内提供了同结构的样例文件：

```text
configs/config.example.yaml
```

字段说明：

- `repositories`: 已注册的 skill 仓库列表。未显式传入 repo 时，`search`、`install` 和 `browse` 默认从这里读取仓库。
- `repositories[].name`: 本地 registry 名称，需要唯一。
- `repositories[].url`: Git 仓库地址。
- `repositories[].skillDir`: 仓库中承载 skill 子目录的相对路径，为空时默认 `skills`。如果写成 `/a/b`，会按仓库内相对路径 `a/b` 处理。
- `repo.downloadDir`: 远程仓库 clone 的父目录。为空时使用当前默认逻辑，也就是系统临时目录，并在命令结束后清理。

如果设置了 `repo.downloadDir`，repo 模块会在该目录下创建本次命令的临时 worktree；命令结束后仍默认清理。该字段只改变下载位置，不表示启用长期缓存。

安装目录不放在配置文件里。未传 `--to` 时，默认安装到命令启动目录下的 `.codex/skills`。

后续会增加 TUI 入口，用于交互式搜索、选择和安装：

```bash
skilldown browse <repo>
```

## 技术栈

- Go
- Cobra: CLI 命令与参数
- Bubble Tea: TUI 状态模型
- Bubbles: TUI 组件
- Lip Gloss: TUI 样式
- Huh: 简单交互表单
- Viper: 用户配置文件读写
- yaml.v3: 配置或元数据解析

## 项目结构

项目采用 `golang-standards/project-layout` 的最小子集，只保留第一版需要的目录。

```text
cmd/skilldown/              # CLI 依赖组装入口
internal/skilldown/app/     # 应用用例层：registry、search 和 install 流程编排
internal/skilldown/cli/     # Cobra 命令行输入适配层
internal/skilldown/tui/     # 交互式 TUI 入口接口
internal/skilldown/repo/    # 远程仓库临时 clone 和清理
internal/skilldown/config/  # 用户配置文件读取和写入
internal/skilldown/registry/# 已注册 skill 仓库管理
internal/skilldown/skill/   # 本地 worktree 中的 skill 发现、元数据解析和内容读取
internal/skilldown/install/ # skill 安装与覆盖策略
internal/skilldown/output/  # 终端 CLI 文本输出
configs/                    # 配置样例或默认配置
docs/                       # 设计文档和任务文档
scripts/                    # 构建、测试、发布脚本
test/testdata/              # 测试数据
```

## 模块边界

`repo`、`registry`、`skill`、`install` 几个模块在流程上相邻，但职责不同：

- `repo`: 负责把远程 Git 仓库临时 clone 到本地 worktree，并在命令结束后清理；不理解 `SKILL.md`、搜索规则或安装规则。
- `config`: 负责读取和写入用户级 `config.yaml`，包含 `repositories` 列表和 repo 下载目录设置；这是唯一知道 Viper、YAML 和配置文件路径的模块。
- `registry`: 负责管理内存中的 `repositories` 列表规则，例如 add/list/remove/resolve、名称唯一校验和默认 `skillDir`；它通过 `config.Store` 做持久化，不直接读取 YAML，不知道 Viper 或配置文件路径。
- `skill`: 接收本地 worktree 路径和搜索目录路径，负责识别和描述 skill。先处理 `<dirPath>/<name>` 目录发现；如果没有发现任何 skill，再尝试 `<dirPath>/skills/<name>`；同时完成 `SKILL.md` 判断、名称和描述解析，并返回完整 `SKILL.md` 内容；不接收 repo URL，不调用 git，不负责写入本地目标目录。`dirPath` 为空时默认使用 `skills`，允许 `a/b` 这种仓库内相对路径，前导 `/` 会被去掉。
- `install`: 负责本地安装策略，只处理目标路径、已存在目录、`--force` 覆盖策略和安装结果；使用 `skill` 模块返回的 `Name`、`Path` 定位源目录。
- `cli`: 负责 Cobra 参数解析，把命令和 flags 转换成 app 请求，并渲染普通终端输出。
- `output`: 负责普通终端 CLI 可复用的文本输出辅助，例如进度消息；不承载 Bubble Tea/TUI 交互。

调用关系保持单向：

```text
app -> registry
app -> repo
app -> skill
app -> install
registry -> config
```

也就是说：

```text
repo     = 怎么把远程仓库变成本地可读 worktree
config   = 怎么读写用户级 config.yaml
registry = 怎么管理已注册仓库列表，以及默认从哪些仓库搜索
skill    = 怎么在本地 worktree 的指定目录路径下识别 skill，并读取完整 SKILL.md
install  = 怎么把一个已选中的 skill 目录复制到目标位置
```

registry 的持久化流程：

```text
registry.Add/Remove/List/Resolve
    cfg = config.Store.Load()
    修改或读取 cfg.Repositories
    config.Store.Save(cfg)
```

因此 registry 不处理 YAML 文件格式；YAML 读写只在 config 模块里完成。

典型流程：

```text
dirPath = app 解析搜索目录路径，默认 "skills"
repoURLs = app 使用显式 repo；如果未传 repo，则读取 registry 中的仓库列表

for each repoURL:
    worktree = repo.Clone(repoURL, SparsePaths: [dirPath])
    skills = skill.Discover(worktree.Dir, dirPath)

search:
    cli renders grouped app.SearchResult

install:
    selected = app 按 --skill 筛选，未指定时选择全部
    app 调用 install.CopyDir(sourceDir, targetDir, force)
```

## 开发

安装依赖：

```bash
go mod download
```

运行测试：

```bash
go test ./...
```

当前核心模块已具备单元测试，后续功能开发应继续通过 `go test ./...` 验证。

## 文档

第一版功能草案见：

- `docs/tasks/skill_down_feature.md`
- `docs/tasks/archived/repo_feature.md`
- `docs/tasks/archived/skill_feature.md`
- `docs/tasks/install_feature.md`
