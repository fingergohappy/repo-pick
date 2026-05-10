package tui

import "github.com/charmbracelet/lipgloss"

// selectedLineStyle 是列表选中项的高亮样式。
var selectedLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Bold(true)

// modalBranchSelectedBaseStyle 是分支列表选中项的整行高亮基础样式。
var modalBranchSelectedBaseStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Background(lipgloss.Color("24")).
	Bold(true)

// paneTitleFocusedStyle 是聚焦栏目标题样式。
var paneTitleFocusedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// paneTitleMutedStyle 是未聚焦栏目标题样式。
var paneTitleMutedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("246")).
	Bold(true)

// treeMetaStyle 是右侧目录上下文信息的样式。
var treeMetaStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("246"))

// treeMetaLabelStyle 是右侧目录上下文标签的样式。
var treeMetaLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("75")).
	Bold(true)

// treeSeparatorStyle 是右侧区域分隔线样式。
var treeSeparatorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

// treeHeaderStyle 是右侧内容表头样式。
var treeHeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("111")).
	Bold(true)

// treeLoadingTitleStyle 是右侧加载态标题样式。
var treeLoadingTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// treeLoadingTextStyle 是右侧加载态进度文本样式。
var treeLoadingTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("248"))

// treeLoadingHintStyle 是右侧加载态说明样式。
var treeLoadingHintStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// dirEntryStyle 是目录条目的样式。
var dirEntryStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("75")).
	Bold(true)

// fileEntryStyle 是文件条目的样式。
var fileEntryStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("250"))

// treeRootEntryStyle 是右侧 tree root 条目的样式。
var treeRootEntryStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("111")).
	Bold(true)

// emptyEntryStyle 是空列表提示的样式。
var emptyEntryStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244")).
	Italic(true)

// registryEmptyTitleStyle 是 registry 空状态标题样式。
var registryEmptyTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	Bold(true)

// registryEmptyHintStyle 是 registry 空状态说明样式。
var registryEmptyHintStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// registryEmptyCardStyle 是 registry 空状态提示块样式。
var registryEmptyCardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("238")).
	Padding(0, 1)

// registryEmptyKeyStyle 是 registry 空状态快捷键样式。
var registryEmptyKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Background(lipgloss.Color("24")).
	Bold(true).
	Padding(0, 1)

// registryEmptyActionStyle 是 registry 空状态动作文本样式。
var registryEmptyActionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("250"))

// modalTitleStyle 是弹框标题样式。
var modalTitleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// modalDescStyle 是弹框说明文本样式。
var modalDescStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// modalFieldLabelStyle 是弹框字段标签样式。
var modalFieldLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// modalFieldValueStyle 是弹框字段值样式。
var modalFieldValueStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

// modalHintStyle 是弹框底部快捷键提示样式。
var modalHintStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// modalDividerStyle 是弹框内部分隔线样式。
var modalDividerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("238"))

// statusTextStyle 是底部状态消息的样式。
var statusTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252"))

// statusHelpStyle 是底部快捷键提示的弱化样式。
var statusHelpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

// statusLineStyle 是底部状态栏文本布局样式。
var statusLineStyle = lipgloss.NewStyle()

// helpSectionStyle 是帮助视图分组标题样式。
var helpSectionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("39")).
	Bold(true)

// helpKeyStyle 是帮助视图按键样式。
var helpKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("255")).
	Reverse(true).
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
		return style.BorderForeground(lipgloss.Color("39"))
	}
	return style.BorderForeground(lipgloss.Color("238"))
}

// modalStyle 构造弹框容器样式：统一边框、内边距和颜色。
func modalStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))
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
		Foreground(lipgloss.Color("75")).
		Bold(true)
}

// modalBranchSelectedLineStyle 返回分支列表选中项的定宽高亮样式。
func modalBranchSelectedLineStyle(width int) lipgloss.Style {
	return modalBranchSelectedBaseStyle.Width(width)
}
