# TUI Lip Gloss 最优重构方案

## 结论

不考虑兼容现有 view 写法时，最优方案是把 view 收敛成一套“块级 renderer”，让 Lip Gloss 负责布局、边框、对齐、测量和固定列渲染，业务层只提供状态和数据。

目标不是把每一行都改成 Lip Gloss，而是明确分工:

- 屏幕布局、pane、modal、空状态、loading 块: 使用 `lipgloss.Style`、`JoinHorizontal`、`JoinVertical`、`Place`。
- 固定列数据: 使用 `lipgloss/table`。
- 交互树、滚动窗口、选中态、省略号: 保留自定义 renderer，不使用 `lipgloss/tree` 或 `lipgloss/list` 强行替代。

最终 view 应该从“拼 `[]string`”变成“渲染组件块”。

## 参考能力

当前依赖是 `github.com/charmbracelet/lipgloss v1.1.0`。

可用能力:

- `lipgloss.Style`: `Width`、`Height`、`MaxWidth`、`Align`、`AlignVertical`、`Padding`、`Border`、`Render`
- 布局: `JoinHorizontal`、`JoinVertical`、`Place`、`PlaceHorizontal`、`PlaceVertical`
- 测量: `Width`、`Height`、`Size`
- 固定列: `github.com/charmbracelet/lipgloss/table`
- 静态列表/树: `github.com/charmbracelet/lipgloss/list`、`github.com/charmbracelet/lipgloss/tree`

官方文档:

- https://github.com/charmbracelet/lipgloss
- https://pkg.go.dev/github.com/charmbracelet/lipgloss
- https://pkg.go.dev/github.com/charmbracelet/lipgloss/table
- https://pkg.go.dev/github.com/charmbracelet/lipgloss/tree
- https://pkg.go.dev/github.com/charmbracelet/lipgloss/list

## 最优架构

建议把 view 层整理成这些 renderer:

```text
view.go              // View 入口，只组装 screen
view_styles.go       // 全局样式 token
view_layout.go       // screen、pane、modal、status 的布局 primitive
view_registry.go     // registry pane renderer
view_tree.go         // repository tree pane renderer
view_modal.go        // add/edit/delete/help modal renderer
view_table.go        // table helper: registry/search/meta/help/branch
view_text.go         // 单行省略、可见宽度、窗口切片
```

核心原则:

- `View()` 只做 screen 级组装，不直接拼业务行。
- pane renderer 返回完整 string block，不把所有组件都压成 `[]string`。
- 列式内容统一走 `lipgloss/table`，不要继续用 `fmt.Sprintf("%-8s")` 做列。
- 所有 frame 样式统一由 style builder 创建，不在业务 renderer 内临时拼 style。
- 交互状态仍由 Bubble Tea model 管，Lip Gloss 只负责渲染。

## 组件最优选择

| 组件 | 最优方案 | 使用 Lip Gloss 能力 | 说明 |
| --- | --- | --- | --- |
| 主屏幕布局 | 保留双 pane + status，但抽成 `renderScreen` | `JoinHorizontal`、`JoinVertical` | `View()` 不再直接知道 pane 细节 |
| Pane 外框 | 统一 `paneStyle(focused)` | `Style.Width`、`Height`、`Padding`、`Border` | 焦点色、边框、padding 集中管理 |
| Pane 标题 | 作为 pane header block | `PlaceHorizontal`、`Align` | 标题居中由 Lip Gloss 做 |
| Pane 分隔线 | 用统一 divider primitive | `Style.Width`、`Render` | 可以仍生成字符，但不要散落在业务函数 |
| Registry 列表 | 改成无边框 table | `lipgloss/table` | 列为 cursor、name、updatedAt |
| Registry 空状态 | 完整 card block | `Style.Border`、`Width`、`Align`、`PlaceHorizontal` | 不再 split 后逐行居中 |
| Tree context | 改成无边框 key/value table | `lipgloss/table` | registry/url/branch/path/search 天然是两列表 |
| Tree 搜索结果 | 改成 table | `lipgloss/table` | type/size/path 是标准固定列 |
| Tree 普通行 | 保留自定义 tree row renderer | `Style.Render`、`Width` | 不用 `lipgloss/tree`，因为交互树需要扁平窗口和选中态 |
| Tree loading | 改成 centered block | `JoinVertical`、`Place` | 不再手动 append 空行 |
| Registry preview | 改成 centered block/card | `JoinVertical`、`Place` | 和 loading 共用 block primitive |
| Add/Edit modal | modal frame + form table | `Style.Border`、`Padding`、`table` | 字段 name/url/branch 用 table，不手写 label 宽度 |
| Delete modal | modal frame + key/value table | `Style.Border`、`table` | 和 add/edit 共用 modal frame |
| Branch selector | table 化，不用 `lipgloss/list` | `lipgloss/table` | cursor + branch name，滚动窗口仍自定义 |
| Help modal | table 化 | `lipgloss/table` | key/desc 固定两列 |
| Status line | left/right block renderer | `JoinHorizontal`、`PlaceHorizontal`、`Width` | 保留降级逻辑，但布局由 helper 管 |
| 单行截断 | 保留自定义 `truncateVisible` | `lipgloss.Width` | Lip Gloss 不替代省略号策略 |

## 为什么不用 `lipgloss/tree`

