package wordpress

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	BaseURL     string
	Username    string
	AppPassword string
	HTTPClient  *http.Client
	Nonce       string
	Cookies     []*http.Cookie
}

type Post struct {
	ID            int        `json:"id,omitempty"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	Excerpt       string     `json:"excerpt,omitempty"`
	Status        string     `json:"status"` // draft, publish, pending, private
	Categories    []int      `json:"categories,omitempty"`
	Tags          []int      `json:"tags,omitempty"`
	Date          *time.Time `json:"date,omitempty"`
	DateGMT       *time.Time `json:"date_gmt,omitempty"`
	Slug          string     `json:"slug,omitempty"`
	Author        int        `json:"author,omitempty"`
	CommentStatus string     `json:"comment_status,omitempty"`
	PingStatus    string     `json:"ping_status,omitempty"`
}

// WordPressTime 处理WordPress API返回的时间格式
type WordPressTime struct {
	time.Time
}

// UnmarshalJSON 解析WordPress时间格式
func (wt *WordPressTime) UnmarshalJSON(data []byte) error {
	str := string(data)
	if len(str) >= 2 {
		str = str[1 : len(str)-1] // 移除引号
	}

	// 尝试多种时间格式
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			wt.Time = t
			return nil
		}
	}

	// 如果所有格式都失败，返回零时间
	wt.Time = time.Time{}
	return nil
}

type PostResponse struct {
	ID         int             `json:"id"`
	Title      TitleResponse   `json:"title"`
	Content    ContentResponse `json:"content"`
	Excerpt    ExcerptResponse `json:"excerpt"`
	Status     string          `json:"status"`
	Categories []int           `json:"categories"`
	Tags       []int           `json:"tags"`
	Date       WordPressTime   `json:"date"`
	DateGMT    WordPressTime   `json:"date_gmt"`
	Link       string          `json:"link"`
	Slug       string          `json:"slug"`
	Author     int             `json:"author"`
}

// TitleResponse 处理标题字段，可能是字符串或对象
type TitleResponse struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

// ContentResponse 处理内容字段
type ContentResponse struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

// ExcerptResponse 处理摘要字段
type ExcerptResponse struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

// GetTitle 获取标题文本
func (tr *TitleResponse) GetTitle() string {
	if tr.Raw != "" {
		return tr.Raw
	}
	return tr.Rendered
}

// GetContent 获取内容文本
func (cr *ContentResponse) GetContent() string {
	if cr.Raw != "" {
		return cr.Raw
	}
	return cr.Rendered
}

// GetExcerpt 获取摘要文本
func (er *ExcerptResponse) GetExcerpt() string {
	if er.Raw != "" {
		return er.Raw
	}
	return er.Rendered
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type ApplicationPassword struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Created  string `json:"created"`
	LastUsed string `json:"last_used"`
	LastIP   string `json:"last_ip"`
}

func NewClient(baseURL, username, appPassword string) *Client {
	jar, _ := cookiejar.New(nil)

	return &Client{
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		Username:    username,
		AppPassword: appPassword,
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) getAuthHeader() string {
	auth := fmt.Sprintf("%s:%s", c.Username, c.AppPassword)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	return fmt.Sprintf("Basic %s", encoded)
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.getAuthHeader())

	// 如果有 nonce，添加到请求头
	if c.Nonce != "" {
		req.Header.Set("X-WP-Nonce", c.Nonce)
	}

	// 如果有 cookies，添加到请求
	if c.Cookies != nil {
		for _, cookie := range c.Cookies {
			req.AddCookie(cookie)
		}
	}

	return c.HTTPClient.Do(req)
}

func (c *Client) CreatePost(post *Post) (*PostResponse, error) {
	postJSON, err := json.Marshal(post)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", "/wp-json/wp/v2/posts", bytes.NewReader(postJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建文章失败: %s, 响应: %s", resp.Status, string(body))
	}

	var postResp PostResponse
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return nil, err
	}

	return &postResp, nil
}

func (c *Client) GetPosts(page, perPage int) ([]PostResponse, error) {
	path := fmt.Sprintf("/wp-json/wp/v2/posts?page=%d&per_page=%d", page, perPage)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取文章失败: %s, 响应: %s", resp.Status, string(body))
	}

	var posts []PostResponse
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	return posts, nil
}

// GetPostsCount 获取文章总数
func (c *Client) GetPostsCount() (int, error) {
	// 获取第一页，每页1篇文章，从响应头中获取总数
	path := "/wp-json/wp/v2/posts?page=1&per_page=1"

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("获取文章总数失败: %s", resp.Status)
	}

	// 从响应头获取总数
	totalHeader := resp.Header.Get("X-WP-Total")
	if totalHeader == "" {
		// 如果没有X-WP-Total头，返回0但不报错
		return 0, nil
	}

	count, err := strconv.Atoi(totalHeader)
	if err != nil {
		return 0, nil // 解析失败也返回0
	}

	return count, nil
}

func (c *Client) GetCategories() ([]Category, error) {
	resp, err := c.doRequest("GET", "/wp-json/wp/v2/categories", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取分类失败: %s, 响应: %s", resp.Status, string(body))
	}

	var categories []Category
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (c *Client) GetCategoryByName(name string) (*Category, error) {
	categories, err := c.GetCategories()
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if strings.EqualFold(cat.Name, name) {
			return &cat, nil
		}
		if strings.EqualFold(cat.Slug, name) {
			return &cat, nil
		}
	}

	return nil, fmt.Errorf("分类 '%s' 不存在", name)
}

func (c *Client) CreateCategory(name, description string) (*Category, error) {
	category := map[string]string{
		"name":        name,
		"description": description,
	}

	categoryJSON, err := json.Marshal(category)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest("POST", "/wp-json/wp/v2/categories", bytes.NewReader(categoryJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建分类失败: %s, 响应: %s", resp.Status, string(body))
	}

	var cat Category
	if err := json.NewDecoder(resp.Body).Decode(&cat); err != nil {
		return nil, err
	}

	return &cat, nil
}

func (c *Client) TestConnection() error {
	// 尝试获取当前用户信息来测试连接
	resp, err := c.doRequest("GET", "/wp-json/wp/v2/users/me", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("连接测试失败: %s", resp.Status)
	}

	return nil
}

func (c *Client) LoginWithCredentials(username, password string) (string, error) {
	// 使用用户名密码登录获取 cookie 和 nonce
	loginData := url.Values{
		"log":         []string{username},
		"pwd":         []string{password},
		"redirect_to": []string{fmt.Sprintf("%s/wp-admin/", c.BaseURL)},
	}

	loginURL := fmt.Sprintf("%s/wp-login.php", c.BaseURL)
	resp, err := c.HTTPClient.PostForm(loginURL, loginData)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("登录失败: %s", resp.Status)
	}

	// 保存 cookies
	c.Cookies = resp.Cookies()

	// 获取 nonce
	nonceURL := fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=rest-nonce", c.BaseURL)
	req, err := http.NewRequest("GET", nonceURL, nil)
	if err != nil {
		return "", err
	}

	for _, cookie := range c.Cookies {
		req.AddCookie(cookie)
	}

	nonceResp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer nonceResp.Body.Close()

	nonceBytes, err := io.ReadAll(nonceResp.Body)
	if err != nil {
		return "", err
	}

	c.Nonce = strings.TrimSpace(string(nonceBytes))
	return c.Nonce, nil
}

func (c *Client) CreateApplicationPassword(name string) (*ApplicationPassword, error) {
	if c.Nonce == "" || c.Cookies == nil {
		return nil, fmt.Errorf("需要先使用用户名密码登录")
	}

	appPassData := map[string]string{
		"name": name,
	}

	appPassJSON, err := json.Marshal(appPassData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/wp-json/wp/v2/users/me/application-passwords", c.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(appPassJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WP-Nonce", c.Nonce)
	for _, cookie := range c.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建应用密码失败: %s, 响应: %s", resp.Status, string(body))
	}

	var appPass ApplicationPassword
	if err := json.NewDecoder(resp.Body).Decode(&appPass); err != nil {
		return nil, err
	}

	return &appPass, nil
}
