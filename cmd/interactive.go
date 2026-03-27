package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/whosm123/WPoster/internal/config"
	"github.com/whosm123/WPoster/internal/markdown"
	"github.com/whosm123/WPoster/internal/ui"
	"github.com/whosm123/WPoster/internal/wordpress"
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
			"查看文章详情",
			"搜索文章",
			"删除文章",
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
		case 2: // 查看文章详情
			if err := a.viewPostDetailFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("查看文章详情失败: %v", err))
			}
		case 3: // 搜索文章
			if err := a.searchPostsFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("搜索文章失败: %v", err))
			}
		case 4: // 删除文章
			if err := a.deletePostFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("删除文章失败: %v", err))
			}
		case 5: // 管理分类
			if err := a.manageCategoriesFlow(); err != nil {
				ui.PrintError(fmt.Sprintf("管理分类失败: %v", err))
			}
		case 6: // 切换用户
			a.client = nil
			a.user = config.UserConfig{}
			ui.PrintInfo("已退出当前用户")
		case 7: // 退出程序
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

	// 使用滚动列表选择文章
	postID, err := a.showPostsScrollList("请选择文章查看详情（使用上下键滚动）")
	if err != nil {
		return err
	}

	if postID == 0 {
		// 用户选择了返回
		return nil
	}

	// 获取文章详情
	stopSpinner := ui.ShowSpinner("加载文章详情")
	postDetail, err := a.client.GetPostByID(postID)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("获取文章详情失败: %v", err)
	}

	// 显示文章详情
	return a.showPostDetail(*postDetail)
}

// showPostsScrollList 显示文章滚动列表并返回选择的文章ID
func (a *App) showPostsScrollList(prompt string) (int, error) {
	// 获取文章总数
	stopSpinner := ui.ShowSpinner("获取文章总数")
	totalCount, err := a.client.GetPostsCount()
	stopSpinner()

	if err != nil {
		return 0, fmt.Errorf("获取文章总数失败: %v", err)
	}

	if totalCount == 0 {
		ui.PrintInfo("没有找到文章")
		return 0, nil
	}

	ui.PrintInfo(fmt.Sprintf("共有 %d 篇文章", totalCount))

	// 获取所有文章（限制最大数量，避免性能问题）
	maxPosts := 100
	perPage := 50
	allPosts := []wordpress.PostResponse{}

	// 计算需要获取的页数
	pagesToFetch := (totalCount + perPage - 1) / perPage
	if pagesToFetch > maxPosts/perPage {
		pagesToFetch = maxPosts / perPage
		ui.PrintInfo(fmt.Sprintf("文章较多，只显示前 %d 篇", maxPosts))
	}

	// 分批获取文章
	for page := 1; page <= pagesToFetch; page++ {
		stopSpinner = ui.ShowSpinner(fmt.Sprintf("加载文章 (已加载 %d 篇)", len(allPosts)))
		posts, err := a.client.GetPosts(page, perPage)
		stopSpinner()

		if err != nil {
			return 0, fmt.Errorf("获取文章失败: %v", err)
		}

		if len(posts) == 0 {
			break
		}

		allPosts = append(allPosts, posts...)
	}

	// 创建文章选项列表
	options := make([]string, len(allPosts))
	for i, post := range allPosts {
		var dateStr string
		if !post.Date.Time.IsZero() {
			dateStr = post.Date.Format("2006-01-02")
		} else {
			dateStr = "未知日期"
		}
		options[i] = fmt.Sprintf("[%d] %s (%s)", post.ID, post.Title.Rendered, dateStr)
	}

	// 添加返回选项
	options = append(options, "返回上级")

	// 显示滚动列表
	choice, _, err := ui.SelectFromList(prompt, options)
	if err != nil {
		return 0, err
	}

	// 处理选择
	if choice == len(options)-1 {
		// 选择"返回上级"
		return 0, nil
	}

	// 返回选择的文章ID
	return allPosts[choice].ID, nil
}

