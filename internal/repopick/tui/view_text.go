package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// selectedLine 按需渲染选中行高亮。
func selectedLine(line string, selected bool) string {
	if !selected {
		return line
	}
	return selectedLineStyle.Render(line)
}

// selectionCursor 返回当前选中行的动画光标。
func (m model) selectionCursor() string {
	frames := []string{">", "›", "»", "›"}
	return frames[m.selectionCursorFrame%len(frames)]
}

// centerLine 按可见宽度居中渲染单行文本。
func centerLine(line string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(line)
}

// fitPlainLine 将普通文本限制为单行并按宽度截断。
func fitPlainLine(line string, width int) string {
	return truncateVisible(firstLine(line), width)
}

// firstLine 取首行文本，避免多行错误或输入破坏状态栏布局。
func firstLine(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return text[:index]
	}
	return text
}

// splitRenderedLines 按行拆分已渲染字符串。
func splitRenderedLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

// truncateVisible 按终端可见宽度截断单行文本。
func truncateVisible(text string, width int) string {
	text = firstLine(text)
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	suffix := "..."
	limit := width - lipgloss.Width(suffix)
	var builder strings.Builder
	for _, char := range text {
		next := builder.String() + string(char)
		if lipgloss.Width(next) > limit {
			break
		}
		builder.WriteRune(char)
	}
	return builder.String() + suffix
}

// limitLines 截取最多 limit 行。
func limitLines(lines []string, limit int) []string {
	if limit < 0 {
		limit = 0
	}
	if len(lines) <= limit {
		return lines
	}
	return lines[:limit]
}

// visibleWindow 返回围绕当前光标的可见列表区间。
func visibleWindow(total int, selected int, limit int) (int, int) {
	if total <= 0 || limit <= 0 {
		return 0, 0
	}
	if total <= limit {
		return 0, total
	}
	selected = clampCursor(selected, total)
	start := selected - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > total {
		start = total - limit
	}
	return start, start + limit
}
