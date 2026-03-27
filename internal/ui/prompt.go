package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/manifoldco/promptui"
)

var (
	// 样式定义
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	// 分隔线
	Divider = strings.Repeat("─", 60)
)

func PrintTitle(title string) {
	fmt.Println(TitleStyle.Render("┌─ " + title + " ─┐"))
}

func PrintSuccess(message string) {
	fmt.Println(SuccessStyle.Render("✓ " + message))
}

func PrintError(message string) {
	fmt.Println(ErrorStyle.Render("✗ " + message))
}

func PrintInfo(message string) {
	fmt.Println(InfoStyle.Render("ℹ " + message))
}

func PrintWarning(message string) {
	fmt.Println(WarningStyle.Render("⚠ " + message))
}

func PrintDivider() {
	fmt.Println(Divider)
}

func PrintBlankLine() {
	fmt.Println()
}

func SelectFromList(label string, items []string) (int, string, error) {
	if len(items) == 0 {
		return -1, "", fmt.Errorf("没有可选项")
	}

	prompt := promptui.Select{
		Label: label,
		Items: items,
		Size:  10,
	}

	index, result, err := prompt.Run()
	if err != nil {
		return -1, "", err
	}

	return index, result, nil
}

func SelectFromListWithDetails(label string, items []string, descriptions []string) (int, string, error) {
	// 简化版本：忽略描述，使用普通选择
	return SelectFromList(label, items)
}

