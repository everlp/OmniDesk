package agent

import (
	"OmniDesk/internal/llm"
	"OmniDesk/internal/tools"
	"context"
	"fmt"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// OmniDeskAgent 封装 ADK 运行环境
type OmniDeskAgent struct {
	runner *runner.Runner
}

// NewOmniDeskAgent 初始化智能客服 Agent
func NewOmniDeskAgent(apiKey string) (*OmniDeskAgent, error) {
	// 初始化模型适配器
	modelAdapter, err := llm.NewGeminiADKAdapter(apiKey, "gemini-2.5-flash")
	if err != nil {
		return nil, fmt.Errorf("无法初始化 Gemini: %v", err)
	}

	// 定义 System Prompt
	sysPrompt := `你是 OmniDesk，企业内部智能客服系统。负责帮员工解答 HR 政策、IT 支持和报修等问题。

【基本原则】
- 态度友好、回答简洁、保持专业。
- 如果用户询问你不知道的信息，必须优先使用 search_knowledge 工具进行查询。
- 如果知识库查询不到，建议使用 create_ticket 工具创建工单。`

	customerServiceAgent, err := llmagent.New(llmagent.Config{
		Name:        "OmniDesk_Agent",
		Description: "企业内部智能客服中心，能够回答文档问题并创建工单。",
		Model:       modelAdapter,
		Instruction: sysPrompt,
		Tools: []tool.Tool{
			tools.SearchKnowledgeTool(),
			tools.CreateTicketTool(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Agent 创建失败: %v", err)
	}

	// 配置运行器
	rn, err := runner.New(runner.Config{
		AppName:           "OmniDesk",
		Agent:             customerServiceAgent,
		SessionService:    session.InMemoryService(),
		ArtifactService:   artifact.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("Runner 创建失败: %v", err)
	}

	return &OmniDeskAgent{runner: rn}, nil
}

// Chat 允许在会话中持续对话
func (a *OmniDeskAgent) Chat(ctx context.Context, sessionID string, userInput string) (string, error) {
	events := a.runner.Run(ctx, "user_1", sessionID, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: userInput}},
	}, agent.RunConfig{})

	var finalResponse string
	for event, err := range events {
		if err != nil {
			return "", fmt.Errorf("ADK Run Error: %v", err)
		}
		if event.Content != nil {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					finalResponse = p.Text
				}
			}
		}
	}

	if finalResponse == "" {
		return "", fmt.Errorf("没有收到有效的回复内容")
	}

	return finalResponse, nil
}
