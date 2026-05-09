package tui

import "fmt"

// registryPaneTitle 返回左侧 registry 栏标题。
func (m model) registryPaneTitle() string {
	return fmt.Sprintf("Registry (%d)", len(m.repositories))
}

// treePaneTitle 返回右侧目录树栏标题。
func (m model) treePaneTitle() string {
	if m.showingSearch {
		return fmt.Sprintf("Repository Tree - Search (%d)", len(m.searchResults))
	}
	if m.repoOpened {
		return fmt.Sprintf("Repository Tree - %s", m.openedRepositoryName())
	}
	return "Repository Tree"
}

// paneHeight 返回主栏目可用高度。
func (m model) paneHeight() int {
	return max(8, m.height-3)
}

// paneBodyRows 返回栏目边框内的可用文本行数。
func (m model) paneBodyRows() int {
	return max(1, m.paneHeight()-2)
}

// paneItemRows 返回扣除栏目标题和分隔线后的可用列表行数。
func (m model) paneItemRows() int {
	return max(1, m.paneBodyRows()-2)
}

// paneContentWidth 返回栏目边框和左右 padding 内的文本宽度。
func paneContentWidth(width int) int {
	return max(1, width-4)
}

// paneWidths 返回左右两栏宽度，registry 保持较窄导航区域。
func paneWidths(totalWidth int) (int, int) {
	leftWidth := max(20, totalWidth*25/100)
	rightWidth := max(40, totalWidth-leftWidth-paneGapAllowance)
	return leftWidth, rightWidth
}
