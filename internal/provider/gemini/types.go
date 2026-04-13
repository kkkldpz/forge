package gemini

import "encoding/json"

// --- 请求类型 ---

// GenerateContentRequest 是 Gemini GenerateContent API 请求体。
type GenerateContentRequest struct {
	Contents          []Content       `json:"contents"`
	SystemInstruction *Content        `json:"systemInstruction,omitempty"`
	Tools             []ToolDecl      `json:"tools,omitempty"`
	ToolConfig        *ToolConfig     `json:"toolConfig,omitempty"`
	GenerationConfig  *GenConfig      `json:"generationConfig,omitempty"`
}

// Content 是 Gemini 格式的消息内容。
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part 是消息中的一个内容段。
type Part struct {
	Text          string           `json:"text,omitempty"`
	FunctionCall  *FunctionCall    `json:"functionCall,omitempty"`
	FunctionResp  *FunctionResp    `json:"functionResponse,omitempty"`
	Thought       bool             `json:"thought,omitempty"`
}

// FunctionCall 表示模型发起的函数调用。
type FunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// FunctionResp 表示函数调用的结果。
type FunctionResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

// ToolDecl 是 Gemini 的工具声明。
type ToolDecl struct {
	FunctionDecls []FuncDecl `json:"functionDeclarations"`
}

// FuncDecl 是单个函数的声明。
type FuncDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ToolConfig 控制函数调用的行为。
type ToolConfig struct {
	FunctionCallingConfig *FuncCallingConfig `json:"functionCallingConfig,omitempty"`
}

// FuncCallingConfig 指定函数调用模式和允许的函数列表。
type FuncCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

// GenConfig 控制生成行为的参数。
type GenConfig struct {
	Temperature      float64         `json:"temperature,omitempty"`
	MaxOutputTokens  int             `json:"maxOutputTokens,omitempty"`
	ThinkingConfig   *ThinkingConfig `json:"thinkingConfig,omitempty"`
}

// ThinkingConfig 控制 Gemini 的 thinking 行为。
type ThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

// --- 响应类型 ---

// GenerateContentResponse 是 Gemini GenerateContent API 的完整响应。
type GenerateContentResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	UsageMetadata  *UsageMetadata `json:"usageMetadata,omitempty"`
}

// Candidate 是模型生成的一个候选结果。
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason"`
}

// UsageMetadata 记录 Gemini API 调用的 token 用量。
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
