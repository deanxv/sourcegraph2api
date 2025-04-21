package sourcegraphapi

import (
	"fmt"
	"github.com/deanxv/CycleTLS/cycletls"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sourcegraph2api/common"
	"sourcegraph2api/common/config"
	logger "sourcegraph2api/common/loggger"
)

const (
	baseURL      = "https://sourcegraph.com"
	chatEndpoint = baseURL + "/.api/completions/stream?api-version=9&client-name=vscode&client-version=1.82.0"
)

func MakeStreamChatRequest(c *gin.Context, client cycletls.CycleTLS, jsonData []byte, cookie string) (<-chan cycletls.SSEResponse, error) {
	traceParent, err := common.GenerateTraceParent()
	if err != nil {
		logger.Errorf(c, "Failed to generate traceparent: %v", err)
		return nil, fmt.Errorf("failed to generate traceparent: %v", err)
	}

	options := cycletls.Options{
		Timeout: 10 * 60 * 60,
		Proxy:   config.ProxyUrl, // 在每个请求中设置代理
		Body:    string(jsonData),
		Method:  "POST",
		Headers: map[string]string{
			"accept-encoding":              "gzip;q=0",
			"authorization":                "token " + cookie,
			"connection":                   "keep-alive",
			"content-type":                 "application/json",
			"traceparent":                  traceParent,
			"user-agent":                   "vscode/1.86.0 (Node.js v20.18.3)",
			"x-requested-with":             "vscode 1.86.0",
			"x-sourcegraph-interaction-id": uuid.New().String(),
			"Host":                         "sourcegraph.com",
			"Transfer-Encoding":            "chunked",
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
