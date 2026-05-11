package tui

import "github.com/charmbracelet/lipgloss"

const (
	// Tokyo Night Moon 调色板基础色。
	tokyoMoonText          lipgloss.Color = "#c8d3f5"
	tokyoMoonFgDark        lipgloss.Color = "#828bb8"
	tokyoMoonDark5         lipgloss.Color = "#737aa2"
	tokyoMoonComment       lipgloss.Color = "#636da6"
	tokyoMoonTerminalBlack lipgloss.Color = "#444a73"
	tokyoMoonBgHighlight   lipgloss.Color = "#2f334d"
	tokyoMoonBlue          lipgloss.Color = "#82aaff"
	tokyoMoonBlue1         lipgloss.Color = "#65bcff"
	tokyoMoonCyan          lipgloss.Color = "#86e1fc"
	tokyoMoonTeal          lipgloss.Color = "#4fd6be"
	tokyoMoonGreen         lipgloss.Color = "#c3e88d"
	tokyoMoonYellow        lipgloss.Color = "#ffc777"
	tokyoMoonOrange        lipgloss.Color = "#ff966c"
	tokyoMoonMagenta       lipgloss.Color = "#c099ff"
	tokyoMoonPurple        lipgloss.Color = "#fca7ea"
)

// selectedLineStyle 是列表选中项的高亮样式。
var selectedLineStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonText).
	Bold(true)

// paneTitleFocusedStyle 是聚焦栏目标题样式。
var paneTitleFocusedStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue).
	Bold(true)

// paneTitleMutedStyle 是未聚焦栏目标题样式。
var paneTitleMutedStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonDark5).
	Bold(true)

// treeMetaStyle 是右侧目录上下文信息的样式。
var treeMetaStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonDark5)

// treeMetaLabelStyle 是右侧目录上下文标签的样式。
var treeMetaLabelStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue1).
	Bold(true)

// treeSeparatorStyle 是右侧区域分隔线样式。
var treeSeparatorStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonTerminalBlack)

// treeHeaderStyle 是右侧内容表头样式。
var treeHeaderStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonPurple).
	Bold(true)

// treeLoadingTitleStyle 是右侧加载态标题样式。
var treeLoadingTitleStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue).
	Bold(true)

// treeLoadingTextStyle 是右侧加载态进度文本样式。
var treeLoadingTextStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonFgDark)

// treeLoadingHintStyle 是右侧加载态说明样式。
var treeLoadingHintStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment)

// dirEntryStyle 是目录条目的样式。
var dirEntryStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue1).
	Bold(true)

// fileEntryStyle 是文件条目的样式。
var fileEntryStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonFgDark)

// treeRootEntryStyle 是右侧 tree root 条目的样式。
var treeRootEntryStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonPurple).
	Bold(true)

// emptyEntryStyle 是空列表提示的样式。
var emptyEntryStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment).
	Italic(true)

// registryEmptyTitleStyle 是 registry 空状态标题样式。
var registryEmptyTitleStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonMagenta).
	Bold(true)

// registryEmptyHintStyle 是 registry 空状态说明样式。
var registryEmptyHintStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment)

// registryEmptyCardStyle 是 registry 空状态提示块样式。
var registryEmptyCardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(tokyoMoonBgHighlight).
	Padding(0, 1)

// registryEmptyKeyStyle 是 registry 空状态快捷键样式。
var registryEmptyKeyStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonText).
	Background(tokyoMoonBgHighlight).
	Bold(true).
	Padding(0, 1)

// registryEmptyActionStyle 是 registry 空状态动作文本样式。
var registryEmptyActionStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonFgDark)

// modalTitleStyle 是弹框标题样式。
var modalTitleStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue).
	Bold(true)

// modalDescStyle 是弹框说明文本样式。
var modalDescStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment)

// modalFieldLabelStyle 是弹框字段标签样式。
var modalFieldLabelStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue).
	Bold(true)

