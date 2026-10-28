package assurance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

func knowledgeChunks(doc Object) []Object {
	content := []rune(dStr(doc["content"]))
	chunks := []Object{}
	hash := Digest(Object{"content": doc["content"], "edition": doc["edition"], "source": doc["source"]})
	for start := 0; start < len(content); start += 680 {
		end := start + 800
		if end > len(content) {
			end = len(content)
		}
		index := len(chunks)
		chunks = append(chunks, Object{"id": fmt.Sprintf("%s:%s:%d", doc["id"], hash[:12], index), "document_id": doc["id"], "name": doc["name"], "source": doc["source"], "edition": doc["edition"], "workspace_id": doc["workspace_id"], "content_hash": hash, "chunk": index + 1, "text": string(content[start:end])})
		if end == len(content) {
			break
		}
	}
	return chunks
}
func knowledgeIndexPublic(doc, index Object) Object {
	o := Clone(doc)
	o["chunks"] = len(knowledgeChunks(doc))
	o["index_state"] = "not_indexed"
	if index != nil {
		chunks := knowledgeChunks(doc)
		if len(chunks) > 0 && index["content_hash"] == chunks[0]["content_hash"] {
			o["index_state"] = "indexed"
		} else {
			o["index_state"] = "stale"
		}
		o["index_provider_id"], o["index_model"], o["indexed_at"] = index["provider_id"], index["model"], index["updated_at"]
	}
	return o
}
func (a *AIService) indexKnowledge(ctx context.Context, documentID, providerID string, allowExternal bool) (Object, error) {
	doc, e := a.repo.platformGet(ctx, "knowledge", documentID)
	if e != nil {
		return nil, e
	}
	if !dBool(doc["enabled"]) {
		return nil, dError("资料已停用")
	}
	p, e := a.taskProvider(ctx, providerID)
	if e != nil {
		return nil, e
	}
	if !dBool(p["enabled"]) {
		return nil, dError("向量服务已停用")
	}
	if p["network_scope"] == "external" && !allowExternal {
		return nil, dError("建立索引将向外部向量服务发送资料正文，请先明确允许")
	}
	chunks := knowledgeChunks(doc)
	vectors := [][]float64{}
	for start := 0; start < len(chunks); start += 16 {
		end := start + 16
		if end > len(chunks) {
			end = len(chunks)
		}
		texts := []string{}
		for _, c := range chunks[start:end] {
			texts = append(texts, dStr(c["text"]))
		}
		got, e := a.embed(ctx, p, texts)
		if e != nil {
			return nil, e
		}
		if len(vectors) > 0 && len(got) > 0 && len(vectors[0]) != len(got[0]) {
			return nil, dError("向量模型维数在索引期间发生变化")
		}
		vectors = append(vectors, got...)
	}
	tx, e := a.repo.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	current, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind='knowledge' AND id=?", documentID))
	if e != nil {
		return nil, e
	}
	if current["revision"] != doc["revision"] {
		return nil, dError("资料已修改，请基于新版本重新建立索引")
	}
	currentP, e := readObject(tx.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind='provider' AND id=?", providerID))
	if e != nil {
		return nil, e
	}
	if providerSignature(currentP) != providerSignature(p) {
		return nil, dError("向量服务配置已变化，请重新建立索引")
	}
	result := Object{"id": documentID, "provider_id": providerID, "model": dObj(p["models"])["embedding"], "signature": providerSignature(p), "content_hash": chunks[0]["content_hash"], "vectors": vectors, "updated_at": Stamp(), "dimensions": len(vectors[0])}
	raw, _ := repoBody(result)
	_, e = tx.ExecContext(ctx, "INSERT INTO platform_resources(kind,id,body) VALUES('knowledge_index',?,?) ON CONFLICT(kind,id) DO UPDATE SET body=excluded.body", documentID, raw)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	delete(result, "vectors")
	result["chunks"] = len(chunks)
	return result, nil
}
func retrievalTokens(text string) map[string]int {
	out := map[string]int{}
	r := []rune(strings.ToLower(text))
	word := []rune{}
	flush := func() {
		if len(word) > 0 {
			out[string(word)]++
			word = nil
		}
	}
	for i, c := range r {
		if unicode.Is(unicode.Han, c) {
			flush()
			out[string(c)]++
			if i+1 < len(r) && unicode.Is(unicode.Han, r[i+1]) {
				out[string([]rune{c, r[i+1]})]++
			}
		} else if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
			word = append(word, c)
		} else {
			flush()
		}
	}
	flush()
	return out
}
func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	dot, x, y := 0.0, 0.0, 0.0
	for i, v := range a {
		dot += v * b[i]
		x += v * v
		y += b[i] * b[i]
	}
	if x == 0 || y == 0 {
		return 0
	}
	return dot / math.Sqrt(x*y)
}
func decodeVectors(value any) [][]float64 {
	b, _ := json.Marshal(value)
	var out [][]float64
	json.Unmarshal(b, &out)
	return out
}
func (a *AIService) searchKnowledge(ctx context.Context, workspaceID, query string, documentIDs []string, providerID string, allowExternal bool, limit int) (Object, error) {
	if limit < 1 || limit > 20 {
		return nil, dError("检索条数须为 1–20")
	}
	if strings.TrimSpace(query) == "" || len([]rune(query)) > 4000 {
		return nil, dError("检索内容须为 1–4000 字")
	}
	docs, e := a.repo.platformList(ctx, "knowledge")
	if e != nil {
		return nil, e
	}
	wanted := map[string]bool{}
	for _, id := range documentIDs {
		wanted[id] = true
	}
	found := map[string]bool{}
	chunks := []Object{}
	vectors := map[string][]float64{}
	warnings := []any{}
	mode := "lexical"
	var provider Object
	var qVector []float64
	if providerID != "" {
		provider, e = a.taskProvider(ctx, providerID)
		if e != nil {
			return nil, e
		}
		if !dBool(provider["enabled"]) {
			return nil, dError("向量服务已停用")
		}
		if provider["network_scope"] == "external" && !allowExternal {
			return nil, dError("此检索未允许向外部向量服务发送查询")
		}
		v, e := a.embed(ctx, provider, []string{query})
		if e != nil {
			return nil, e
		}
		qVector = v[0]
	}
	for _, doc := range docs {
		scope := dStr(doc["workspace_id"])
		if !dBool(doc["enabled"]) || (scope != "" && scope != workspaceID) || (len(wanted) > 0 && !wanted[dStr(doc["id"])]) {
			continue
		}
		found[dStr(doc["id"])] = true
		parts := knowledgeChunks(doc)
		chunks = append(chunks, parts...)
		if provider != nil {
			index, e := a.repo.platformGet(ctx, "knowledge_index", dStr(doc["id"]))
			if e == nil && len(parts) > 0 && index["signature"] == providerSignature(provider) && index["content_hash"] == parts[0]["content_hash"] {
				v := decodeVectors(index["vectors"])
				if len(v) == len(parts) && len(v) > 0 && len(v[0]) == len(qVector) {
					for i, c := range parts {
						vectors[dStr(c["id"])] = v[i]
					}
				} else {
					warnings = append(warnings, doc["name"].(string)+"：向量维数或片段不匹配，使用关键词检索")
				}
			} else {
				warnings = append(warnings, dStr(doc["name"])+"：尚无当前模型的有效索引，使用关键词检索")
			}
		}
	}
	for id := range wanted {
		if !found[id] {
			return nil, dError("所选资料不存在、已停用或不属于当前项目")
		}
	}
	// Only explicitly selected project clauses enter retrieval; no historic private library is read.
	if workspaceID != "" && a.catalog != nil {
		ws, e := a.repo.Get(ctx, workspaceID)
		if e != nil {
			return nil, e
		}
		for _, key := range dList(ws["regulation_refs"]) {
			clause, e := a.catalog.Get(dStr(key))
			if e != nil || dBool(clause["withdrawn"]) {
				continue
			}
			doc := Object{"id": key, "name": clause["title"], "source": dObj(clause["facts"])["正文来源"], "edition": dObj(clause["facts"])["版本"], "content": clause["description"], "workspace_id": workspaceID}
			for _, c := range knowledgeChunks(doc) {
				c["regulation_key"] = key
				chunks = append(chunks, c)
			}
		}
	}
	if len(chunks) > 20000 {
		return nil, dError("检索范围超过 20,000 个片段，请选择具体资料范围")
	}
	queryTokens := retrievalTokens(query)
	tokens := make([]map[string]int, len(chunks))
	df := map[string]int{}
	avgLen := 0.0
	for i, c := range chunks {
		tokens[i] = retrievalTokens(dStr(c["name"]) + " " + dStr(c["text"]))
		avgLen += float64(len(tokens[i]))
		for key := range queryTokens {
			if tokens[i][key] > 0 {
				df[key]++
			}
		}
	}
	if len(chunks) > 0 {
		avgLen /= float64(len(chunks))
	}
	if avgLen == 0 {
		avgLen = 1
	}
	lexical := make([]float64, len(chunks))
	semantic := make([]float64, len(chunks))
	for i, c := range chunks {
		score := 0.0
		for key := range queryTokens {
			tf := float64(tokens[i][key])
			if tf == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(chunks)-df[key])+0.5)/(float64(df[key])+0.5))
			score += idf * tf * 2.2 / (tf + 1.2*(0.25+0.75*float64(len(tokens[i]))/avgLen))
		}
		lexical[i] = score
		if v := vectors[dStr(c["id"])]; v != nil {
			semantic[i] = cosine(qVector, v)
		}
	}
	rank := func(values []float64) map[int]int {
		ids := []int{}
		for i, v := range values {
			if v > 0 {
				ids = append(ids, i)
			}
		}
		sort.SliceStable(ids, func(i, j int) bool { return values[ids[i]] > values[ids[j]] })
		out := map[int]int{}
		for n, i := range ids {
			out[i] = n + 1
		}
		return out
	}
	lr, sr := rank(lexical), rank(semantic)
	if len(vectors) > 0 {
		mode = "hybrid"
	}
	scored := []Object{}
	for i, c := range chunks {
		score := 0.0
		if r := lr[i]; r > 0 {
			score += 1 / float64(60+r)
		}
		if r := sr[i]; r > 0 {
			score += 1 / float64(60+r)
		}
		if score == 0 {
			continue
		}
		o := Clone(c)
		o["score"] = score
		o["retrieval"] = mode
		scored = append(scored, o)
	}
	sort.SliceStable(scored, func(i, j int) bool { return numberFloat(scored[i]["score"]) > numberFloat(scored[j]["score"]) })
	if len(scored) > limit {
		scored = scored[:limit]
	}
	return Object{"items": scored, "mode": mode, "warnings": warnings, "candidate_chunks": len(chunks)}, nil
}
