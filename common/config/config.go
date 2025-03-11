package config

import (
	"errors"
	"math/rand"
	"os"
	"sourcegraph2api/common/env"
	"strings"
	"sync"
	"time"
)

var ApiSecret = os.Getenv("API_SECRET")
var ApiSecrets = strings.Split(os.Getenv("API_SECRET"), ",")
var SGCookie = os.Getenv("SG_COOKIE")
var IpBlackList = strings.Split(os.Getenv("IP_BLACK_LIST"), ",")
var AutoDelChat = env.Int("AUTO_DEL_CHAT", 0)
var ProxyUrl = env.String("PROXY_URL", "")

// 隐藏思考过程
var ReasoningHide = env.Int("REASONING_HIDE", 0)

// 前置message
var PRE_MESSAGES_JSON = env.String("PRE_MESSAGES_JSON", "")
var RateLimitCookieLockDuration = env.Int("RATE_LIMIT_COOKIE_LOCK_DURATION", 60)

// 路由前缀
var RoutePrefix = env.String("ROUTE_PREFIX", "")
var AllDialogRecordEnable = os.Getenv("ALL_DIALOG_RECORD_ENABLE")
var SwaggerEnable = os.Getenv("SWAGGER_ENABLE")
var DebugEnabled = os.Getenv("DEBUG") == "true"

var RateLimitKeyExpirationDuration = 20 * time.Minute

var RequestOutTimeDuration = 5 * time.Minute

var (
	RequestRateLimitNum            = env.Int("REQUEST_RATE_LIMIT", 60)
	RequestRateLimitDuration int64 = 1 * 60
)

type RateLimitCookie struct {
	ExpirationTime time.Time // 过期时间
}

var (
	rateLimitCookies sync.Map // 使用 sync.Map 管理限速 Cookie
)

func AddRateLimitCookie(cookie string, expirationTime time.Time) {
	rateLimitCookies.Store(cookie, RateLimitCookie{
		ExpirationTime: expirationTime,
	})
	//fmt.Printf("Storing cookie: %s with value: %+v\n", cookie, RateLimitCookie{ExpirationTime: expirationTime})
}

type CookieManager struct {
	Cookies      []string
	currentIndex int
	mu           sync.Mutex
}

var (
	SGCookies    []string   // 存储所有的 cookies
	cookiesMutex sync.Mutex // 保护 SGCookies 的互斥锁
)

// InitSGCookies 初始化 SGCookies
func InitSGCookies() {
	cookiesMutex.Lock()
	defer cookiesMutex.Unlock()

	SGCookies = []string{}

	// 从环境变量中读取 SG_COOKIE 并拆分为切片
	cookieStr := os.Getenv("SG_COOKIE")
	if cookieStr != "" {

		for _, cookie := range strings.Split(cookieStr, ",") {
			// 如果 cookie 不包含 "session_id="，则添加前缀
			if !strings.Contains(cookie, "sgs=") {
				cookie = "sgs=" + cookie
			}
			SGCookies = append(SGCookies, cookie)
		}
	}
}

// RemoveCookie 删除指定的 cookie（支持并发）
func RemoveCookie(cookieToRemove string) {
	cookiesMutex.Lock()
	defer cookiesMutex.Unlock()

	// 创建一个新的切片，过滤掉需要删除的 cookie
	var newCookies []string
	for _, cookie := range GetSGCookies() {
		if cookie != cookieToRemove {
			newCookies = append(newCookies, cookie)
		}
	}

	// 更新 SGCookies
	SGCookies = newCookies
}

// GetSGCookies 获取 SGCookies 的副本
func GetSGCookies() []string {
	//cookiesMutex.Lock()
	//defer cookiesMutex.Unlock()

	// 返回 SGCookies 的副本，避免外部直接修改
	cookiesCopy := make([]string, len(SGCookies))
	copy(cookiesCopy, SGCookies)
	return cookiesCopy
}

// NewCookieManager 创建 CookieManager
func NewCookieManager() *CookieManager {
	var validCookies []string
	// 遍历 SGCookies
	for _, cookie := range GetSGCookies() {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue // 忽略空字符串
		}

		// 检查是否在 RateLimitCookies 中
		if value, ok := rateLimitCookies.Load(cookie); ok {
			rateLimitCookie, ok := value.(RateLimitCookie) // 正确转换为 RateLimitCookie
			if !ok {
				continue
			}
			if rateLimitCookie.ExpirationTime.After(time.Now()) {
				// 如果未过期，忽略该 cookie
				continue
			} else {
				// 如果已过期，从 RateLimitCookies 中删除
				rateLimitCookies.Delete(cookie)
			}
		}

		// 添加到有效 cookie 列表
		validCookies = append(validCookies, cookie)
	}

	return &CookieManager{
		Cookies:      validCookies,
		currentIndex: 0,
	}
}

func IsRateLimited(cookie string) bool {
	if value, ok := rateLimitCookies.Load(cookie); ok {
		rateLimitCookie := value.(RateLimitCookie)
		return rateLimitCookie.ExpirationTime.After(time.Now())
	}
	return false
}

func (cm *CookieManager) RemoveCookie(cookieToRemove string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.Cookies) == 0 {
		return errors.New("no cookies available")
	}

	// 查找要删除的cookie的索引
	index := -1
	for i, cookie := range cm.Cookies {
		if cookie == cookieToRemove {
			index = i
			break
		}
	}

	// 如果没找到要删除的cookie
	if index == -1 {
		return errors.New("RemoveCookie -> cookie not found")
	}

	// 从切片中删除cookie
	cm.Cookies = append(cm.Cookies[:index], cm.Cookies[index+1:]...)

	// 如果当前索引大于或等于删除后的切片长度，重置为0
	if cm.currentIndex >= len(cm.Cookies) {
		cm.currentIndex = 0
	}

	return nil
}

func (cm *CookieManager) GetNextCookie() (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.Cookies) == 0 {
		return "", errors.New("no cookies available")
	}

	cm.currentIndex = (cm.currentIndex + 1) % len(cm.Cookies)
	return cm.Cookies[cm.currentIndex], nil
}

func (cm *CookieManager) GetRandomCookie() (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.Cookies) == 0 {
		return "", errors.New("no cookies available")
	}

	// 生成随机索引
	randomIndex := rand.Intn(len(cm.Cookies))
	// 更新当前索引
	cm.currentIndex = randomIndex

	return cm.Cookies[randomIndex], nil
}
