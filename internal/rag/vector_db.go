package rag

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"strings"
)

type Chunk struct {
	ID      string    `json:"id"`
	File    string    `json:"file"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
	Vector  []float32 `json:"vector"`
}

type VectorStore struct {
	Chunks []Chunk `json:"chunks"`
}

var DB *VectorStore
var DBPath = "data/vector_store.json"

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Search 检索最相关的 TopK 分块
func Search(apiKey string, query string, topK int) ([]Chunk, error) {
	if DB == nil || len(DB.Chunks) == 0 {
		return nil, fmt.Errorf("知识库为空")
	}

	queryVector, err := GetEmbedding(apiKey, query)
	if err != nil {
		return nil, err
	}

	type ScoredChunk struct {
		Score float32
		Chunk Chunk
	}

	var scored []ScoredChunk
	for _, chunk := range DB.Chunks {
		score := cosineSimilarity(queryVector, chunk.Vector)
		scored = append(scored, ScoredChunk{Score: score, Chunk: chunk})
	}

	// 降序排列
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	var results []Chunk
	for i := 0; i < topK && i < len(scored); i++ {
		// 设定一个相似度阈值，过滤掉完全不相干的
		if scored[i].Score > 0.4 {
			results = append(results, scored[i].Chunk)
		}
	}
	return results, nil
}

// InitRAG 系统启动时扫描文档并自动生成向量
func InitRAG(apiKey string) error {
	DB = &VectorStore{Chunks: []Chunk{}}
	
	// 如果由于某种原因不需要每次启动都生成，可以写逻辑读取 JSON，但为了热更新，我们直接扫描并重建
	// 生产环境可以对比文件修改时间来做增量更新
	
	files, err := filepath.Glob("data/knowledge/*.md")
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		text := string(content)
		// 按二级标题拆分 Chunk
		sections := strings.Split(text, "\n## ")
		fileBase := filepath.Base(file)

		for i, section := range sections {
			if strings.TrimSpace(section) == "" {
				continue
			}

			// 如果是第一段（即包含 # 的开头），去除 # 号
			if i == 0 {
				section = strings.Replace(section, "# ", "", 1)
			} else {
				section = "## " + section // 补回被 split 砍掉的符号
			}

			// 提取标题，简单处理取第一行
			lines := strings.Split(section, "\n")
			title := strings.TrimSpace(lines[0])

			// 组装文本块进行向量化
			chunkContent := fmt.Sprintf("【文件来源: %s】\n%s", fileBase, strings.TrimSpace(section))
			
			vec, err := GetEmbedding(apiKey, chunkContent)
			if err != nil {
				fmt.Printf("⚠️ 警告：文档 %s 向量化失败: %v\n", title, err)
				continue
			}

			DB.Chunks = append(DB.Chunks, Chunk{
				ID:      fmt.Sprintf("%s-%d", fileBase, i),
				File:    fileBase,
				Title:   title,
				Content: chunkContent,
				Vector:  vec,
			})
		}
	}

	// 保存到磁盘
	jsonData, _ := json.MarshalIndent(DB, "", "  ")
	_ = ioutil.WriteFile(DBPath, jsonData, 0644)

	fmt.Printf("✅ 本地 RAG 向量库初始化完成，共加载 %d 个知识分块\n", len(DB.Chunks))
	return nil
}
