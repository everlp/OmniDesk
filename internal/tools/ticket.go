package tools

import (
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// CreateTicketTool 用于创建工单的工具
func CreateTicketTool() tool.Tool {
	t, err := functiontool.New(functiontool.Config{
		Name:        "create_ticket",
		Description: "当无法解决用户问题，或者用户要求转人工、硬件损坏需要报修时，调用此工具创建一个 IT/HR 支持工单。",
	}, func(ctx tool.Context, args struct {
		Category    string `json:"category" desc:"工单分类，如 'IT', 'HR', 'Admin'"`
		Description string `json:"description" desc:"用户遇到的具体问题描述"`
		Urgency     string `json:"urgency" desc:"紧急程度: 'Low', 'Medium', 'High'"`
	}) (string, error) {
		rand.Seed(time.Now().UnixNano())
		ticketID := fmt.Sprintf("TICKET-%06d", rand.Intn(1000000))
		
		response := fmt.Sprintf("✅ 工单 %s 已成功创建。\n分类: %s\n紧急程度: %s\n描述: %s\n我们的客服专员将尽快与用户联系。", 
			ticketID, args.Category, args.Urgency, args.Description)

		return response, nil
	})
	if err != nil {
		panic(fmt.Sprintf("初始化 CreateTicketTool 失败: %v", err))
	}
	return t
}
