package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/wposter/internal/config"
	"github.com/yourusername/wposter/internal/markdown"
	"github.com/yourusername/wposter/internal/ui"
	"github.com/yourusername/wposter/internal/wordpress"
)

type App struct {
	config *config.Config
	client *wordpress.Client
	user   config.UserConfig
}

func NewApp() (*App, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %v", err)
	}

	return &App{
		config: cfg,
	}, nil
}

func (a *App) Run() error {
	ui.DisplayWelcome()

	// 主循环
	for {
		if a.client == nil {
			// 需要登录
			if err := a.loginFlow(); err != nil {
				if err.Error() == "用户取消" {
					ui.PrintInfo("程序退出")
					return nil
				}
				ui.PrintError(fmt.Sprintf("登录失败: %v", err))
				continue
			}
		}

		// 显示用户信息和文章
		a.showUserInfo()

		// 显示主菜单
		choice, err := ui.ShowMenu("主菜单", []string{
			"发布新文章",
			"查看文章列表",
			"管理分类",
			"切换用户",
			"退出程序",
		})
		if err != nil {
			ui.PrintError(fmt.Sprintf("选择失败: %v", err))
			continue
		}

		switch choice {
		case 0: // 发布新文章
			if err := a.createPostFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("发布文章失败: %v", err))
			}
		case 1: // 查看文章列表
			if err := a.listPostsFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("获取文章列表失败: %v", err))
			}
		case 2: // 管理分类
			if err := a.manageCategoriesFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("管理分类失败: %v", err))
			}
		case 3: // 切换用户
			a.client = nil
			a.user = config.UserConfig{}
			ui.PrintInfo("已退出当前用户")
		case 4: // 退出程序
			ui.PrintInfo("感谢使用 WPoster，再见！")
			return nil
		}
	}
}

func (a *App) loginFlow() error {
	ui.PrintTitle("用户登录")

	// 检查是否有已保存的用户
	users := a.config.ListUsers()
	if len(users) > 0 {
		choice, _, err := ui.SelectFromList("请选择登录方式", []string{
			"使用已有账户登录",
			"新建账户登录",
			"退出程序",
		})
		if err != nil {
			return err
		}

		switch choice {
		case 0: // 使用已有账户
			return a.loginWithExistingUser()
		case 1: // 新建账户
			return a.loginWithNewUser()
		case 2: // 退出程序
			return fmt.Errorf("用户取消")
		}
	} else {
		// 没有已保存的用户，直接进入新建账户流程
		ui.PrintInfo("没有找到已保存的用户，请新建账户")
		return a.loginWithNewUser()
	}

	return nil
}

func (a *App) loginWithExistingUser() error {
	users := a.config.ListUsers()
	if len(users) == 0 {
		return fmt.Errorf("没有已保存的用户")
	}

	// 选择用户
	_, username, err := ui.SelectFromList("请选择用户", users)
	if err != nil {
		return err
	}

	// 加载用户配置
	user, err := a.config.GetUser(username)
	if err != nil {
		return fmt.Errorf("加载用户配置失败: %v", err)
	}

	// 创建客户端并测试连接
	ui.PrintInfo(fmt.Sprintf("正在连接到 %s...", user.BaseURL))
	client := wordpress.NewClient(user.BaseURL, user.Username, user.AppPassword)

	stopSpinner := ui.ShowSpinner("测试连接")
	if err := client.TestConnection(); err != nil {
		stopSpinner()
		return fmt.Errorf("连接测试失败: %v", err)
	}
	stopSpinner()

	// 更新最后登录时间
	if err := a.config.UpdateLastLogin(username); err != nil {
		ui.PrintWarning(fmt.Sprintf("更新登录时间失败: %v", err))
	}

	a.client = client
	a.user = user

	ui.PrintSuccess(fmt.Sprintf("登录成功！欢迎回来 %s", user.Username))
	return nil
}

