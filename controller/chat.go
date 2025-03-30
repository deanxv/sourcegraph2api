package controller

import (
	"encoding/json"
	"fmt"
	"github.com/deanxv/CycleTLS/cycletls"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"sourcegraph2api/common"
	"sourcegraph2api/common/config"
	logger "sourcegraph2api/common/loggger"
	"sourcegraph2api/model"
	"sourcegraph2api/sourcegraphapi"
	"strings"
	"time"
)

const (
	responseIDFormat = "chatcmpl-%s"
)

// ChatForOpenAI @Summary OpenAI对话接口
// @Description OpenAI对话接口
// @Tags OpenAI
// @Accept json
// @Produce json
// @Param req body model.OpenAIChatCompletionRequest true "OpenAI对话请求"
// @Param Authorization header string true "Authorization API-KEY"
// @Router /v1/chat/completions [post]
func ChatForOpenAI(c *gin.Context) {
	client := cycletls.Init()
	defer safeClose(client)

	var openAIReq model.OpenAIChatCompletionRequest
	if err := c.BindJSON(&openAIReq); err != nil {
		logger.Errorf(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, model.OpenAIErrorResponse{
			OpenAIError: model.OpenAIError{
				Message: "Invalid request parameters",
				Type:    "request_error",
				Code:    "500",
			},
		})
		return
	}

	openAIReq.RemoveEmptyContentMessages()

	if openAIReq.Stream {
		handleStreamRequest(c, client, openAIReq)
	} else {
		handleNonStreamRequest(c, client, openAIReq)
	}
}

func handleNonStreamRequest(c *gin.Context, client cycletls.CycleTLS, openAIReq model.OpenAIChatCompletionRequest) {
	responseId := fmt.Sprintf(responseIDFormat, time.Now().Format("20060102150405"))
	ctx := c.Request.Context()
	cookieManager := config.NewCookieManager()
	maxRetries := len(cookieManager.Cookies)
	cookie, err := cookieManager.GetRandomCookie()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	for attempt := 0; attempt < maxRetries; attempt++ {
		requestBody, err := createRequestBody(c, &openAIReq)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to marshal request body"})
			return
		}
		sseChan, err := sourcegraphapi.MakeStreamChatRequest(c, client, jsonData, cookie)
		if err != nil {
			logger.Errorf(ctx, "MakeStreamChatRequest err on attempt %d: %v", attempt+1, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		isRateLimit := false
		var delta string
		var assistantMsgContent string
		var shouldContinue bool
		thinkStartType := new(bool) // 初始值为false
		for response := range sseChan {
			if response.Done {
				logger.Debugf(ctx, response.Data)
				return
			}

			data := response.Data
			if data == "" {
				continue
			}

			logger.Debug(ctx, strings.TrimSpace(data))

			switch {
			case common.IsCloudflareChallenge(data):
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cf challenge"})
				return
			case common.IsNotLogin(data):
				isRateLimit = true
				logger.Warnf(ctx, "Cookie Not Login, switching to next cookie, attempt %d/%d, COOKIE:%s", attempt+1, maxRetries, cookie)
				// 删除cookie
				//config.RemoveCookie(cookie)
				break
			}

			streamDelta, streamShouldContinue := processNoStreamData(c, data, responseId, openAIReq.Model, jsonData, thinkStartType)
			delta = streamDelta
			shouldContinue = streamShouldContinue
			// 处理事件流数据
			if !shouldContinue {
				promptTokens := model.CountTokenText(string(jsonData), openAIReq.Model)
				completionTokens := model.CountTokenText(assistantMsgContent, openAIReq.Model)
				finishReason := "stop"

				c.JSON(http.StatusOK, model.OpenAIChatCompletionResponse{
					ID:      fmt.Sprintf(responseIDFormat, time.Now().Format("20060102150405")),
					Object:  "chat.completion",
					Created: time.Now().Unix(),
					Model:   openAIReq.Model,
					Choices: []model.OpenAIChoice{{
						Message: model.OpenAIMessage{
							Role:    "assistant",
							Content: assistantMsgContent,
						},
						FinishReason: &finishReason,
					}},
					Usage: model.OpenAIUsage{
						PromptTokens:     promptTokens,
						CompletionTokens: completionTokens,
						TotalTokens:      promptTokens + completionTokens,
					},
				})

				return
			} else {
				//if strings.TrimSpace(delta) != "" {
				assistantMsgContent = assistantMsgContent + delta

				//}
			}
		}
		if !isRateLimit {
			return
		}

		// 获取下一个可用的cookie继续尝试
		cookie, err = cookieManager.GetNextCookie()
		if err != nil {
			logger.Errorf(ctx, "No more valid cookies available after attempt %d", attempt+1)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	}
	logger.Errorf(ctx, "All cookies exhausted after %d attempts", maxRetries)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "All cookies are temporarily unavailable."})
	return
}

