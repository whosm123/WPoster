# WPoster - WordPress文章远程管理CLI工具

一个用Go编写的Windows命令行工具，用于远程管理和发布markdown格式的WordPress文章及分类，支持中文内容、Markdown转换和交互式操作。

## 功能特性

-  **交互式CLI界面** - 无需记忆复杂命令，菜单驱动操作
-  **Markdown支持** - 自动将Markdown转换为HTML发布
-  **多用户配置** - 支持多个WordPress站点和用户
-  **一键发布** - 快速发布文章到指定分类
-  **自动抓取应用密码** - 无需手动创建，输入用户密码即可自动抓取您的应用密码

## 安装

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/whosm123/WPoster.git
cd WPoster

# 安装依赖并编译
go mod download
go build -o wposter.exe
```

### 直接下载

从 [Releases](https://github.com/whosm123/WPoster/releases) 页面下载预编译的 `wposter.exe`。

## 使用方法

### 首次运行

```bash
wposter.exe
```

程序会引导你：
1. 选择登录方式（新建账户或使用已有账户）
2. 输入WordPress站点URL
3. 输入用户名/密码或应用密码
<img width="893" height="371" alt="image" src="https://github.com/user-attachments/assets/c6f52a94-5b19-42a2-9fd6-76c922f7d805" />

### 主要功能

1. **查看最近文章** - 显示最近的10篇文章（支持中文标题截断）
2. **查看分类列表** - 显示站点所有分类
3. **发布新文章** - 从Markdown文件发布文章
4. **退出程序**
<img width="807" height="1056" alt="image" src="https://github.com/user-attachments/assets/70450d5b-9ae2-4027-8663-07ec042e98b7" />


### 发布文章流程

1. 输入文章标题
2. 选择或创建分类
3. 选择Markdown文件路径
4. 选择文章状态（草稿/立即发布/待审核/私密）
5. 自动转换Markdown为HTML并发布
 <img width="888" height="960" alt="image" src="https://github.com/user-attachments/assets/751ae9f4-a673-46b8-98fa-654b816c5260" />


## 配置
所有凭据都在本地存储，且存储的是您输入或者根据您的用户名+密码创建的应用密码。
配置文件存储在 `~/.wposter/users.json`，包含：
- 站点URL和默认用户
- 用户凭证（应用密码加密存储）
- 最后登录时间

## 技术架构

```
WPoster/
├── cmd/interactive.go      # 交互式CLI主程序
├── internal/
│   ├── config/             # JSON配置管理（多用户支持）
│   ├── wordpress/          # WordPress REST API客户端
│   ├── markdown/           # Markdown到HTML转换
│   └── ui/                 # 用户界面和表格显示
├── main.go                 # 程序入口点
└── README.md               # 本文档
```

### 核心技术

- **表格显示算法** - 准确计算中文字符显示宽度（2列），英文字符宽度（1列）
- **时间戳处理** - 正确处理WordPress时间格式，避免文章"消失"问题
- **错误处理** - 完善的错误处理和用户反馈
- **安全存储** - 应用密码加密存储

## 开发

欢迎提交PR！

### 依赖

- Go 1.19+
- WordPress 5.6+（支持应用密码）

### 构建

```bash
go build -o wposter.exe
```

### 测试

```bash
go test ./...
```

## 常见问题


### Q: 中文标题显示不完整？
A: 表格显示算法会自动截断过长的中文标题，在完整字符边界添加"..."。

### Q: 如何获取WordPress应用密码？
A: 您可以直接提供用户名和密码，WPoster会自动创建应用密码来访问RestAPI。
您也可以自己创建应用密码：在WordPress后台：用户 → 编辑用户 → 应用密码 → 添加新应用密码。
<img width="2548" height="986" alt="image" src="https://github.com/user-attachments/assets/2ee7ba87-0c85-47d3-8959-5a6adb8a4c77" />


### Q: 支持多个WordPress站点吗？
A: 是的，支持多个站点和用户，同一站点多个用户，配置自动管理。

## 贡献

欢迎提交Issue和Pull Request！

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

## 许可证

MIT 

## 作者

Makari

## 致谢

- [gomarkdown/markdown](https://github.com/gomarkdown/markdown) - Markdown解析器
- [WordPress REST API](https://developer.wordpress.org/rest-api/) - WordPress API文档
