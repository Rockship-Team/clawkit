package main

import (
	"os"
	"strings"
)

// KnowledgeChunk is a financial knowledge entry for RAG-style retrieval.
type KnowledgeChunk struct {
	ID       string `json:"id"`
	Topic    string `json:"topic"`
	Category string `json:"category"`
	Level    string `json:"level"`
	Content  string `json:"content"`
	Tags     string `json:"tags"`
	Source   string `json:"source,omitempty"`
}

func loadKnowledge() []KnowledgeChunk {
	var kb []KnowledgeChunk
	readJSON(dataPath("knowledge-base.json"), &kb)
	return kb
}

func cmdKnowledge(args []string) {
	if len(args) == 0 {
		errOut("usage: knowledge search|list|get")
		os.Exit(1)
	}

	switch args[0] {
	case "search":
		if len(args) < 2 {
			errOut("usage: knowledge search <query>")
			os.Exit(1)
		}
		query := strings.ToLower(strings.Join(args[1:], " "))
		knowledgeSearch(query)

	case "list":
		category := ""
		if len(args) > 1 {
			category = args[1]
		}
		knowledgeList(category)

	case "get":
		if len(args) < 2 {
			errOut("usage: knowledge get <id>")
			os.Exit(1)
		}
		knowledgeGet(args[1])

	default:
		errOut("unknown knowledge command: " + args[0])
		os.Exit(1)
	}
}

func knowledgeSearch(query string) {
	kb := loadKnowledge()
	if len(kb) == 0 {
		errOut("no knowledge base found in data/knowledge-base.json")
		os.Exit(1)
	}

	words := strings.Fields(query)
	type scored struct {
		Chunk KnowledgeChunk `json:"chunk"`
		Score int            `json:"score"`
	}

	var results []scored
	for _, chunk := range kb {
		score := 0
		searchable := strings.ToLower(chunk.Topic + " " + chunk.Tags + " " + chunk.Content)
		for _, w := range words {
			if strings.Contains(searchable, w) {
				score++
			}
			// Bonus for topic/tag match
			if strings.Contains(strings.ToLower(chunk.Topic), w) {
				score += 2
			}
			if strings.Contains(strings.ToLower(chunk.Tags), w) {
				score++
			}
		}
		if score > 0 {
			results = append(results, scored{Chunk: chunk, Score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top 5
	if len(results) > 5 {
		results = results[:5]
	}

	var chunks []KnowledgeChunk
	for _, r := range results {
		chunks = append(chunks, r.Chunk)
	}

	okOut(map[string]interface{}{"results": chunks, "count": len(chunks), "query": query})
}

func knowledgeList(category string) {
	kb := loadKnowledge()
	if category == "" {
		// Group by category
		cats := map[string]int{}
		for _, chunk := range kb {
			cats[chunk.Category]++
		}
		okOut(map[string]interface{}{"categories": cats, "total": len(kb)})
		return
	}

	var filtered []KnowledgeChunk
	for _, chunk := range kb {
		if chunk.Category == category {
			filtered = append(filtered, chunk)
		}
	}
	okOut(map[string]interface{}{"chunks": filtered, "count": len(filtered), "category": category})
}

func knowledgeGet(id string) {
	kb := loadKnowledge()
	for _, chunk := range kb {
		if chunk.ID == id {
			okOut(map[string]interface{}{"chunk": chunk})
			return
		}
	}
	errOut("chunk not found: " + id)
	os.Exit(1)
}
