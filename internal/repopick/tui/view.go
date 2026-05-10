package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

const (
	paneGapAllowance  = 3
	minTerminalWidth  = 80
	minTerminalHeight = 24
)

// View 渲染双栏主界面和底部状态。
func (m model) View() string {
	if m.terminalTooSmall() {
		return m.narrowView()
	}

	leftWidth, rightWidth := paneWidths(m.width)
	left := renderPane(m.registryPaneTitle(), m.registryLinesView(), paneRenderOptions{
		width:   leftWidth,
		height:  m.paneHeight(),
		focused: m.focus == focusRegistry,
	})
	right := renderPane(m.treePaneTitle(), m.treeLinesView(), paneRenderOptions{
		width:   rightWidth,
		height:  m.paneHeight(),
		focused: m.focus == focusTree,
	})
	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bodyHeight := max(8, m.height-3)
	if m.showHelp {
		main = placeOverlay(main, m.helpView(), m.width, bodyHeight)
	} else if m.isAddMode() {
		main = placeOverlay(main, m.addRepositoryModalView(), m.width, bodyHeight)
	} else if m.mode == modeConfirmDelete {
		main = placeOverlay(main, m.deleteRepositoryConfirmModalView(), m.width, bodyHeight)
	}

	return renderScreen(main, m.statusLine())
}

// narrowView 渲染终端过窄时的提示。
func (m model) narrowView() string {
	message := fmt.Sprintf("terminal too small: need at least %dx%d", minTerminalWidth, minTerminalHeight)
	return fitPlainLine(message, m.width) + "\n" + renderStatusLine(m.status, "? help", m.width)
}

// terminalTooSmall 判断当前终端是否低于可用尺寸。
func (m model) terminalTooSmall() bool {
	return (m.width > 0 && m.width < minTerminalWidth) || (m.height > 0 && m.height < minTerminalHeight)
}