func createRequestBody(c *gin.Context, req *model.OpenAIChatCompletionRequest) (map[string]interface{}, error) {
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {

		speaker := msg.Role
		if msg.Role == "user" {
			speaker = "human"
		}

		messages[i] = map[string]interface{}{
			"speaker": speaker,
			"text":    msg.Content,
		}
	}

	modelInfo, b := common.GetSGModelInfo(req.Model)
	if !b {
		return nil, fmt.Errorf("model %s not found", req.Model)
	}
	requestBody := map[string]interface{}{
		"model": modelInfo.ModelRef,
		//"stream":            req.Stream,
		"messages":          messages,
		"maxTokensToSample": 4000,
		"temperature":       0.2,
		"topP":              -1,
		"topK":              -1,
	}

	logger.Debug(c.Request.Context(), fmt.Sprintf("RequestBody: %v", requestBody))

	return requestBody, nil
}

// createStreamResponse 创建流式响应
func createStreamResponse(responseId, modelName string, jsonData []byte, delta model.OpenAIDelta, finishReason *string) model.OpenAIChatCompletionResponse {
	promptTokens := model.CountTokenText(string(jsonData), modelName)
	completionTokens := model.CountTokenText(delta.Content, modelName)
	return model.OpenAIChatCompletionResponse{
		ID:      responseId,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []model.OpenAIChoice{
			{
				Index:        0,
				Delta:        delta,
				FinishReason: finishReason,
			},
		},
		Usage: model.OpenAIUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

// handleDelta 处理消息字段增量
func handleDelta(c *gin.Context, delta string, responseId, modelName string, jsonData []byte) error {
	// 创建基础响应
	createResponse := func(content string) model.OpenAIChatCompletionResponse {
		return createStreamResponse(
			responseId,
			modelName,
			jsonData,
			model.OpenAIDelta{Content: content, Role: "assistant"},
			nil,
		)
	}

	// 发送基础事件
	var err error
	if err = sendSSEvent(c, createResponse(delta)); err != nil {
		return err
	}

	return err
}

// handleMessageResult 处理消息结果
func handleMessageResult(c *gin.Context, responseId, modelName string, jsonData []byte) bool {
	finishReason := "stop"
	var delta string

	streamResp := createStreamResponse(responseId, modelName, jsonData, model.OpenAIDelta{Content: delta, Role: "assistant"}, &finishReason)
	if err := sendSSEvent(c, streamResp); err != nil {
		logger.Warnf(c.Request.Context(), "sendSSEvent err: %v", err)
		return false
	}
	c.SSEvent("", " [DONE]")
	return false
}

// sendSSEvent 发送SSE事件
func sendSSEvent(c *gin.Context, response model.OpenAIChatCompletionResponse) error {
	jsonResp, err := json.Marshal(response)
	if err != nil {
		logger.Errorf(c.Request.Context(), "Failed to marshal response: %v", err)
		return err
	}
	c.SSEvent("", " "+string(jsonResp))
	c.Writer.Flush()
	return nil
}

func handleStreamRequest(c *gin.Context, client cycletls.CycleTLS, openAIReq model.OpenAIChatCompletionRequest) {

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	responseId := fmt.Sprintf(responseIDFormat, time.Now().Format("20060102150405"))
	ctx := c.Request.Context()
	cookieManager := config.NewCookieManager()
	maxRetries := len(cookieManager.Cookies)
	cookie, err := cookieManager.GetRandomCookie()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Stream(func(w io.Writer) bool {
		for attempt := 0; attempt < maxRetries; attempt++ {
			requestBody, err := createRequestBody(c, &openAIReq)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return false
			}

			jsonData, err := json.Marshal(requestBody)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to marshal request body"})
				return false
			}
			sseChan, err := sourcegraphapi.MakeStreamChatRequest(c, client, jsonData, cookie)
			if err != nil {
				logger.Errorf(ctx, "MakeStreamChatRequest err on attempt %d: %v", attempt+1, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return false
			}

			isRateLimit := false
			thinkStartType := new(bool) // 初始值为false
		SSELoop:
			for response := range sseChan {

				if response.Status == 400 {
					isRateLimit = true
					logger.Errorf(ctx, fmt.Sprintf("No permission to call this model:%s", openAIReq.Model))
					break SSELoop // 使用 label 跳出 SSE 循环
				}

				if response.Done {
					logger.Debugf(ctx, response.Data)
					return false
				}

				data := response.Data
				if data == "" {
					continue
				}

				logger.Debug(ctx, strings.TrimSpace(data))

				switch {
				case common.IsCloudflareChallenge(data):
					c.JSON(http.StatusInternalServerError, gin.H{"error": "cf challenge"})
					return false
				case common.IsRateLimit(data):
					isRateLimit = true
					logger.Warnf(ctx, "Cookie rate limited, switching to next cookie, attempt %d/%d, COOKIE:%s", attempt+1, maxRetries, cookie)
					config.AddRateLimitCookie(cookie, time.Now().Add(time.Duration(config.RateLimitCookieLockDuration)*time.Second))
					break SSELoop // 使用 label 跳出 SSE 循环
				case common.IsNotLogin(data):
					isRateLimit = true
					logger.Warnf(ctx, "Cookie Not Login, switching to next cookie, attempt %d/%d, COOKIE:%s", attempt+1, maxRetries, cookie)
					// 删除cookie
					//config.RemoveCookie(cookie)
					break SSELoop // 使用 label 跳出 SSE 循环
				}

				_, shouldContinue := processStreamData(c, data, responseId, openAIReq.Model, jsonData, thinkStartType)
				// 处理事件流数据

				if !shouldContinue {
					return false
				}
			}

			if !isRateLimit {
				return true
			}
			// 获取下一个可用的cookie继续尝试
			cookie, err = cookieManager.GetNextCookie()
			if err != nil {
				logger.Errorf(ctx, "No more valid cookies available after attempt %d", attempt+1)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return false
			}

		}

		logger.Errorf(ctx, "All cookies exhausted after %d attempts", maxRetries)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "All cookies are temporarily unavailable."})
		return false
	})
}

