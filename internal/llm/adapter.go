package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// GeminiADKAdapter 桥接 Gemini Client 到 ADK 的 model.LLM 接口
type GeminiADKAdapter struct {
	client *genai.Client
	model  string
}

func NewGeminiADKAdapter(apiKey string, modelName string) (*GeminiADKAdapter, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}
	return &GeminiADKAdapter{client: client, model: modelName}, nil
}

func (a *GeminiADKAdapter) Name() string {
	return a.model
}

type declarableTool interface {
	Declaration() *genai.FunctionDeclaration
}

// GenerateContent 实现 ADK 的内容生成接口
func (a *GeminiADKAdapter) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		config := &genai.GenerateContentConfig{}

		// 1. 系统指令
		if req.Config != nil && req.Config.SystemInstruction != nil {
			config.SystemInstruction = req.Config.SystemInstruction
		}

		// 2. 工具定义
		if len(req.Tools) > 0 {
			var genaiTools []*genai.Tool
			gt := &genai.Tool{}
			for _, tAny := range req.Tools {
				if dt, ok := tAny.(declarableTool); ok {
					gt.FunctionDeclarations = append(gt.FunctionDeclarations, dt.Declaration())
				} else if gTool, ok := tAny.(*genai.Tool); ok {
					gt.FunctionDeclarations = append(gt.FunctionDeclarations, gTool.FunctionDeclarations...)
				}
			}
			if len(gt.FunctionDeclarations) > 0 {
				genaiTools = append(genaiTools, gt)
			}
			config.Tools = genaiTools
		}

		// 3. 组织对话历史
		var contents []*genai.Content
		for _, c := range req.Contents {
			// Convert to genai.Content
			// Usually ADK req.Contents are already *genai.Content, or close to it
			contents = append(contents, c)
		}

		// 4. 调用 Gemini 接口
		resp, err := a.client.Models.GenerateContent(ctx, a.model, contents, config)
		if err != nil {
			log.Printf("Gemini API Error: %v", err)
			yield(nil, err)
			return
		}

		// 5. 将响应传回 ADK 格式
		if len(resp.Candidates) == 0 {
			yield(nil, fmt.Errorf("Gemini returned no candidates"))
			return
		}

		candidate := resp.Candidates[0]
		if candidate.Content == nil {
			yield(nil, fmt.Errorf("Gemini returned empty content"))
			return
		}

		yield(&model.LLMResponse{Content: candidate.Content}, nil)
	}
}

func (a *GeminiADKAdapter) marshalArgs(args interface{}) string {
	b, _ := json.Marshal(args)
	return string(b)
}