// showPostDetail 显示文章详情
func (a *App) showPostDetail(post wordpress.PostResponse) error {
	ui.PrintTitle("文章详情")

	// 显示文章基本信息
	fmt.Printf("ID: %d\n", post.ID)
	fmt.Printf("标题: %s\n", post.Title.Rendered)

	var dateStr string
	if !post.Date.Time.IsZero() {
		dateStr = post.Date.Format("2006-01-02 15:04:05")
	} else {
		dateStr = "未知"
	}
	fmt.Printf("发布时间: %s\n", dateStr)
	fmt.Printf("状态: %s\n", post.Status)
	fmt.Printf("链接: %s\n", post.Link)

	// 显示分类信息
	if len(post.Categories) > 0 {
		stopSpinner := ui.ShowSpinner("获取分类信息")
		categories, err := a.client.GetCategories()
		stopSpinner()

		if err == nil {
			categoryNames := []string{}
			for _, catID := range post.Categories {
				for _, cat := range categories {
					if cat.ID == catID {
						categoryNames = append(categoryNames, cat.Name)
						break
					}
				}
			}
			if len(categoryNames) > 0 {
				fmt.Printf("分类: %s\n", strings.Join(categoryNames, ", "))
			}
		}
	}

	ui.PrintDivider()

	// 显示文章内容（清理HTML标签）
	content := cleanHTML(post.Content.Rendered)
	if len(content) > 500 {
		content = content[:500] + "..."
	}
	fmt.Printf("内容预览:\n%s\n", content)

	ui.PrintDivider()

	// 等待用户按键
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
		"删除分类",
		"返回上级",
	})
	if err != nil {
		return err
	}

	switch choice {
	case 0: // 新建分类
		return a.createCategoryFlow()
	case 1: // 删除分类
		return a.deleteCategoryFlow(categories)
	case 2: // 返回
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

// viewPostDetailFlow 查看文章详情流程
func (a *App) viewPostDetailFlow() error {
	ui.PrintTitle("查看文章详情")

	// 先显示文章滚动列表
	postID, err := a.showPostsScrollList("请选择要查看的文章（使用上下键滚动）")
	if err != nil {
		return err
	}

	if postID == 0 {
		// 用户选择了返回
		return nil
	}

	// 获取文章详情
	stopSpinner := ui.ShowSpinner("加载文章详情")
	postDetail, err := a.client.GetPostByID(postID)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("获取文章详情失败: %v", err)
	}

	// 显示文章详情
	ui.PrintTitle("文章详情")
	fmt.Printf("ID: %d\n", postID)
	fmt.Printf("标题: %s\n", postDetail.Title.GetTitle())
	fmt.Printf("状态: %s\n", getStatusChinese(postDetail.Status))
	fmt.Printf("发布时间: %s\n", postDetail.Date.Time.Format("2006-01-02 15:04:05"))
	fmt.Printf("链接: %s\n", postDetail.Link)

	// 显示分类
	if len(postDetail.Categories) > 0 {
		fmt.Printf("分类: ")
		// 这里可以获取分类名称，暂时显示ID
		for i, catID := range postDetail.Categories {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%d", catID)
		}
		fmt.Println()
	}

	// 显示内容 - 扩大显示窗口为10行
	content := postDetail.Content.GetContent()

	// 清理HTML标签，只显示文本内容
	content = cleanHTML(content)

	// 按行分割内容
	lines := strings.Split(content, "\n")

	fmt.Printf("\n内容预览（显示前%d行）:\n", len(lines))
	fmt.Println(strings.Repeat("─", 60))

	// 显示前10行或全部行（如果少于10行）
	displayLines := 10
	if len(lines) < displayLines {
		displayLines = len(lines)
	}

	for i := 0; i < displayLines && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			fmt.Printf("%2d. %s\n", i+1, line)
		}
	}

	if len(lines) > displayLines {
		fmt.Printf("...（还有 %d 行未显示）\n", len(lines)-displayLines)
	}

	fmt.Println(strings.Repeat("─", 60))

	ui.WaitForEnter("\n按 Enter 键返回...")
	return nil
}

// cleanHTML 简单清理HTML标签，只保留文本内容
func cleanHTML(html string) string {
	// 简单的HTML标签清理
	html = strings.ReplaceAll(html, "<p>", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n")
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br />", "\n")

	// 移除其他HTML标签
	var result strings.Builder
	inTag := false

	for _, ch := range html {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(ch)
		}
	}

	// 清理多余的空格和换行
	cleaned := result.String()
	cleaned = strings.ReplaceAll(cleaned, "&nbsp;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")

	// 合并多个换行
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(cleaned)
}

// deletePostFlow 删除文章流程
func (a *App) deletePostFlow() error {
	ui.PrintTitle("删除文章")

	// 使用滚动列表选择文章
	postID, err := a.showPostsScrollList("请选择要删除的文章（使用上下键滚动）")
	if err != nil {
		return err
	}

	if postID == 0 {
		// 用户选择了返回
		return nil
	}

	// 获取文章详情
	stopSpinner := ui.ShowSpinner("加载文章详情")
	postDetail, err := a.client.GetPostByID(postID)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("获取文章详情失败: %v", err)
	}

	// 确认并删除文章
	return a.confirmAndDeletePost(*postDetail)
}

// confirmAndDeletePost 确认并删除文章
func (a *App) confirmAndDeletePost(selectedPost wordpress.PostResponse) error {
	// 先显示文章详情
	ui.PrintTitle("文章详情")
	fmt.Printf("ID: %d\n", selectedPost.ID)
	fmt.Printf("标题: %s\n", selectedPost.Title.GetTitle())
	fmt.Printf("状态: %s\n", getStatusChinese(selectedPost.Status))
	fmt.Printf("发布时间: %s\n", selectedPost.Date.Time.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// 确认删除
	confirm, err := ui.Confirm(fmt.Sprintf("确定要删除这篇文章吗？"))
	if err != nil {
		return err
	}

	if !confirm {
		ui.PrintInfo("已取消删除")
		return nil
	}

	// 选择删除方式
	deleteChoice, _, err := ui.SelectFromList("请选择删除方式", []string{
		"移动到回收站（可恢复）",
		"永久删除（不可恢复）",
		"取消",
	})
	if err != nil {
		return err
	}

	if deleteChoice == 2 {
		ui.PrintInfo("已取消删除")
		return nil
	}

	forceDelete := (deleteChoice == 1)

	// 执行删除
	stopSpinner := ui.ShowSpinner("删除文章中")
	err = a.client.DeletePost(selectedPost.ID, forceDelete)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("删除文章失败: %v", err)
	}

	if forceDelete {
		ui.PrintSuccess("文章已永久删除")
	} else {
		ui.PrintSuccess("文章已移动到回收站")
	}

	ui.WaitForEnter("按 Enter 键继续...")
	return nil
}

