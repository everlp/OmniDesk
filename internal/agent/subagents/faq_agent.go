package subagents

import (
	"OmniDesk/internal/db"
	"fmt"
	"strings"
)

// FAQAgent 负责拦截高频问题，节约底层模型 Token
type FAQAgent struct{}

func NewFAQAgent() *FAQAgent {
	return &FAQAgent{}
}

// Check 如果是极短提问且匹配数据库 FAQ，直接短路返回
func (f *FAQAgent) Check(userInput string) string {
	if len(userInput) > 3 && len(userInput) < 20 {
		var faqs []db.FAQ
		db.DB.Find(&faqs)
		for _, faq := range faqs {
			// 简单的模糊匹配
			if strings.Contains(userInput, faq.Question) || strings.Contains(faq.Question, userInput) {
				return fmt.Sprintf("这是高频问题拦截：\n**Q: %s**\n**A:** %s\n\n如果这不是您想要的，请详细描述您的问题。", faq.Question, faq.Answer)
			}
		}
	}
	return ""
}
