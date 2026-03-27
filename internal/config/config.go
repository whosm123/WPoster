package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrUserNotFound = errors.New("用户不存在")
	ErrURLNotFound  = errors.New("URL不存在")
)

type Config struct {
	BaseURLs map[string]BaseURLConfig `json:"base_urls"`
	Users    map[string]UserConfig    `json:"users"`
	filePath string
	key      []byte
}

type BaseURLConfig struct {
	DefaultUser string `json:"default_user"`
}

type UserConfig struct {
	BaseURL     string `json:"base_url"`
	Username    string `json:"username"`
	AppPassword string `json:"app_password"` // 加密存储
	LastLogin   string `json:"last_login"`
}

func NewConfig() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "users.json")
	keyPath := filepath.Join(configDir, ".key")

	// 生成或加载加密密钥
	key, err := getOrCreateKey(keyPath)
	if err != nil {
		return nil, err
	}

	config := &Config{
		BaseURLs: make(map[string]BaseURLConfig),
		Users:    make(map[string]UserConfig),
		filePath: configPath,
		key:      key,
	}

	// 如果配置文件存在，加载它
	if _, err := os.Stat(configPath); err == nil {
		if err := config.Load(); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".wposter")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}
	return configDir, nil
}

func getOrCreateKey(keyPath string) ([]byte, error) {
	// 如果密钥文件存在，读取它
	if _, err := os.Stat(keyPath); err == nil {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		if len(key) == 32 {
			return key, nil
		}
	}

	// 生成新的32字节密钥
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// 保存密钥文件
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

func (c *Config) Load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		// 如果文件不存在，创建空配置
		if os.IsNotExist(err) {
			c.BaseURLs = make(map[string]BaseURLConfig)
			c.Users = make(map[string]UserConfig)
			return nil
		}
		return err
	}

	// 空文件处理
	if len(data) == 0 {
		c.BaseURLs = make(map[string]BaseURLConfig)
		c.Users = make(map[string]UserConfig)
		return nil
	}

	// 解析JSON
	return json.Unmarshal(data, c)
}

func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath, data, 0600)
}

func (c *Config) AddUser(username, baseURL, usernameVal, appPassword string) error {
	encryptedPassword, err := c.encrypt(appPassword)
	if err != nil {
		return err
	}

	c.Users[username] = UserConfig{
		BaseURL:     baseURL,
		Username:    usernameVal,
		AppPassword: encryptedPassword,
		LastLogin:   time.Now().Format(time.RFC3339),
	}

	// 如果这是该URL的第一个用户，设置为默认用户
	if _, exists := c.BaseURLs[baseURL]; !exists {
		c.BaseURLs[baseURL] = BaseURLConfig{
			DefaultUser: username,
		}
	}

	return c.Save()
}

func (c *Config) GetUser(username string) (UserConfig, error) {
	user, exists := c.Users[username]
	if !exists {
		return UserConfig{}, ErrUserNotFound
	}

	// 解密密码
	decryptedPassword, err := c.decrypt(user.AppPassword)
	if err != nil {
		return UserConfig{}, err
	}

	user.AppPassword = decryptedPassword
	return user, nil
}

func (c *Config) ListUsers() []string {
	users := make([]string, 0, len(c.Users))
	for username := range c.Users {
		users = append(users, username)
	}
	return users
}

func (c *Config) ListBaseURLs() []string {
	urls := make([]string, 0, len(c.BaseURLs))
	for url := range c.BaseURLs {
		urls = append(urls, url)
	}
	return urls
}

func (c *Config) GetUsersByBaseURL(baseURL string) []string {
	users := make([]string, 0)
	for username, user := range c.Users {
		if user.BaseURL == baseURL {
			users = append(users, username)
		}
	}
	return users
}

func (c *Config) UpdateLastLogin(username string) error {
	user, exists := c.Users[username]
	if !exists {
		return ErrUserNotFound
	}

	user.LastLogin = time.Now().Format(time.RFC3339)
	c.Users[username] = user
	return c.Save()
}

func (c *Config) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *Config) decrypt(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文太短")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (c *Config) Validate() error {
	for username, user := range c.Users {
		if user.BaseURL == "" {
			return fmt.Errorf("用户 %s 缺少 base_url", username)
		}
		if user.Username == "" {
			return fmt.Errorf("用户 %s 缺少 username", username)
		}
		if user.AppPassword == "" {
			return fmt.Errorf("用户 %s 缺少 app_password", username)
		}
	}
	return nil
}
