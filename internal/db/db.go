package db

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Message 存储聊天记录
type Message struct {
	gorm.Model
	SessionID string `gorm:"index"`
	Role      string // "user" or "agent"
	Content   string
}

// Feedback 存储用户点赞踩与反馈
type Feedback struct {
	gorm.Model
	SessionID string `gorm:"index"`
	MessageID uint   // 关联具体消息ID
	Rating    string // "up" or "down"
	Comments  string // 具体建议
}

// Ticket 存储工单记录
type Ticket struct {
	gorm.Model
	TicketID    string `gorm:"uniqueIndex"`
	SessionID   string
	Category    string
	Description string
	Urgency     string
	Status      string // "Open", "Closed"
	ChatHistory string // 工单附带的近期聊天记录
}

// FAQ 存储高频问题
type FAQ struct {
	gorm.Model
	Question string `gorm:"uniqueIndex"`
	Answer   string
}

// SessionState 记录会话的完结状态
type SessionState struct {
	gorm.Model
	SessionID  string `gorm:"uniqueIndex"`
	IsResolved bool
}

var DB *gorm.DB

// InitDB 初始化并自动迁移 SQLite 数据库
func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("omnidesk.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("无法连接 SQLite 数据库: %v", err)
	}

	// 自动迁移 schemas
	err = DB.AutoMigrate(&Message{}, &Feedback{}, &Ticket{}, &FAQ{}, &SessionState{})
	if err != nil {
		return fmt.Errorf("无法自动迁移数据库表: %v", err)
	}

	log.Println("✅ SQLite 数据库初始化成功")
	
	// 初始化默认高频问题 (如果为空)
	initDefaultFAQs()

	return nil
}

func initDefaultFAQs() {
	var count int64
	DB.Model(&FAQ{}).Count(&count)
	if count == 0 {
		defaultFAQs := []FAQ{
			{Question: "连不上WiFi", Answer: "请确认您已连接到 'Corp_Guest' 或 'Corp_Staff'。如果是员工，请使用 OA 账号密码认证。如果一直失败，请尝试重启电脑或修改 DNS 为 114.114.114.114。"},
			{Question: "年假有几天", Answer: "根据公司 HR 政策，入职满一年可享受 5 天年假，满三年 10 天，满十年 15 天。具体请查阅 HR_Policy.md。"},
			{Question: "怎么报销", Answer: "所有报销均通过内部 ERP 系统提交。贴票要求：电子发票直接上传，纸质发票请邮寄给财务部。每月 25 日前提交当月报销。"},
		}
		DB.Create(&defaultFAQs)
		log.Println("✅ 注入默认高频 FAQ 数据")
	}
}