// updateManageCategoriesFlow 更新管理分类流程，添加删除功能
func (a *App) updateManageCategoriesFlow() error {
	// 先调用原有的管理分类流程
	if err := a.manageCategoriesFlow(); err != nil {
		return err
	}

	// 这里可以添加额外的分类管理功能
	return nil
}

// searchPostsFlow 搜索文章流程
func (a *App) searchPostsFlow() error {
	ui.PrintTitle("搜索文章")

	// 输入搜索关键词
	query, err := ui.InputText("请输入搜索关键词", "")
	if err != nil {
		return err
	}

	if query == "" {
		return fmt.Errorf("搜索关键词不能为空")
	}

	// 执行搜索
	stopSpinner := ui.ShowSpinner("搜索文章中")
	posts, err := a.client.SearchPosts(query, 1, 20)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("搜索失败: %v", err)
	}

	if len(posts) == 0 {
		ui.PrintInfo(fmt.Sprintf("没有找到包含 '%s' 的文章", query))
		ui.WaitForEnter("按 Enter 键返回...")
		return nil
	}

	// 显示搜索结果
	ui.PrintSuccess(fmt.Sprintf("找到 %d 篇相关文章:", len(posts)))
	ui.DisplayPosts(posts)

	// 提供操作选项
	choice, _, err := ui.SelectFromList("请选择操作", []string{
		"查看文章详情",
		"返回上级",
	})
	if err != nil {
		return err
	}

	if choice == 0 {
		// 这里可以添加查看搜索结果中文章详情的功能
		// 暂时简化处理
		ui.PrintInfo("查看文章详情功能请从主菜单选择")
	}

	ui.WaitForEnter("按 Enter 键返回...")
	return nil
}

// deleteCategoryFlow 删除分类流程
func (a *App) deleteCategoryFlow(categories []wordpress.Category) error {
	ui.PrintTitle("删除分类")

	if len(categories) == 0 {
		return fmt.Errorf("没有可删除的分类")
	}

	// 创建分类选择列表
	var categoryOptions []string
	for _, cat := range categories {
		option := fmt.Sprintf("%d: %s (%d篇文章)", cat.ID, cat.Name, cat.Count)
		categoryOptions = append(categoryOptions, option)
	}

	choice, _, err := ui.SelectFromList("请选择要删除的分类", categoryOptions)
	if err != nil {
		return err
	}

	selectedCat := categories[choice]

	// 检查分类是否有文章
	if selectedCat.Count > 0 {
		ui.PrintWarning(fmt.Sprintf("分类 '%s' 包含 %d 篇文章，删除前需要先移动或删除这些文章。",
			selectedCat.Name, selectedCat.Count))

		confirm, err := ui.Confirm("确定要删除这个有文章的分类吗？")
		if err != nil {
			return err
		}
		if !confirm {
			ui.PrintInfo("已取消删除")
			return nil
		}
	}

	// 确认删除
	confirm, err := ui.Confirm(fmt.Sprintf("确定要删除分类 '%s' (ID: %d) 吗？",
		selectedCat.Name, selectedCat.ID))
	if err != nil {
		return err
	}

	if !confirm {
		ui.PrintInfo("已取消删除")
		return nil
	}

	// 执行删除
	stopSpinner := ui.ShowSpinner("删除分类中")

	// WordPress分类不支持回收站，总是使用force=true
	// 即使分类没有文章，也需要force参数
	forceDelete := true
	err = a.client.DeleteCategory(selectedCat.ID, forceDelete)
	stopSpinner()

	if err != nil {
		return fmt.Errorf("删除分类失败: %v", err)
	}

	if selectedCat.Count > 0 {
		ui.PrintSuccess(fmt.Sprintf("分类 '%s' 已强制删除（包含%d篇文章）", selectedCat.Name, selectedCat.Count))
	} else {
		ui.PrintSuccess(fmt.Sprintf("分类 '%s' 已删除", selectedCat.Name))
	}
	ui.WaitForEnter("按 Enter 键继续...")
	return nil
}

// 辅助函数：获取状态的中文显示
func getStatusChinese(status string) string {
	switch status {
	case "publish":
		return "已发布"
	case "draft":
		return "草稿"
	case "pending":
		return "待审核"
	case "private":
		return "私密"
	case "trash":
		return "回收站"
	default:
		return status
	}
}
