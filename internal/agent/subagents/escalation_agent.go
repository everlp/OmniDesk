package subagents

import (
	"OmniDesk/internal/db"
	"fmt"
)

// EscalationAgent 负责分析对话轮次，决定是否自动转交人工并拉起工单
type EscalationAgent struct{}

func NewEscalationAgent() *EscalationAgent {
	return &EscalationAgent{}
}

// Check 判断是否达到 10 轮交互，若是则中断循环，直接创建 DB Ticket
func (e *EscalationAgent) Check(sessionID string) string {
	var count int64
	db.DB.Model(&db.Message{}).Where("session_id = ?", sessionID).Count(&count)
	
	// 如果达到10轮（User和Agent各一条，计20条），检查是否已经创建过工单
	if count >= 20 {
		var ticketCount int64
		db.DB.Model(&db.Ticket{}).Where("session_id = ?", sessionID).Count(&ticketCount)
		if ticketCount == 0 {
			// 自动拦截并创建工单
			ticketID := fmt.Sprintf("ESCALATION-%s", sessionID[:5])
			
			// 检索该 session 的最近 10 条聊天记录作为工单上下文
			var msgs []db.Message
			db.DB.Where("session_id = ?", sessionID).Order("created_at desc").Limit(10).Find(&msgs)
			historyContext := ""
			for i := len(msgs) - 1; i >= 0; i-- {
				historyContext += fmt.Sprintf("[%s]: %s\n", msgs[i].Role, msgs[i].Content)
			}

			ticket := db.Ticket{
				TicketID:    ticketID,
				SessionID:   sessionID,
				Category:    "Escalation",
				Description: "自动转接：10轮对话仍未解决问题",
				Urgency:     "High",
				Status:      "Open",
				ChatHistory: historyContext,
			}
			
			if err := db.DB.Create(&ticket).Error; err != nil {
				return "由于多次对话未能解决您的问题，我尝试为您创建工单但失败了，请联系管理员。"
			}
			return fmt.Sprintf("抱歉，我们进行了多轮对话似乎仍未解决您的问题。我已经主动为您转交人工处理：✅ 工单 %s 已成功创建。我们的客服专员将尽快与您联系。", ticketID)
		}
	}
	return ""
}
