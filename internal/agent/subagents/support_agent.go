package subagents

import (
	"OmniDesk/internal/llm"
	"OmniDesk/internal/tools"
	"context"
	"fmt"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// SupportAgent 负责执行大模型推理与工具调用
type SupportAgent struct {
	runner *runner.Runner
}

func NewSupportAgent(apiKey string) (*SupportAgent, error) {
	modelAdapter, err := llm.NewGeminiADKAdapter(apiKey, "gemini-2.5-flash")
	if err != nil {
		return nil, fmt.Errorf("无法初始化 Gemini: %v", err)
	}

	sysPrompt := `你是 OmniDesk，企业内部智能客服系统。负责帮员工解答 HR 政策、IT 支持和报修等问题。

【基本原则 - 核心要求】
1. 态度友好、回答简洁、保持专业。
2. 绝对不能向用户许诺未包含在知识库或服务手册中的内容。
3. 遇到关于业务、网络(WiFi)、HR、报销、IT 等提问时，你必须且只能首先使用 search_knowledge 工具去搜索！
4. 只有当你调用过 search_knowledge 工具且真的没有找到结果时，才必须委婉地回答：“很抱歉，我们目前还不支持这部分功能，您的反馈我们已经收到，会进行跟进。” 严禁随意编造！
5. 只有当用户主动明确要求转人工服务/报修，或者明确硬件物理损坏时，才能主动调用 create_ticket 工具创建工单。
6. 你可以通过工具拿到当前上下文的 SessionID 并传给工单工具。`

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "OmniDesk_Support_Agent",
		Description: "通用解答专家，负责处理标准的用户求助与查询。",
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

	rn, err := runner.New(runner.Config{
		AppName:           "OmniDesk_Support",
		Agent:             llmAgent,
		SessionService:    session.InMemoryService(),
		ArtifactService:   artifact.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("Runner 创建失败: %v", err)
	}

	return &SupportAgent{runner: rn}, nil
}

// Check 执行深度模型推理
func (s *SupportAgent) Check(ctx context.Context, sessionID string, userInput string) (string, error) {
	enrichedInput := fmt.Sprintf("[当前会话SessionID: %s]\n用户提问: %s", sessionID, userInput)

	// 触发前默认沉睡 1 秒，缓解免费额度 API 的并发频率限制 (Rate Limit)
	time.Sleep(1 * time.Second)

	events := s.runner.Run(ctx, "user_1", sessionID, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: enrichedInput}},
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
