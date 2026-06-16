package agent

import (
	"OmniDesk/internal/agent/subagents"
	"context"
	"fmt"
)

// Orchestrator 负责编排各个子 Agent
type Orchestrator struct {
	guardAgent      *subagents.GuardAgent
	escalationAgent *subagents.EscalationAgent
	faqAgent        *subagents.FAQAgent
	supportAgent    *subagents.SupportAgent
}

// NewOmniDeskAgent 初始化智能客服编排引擎
func NewOmniDeskAgent(apiKey string) (*Orchestrator, error) {
	supportAgent, err := subagents.NewSupportAgent(apiKey)
	if err != nil {
		return nil, fmt.Errorf("SupportAgent 初始化失败: %v", err)
	}

	return &Orchestrator{
		guardAgent:      subagents.NewGuardAgent(),
		escalationAgent: subagents.NewEscalationAgent(),
		faqAgent:        subagents.NewFAQAgent(),
		supportAgent:    supportAgent,
	}, nil
}

// Chat 流水线调度各个 Agent
func (o *Orchestrator) Chat(ctx context.Context, sessionID string, userInput string) (string, error) {
	// 1. GuardAgent: 防骂/消极情绪检测拦截
	if blockMsg := o.guardAgent.Check(userInput); blockMsg != "" {
		return blockMsg, nil
	}

	// 2. FAQAgent: 高频问题拦截 (短路后续处理)
	if faqMsg := o.faqAgent.Check(userInput); faqMsg != "" {
		return faqMsg, nil
	}

	// 3. EscalationAgent: 多轮拦截判定 (兜底强转工单)
	if escalateMsg := o.escalationAgent.Check(sessionID); escalateMsg != "" {
		return escalateMsg, nil
	}

	// 4. SupportAgent: 执行耗时的 LLM 深度推理
	response, err := o.supportAgent.Check(ctx, sessionID, userInput)
	if err != nil {
		return "", err
	}

	return response, nil
}
