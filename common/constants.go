package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v1.1.3"            // this hard coding will be replaced automatically when building, no need to manually change

type SGModelInfo struct {
	Model     string
	ModelRef  string
	MaxTokens int
}

// 创建映射表（假设用 model 名称作为 key）
var modelRegistry = map[string]SGModelInfo{
	"o4-mini":                             {"o4-mini", "openai::2024-02-01::gpt-4o-mini", 64000},
	"claude-3-7-sonnet-latest":            {"claude-3-7-sonnet", "anthropic::2024-10-22::claude-3-7-sonnet-latest", 64000},
	"claude-3-7-sonnet-extended-thinking": {"claude-3-7-sonnet-extended-thinking", "anthropic::2024-10-22::claude-3-7-sonnet-extended-thinking", 64000},
	"claude-3-5-sonnet-latest":            {"claude-3-5-sonnet-latest", "anthropic::2024-10-22::claude-3-5-sonnet-latest", 64000},
	"gpt-4o":                              {"gpt-4o", "openai::2024-02-01::gpt-4o", 64000},
	"gpt-4.1":                             {"gpt-4.1", "openai::2024-02-01::gpt-4.1", 64000},
	"o3":                                  {"o3", "openai::2024-02-01::o3", 64000},
	"gemini-1.5-pro":                      {"gemini-1.5-pro", "google::v1::gemini-1.5-pro", 64000},
	"gemini-2.5-pro-preview-03-25":        {"gemini-2.5-pro-preview-03-25", "google::v1::gemini-2.5-pro-preview-03-25", 64000},
	"claude-3-5-haiku-latest":             {"claude-3-5-haiku-latest", "anthropic::2024-10-22::claude-3-5-haiku-latest", 64000},
	"gemini-2.0-flash-exp":                {"gemini-2.0-flash-exp", "google::v1::gemini-2.0-flash-exp", 64000},
	"gemini-2.0-flash-lite":               {"gemini-2.0-flash-lite", "google::v1::gemini-2.0-flash-lite", 64000},
	"gpt-4o-mini":                         {"gpt-4o-mini", "openai::2024-02-01::gpt-4o-mini", 64000},
	"gpt-4.1-mini":                        {"gpt-4.1-mini", "openai::2024-02-01::gpt-4.1-mini", 64000},
	"gpt-4.1-nano":                        {"gpt-4.1-nano", "openai::2024-02-01::gpt-4.1-nano", 64000},
}

// 通过 model 名称查询的方法
func GetSGModelInfo(modelName string) (SGModelInfo, bool) {
	info, exists := modelRegistry[modelName]
	return info, exists
}

func GetSGModelList() []string {
	var modelList []string
	for k := range modelRegistry {
		modelList = append(modelList, k)
	}
	return modelList
}
