package common

import (
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	_ "github.com/pkoukk/tiktoken-go"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// splitStringByBytes 将字符串按照指定的字节数进行切割
func SplitStringByBytes(s string, size int) []string {
	var result []string

	for len(s) > 0 {
		// 初始切割点
		l := size
		if l > len(s) {
			l = len(s)
		}

		// 确保不在字符中间切割
		for l > 0 && !utf8.ValidString(s[:l]) {
			l--
		}

		// 如果 l 减到 0，说明 size 太小，无法容纳一个完整的字符
		if l == 0 {
			l = len(s)
			for l > 0 && !utf8.ValidString(s[:l]) {
				l--
			}
		}

		result = append(result, s[:l])
		s = s[l:]
	}

	return result
}

func Obj2Bytes(obj interface{}) ([]byte, error) {
	// 创建一个jsonIter的Encoder
	configCompatibleWithStandardLibrary := jsoniter.ConfigCompatibleWithStandardLibrary
	// 将结构体转换为JSON文本并保持顺序
	bytes, err := configCompatibleWithStandardLibrary.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func GetUUID() string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	return code
}

// RandomElement 返回给定切片中的随机元素
func RandomElement[T any](slice []T) (T, error) {
	if len(slice) == 0 {
		var zero T
		return zero, fmt.Errorf("empty slice")
	}

	// 确保每次随机都不一样
	rand.Seed(time.Now().UnixNano())

	// 随机选择一个索引
	index := rand.Intn(len(slice))
	return slice[index], nil
}

func SliceContains(slice []string, str string) bool {
	for _, item := range slice {
		if strings.Contains(str, item) {
			return true
		}
	}
	return false
}

func IsImageBase64(s string) bool {
	// 检查字符串是否符合数据URL的格式
	if !strings.HasPrefix(s, "data:image/") || !strings.Contains(s, ";base64,") {
		return false
	}

	if !strings.Contains(s, ";base64,") {
		return false
	}

	// 获取";base64,"后的Base64编码部分
	dataParts := strings.Split(s, ";base64,")
	if len(dataParts) != 2 {
		return false
	}
	base64Data := dataParts[1]

	// 尝试Base64解码
	_, err := base64.StdEncoding.DecodeString(base64Data)
	return err == nil
}

func IsBase64(s string) bool {
	// 检查字符串是否符合数据URL的格式
	//if !strings.HasPrefix(s, "data:image/") || !strings.Contains(s, ";base64,") {
	//	return false
	//}

	if !strings.Contains(s, ";base64,") {
		return false
	}

	// 获取";base64,"后的Base64编码部分
	dataParts := strings.Split(s, ";base64,")
	if len(dataParts) != 2 {
		return false
	}
	base64Data := dataParts[1]

	// 尝试Base64解码
	_, err := base64.StdEncoding.DecodeString(base64Data)
	return err == nil
}

//<h1 data-translate="block_headline">Sorry, you have been blocked</h1>

func IsCloudflareBlock(data string) bool {
	if strings.Contains(data, `<h1 data-translate="block_headline">Sorry, you have been blocked</h1>`) {
		return true
	}

	return false
}

func IsCloudflareChallenge(data string) bool {
	// 检查基本的 HTML 结构
	htmlPattern := `^<!DOCTYPE html><html.*?><head>.*?</head><body.*?>.*?</body></html>$`

	// 检查 Cloudflare 特征
	cfPatterns := []string{
		`<title>Just a moment\.\.\.</title>`,          // 标题特征
		`window\._cf_chl_opt`,                         // CF 配置对象
		`challenge-platform/h/b/orchestrate/chl_page`, // CF challenge 路径
		`cdn-cgi/challenge-platform`,                  // CDN 路径特征
		`<meta http-equiv="refresh" content="\d+">`,   // 刷新 meta 标签
	}

	// 首先检查整体 HTML 结构
	matched, _ := regexp.MatchString(htmlPattern, strings.TrimSpace(data))
	if !matched {
		return false
	}

	// 检查是否包含 Cloudflare 特征
	for _, pattern := range cfPatterns {
		if matched, _ := regexp.MatchString(pattern, data); matched {
			return true
		}
	}

	return false
}

func IsRateLimit(data string) bool {
	if strings.HasPrefix(data, `{"error":"OpenAI (Sourcegraph Cody Gateway): unexpected status code 429: you have exceeded the concurrency limit of 3 requests for \"chat_completions\". Retry after`) ||
		strings.HasPrefix(data, `{"error":"Anthropic (Sourcegraph Cody Gateway): unexpected status code 429: you have exceeded the concurrency limit of 3 requests for \"chat_completions\". Retry after`) ||
		strings.HasPrefix(data, `{"error":"OpenAI (Sourcegraph Cody Gateway): unexpected status code 429: you have exceeded the concurrency limit of 3 requests for \"chat_completions\". Retry after`) {
		return true
	}

	return false
}

func IsNotLogin(data string) bool {
	if strings.Contains(data, `{"status":-5,"message":"not login","data":{}}`) {
		return true
	}

	return false
}

func IsServerError(data string) bool {
	if data == "Internal Server Error" {
		return true
	}

	return false
}

// 使用 MD5 算法
func StringToMD5(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

// 使用 SHA1 算法
func StringToSHA1(str string) string {
	hash := sha1.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

// 使用 SHA256 算法
func StringToSHA256(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}

// GenerateTraceParent 生成 transparent
func GenerateTraceParent() (string, error) {
	// 生成 16 字节的 trace ID
	traceID := make([]byte, 16)
	_, err := cryptorand.Read(traceID)
	if err != nil {
		return "", fmt.Errorf("failed to generate trace ID: %w", err)
	}

	// 生成 8 字节的 parent ID
	parentID := make([]byte, 8)
	_, err = cryptorand.Read(parentID)
	if err != nil {
		return "", fmt.Errorf("failed to generate parent ID: %w", err)
	}

	// 格式: version-traceid-parentid-flags
	// 00 是版本，01 是标志位（表示采样）
	return fmt.Sprintf("00-%s-%s-01",
		hex.EncodeToString(traceID),
		hex.EncodeToString(parentID)), nil
}
