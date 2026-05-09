package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// tableStyleResolver 用于按单元格决定 table 样式。
type tableStyleResolver func(row int, col int) lipgloss.Style

// renderTableRows 使用无边框 table 渲染多列内容。
func renderTableRows(rows [][]string, width int, resolver tableStyleResolver) string {
	if width <= 0 || len(rows) == 0 {
		return ""
	}

	columnWidths := tableColumnWidths(rows, width)
	cleanRows := make([][]string, len(rows))
	for i, row := range rows {
		if len(row) == 0 {
			cleanRows[i] = []string{""}
			continue
		}
		cleanRows[i] = make([]string, len(row))
		for col, value := range row {
			cleanRows[i][col] = truncateVisible(firstLine(value), columnWidths[col])
		}
	}

	if resolver == nil {
		resolver = func(row int, col int) lipgloss.Style {
			return lipgloss.NewStyle()
		}
	}

	t := table.New().
		Rows(cleanRows...).
		Width(width).
		Wrap(false).
		StyleFunc(table.StyleFunc(resolver)).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderRow(false).
		BorderColumn(false)
	return t.Render()
}

// renderKeyValueTable 渲染两列 key/value 布局。
func renderKeyValueTable(rows [][2]string, width int, resolver tableStyleResolver) string {
	items := make([][]string, len(rows))
	for i, row := range rows {
		items[i] = []string{row[0], row[1]}
	}
	return renderTableRows(items, width, resolver)
}

// renderSelectableTable 渲染可选表格并对 selected 行使用高亮。
func renderSelectableTable(rows [][]string, selected int, width int, resolver tableStyleResolver) string {
	resolved := func(row int, col int) lipgloss.Style {
		if row == selected {
			return selectedLineStyle
		}
		if resolver == nil {
			return lipgloss.NewStyle()
		}
		return resolver(row, col)
	}
	return renderTableRows(rows, width, resolved)
}

// renderHelpTable 渲染帮助列表。
func renderHelpTable(bindings [][2]string, width int) string {
	rows := make([][]string, len(bindings))
	for i, binding := range bindings {
		rows[i] = []string{binding[0], binding[1]}
	}
	return renderTableRows(rows, width, nil)
}

// tableColumnWidths 按宽度预算每列等宽切分。
func tableColumnWidths(rows [][]string, totalWidth int) []int {
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	if maxCols == 0 {
		return nil
	}

	widths := make([]int, maxCols)
	remaining := totalWidth
	for i := 0; i < maxCols; i++ {
		if i == maxCols-1 {
			widths[i] = max(1, remaining)
			break
		}
		contentWidth := 1
		for _, row := range rows {
			if i < len(row) {
				contentWidth = max(contentWidth, lipgloss.Width(firstLine(row[i])))
			}
		}
		maxBeforeLast := max(1, remaining-(maxCols-i-1))
		widths[i] = max(1, min(contentWidth, maxBeforeLast))
		remaining -= widths[i]
	}
	return widths
}