// modalFieldValueStyle 是弹框字段值样式。
var modalFieldValueStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonText)

// modalHintStyle 是弹框底部快捷键提示样式。
var modalHintStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment)

// modalDividerStyle 是弹框内部分隔线样式。
var modalDividerStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBgHighlight)

// statusTextStyle 是底部状态消息的样式。
var statusTextStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonTeal)

// statusHelpStyle 是底部快捷键提示的弱化样式。
var statusHelpStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonComment)

// statusHelpKeyStyle 是底部快捷键的强调样式。
var statusHelpKeyStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonYellow).
	Bold(true)

// statusHelpDescStyle 是底部快捷键说明的样式。
var statusHelpDescStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonDark5)

// statusLineStyle 是底部状态栏文本布局样式。
var statusLineStyle = lipgloss.NewStyle()

// helpSectionStyle 是帮助视图分组标题样式。
var helpSectionStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonBlue).
	Bold(true)

// helpKeyStyle 是帮助视图按键样式。
var helpKeyStyle = lipgloss.NewStyle().
	Foreground(tokyoMoonText).
	Background(tokyoMoonBgHighlight).
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
		return style.BorderForeground(tokyoMoonBlue)
	}
	return style.BorderForeground(tokyoMoonBgHighlight)
}

// modalStyle 构造弹框容器样式：统一边框、内边距和颜色。
func modalStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tokyoMoonBlue)
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
	style := lipgloss.NewStyle().Foreground(tokyoMoonMagenta)
	if selected {
		return style.Bold(true)
	}
	return style
}

// registryUpdatedAtStyle 返回 registry 更新时间列样式。
func registryUpdatedAtStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(tokyoMoonYellow)
	if selected {
		return style.Bold(true)
	}
	return style
}

// treeMetaValueStyle 返回 worktree 上方元信息 value 的分组样式。
func treeMetaValueStyle(key string) lipgloss.Style {
	switch key {
	case "registry":
		return lipgloss.NewStyle().Foreground(tokyoMoonMagenta)
	case "url":
		return lipgloss.NewStyle().Foreground(tokyoMoonCyan)
	case "branch":
		return lipgloss.NewStyle().Foreground(tokyoMoonGreen)
	case "path":
		return lipgloss.NewStyle().Foreground(tokyoMoonOrange)
	case "search":
		return lipgloss.NewStyle().Foreground(tokyoMoonYellow)
	default:
		return treeMetaStyle
	}
}

// modalRegistryTitleStyle 返回新增或编辑 registry 弹框标题样式。
func modalRegistryTitleStyle(editing bool) lipgloss.Style {
	if editing {
		return lipgloss.NewStyle().Foreground(tokyoMoonOrange).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(tokyoMoonGreen).Bold(true)
}

// modalRegistryFieldValueStyle 返回新增或编辑 registry 字段值样式。
func modalRegistryFieldValueStyle(field string, active bool) lipgloss.Style {
	if active {
		return lipgloss.NewStyle().Foreground(tokyoMoonYellow).Bold(true)
	}
	switch field {
	case "Name":
		return lipgloss.NewStyle().Foreground(tokyoMoonMagenta)
	case "URL":
		return lipgloss.NewStyle().Foreground(tokyoMoonCyan)
	case "Branch":
		return lipgloss.NewStyle().Foreground(tokyoMoonGreen)
	default:
		return modalFieldValueStyle
	}
}

// modalBranchChoiceStyle 返回新增或编辑 registry 分支候选项样式。
func modalBranchChoiceStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().Foreground(tokyoMoonGreen)
	if selected {
		return style.Background(tokyoMoonBgHighlight).Bold(true)
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
		Foreground(tokyoMoonBlue1).
		Bold(true)
}

// modalBranchSelectedLineStyle 返回分支列表选中项的定宽高亮样式。
func modalBranchSelectedLineStyle(width int) lipgloss.Style {
	return modalBranchChoiceStyle(true).Width(width)
}