func (a *App) loginWithNewUser() error {
	ui.PrintTitle("新建账户登录")

	// 选择或输入站点URL
	var baseURL string
	baseURLs := a.config.ListBaseURLs()

	if len(baseURLs) > 0 {
		choice, selectedURL, err := ui.SelectFromList("请选择站点", append(baseURLs, "添加新站点"))
		if err != nil {
			return err
		}

		if choice < len(baseURLs) {
			baseURL = selectedURL
		} else {
			// 添加新站点
			baseURL, err = ui.InputText("请输入 WordPress 站点 URL (例如: https://example.com)", "")
			if err != nil {
				return err
			}

			// 确保 URL 格式正确
			if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
				baseURL = "https://" + baseURL
			}
		}
	} else {
		// 没有已保存的站点，需要输入
		var err error
		baseURL, err = ui.InputText("请输入 WordPress 站点 URL (例如: https://example.com)", "")
		if err != nil {
			return err
		}

		// 确保 URL 格式正确
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			baseURL = "https://" + baseURL
		}
	}

	// 选择登录方式
	choice, _, err := ui.SelectFromList("请选择登录方式", []string{
		"使用用户名和密码登录（自动生成应用密码）",
		"直接使用应用密码登录",
		"返回上级",
	})
	if err != nil {
		return err
	}

	switch choice {
	case 0: // 用户名密码登录
		return a.loginWithUsernamePassword(baseURL)
	case 1: // 应用密码登录
		return a.loginWithAppPassword(baseURL)
	case 2: // 返回
		return fmt.Errorf("用户取消")
	}

	return nil
}

func (a *App) loginWithUsernamePassword(baseURL string) error {
	ui.PrintTitle("用户名密码登录")

	// 输入用户名
	username, err := ui.InputText("请输入 WordPress 用户名/邮箱", "")
	if err != nil {
		return err
	}

	// 输入密码
	password, err := ui.InputPassword("请输入密码")
	if err != nil {
		return err
	}

	// 输入要保存的配置名称
	configName, err := ui.InputText("请输入要保存的配置名称", fmt.Sprintf("%s@%s",
		strings.Split(username, "@")[0],
		strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")))
	if err != nil {
		return err
	}

	// 创建临时客户端用于登录
	tempClient := wordpress.NewClient(baseURL, "", "")

	stopSpinner := ui.ShowSpinner("正在登录并生成应用密码")

	// 使用用户名密码登录获取 nonce
	_, err = tempClient.LoginWithCredentials(username, password)
	if err != nil {
		stopSpinner()
		return fmt.Errorf("登录失败: %v", err)
	}

	// 生成应用密码
	appPass, err := tempClient.CreateApplicationPassword("WPoster CLI")
	if err != nil {
		stopSpinner()
		return fmt.Errorf("生成应用密码失败: %v", err)
	}

	stopSpinner()

	ui.PrintSuccess(fmt.Sprintf("应用密码生成成功: %s", appPass.Password))

	// 保存用户配置
	if err := a.config.AddUser(configName, baseURL, username, appPass.Password); err != nil {
		return fmt.Errorf("保存用户配置失败: %v", err)
	}

	// 创建正式客户端
	client := wordpress.NewClient(baseURL, username, appPass.Password)

	stopSpinner = ui.ShowSpinner("测试连接")
	if err := client.TestConnection(); err != nil {
		stopSpinner()
		return fmt.Errorf("连接测试失败: %v", err)
	}
	stopSpinner()

	a.client = client
	a.user = config.UserConfig{
		BaseURL:     baseURL,
		Username:    username,
		AppPassword: appPass.Password,
	}

	ui.PrintSuccess(fmt.Sprintf("登录成功！欢迎 %s", username))
	return nil
}

func (a *App) loginWithAppPassword(baseURL string) error {
	ui.PrintTitle("应用密码登录")

	// 输入用户名
	username, err := ui.InputText("请输入 WordPress 用户名/邮箱", "")
	if err != nil {
		return err
	}

	// 输入应用密码
	appPassword, err := ui.InputPassword("请输入应用密码")
	if err != nil {
		return err
	}

	// 输入要保存的配置名称
	configName, err := ui.InputText("请输入要保存的配置名称", fmt.Sprintf("%s@%s",
		strings.Split(username, "@")[0],
		strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")))
	if err != nil {
		return err
	}

	// 测试连接
	ui.PrintInfo(fmt.Sprintf("正在连接到 %s...", baseURL))
	client := wordpress.NewClient(baseURL, username, appPassword)

	stopSpinner := ui.ShowSpinner("测试连接")
	if err := client.TestConnection(); err != nil {
		stopSpinner()
		return fmt.Errorf("连接测试失败: %v", err)
	}
	stopSpinner()

	// 保存用户配置
	if err := a.config.AddUser(configName, baseURL, username, appPassword); err != nil {
		return fmt.Errorf("保存用户配置失败: %v", err)
	}

	a.client = client
	a.user = config.UserConfig{
		BaseURL:     baseURL,
		Username:    username,
		AppPassword: appPassword,
	}

	ui.PrintSuccess(fmt.Sprintf("登录成功！欢迎 %s", username))
	return nil
}

func (a *App) showUserInfo() {
	// 获取文章总数
	postCount := 0
	if a.client != nil {
		count, err := a.client.GetPostsCount()
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("获取文章总数失败: %v", err))
		} else {
			postCount = count
		}
	}

	// 显示用户信息
	ui.DisplayUserInfo(a.user.Username, a.user.BaseURL, postCount)

	// 获取最近10篇文章
	if a.client != nil {
		posts, err := a.client.GetPosts(1, 10)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("获取最近文章失败: %v", err))
			return
		}

		// 显示最近文章
		if len(posts) > 0 {
			ui.DisplayPosts(posts)
			ui.PrintBlankLine()
		}
	}
}

