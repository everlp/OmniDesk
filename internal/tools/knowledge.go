package tools

import (
	"OmniDesk/internal/rag"
	"fmt"
	"os"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// SearchKnowledgeTool 基于纯 Go 本地向量数据库的 RAG 检索工具
func SearchKnowledgeTool() tool.Tool {
	t, err := functiontool.New(functiontool.Config{
		Name:        "search_knowledge",
		Description: "检索企业内部知识库。你可以传入完整的自然语言问题（如：‘WiFi 连不上怎么办？’）进行语义搜索。",
	}, func(ctx tool.Context, args struct {
		Query string `json:"query" desc:"用户的完整自然语言提问"`
	}) (string, error) {
		apiKey := os.Getenv("GEMINI_API_KEY")
		
		chunks, err := rag.Search(apiKey, args.Query, 3)
		if err != nil {
			return "", fmt.Errorf("RAG 检索失败: %v", err)
		}

		if len(chunks) == 0 {
			return "知识库中未找到与该查询相关的内容。", nil
		}

		var results []string
		for _, c := range chunks {
			results = append(results, c.Content)
		}

		return strings.Join(results, "\n\n-------------------\n\n"), nil
	})
	if err != nil {
		panic(fmt.Sprintf("初始化 SearchKnowledgeTool 失败: %v", err))
	}
	return t
}
