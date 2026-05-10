package tui

import "github.com/charmbracelet/lipgloss"

const (
	// Catppuccin Mocha 调色板基础色。
	mochaText     lipgloss.Color = "#cdd6f4"
	mochaSubtext1 lipgloss.Color = "#bac2de"
	mochaSubtext0 lipgloss.Color = "#a6adc8"
	mochaOverlay0 lipgloss.Color = "#6c7086"
	mochaSurface1 lipgloss.Color = "#45475a"
	mochaSurface0 lipgloss.Color = "#313244"
	mochaBlue     lipgloss.Color = "#89b4fa"
	mochaSky      lipgloss.Color = "#89dceb"
	mochaTeal     lipgloss.Color = "#94e2d5"
	mochaGreen    lipgloss.Color = "#a6e3a1"
	mochaYellow   lipgloss.Color = "#f9e2af"
	mochaPeach    lipgloss.Color = "#fab387"
	mochaMauve    lipgloss.Color = "#cba6f7"
	mochaSapphire lipgloss.Color = "#74c7ec"
	mochaLavender lipgloss.Color = "#b4befe"
)

// selectedLineStyle 是列表选中项的高亮样式。
var selectedLineStyle = lipgloss.NewStyle().
	Foreground(mochaText).
	Bold(true)

// paneTitleFocusedStyle 是聚焦栏目标题样式。
var paneTitleFocusedStyle = lipgloss.NewStyle().
	Foreground(mochaBlue).
	Bold(true)

// paneTitleMutedStyle 是未聚焦栏目标题样式。
var paneTitleMutedStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext0).
	Bold(true)

// treeMetaStyle 是右侧目录上下文信息的样式。
var treeMetaStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext0)

// treeMetaLabelStyle 是右侧目录上下文标签的样式。
var treeMetaLabelStyle = lipgloss.NewStyle().
	Foreground(mochaSapphire).
	Bold(true)

// treeSeparatorStyle 是右侧区域分隔线样式。
var treeSeparatorStyle = lipgloss.NewStyle().
	Foreground(mochaSurface1)

// treeHeaderStyle 是右侧内容表头样式。
var treeHeaderStyle = lipgloss.NewStyle().
	Foreground(mochaLavender).
	Bold(true)

// treeLoadingTitleStyle 是右侧加载态标题样式。
var treeLoadingTitleStyle = lipgloss.NewStyle().
	Foreground(mochaBlue).
	Bold(true)

// treeLoadingTextStyle 是右侧加载态进度文本样式。
var treeLoadingTextStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext1)

// treeLoadingHintStyle 是右侧加载态说明样式。
var treeLoadingHintStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0)

// dirEntryStyle 是目录条目的样式。
var dirEntryStyle = lipgloss.NewStyle().
	Foreground(mochaSapphire).
	Bold(true)

// fileEntryStyle 是文件条目的样式。
var fileEntryStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext1)

// treeRootEntryStyle 是右侧 tree root 条目的样式。
var treeRootEntryStyle = lipgloss.NewStyle().
	Foreground(mochaLavender).
	Bold(true)

// emptyEntryStyle 是空列表提示的样式。
var emptyEntryStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0).
	Italic(true)

// registryEmptyTitleStyle 是 registry 空状态标题样式。
var registryEmptyTitleStyle = lipgloss.NewStyle().
	Foreground(mochaMauve).
	Bold(true)

// registryEmptyHintStyle 是 registry 空状态说明样式。
var registryEmptyHintStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0)

// registryEmptyCardStyle 是 registry 空状态提示块样式。
var registryEmptyCardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(mochaSurface0).
	Padding(0, 1)

// registryEmptyKeyStyle 是 registry 空状态快捷键样式。
var registryEmptyKeyStyle = lipgloss.NewStyle().
	Foreground(mochaText).
	Background(mochaSurface0).
	Bold(true).
	Padding(0, 1)

// registryEmptyActionStyle 是 registry 空状态动作文本样式。
var registryEmptyActionStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext1)

// modalTitleStyle 是弹框标题样式。
var modalTitleStyle = lipgloss.NewStyle().
	Foreground(mochaBlue).
	Bold(true)

// modalDescStyle 是弹框说明文本样式。
var modalDescStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0)

// modalFieldLabelStyle 是弹框字段标签样式。
var modalFieldLabelStyle = lipgloss.NewStyle().
	Foreground(mochaBlue).
	Bold(true)

// modalFieldValueStyle 是弹框字段值样式。
var modalFieldValueStyle = lipgloss.NewStyle().
	Foreground(mochaText)

// modalHintStyle 是弹框底部快捷键提示样式。
var modalHintStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0)

// modalDividerStyle 是弹框内部分隔线样式。
var modalDividerStyle = lipgloss.NewStyle().
	Foreground(mochaSurface0)

