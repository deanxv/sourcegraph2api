package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v1.1.4"            // this hard coding will be replaced automatically when building, no need to manually change

type SGModelInfo struct {
	Model     string
	ModelRef  string
	MaxTokens int
}

// 创建映射表（假设用 model 名称作为 key）
var modelRegistry = map[string]SGModelInfo{
	"claude-sonnet-4-latest":              {"claude-sonnet-4-latest", "anthropic::2024-10-22::claude-sonnet-4-latest", 64000},
	"claude-sonnet-4-thinking-latest":     {"claude-sonnet-4-thinking-latest", "anthropic::2024-10-22::claude-sonnet-4-thinking-latest", 64000},
	"claude-3-7-sonnet-latest":            {"claude-3-7-sonnet-latest", "anthropic::2024-10-22::claude-3-7-sonnet-latest", 64000},
	"claude-3-7-sonnet-extended-thinking": {"claude-3-7-sonnet-extended-thinking", "anthropic::2024-10-22::claude-3-7-sonnet-extended-thinking", 64000},
	"claude-3-5-sonnet-latest":            {"claude-3-5-sonnet-latest", "anthropic::2024-10-22::claude-3-5-sonnet-latest", 64000},
	"claude-3-opus":                       {"claude-3-opus", "anthropic::2023-06-01::claude-3-opus", 64000},
	"claude-3-5-haiku-latest":             {"claude-3-5-haiku-latest", "anthropic::2024-10-22::claude-3-5-haiku-latest", 64000},
	"claude-3-haiku":                      {"claude-3-haiku", "anthropic::2023-06-01::claude-3-haiku", 64000},
	"claude-3.5-sonnet":                   {"claude-3.5-sonnet", "anthropic::2023-06-01::claude-3.5-sonnet", 64000},
	"claude-3-5-sonnet-20240620":          {"claude-3-5-sonnet-20240620", "anthropic::2023-06-01::claude-3-5-sonnet-20240620", 64000},
	"claude-3-sonnet":                     {"claude-3-sonnet", "anthropic::2023-06-01::claude-3-sonnet", 64000},
	"claude-2.1":                          {"claude-2.1", "anthropic::2023-01-01::claude-2.1", 64000},
	"claude-2.0":                          {"claude-2.0", "anthropic::2023-01-01::claude-2.0", 64000},
	"deepseek-v3":                         {"deepseek-v3", "fireworks::v1::deepseek-v3", 64000},
	"gemini-1.5-pro":                      {"gemini-1.5-pro", "google::v1::gemini-1.5-pro", 64000},
	"gemini-1.5-pro-002":                  {"gemini-1.5-pro-002", "google::v1::gemini-1.5-pro-002", 64000},
	"gemini-2.0-flash-exp":                {"gemini-2.0-flash-exp", "google::v1::gemini-2.0-flash-exp", 64000},
	"gemini-2.0-flash":                    {"gemini-2.0-flash", "google::v1::gemini-2.0-flash", 64000},
	"gemini-2.5-flash-preview-04-17":      {"gemini-2.5-flash-preview-04-17", "google::v1::gemini-2.5-flash-preview-04-17", 64000},
	"gemini-2.0-flash-lite":               {"gemini-2.0-flash-lite", "google::v1::gemini-2.0-flash-lite", 64000},
	"gemini-2.0-pro-exp-02-05":            {"gemini-2.0-pro-exp-02-05", "google::v1::gemini-2.0-pro-exp-02-05", 64000},
	"gemini-2.5-pro-preview-03-25":        {"gemini-2.5-pro-preview-03-25", "google::v1::gemini-2.5-pro-preview-03-25", 64000},
	"gemini-1.5-flash":                    {"gemini-1.5-flash", "google::v1::gemini-1.5-flash", 64000},
	"gemini-1.5-flash-002":                {"gemini-1.5-flash-002", "google::v1::gemini-1.5-flash-002", 64000},
	"mixtral-8x7b-instruct":               {"mixtral-8x7b-instruct", "mistral::v1::mixtral-8x7b-instruct", 64000},
	"mixtral-8x22b-instruct":              {"mixtral-8x22b-instruct", "mistral::v1::mixtral-8x22b-instruct", 64000},
	"gpt-4o":                              {"gpt-4o", "openai::2024-02-01::gpt-4o", 64000},
	"gpt-4.1":                             {"gpt-4.1", "openai::2024-02-01::gpt-4.1", 64000},
	"gpt-4o-mini":                         {"gpt-4o-mini", "openai::2024-02-01::gpt-4o-mini", 64000},
	"gpt-4.1-mini":                        {"gpt-4.1-mini", "openai::2024-02-01::gpt-4.1-mini", 64000},
	"gpt-4.1-nano":                        {"gpt-4.1-nano", "openai::2024-02-01::gpt-4.1-nano", 64000},
	"o3-mini-medium":                      {"o3-mini-medium", "openai::2024-02-01::o3-mini-medium", 64000},
	"o3":                                  {"o3", "openai::2024-02-01::o3", 64000},
	"o4-mini":                             {"o4-mini", "openai::2024-02-01::o4-mini", 64000},
	"o1":                                  {"o1", "openai::2024-02-01::o1", 64000},
	"gpt-4-turbo":                         {"gpt-4-turbo", "openai::2024-02-01::gpt-4-turbo", 64000},
	"gpt-3.5-turbo":                       {"gpt-3.5-turbo", "openai::2024-02-01::gpt-3.5-turbo", 64000},
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