func InputText(label string, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("输入不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

func InputTextOptional(label string, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}

	return prompt.Run()
}

func InputPassword(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Mask:  '•',
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("密码不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

func InputYesNo(label string, defaultValue bool) (bool, error) {
	defaultStr := "是"
	if !defaultValue {
		defaultStr = "否"
	}

	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("%s (是/否)", label),
		Default:   defaultStr,
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}

	return strings.ToLower(result) == "是" || strings.ToLower(result) == "y", nil
}

func InputNumber(label string, min, max int) (int, error) {
	for {
		fmt.Printf("%s (%d-%d): ", label, min, max)

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return min, nil
		}

		num, err := strconv.Atoi(input)
		if err != nil {
			PrintError("请输入有效的数字")
			continue
		}

		if num < min || num > max {
			PrintError(fmt.Sprintf("数字必须在 %d 到 %d 之间", min, max))
			continue
		}

		return num, nil
	}
}

func ShowMenu(title string, options []string) (int, error) {
	PrintTitle(title)
	PrintBlankLine()

	for i, option := range options {
		fmt.Printf("  [%d] %s\n", i+1, option)
	}
	PrintBlankLine()

	choice, err := InputNumber("请选择", 1, len(options))
	if err != nil {
		return -1, err
	}

	return choice - 1, nil
}

func ShowProgress(message string) func() {
	fmt.Printf("⏳ %s...", message)

	return func() {
		fmt.Println(" 完成")
	}
}

func ShowSpinner(message string) func() {
	done := make(chan bool)

	go func() {
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				fmt.Print("\r")
				return
			default:
				fmt.Printf("\r%s %s", spinnerChars[i], message)
				i = (i + 1) % len(spinnerChars)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return func() {
		done <- true
		time.Sleep(100 * time.Millisecond)
		fmt.Println("\r✓ " + message + " 完成")
	}
}

func ShowTable(headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// 计算每列最大宽度
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// 打印表头
	headerLine := "┌"
	for i, width := range colWidths {
		headerLine += strings.Repeat("─", width+2)
		if i < len(colWidths)-1 {
			headerLine += "┬"
		}
	}
	headerLine += "┐"
	fmt.Println(headerLine)

	fmt.Print("│")
	for i, header := range headers {
		fmt.Printf(" %-*s │", colWidths[i], header)
	}
	fmt.Println()

	// 打印分隔线
	sepLine := "├"
	for i, width := range colWidths {
		sepLine += strings.Repeat("─", width+2)
		if i < len(colWidths)-1 {
			sepLine += "┼"
		}
	}
	sepLine += "┤"
	fmt.Println(sepLine)

	// 打印数据行
	for _, row := range rows {
		fmt.Print("│")
		for i := 0; i < len(colWidths); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Printf(" %-*s │", colWidths[i], cell)
		}
		fmt.Println()
	}

	// 打印底部边框
	footerLine := "└"
	for i, width := range colWidths {
		footerLine += strings.Repeat("─", width+2)
		if i < len(colWidths)-1 {
			footerLine += "┴"
		}
	}
	footerLine += "┘"
	fmt.Println(footerLine)
}

func WaitForEnter(message string) {
	if message != "" {
		fmt.Print(message)
	} else {
		fmt.Print("按 Enter 键继续...")
	}

	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

func DisplayWelcome() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    WPoster - WordPress CLI               ║")
	fmt.Println("║                 WordPress 文章管理工具                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func DisplayUserInfo(username, baseURL string, postCount int) {
	fmt.Println("┌──────────────────────────────────────────────────────────┐")
	fmt.Printf("│ 用户: %-50s │\n", username)
	fmt.Printf("│ 站点: %-50s │\n", baseURL)
	fmt.Printf("│ 文章数: %-48d │\n", postCount)
	fmt.Println("└──────────────────────────────────────────────────────────┘")
	fmt.Println()
}

// PromptForBaseURL prompts user for WordPress base URL
func PromptForBaseURL() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入 WordPress 网站地址 (例如: https://example.com)",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("网站地址不能为空")
			}
			if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
				return fmt.Errorf("请输入有效的 URL (以 http:// 或 https:// 开头)")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForLoginMethod prompts user to choose login method
func PromptForLoginMethod() (string, error) {
	prompt := promptui.Select{
		Label: "选择登录方式",
		Items: []string{
			"使用用户名和密码登录 (自动生成应用密码)",
			"直接使用应用密码登录",
		},
	}

	_, result, err := prompt.Run()
	return result, err
}

// PromptForUsername prompts for WordPress username
func PromptForUsername() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入 WordPress 用户名",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("用户名不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForPassword prompts for WordPress password
func PromptForPassword() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入 WordPress 密码",
		Mask:  '*',
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("密码不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForAppPassword prompts for application password
func PromptForAppPassword() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入应用密码",
		Mask:  '*',
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("应用密码不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForConfigName prompts for configuration name
func PromptForConfigName(defaultName string) (string, error) {
	prompt := promptui.Prompt{
		Label:   "请输入配置名称",
		Default: defaultName,
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("配置名称不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForMainMenu shows main menu and returns selection
func PromptForMainMenu() (string, error) {
	prompt := promptui.Select{
		Label: "请选择操作",
		Items: []string{
			"查看最近文章",
			"发布新文章",
			"切换用户",
			"退出",
		},
	}

	_, result, err := prompt.Run()
	return result, err
}

// PromptForPostTitle prompts for post title
func PromptForPostTitle() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入文章标题",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("文章标题不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForCategory prompts for category name or ID
func PromptForCategory() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入分类名称或ID (多个用逗号分隔)",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("分类不能为空")
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForMarkdownFile prompts for markdown file path
func PromptForMarkdownFile() (string, error) {
	prompt := promptui.Prompt{
		Label: "请输入 Markdown 文件路径",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("文件路径不能为空")
			}
			if _, err := os.Stat(input); os.IsNotExist(err) {
				return fmt.Errorf("文件不存在: %s", input)
			}
			return nil
		},
	}

	return prompt.Run()
}

// PromptForPostStatus prompts for post status
func PromptForPostStatus() (string, error) {
	prompt := promptui.Select{
		Label: "选择文章状态",
		Items: []string{
			"draft",
			"publish",
			"pending",
			"private",
		},
	}

	_, result, err := prompt.Run()
	return result, err
}

// PromptYesNo prompts for yes/no confirmation
func PromptYesNo(question string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     question + " (y/n)",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}

	return strings.ToLower(result) == "y", nil
}
