package subagents

import (
	"strings"
)

// GuardAgent 负责系统安全和风控过滤
type GuardAgent struct{}

func NewGuardAgent() *GuardAgent {
	return &GuardAgent{}
}

// Check 检查输入是否包含恶意词汇，如果包含则拦截并返回警告
func (g *GuardAgent) Check(userInput string) string {
	toxicWords := []string{"傻逼", "弱智", "去死", "废物", "垃圾", "智障"}
	for _, word := range toxicWords {
		if strings.Contains(userInput, word) {
			return "⚠️ 请注意您的回答预期和沟通用语。我们是一套企业专业系统，拒绝负面和辱骂词汇。如有具体问题，请友善描述。"
		}
	}
	return ""
}
