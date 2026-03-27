package ui

import (
	"fmt"
	"strings"

	"github.com/whosm123/WPoster/internal/wordpress"
)

// displayWidth 计算字符串在终端中的显示宽度
// 中文字符宽度为2，英文字符宽度为1
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r <= 127 {
			width += 1 // ASCII字符
		} else {
			width += 2 // 中文字符
		}
	}
	return width
}

// truncateToWidth 截断字符串到指定显示宽度，考虑中文字符宽度
func truncateToWidth(s string, maxWidth int) string {
	if displayWidth(s) <= maxWidth {
		return s
	}

	// 需要截断
	currentWidth := 0
	runes := []rune(s)
	result := []rune{}

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		charWidth := 1
		if r > 127 {
			charWidth = 2
		}

		// 检查添加当前字符和省略号是否会超过最大宽度
		if currentWidth+charWidth+3 > maxWidth { // 3是"..."的宽度
			break
		}

		result = append(result, r)
		currentWidth += charWidth
	}

	return string(result) + "..."
}

// padToWidth 填充字符串到指定显示宽度
func padToWidth(s string, width int) string {
	displayW := displayWidth(s)
	if displayW >= width {
		return s
	}

	// 需要填充空格（空格总是占用1个显示宽度）
	return s + strings.Repeat(" ", width-displayW)
}

// DisplayPosts 显示文章列表表格
func DisplayPosts(posts []wordpress.PostResponse) {
	fmt.Println("最近文章:")

	// 表格宽度定义（显示宽度）
	idWidth := 4     // ID列显示宽度
	titleWidth := 30 // 标题列显示宽度
	statusWidth := 8 // 状态列显示宽度

	// 打印表头
	printTableHeader(idWidth, titleWidth, statusWidth)

	// 表头内容
	idHeader := padToWidth("ID", idWidth)
	titleHeader := padToWidth("标题", titleWidth)
	statusHeader := padToWidth("状态", statusWidth)
	fmt.Printf("│ %s │ %s │ %s │\n", idHeader, titleHeader, statusHeader)

	// 分隔线
	printTableSeparator(idWidth, titleWidth, statusWidth)

	for _, post := range posts {
		// 获取标题
		title := post.Title.GetTitle()
		if title == "" {
			title = "(无标题)"
		}

		// 截断标题到固定显示宽度
		title = truncateToWidth(title, titleWidth)

		// 格式化状态
		status := post.Status
		switch status {
		case "publish":
			status = "已发布"
		case "draft":
			status = "草稿"
		case "pending":
			status = "待审核"
		case "private":
			status = "私密"
		}

		// 准备各列内容
		idStr := padToWidth(fmt.Sprintf("%d", post.ID), idWidth)
		titleStr := padToWidth(title, titleWidth)
		statusStr := padToWidth(status, statusWidth)

		// 打印行
		fmt.Printf("│ %s │ %s │ %s │\n", idStr, titleStr, statusStr)
	}

	// 底部边框
	printTableFooter(idWidth, titleWidth, statusWidth)
}

// printTableHeader 打印表格头部边框
func printTableHeader(idWidth, titleWidth, statusWidth int) {
	line := "┌"
	line += strings.Repeat("─", idWidth+2)
	line += "┬"
	line += strings.Repeat("─", titleWidth+2)
	line += "┬"
	line += strings.Repeat("─", statusWidth+2)
	line += "┐"
	fmt.Println(line)
}

// printTableSeparator 打印表格分隔线
func printTableSeparator(idWidth, titleWidth, statusWidth int) {
	line := "├"
	line += strings.Repeat("─", idWidth+2)
	line += "┼"
	line += strings.Repeat("─", titleWidth+2)
	line += "┼"
	line += strings.Repeat("─", statusWidth+2)
	line += "┤"
	fmt.Println(line)
}

// printTableFooter 打印表格底部边框
func printTableFooter(idWidth, titleWidth, statusWidth int) {
	line := "└"
	line += strings.Repeat("─", idWidth+2)
	line += "┴"
	line += strings.Repeat("─", titleWidth+2)
	line += "┴"
	line += strings.Repeat("─", statusWidth+2)
	line += "┘"
	fmt.Println(line)
}

// DisplayCategories 显示分类列表表格
func DisplayCategories(categories []wordpress.Category) {
	fmt.Println("分类列表:")

	// 表格宽度定义
	idWidth := 4    // ID列显示宽度
	nameWidth := 20 // 名称列显示宽度
	countWidth := 8 // 文章数列显示宽度
	descWidth := 10 // 描述列显示宽度

	// 打印表头
	line := "┌"
	line += strings.Repeat("─", idWidth+2)
	line += "┬"
	line += strings.Repeat("─", nameWidth+2)
	line += "┬"
	line += strings.Repeat("─", countWidth+2)
	line += "┬"
	line += strings.Repeat("─", descWidth+2)
	line += "┐"
	fmt.Println(line)

	// 表头内容
	idHeader := padToWidth("ID", idWidth)
	nameHeader := padToWidth("名称", nameWidth)
	countHeader := padToWidth("文章数", countWidth)
	descHeader := padToWidth("描述", descWidth)
	fmt.Printf("│ %s │ %s │ %s │ %s │\n", idHeader, nameHeader, countHeader, descHeader)

	// 分隔线
	line = "├"
	line += strings.Repeat("─", idWidth+2)
	line += "┼"
	line += strings.Repeat("─", nameWidth+2)
	line += "┼"
	line += strings.Repeat("─", countWidth+2)
	line += "┼"
	line += strings.Repeat("─", descWidth+2)
	line += "┤"
	fmt.Println(line)

	for _, cat := range categories {
		// 截断名称和描述到固定显示宽度
		name := truncateToWidth(cat.Name, nameWidth)
		desc := truncateToWidth(cat.Description, descWidth)
		if desc == "" {
			desc = "-"
		}

		// 准备各列内容
		idStr := padToWidth(fmt.Sprintf("%d", cat.ID), idWidth)
		nameStr := padToWidth(name, nameWidth)
		countStr := padToWidth(fmt.Sprintf("%d", cat.Count), countWidth)
		descStr := padToWidth(desc, descWidth)

		// 打印行
		fmt.Printf("│ %s │ %s │ %s │ %s │\n", idStr, nameStr, countStr, descStr)
	}

	// 底部边框
	line = "└"
	line += strings.Repeat("─", idWidth+2)
	line += "┴"
	line += strings.Repeat("─", nameWidth+2)
	line += "┴"
	line += strings.Repeat("─", countWidth+2)
	line += "┴"
	line += strings.Repeat("─", descWidth+2)
	line += "┘"
	fmt.Println(line)
}