// 处理流式数据的辅助函数，返回bool表示是否继续处理
func processStreamData(c *gin.Context, data string, responseId, model string, jsonData []byte, thinkStartType *bool) (string, bool) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "data: ")

	if data == `{"stopReason":"end_turn"}` {
		handleMessageResult(c, responseId, model, jsonData)
		return "", false
	}

	if !strings.HasPrefix(data, "{\"deltaText\":") {
		return "", true
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		logger.Errorf(c.Request.Context(), "Failed to unmarshal event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return "", false
	}
	delta, ok := event["deltaText"].(string)
	if ok {
		if err := handleDelta(c, delta, responseId, model, jsonData); err != nil {
			logger.Errorf(c.Request.Context(), "handleDelta err: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return "", false
		}
		return delta, true
	}
	//delta, ok = event["reasoning_content"].(string)
	//if ok {
	//	if !*thinkStartType {
	//		delta = "<think>\n" + delta
	//		*thinkStartType = true
	//	}
	//	if err := handleDelta(c, delta, responseId, model, jsonData); err != nil {
	//		logger.Errorf(c.Request.Context(), "handleDelta err: %v", err)
	//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	//		return "", false
	//	}
	//	return delta, true
	//}
	//_, ok = event["thinking_time"].(float64)
	//if ok {
	//	delta = "\n</think>"
	//	if err := handleDelta(c, delta, responseId, model, jsonData); err != nil {
	//		logger.Errorf(c.Request.Context(), "handleDelta err: %v", err)
	//		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	//		return "", false
	//	}
	//	return delta, true
	//}

	return "", true

}

func processNoStreamData(c *gin.Context, data string, responseId, model string, jsonData []byte, thinkStartType *bool) (string, bool) {
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "data: ")

	if data == `{"stopReason":"end_turn"}` {
		return "", false
	}

	if !strings.HasPrefix(data, "{\"deltaText\":") {
		return "", true
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		logger.Errorf(c.Request.Context(), "Failed to unmarshal event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return "", false
	}
	delta, ok := event["deltaText"].(string)
	if ok {
		return delta, true
	}
	//delta, ok = event["reasoning_content"].(string)
	//if ok {
	//	if !*thinkStartType {
	//		delta = "<think>\n" + delta
	//		*thinkStartType = true
	//	}
	//	return delta, true
	//}
	//_, ok = event["thinking_time"].(float64)
	//if ok {
	//	delta = "\n</think>"
	//	return delta, true
	//}

	return "", true

}

func OpenaiModels(c *gin.Context) {
	var modelsResp []string

	modelsResp = common.GetSGModelList()

	var openaiModelListResponse model.OpenaiModelListResponse
	var openaiModelResponse []model.OpenaiModelResponse
	openaiModelListResponse.Object = "list"

	for _, modelResp := range modelsResp {
		openaiModelResponse = append(openaiModelResponse, model.OpenaiModelResponse{
			ID:     modelResp,
			Object: "model",
		})
	}
	openaiModelListResponse.Data = openaiModelResponse
	c.JSON(http.StatusOK, openaiModelListResponse)
	return
}

func safeClose(client cycletls.CycleTLS) {
	if client.ReqChan != nil {
		close(client.ReqChan)
	}
	if client.RespChan != nil {
		close(client.RespChan)
	}
}
