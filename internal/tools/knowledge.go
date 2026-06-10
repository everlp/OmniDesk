package tools

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// SearchKnowledgeTool 一个简易的 RAG 检索工具，基于本地 Markdown 文件
func SearchKnowledgeTool() tool.Tool {
	t, err := functiontool.New(functiontool.Config{
		Name:        "search_knowledge",
		Description: "检索企业内部知识库，例如 HR 政策、IT 支持指南等。你可以提供关键词，比如 '年假', '打印机', 'WiFi' 等。",
	}, func(ctx tool.Context, args struct {
		Query string `json:"query" desc:"要检索的关键词"`
	}) (string, error) {
		files, err := filepath.Glob("data/knowledge/*.md")
		if err != nil {
			return "", fmt.Errorf("无法读取知识库目录: %v", err)
		}

		var results []string
		queryLower := strings.ToLower(args.Query)

		for _, file := range files {
			content, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}

			text := string(content)
			if strings.Contains(strings.ToLower(text), queryLower) {
				snippet := text
				if len(snippet) > 1000 {
					snippet = snippet[:1000] + "...(被截断)"
				}
				results = append(results, fmt.Sprintf("--- 知识库来源 [%s] ---\n%s\n", filepath.Base(file), snippet))
			}
		}

		if len(results) == 0 {
			return "知识库中未找到与该查询相关的内容。", nil
		}

		return strings.Join(results, "\n"), nil
	})
	if err != nil {
		panic(fmt.Sprintf("初始化 SearchKnowledgeTool 失败: %v", err))
	}
	return t
}
