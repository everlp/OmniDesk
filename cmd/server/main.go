package main

import (
	"OmniDesk/internal/agent"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
}

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Println("警告: 未设置 GEMINI_API_KEY，模型调用可能会失败。")
	}

	// 1. 初始化 OmniDesk 客服 Agent
	deskAgent, err := agent.NewOmniDeskAgent(apiKey)
	if err != nil {
		log.Fatalf("初始化 OmniDesk 失败: %v", err)
	}

	// 2. 启动 HTTP 服务
	r := gin.Default()

	// 简单的 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

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

		ctx := context.Background()
		response, err := deskAgent.Chat(ctx, req.SessionID, req.Message)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ChatResponse{Error: "Agent 返回错误: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{Reply: response})
	})

	fmt.Println("=====================================================")
	fmt.Println("🚀 OmniDesk API 已启动，监听端口 8081")
	fmt.Println("👉 后端接口: http://localhost:8081/api/chat")
	fmt.Println("=====================================================")

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
