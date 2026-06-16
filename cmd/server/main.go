package main

import (
	"OmniDesk/internal/agent"
	"OmniDesk/internal/db"
	"OmniDesk/internal/rag"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Reply     string `json:"reply"`
	MessageID uint   `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type FeedbackRequest struct {
	SessionID string `json:"session_id"`
	MessageID uint   `json:"message_id"`
	Rating    string `json:"rating"` // "up" or "down"
	Comments  string `json:"comments"`
}

// 简单的基于 SessionID 的限流器
var (
	visitors = make(map[string]*rate.Limiter)
	mu       sync.Mutex
)

func getVisitor(sessionID string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, exists := visitors[sessionID]
	if !exists {
		// 每分钟 20 次 = 每 3 秒 1 次
		limiter = rate.NewLimiter(rate.Every(3*time.Second), 20)
		visitors[sessionID] = limiter
	}
	return limiter
}

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			sessionID = c.ClientIP() // fallback
		}
		limiter := getVisitor(sessionID)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试 (1分钟内最多20次)"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("警告: 未设置 GEMINI_API_KEY，模型调用可能会失败。")
	}

	// 初始化数据库
	if err := db.InitDB(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("✅ SQLite 数据库初始化成功")

	// 2. 初始化本地 RAG 向量数据库
	if err := rag.InitRAG(apiKey); err != nil {
		log.Fatalf("RAG 向量库初始化失败: %v", err)
	}

	// 3. 注入测试的 FAQ 数据
	db.DB.FirstOrCreate(&db.FAQ{Question: "WiFi 密码是什么", Answer: "公司访客 WiFi 密码是：Welcome2026"})
	db.DB.FirstOrCreate(&db.FAQ{Question: "怎么请假", Answer: "请假请登录 OA 系统，在首页点击“我的考勤”->“休假申请”。"})
	
	// 4. 初始化多智能体系统
	deskAgent, err := agent.NewOmniDeskAgent(apiKey)
	if err != nil {
		log.Fatalf("初始化 OmniDesk 失败: %v", err)
	}

	// 2. 启动 HTTP 服务
	r := gin.Default()

	// 简单的 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Session-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.Use(rateLimitMiddleware())

	r.POST("/api/chat", func(c *gin.Context) {
		var req ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ChatResponse{Error: "参数错误: " + err.Error()})
			return
		}

		if req.SessionID == "" {
			c.JSON(http.StatusBadRequest, ChatResponse{Error: "session_id 不能为空"})
			return
		}

		// 存储用户提问
		userMsg := db.Message{
			SessionID: req.SessionID,
			Role:      "user",
			Content:   req.Message,
		}
		db.DB.Create(&userMsg)

		ctx := context.Background()
		response, err := deskAgent.Chat(ctx, req.SessionID, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ChatResponse{Error: "Agent 返回错误: " + err.Error()})
			return
		}

		// 存储 Agent 回复
		agentMsg := db.Message{
			SessionID: req.SessionID,
			Role:      "agent",
			Content:   response,
		}
		db.DB.Create(&agentMsg)

		c.JSON(http.StatusOK, ChatResponse{
			Reply:     response,
			MessageID: agentMsg.ID, // 返回 MessageID 给前端用于 feedback
		})
	})

	r.POST("/api/feedback", func(c *gin.Context) {
		var req FeedbackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		feedback := db.Feedback{
			SessionID: req.SessionID,
			MessageID: req.MessageID,
			Rating:    req.Rating,
			Comments:  req.Comments,
		}
		if err := db.DB.Create(&feedback).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存反馈失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "感谢您的反馈！"})
	})

	type ResolveRequest struct {
		SessionID string `json:"session_id"`
	}

	r.POST("/api/resolve", func(c *gin.Context) {
		var req ResolveRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		state := db.SessionState{
			SessionID:  req.SessionID,
			IsResolved: true,
		}
		
		// 保存或更新
		db.DB.Save(&state)
		
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "会话已标记为解决！"})
	})

	// 获取历史审计接口
	r.GET("/api/history/:session_id", func(c *gin.Context) {
		sessionID := c.Param("session_id")
		var messages []db.Message
		db.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages)
		c.JSON(http.StatusOK, messages)
	})

	fmt.Println("=====================================================")
	fmt.Println("🚀 OmniDesk API 已启动，监听端口 8082")
	fmt.Println("👉 后端接口: http://localhost:8082/api/chat")
	fmt.Println("=====================================================")

	if err := r.Run(":8082"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
