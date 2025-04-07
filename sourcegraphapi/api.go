package sourcegraphapi

import (
	"fmt"
	"github.com/deanxv/CycleTLS/cycletls"
	"github.com/gin-gonic/gin"
	"sourcegraph2api/common/config"
	logger "sourcegraph2api/common/loggger"
)

const (
	baseURL      = "https://sourcegraph.com"
	chatEndpoint = baseURL + "/.api/completions/stream?api-version=2&client-name=web&client-version=0.0.1"
)

func MakeStreamChatRequest(c *gin.Context, client cycletls.CycleTLS, jsonData []byte, cookie string) (<-chan cycletls.SSEResponse, error) {

	options := cycletls.Options{
		Timeout: 10 * 60 * 60,
		Proxy:   config.ProxyUrl, // 在每个请求中设置代理
		Body:    string(jsonData),
		Method:  "POST",
		Headers: map[string]string{
			"Content-Type":     "application/json; charset=utf-8",
			"Accept":           "text/event-stream",
			"Origin":           baseURL,
			"Referer":          baseURL + `/.assets/_sk/_app/immutable/workers/agent.worker-5ySyKmZ8.js`,
			"Cookie":           cookie,
			"x-requested-with": "Sourcegraph",
			"User-Agent":       config.UserAgent,
		},
	}

	logger.Debug(c.Request.Context(), fmt.Sprintf("cookie: %v", cookie))

	sseChan, err := client.DoSSE(chatEndpoint, options, "POST")
	if err != nil {
		logger.Errorf(c, "Failed to make stream request: %v", err)
		return nil, fmt.Errorf("Failed to make stream request: %v", err)
	}
	return sseChan, nil
}
