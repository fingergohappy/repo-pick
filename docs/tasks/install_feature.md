---
title: install
type: feature
date: 2026-04-27
status: accepted
version: "1.0"
summary: 设计 `internal/skilldown/install` 模块的目录复制能力。该模块只接收源目录、目标目录和覆盖开关，负责把源目录完整复制到目标目录。
scope:
  - internal/skilldown/install
  - test/testdata
---

# install

## 目标

`install` 模块只做一件事：把一个本地源目录复制到一个指定目标目录。

调用方负责决定复制哪个 skill、源目录在哪里、目标目录叫什么、是否安装多个 skill。`install` 不关心 repo、CLI、skill 名称、`SKILL.md` 内容或搜索目录路径。

## 边界

`install` 负责：

- [x] 校验源目录存在且是目录。
- [x] 校验目标目录参数有效。
- [x] 目标目录不存在时，创建必要的父目录并复制。
- [x] 目标目录已存在且 `force=false` 时返回失败，不覆盖已有内容。
- [x] 目标目录已存在且 `force=true` 时，只删除目标目录本身，然后重新复制。
- [x] 递归复制源目录下的普通文件和子目录。

`install` 不负责：

- [x] 不 clone 远程仓库。
- [x] 不扫描 skill 目录。
- [x] 不解析 `SKILL.md`。
- [x] 不筛选 skill。
- [x] 不计算 `<to>/<skillName>`。
- [x] 不处理多 skill 安装流程。
- [x] 不输出 CLI 文本。

## 接口设计

```go
type ResultStatus string

const (
    ResultInstalled ResultStatus = "installed"
    ResultFailed    ResultStatus = "failed"
)

type Result struct {
    SourceDir string
    TargetDir string
    Status    ResultStatus
    Err       error
}

type Installer struct{}

func (i Installer) CopyDir(ctx context.Context, sourceDir string, targetDir string, force bool) Result
```

`CopyDir` 的入参语义：

- `sourceDir`: 要复制的本地目录。
- `targetDir`: 最终目标目录，不是目标根目录。
- `force`: 是否允许覆盖已存在的 `targetDir`。

## 复制规则

复制时保留目录结构和普通文件内容。

第一版不处理 Git 元数据、文件所有者、扩展属性或符号链接特殊语义。遇到复制错误时返回失败；不做复杂事务回滚。

`force=true` 时只能删除 `targetDir`，不能删除它的父目录，也不能影响其它目录。

## 验收标准

- [x] `CopyDir(ctx, source, target, false)` 能把整个 `source` 目录复制到 `target`。
- [x] `target` 不存在但父目录可创建时，复制成功。
- [x] `target` 已存在且 `force=false` 时，返回失败并保留原目录内容。
- [x] `target` 已存在且 `force=true` 时，只覆盖 `target`。
- [x] `sourceDir` 不存在或不是目录时，返回失败。
- [x] `install` 包不导入 `repo`、`skill`、`app` 或 `output` 包。
- [x] `go test ./...` 通过，或明确说明失败原因。

## 验收记录

- 2026-04-27: `go test ./internal/skilldown/install` 通过。
- 2026-04-27: `go test ./...` 通过。

## 变更历史

| 版本 | 日期 | 状态 | 说明 |
|------|------|------|------|
| 1.0 | 2026-04-27 | accepted | 按现有 install 实现完成验收 |