即使不考虑兼容性，`lipgloss/tree` 也不是这里的最优解。

当前 repository tree 是交互控件，不是静态树输出:

- 需要按光标位置显示窗口。
- 需要展开/收起目录。
- 需要 root 行特殊语义。
- 需要选中行动画 cursor。
- 需要文件/目录不同样式。
- 需要下载、打开、返回上级等交互动作映射到扁平行。

`lipgloss/tree` 擅长静态树渲染。为了塞进当前交互模型，最终会绕过它的大部分默认行为，复杂度反而更高。

最优方案是保留自定义 tree row renderer，但把样式、截断、缩进符号和选中态封装清楚。

## 为什么不用 `lipgloss/list`

`lipgloss/list` 适合静态列表文本，不适合这里的 registry、branch selector 和 tree 列表。

这些列表都有 Bubble Tea 状态:

- 当前选中项。
- 滚动窗口。
- 搜索过滤。
- loading/error/empty 状态。
- 行级样式。

最优方案是:

- 简单列式列表用 `lipgloss/table`。
- 交互窗口和选中态由 view helper 控制。
- 不引入 `lipgloss/list`。

## 最优重构顺序

### 1. 建立 layout primitives

先建立统一布局函数:

```go
func renderScreen(left string, right string, status string, width int, height int) string
func renderPane(title string, body string, opts paneOptions) string
func renderModal(body string, width int) string
func placeOverlay(base string, overlay string, width int, height int) string
func renderCenteredBlock(width int, height int, lines ...string) string
```

这是最重要的一步。后续所有组件都应该返回 string block，而不是散落地返回 `[]string`。

### 2. 把 frame style 全部集中

建立 style builder:

```go
func paneStyle(width int, height int, focused bool) lipgloss.Style
func modalStyle(width int) lipgloss.Style
func emptyCardStyle(width int) lipgloss.Style
func statusStyle(width int) lipgloss.Style
```

`view_styles.go` 保留颜色 token，builder 负责组合尺寸和状态。

### 3. 引入 table helper

建立无边框 table helper:

```go
func renderKeyValueTable(rows [][2]string, width int) string
func renderSelectableTable(rows [][]string, selected int, width int, columns []columnSpec) string
func renderHelpTable(bindings []helpBinding, width int) string
```

优先替换:

1. tree context。
2. search result。
3. help modal。
4. branch selector。
5. registry list。

这些都是固定列数据，用 table 是最优。

### 4. 重写 modal body

Add/Edit/Delete/Help modal 应该共用:

- modal frame。
- divider。
- key/value table。
- footer hint。

modal 内部不再直接构造边框 style。

### 5. 收敛 tree renderer

普通 tree 不换 `lipgloss/tree`，但要拆成明确层次:

```go
func renderTreeRows(rows []treeRow, selected int, width int) string
func renderTreeRow(row treeRow, selected bool, width int) string
func treeRowPrefix(row treeRow) string
func treeRowStyle(row treeRow, selected bool) lipgloss.Style
```

tree renderer 的目标是保持交互能力清晰，而不是追求使用更多 Lip Gloss 子包。

### 6. 收敛 status line

status line 最优形态是明确左右区域:

```go
func renderStatus(status string, help string, width int) string
```

内部仍保留降级策略:

1. 空间足够: status + full help。
2. 空间不足: status + compact help。
3. 再不足: 只展示截断 status。

区别是布局和样式由 helper 管，不在业务判断里散落。

## 最终文件职责

```text
view.go
  View 入口，只调用 renderScreen/renderOverlay

view_layout.go
  screen/pane/modal/status/centered block primitives

view_styles.go
  color token 和 style builder

view_table.go
  key/value、selectable、help、search table helpers

view_registry.go
  registry pane 数据到 table rows 的转换

view_tree.go
  tree pane 数据到 context/search/tree/loading block 的转换

view_modal.go
  modal body 组装

view_text.go
  truncateVisible、visibleWindow、firstLine 等终端文本工具
```

## 目标效果

重构完成后，view 层应该满足:

- 不再在业务 renderer 里重复创建 border/padding style。
- 不再用 `fmt.Sprintf("%-8s")` 做 UI 列布局。
- 不再用手动空行 padding 做垂直居中。
- 不再让 `View()` 直接拼具体业务行。
- 只有真正需要逐字符控制的地方保留手写逻辑: tree row、visible window、ellipsis。

## 验收标准

只看最终最优形态，不为旧结构保留兼容层:

- `go test ./internal/repopick/tui` 通过。
- `go test ./...` 通过。
- `git diff --check` 通过。
- view 层没有 legacy wrapper、alias、双轨实现。
- 表格类 UI 不再手写固定列空格。
- modal/pane/status 的 frame style 只有一个来源。
- tree 仍保持交互语义清晰，不为了使用 `lipgloss/tree` 牺牲可维护性。

## 最终结论

最优方案是:

1. 用 Lip Gloss 建立统一块级布局系统。
2. 用 `lipgloss/table` 接管所有固定列 UI。
3. 不使用 `lipgloss/tree` 重写交互 tree。
4. 不使用 `lipgloss/list` 重写交互列表。
5. 保留自定义文本截断、滚动窗口和 tree row 渲染。

这比“逐个 helper 小修”更彻底，也比“所有东西都换成 Lip Gloss 子包”更可维护。