// statusTextStyle 是底部状态消息的样式。
var statusTextStyle = lipgloss.NewStyle().
	Foreground(mochaTeal)

// statusHelpStyle 是底部快捷键提示的弱化样式。
var statusHelpStyle = lipgloss.NewStyle().
	Foreground(mochaOverlay0)

// statusHelpKeyStyle 是底部快捷键的强调样式。
var statusHelpKeyStyle = lipgloss.NewStyle().
	Foreground(mochaYellow).
	Bold(true)

// statusHelpDescStyle 是底部快捷键说明的样式。
var statusHelpDescStyle = lipgloss.NewStyle().
	Foreground(mochaSubtext0)

// statusLineStyle 是底部状态栏文本布局样式。
var statusLineStyle = lipgloss.NewStyle()

// helpSectionStyle 是帮助视图分组标题样式。
var helpSectionStyle = lipgloss.NewStyle().
	Foreground(mochaBlue).
	Bold(true)

// helpKeyStyle 是帮助视图按键样式。
var helpKeyStyle = lipgloss.NewStyle().
	Foreground(mochaText).
	Background(mochaSurface0).
	Bold(true).
	Padding(0, 1)

// paneStyle 构造列容器样式：统一边框与尺寸控制。
func paneStyle(width int, height int, focused bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Border(lipgloss.NormalBorder())

	if focused {
		return style.BorderForeground(mochaBlue)
	}
	return style.BorderForeground(mochaSurface0)
}

// modalStyle 构造弹框容器样式：统一边框、内边距和颜色。
func modalStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mochaBlue)
}

// paneHeaderStyle 返回标题行样式。
func paneHeaderStyle(focused bool) lipgloss.Style {
	if focused {
		return paneTitleFocusedStyle
	}
	return paneTitleMutedStyle
}

// paneDividerStyle 返回栏目标题下方分隔线样式。
func paneDividerStyle() lipgloss.Style {
	return treeSeparatorStyle
}

// registryNameStyle 返回 registry 名称列样式。
func registryNameStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(mochaMauve)
	if selected {
		return style.Bold(true)
	}
	return style
}

// registryUpdatedAtStyle 返回 registry 更新时间列样式。
func registryUpdatedAtStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(mochaYellow)
	if selected {
		return style.Bold(true)
	}
	return style
}

// treeMetaValueStyle 返回 worktree 上方元信息 value 的分组样式。
func treeMetaValueStyle(key string) lipgloss.Style {
	switch key {
	case "registry":
		return lipgloss.NewStyle().Foreground(mochaMauve)
	case "url":
		return lipgloss.NewStyle().Foreground(mochaSky)
	case "branch":
		return lipgloss.NewStyle().Foreground(mochaGreen)
	case "path":
		return lipgloss.NewStyle().Foreground(mochaPeach)
	case "search":
		return lipgloss.NewStyle().Foreground(mochaYellow)
	default:
		return treeMetaStyle
	}
}

// modalRegistryTitleStyle 返回新增或编辑 registry 弹框标题样式。
func modalRegistryTitleStyle(editing bool) lipgloss.Style {
	if editing {
		return lipgloss.NewStyle().Foreground(mochaPeach).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(mochaGreen).Bold(true)
}

// modalRegistryFieldValueStyle 返回新增或编辑 registry 字段值样式。
func modalRegistryFieldValueStyle(field string, active bool) lipgloss.Style {
	if active {
		return lipgloss.NewStyle().Foreground(mochaYellow).Bold(true)
	}
	switch field {
	case "Name":
		return lipgloss.NewStyle().Foreground(mochaMauve)
	case "URL":
		return lipgloss.NewStyle().Foreground(mochaSky)
	case "Branch":
		return lipgloss.NewStyle().Foreground(mochaGreen)
	default:
		return modalFieldValueStyle
	}
}

// modalBranchChoiceStyle 返回新增或编辑 registry 分支候选项样式。
func modalBranchChoiceStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(mochaGreen)
	if selected {
		return style.Background(mochaSurface0).Bold(true)
	}
	return style
}

// emptyCardContainerStyle 返回空状态卡片的容器样式。
func emptyCardContainerStyle(width int) lipgloss.Style {
	return registryEmptyCardStyle.
		Width(width)
}

// statusLineContainerStyle 返回底部状态行容器样式。
func statusLineContainerStyle(width int) lipgloss.Style {
	return statusLineStyle.Width(width)
}

// modalBodyStyle 用于弹框内部内容区域的统一样式。
func modalBodyStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

// modalTableHeaderStyle 返回 modal 内部表格头部样式。
func modalTableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(mochaSapphire).
		Bold(true)
}

// modalBranchSelectedLineStyle 返回分支列表选中项的定宽高亮样式。
func modalBranchSelectedLineStyle(width int) lipgloss.Style {
	return modalBranchChoiceStyle(true).Width(width)
}