func (a *App) createPostFlow() error {
	ui.PrintTitle("发布新文章")

	// 输入文章标题
	title, err := ui.InputText("请输入文章标题", "")
	if err != nil {
		return err
	}

	// 选择或输入分类
	categories, err := a.client.GetCategories()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("获取分类失败: %v", err))
		// 继续，用户可以输入分类名称
	}

	var categoryID int
	if len(categories) > 0 {
		// 显示现有分类
		var categoryNames []string
		for _, cat := range categories {
			categoryNames = append(categoryNames, fmt.Sprintf("%s (%d篇文章)", cat.Name, cat.Count))
		}
		categoryNames = append(categoryNames, "新建分类")

		choice, _, err := ui.SelectFromList("请选择文章分类", categoryNames)
		if err != nil {
			return err
		}

		if choice < len(categories) {
			categoryID = categories[choice].ID
		} else {
			// 新建分类
			categoryName, err := ui.InputText("请输入新分类名称", "")
			if err != nil {
				return err
			}

			categoryDesc, _ := ui.InputTextOptional("请输入分类描述（可选）", "")

			stopSpinner := ui.ShowSpinner("创建分类")
			newCategory, err := a.client.CreateCategory(categoryName, categoryDesc)
			if err != nil {
				stopSpinner()
				return fmt.Errorf("创建分类失败: %v", err)
			}
			stopSpinner()

			categoryID = newCategory.ID
			ui.PrintSuccess(fmt.Sprintf("分类创建成功: %s (ID: %d)", newCategory.Name, newCategory.ID))
		}
	} else {
		// 没有分类，需要创建
		categoryName, err := ui.InputText("请输入分类名称", "未分类")
		if err != nil {
			return err
		}

		stopSpinner := ui.ShowSpinner("创建分类")
		newCategory, err := a.client.CreateCategory(categoryName, "")
		if err != nil {
			stopSpinner()
			return fmt.Errorf("创建分类失败: %v", err)
		}
		stopSpinner()

		categoryID = newCategory.ID
		ui.PrintSuccess(fmt.Sprintf("分类创建成功: %s (ID: %d)", newCategory.Name, newCategory.ID))
	}

	// 输入 Markdown 文件路径
	filePath, err := ui.InputText("请输入 Markdown 文件路径", "")
	if err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	// 选择文章状态
	statusChoice, _, err := ui.SelectFromList("请选择文章状态", []string{
		"草稿 (draft)",
		"立即发布 (publish)",
		"待审核 (pending)",
		"私密文章 (private)",
	})
	if err != nil {
		return err
	}

	statusMap := []string{"draft", "publish", "pending", "private"}
	status := statusMap[statusChoice]

	// 转换 Markdown
	stopSpinner := ui.ShowSpinner("转换 Markdown")
	htmlContent, err := markdown.ConvertMarkdownFile(filePath)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("转换 Markdown 失败: %v", err)
	}

	content := string(htmlContent)
	// 简单提取摘要（前200个字符）
	excerpt := ""
	if len(content) > 200 {
		excerpt = content[:200] + "..."
	} else {
		excerpt = content
	}

	// 创建文章
	post := &wordpress.Post{
		Title:      title,
		Content:    content,
		Excerpt:    excerpt,
		Status:     status,
		Categories: []int{categoryID},
		// 不设置Date和DateGMT，让WordPress自动使用当前时间
		// 如果设置为nil，omitempty标签会确保这些字段不被发送
	}

	stopSpinner = ui.ShowSpinner("发布文章")
	postResp, err := a.client.CreatePost(post)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("发布文章失败: %v", err)
	}

	ui.PrintSuccess(fmt.Sprintf("文章发布成功！"))
	ui.PrintInfo(fmt.Sprintf("文章ID: %d", postResp.ID))
	ui.PrintInfo(fmt.Sprintf("文章链接: %s", postResp.Link))
	ui.PrintInfo(fmt.Sprintf("文章状态: %s", postResp.Status))

	ui.WaitForEnter("按 Enter 键继续...")
	return nil
}

func (a *App) listPostsFlow() error {
	ui.PrintTitle("文章列表")

	// 获取文章
	stopSpinner := ui.ShowSpinner("加载文章")
	posts, err := a.client.GetPosts(1, 20)
	stopSpinner()

	if err != nil {
		return err
	}

	if len(posts) == 0 {
		ui.PrintInfo("没有找到文章")
		ui.WaitForEnter("按 Enter 键返回...")
		return nil
	}

	// 显示文章
	ui.DisplayPosts(posts)
	ui.PrintBlankLine()

	// 选择查看文章详情
	choice, err := ui.InputYesNo("是否查看某篇文章的详情？", false)
	if err != nil {
		return err
	}

	if choice {
		postID, err := ui.InputNumber("请输入文章ID", 1, 10000)
		if err != nil {
			return err
		}

		// 这里可以添加查看文章详情的功能
		ui.PrintInfo(fmt.Sprintf("文章ID %d 的详情功能尚未实现", postID))
	}

	ui.WaitForEnter("按 Enter 键返回...")
	return nil
}

func (a *App) manageCategoriesFlow() error {
	ui.PrintTitle("分类管理")

	// 获取分类
	stopSpinner := ui.ShowSpinner("加载分类")
	categories, err := a.client.GetCategories()
	stopSpinner()

	if err != nil {
		return err
	}

	// 显示分类
	ui.DisplayCategories(categories)
	ui.PrintBlankLine()

	// 选择操作
	choice, _, err := ui.SelectFromList("请选择操作", []string{
		"新建分类",
		"返回上级",
	})
	if err != nil {
		return err
	}

	switch choice {
	case 0: // 新建分类
		return a.createCategoryFlow()
	case 1: // 返回
		return nil
	}

	return nil
}

func (a *App) createCategoryFlow() error {
	ui.PrintTitle("新建分类")

	// 输入分类名称
	name, err := ui.InputText("请输入分类名称", "")
	if err != nil {
		return err
	}

	// 输入分类描述
	description, _ := ui.InputTextOptional("请输入分类描述（可选）", "")

	// 创建分类
	stopSpinner := ui.ShowSpinner("创建分类")
	category, err := a.client.CreateCategory(name, description)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("创建分类失败: %v", err)
	}

	ui.PrintSuccess(fmt.Sprintf("分类创建成功！"))
	ui.PrintInfo(fmt.Sprintf("分类ID: %d", category.ID))
	ui.PrintInfo(fmt.Sprintf("分类名称: %s", category.Name))
	ui.PrintInfo(fmt.Sprintf("分类描述: %s", category.Description))

	ui.WaitForEnter("按 Enter 键继续...")
	return nil
}

func (a *App) selectFile() (string, error) {
	// 简单的文件选择器
	fmt.Println("请选择 Markdown 文件:")
	fmt.Println("1. 输入文件路径")
	fmt.Println("2. 浏览当前目录")
	fmt.Println("3. 取消")

	choice, err := ui.InputNumber("请选择", 1, 3)
	if err != nil {
		return "", err
	}

	switch choice {
	case 1: // 输入文件路径
		return ui.InputText("请输入文件路径", "")
	case 2: // 浏览当前目录
		return a.browseDirectory()
	case 3: // 取消
		return "", fmt.Errorf("用户取消")
	}

	return "", nil
}

func (a *App) browseDirectory() (string, error) {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 查找 .md 文件
	var mdFiles []string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			relPath, _ := filepath.Rel(cwd, path)
			mdFiles = append(mdFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(mdFiles) == 0 {
		return "", fmt.Errorf("当前目录没有找到 .md 文件")
	}

	// 选择文件
	choice, _, err := ui.SelectFromList("请选择 Markdown 文件", mdFiles)
	if err != nil {
		return "", err
	}

	return filepath.Join(cwd, mdFiles[choice]), nil
}
