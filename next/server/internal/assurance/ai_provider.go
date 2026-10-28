package assurance

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

func platformModel(model string) map[string]dField {
	switch model {
	case "AIModels":
		return map[string]dField{"analysis": dsd("", 200), "vision": dsd("", 200), "embedding": dsd("", 200)}
	case "AIProvider":
		return map[string]dField{
			"name": ds(1, 120), "protocol": de("openai_chat", "openai_responses", "anthropic", "ollama"), "base_url": ds(1, 1000), "network_scope": de("internal", "external"), "models": dm("AIModels", true), "enabled": db(true, true), "priority": dnd("int", 10, 0, 1000, true), "timeout_seconds": dnd("int", 90, 5, 180, true), "max_output_tokens": dnd("int", 8192, 256, 32768, true), "retries": dnd("int", 1, 0, 2, true), "notes": dsd("", 2000), "api_key": dop(ds(0, 4096)), "clear_key": db(false, true), "input_price_per_million": dnd("number", 0, 0, 100000, true), "output_price_per_million": dnd("number", 0, 0, 100000, true), "currency": ded("USD", "USD", "CNY"), "generation_path": dsd("", 200), "embedding_path": dsd("", 200), "models_path": dsd("", 200), "chat_token_parameter": ded("max_completion_tokens", "max_completion_tokens", "max_tokens"),
		}
	case "AIPrompt":
		return map[string]dField{"name": ds(1, 120), "purpose": de("diagram", "analysis", "review"), "instructions": ds(1, 16000), "enabled": db(true, true)}
	case "KnowledgeDocument":
		return map[string]dField{"name": ds(1, 200), "source": ds(1, 1000), "edition": ds(1, 200), "content": ds(1, 120000), "workspace_id": dsd("", 100), "tags": dl(ds(1, 80), 0, 30, false), "enabled": db(true, true), "reviewed": db(false, true)}
	}
	return nil
}
func validateProvider(value Object) (Object, error) {
	p, e := dValidateModel("AIProvider", value)
	if e != nil {
		return nil, e
	}
	u, e := url.Parse(dStr(p["base_url"]))
	if e != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, dError("服务地址须为不含凭据、查询参数或片段的 HTTP(S) 地址")
	}
	if p["network_scope"] == "external" && u.Scheme != "https" {
		return nil, dError("外部模型服务必须使用 HTTPS")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && (ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) {
		return nil, dError("服务地址不能是未指定、组播或链路本地地址")
	}
	p["base_url"] = strings.TrimRight(u.String(), "/")
	for _, key := range []string{"generation_path", "embedding_path", "models_path"} {
		path := dStr(p[key])
		if strings.ContainsAny(path, "?#\\") || strings.Contains(path, "..") || strings.Contains(path, ":") {
			return nil, dError("接口路径必须为相对路径，不能包含查询参数或上级目录")
		}
	}
	if !dBool(dObj(p["models"])["analysis"]) && !dBool(dObj(p["models"])["vision"]) && !dBool(dObj(p["models"])["embedding"]) {
		return nil, dError("至少配置一个分析、视觉或向量模型")
	}
	if p["protocol"] == "anthropic" && dBool(dObj(p["models"])["embedding"]) {
		return nil, dError("Anthropic Messages 不提供向量接口，请单独配置向量服务")
	}
	return p, nil
}
func publicProvider(p Object) Object {
	o := publicResource(p)
	o["has_key"] = dBool(o["secret_ciphertext"])
	delete(o, "secret_ciphertext")
	delete(o, "api_key")
	delete(o, "clear_key")
	return o
}
func providerSignature(p Object) string {
	return Digest(Object{"id": p["id"], "base_url": p["base_url"], "protocol": p["protocol"], "model": dObj(p["models"])["embedding"], "path": p["embedding_path"]})
}
func (r *Repository) saveProvider(ctx context.Context, id string, revision int64, payload Object, actor string) (Object, error) {
	p, e := validateProvider(payload)
	if e != nil {
		return nil, e
	}
	if id == "" {
		id = Identity("provider")
	}
	if revision >= 0 {
		old, e := r.platformGet(ctx, "provider", id)
		if e != nil {
			return nil, e
		}
		if !dBool(p["clear_key"]) {
			p["secret_ciphertext"] = old["secret_ciphertext"]
		}
	}
	if dBool(p["clear_key"]) {
		delete(p, "secret_ciphertext")
	}
	if key := dStr(p["api_key"]); key != "" {
		if strings.ContainsAny(key, "\r\n") {
			return nil, dError("凭据不能包含换行")
		}
		sealed, e := r.encryptSecret(id, key)
		if e != nil {
			return nil, e
		}
		p["secret_ciphertext"] = sealed
	}
	delete(p, "api_key")
	delete(p, "clear_key")
	if p["secret_ciphertext"] == nil {
		delete(p, "secret_ciphertext")
	}
	out, e := r.platformPut(ctx, "provider", id, revision, p, actor)
	if e != nil {
		return nil, e
	}
	return publicProvider(out), nil
}

type modelInput struct{ System, Text, Image, Mime, Purpose string }
type modelResult struct {
	Text                      string
	InputTokens, OutputTokens int
	ProviderID, Model         string
}
type upstreamError struct {
	Status    int
	Retryable bool
	Message   string
}

func (e *upstreamError) Error() string { return e.Message }

type AIService struct {
	initError error
	repo      *Repository
	catalog   *Catalog
	mu        sync.Mutex
	health    map[string]Object
	cancels   map[string]context.CancelFunc
	slots     chan struct{}
	ctx       context.Context
	stop      context.CancelFunc
	wg        sync.WaitGroup
}

func NewAIService(repo *Repository, catalog *Catalog) *AIService {
	ctx, cancel := context.WithCancel(context.Background())
	a := &AIService{repo: repo, catalog: catalog, health: map[string]Object{}, cancels: map[string]context.CancelFunc{}, slots: make(chan struct{}, 2), ctx: ctx, stop: cancel}
	rows, _ := repo.platformList(ctx, "provider_health")
	for _, h := range rows {
		a.health[dStr(h["id"])] = h
	}
	if e := a.recoverJobs(); e != nil {
		a.initError = e
	}
	return a
}
func (a *AIService) Close() { a.stop(); a.wg.Wait() }
func providerURL(p Object, kind string) string {
	defaults := map[string]string{"generation": "chat/completions", "models": "models", "embedding": "embeddings"}
	if p["protocol"] == "openai_responses" {
		defaults["generation"] = "responses"
	}
	if p["protocol"] == "anthropic" {
		defaults["generation"] = "messages"
	}
	if p["protocol"] == "ollama" {
		defaults = map[string]string{"generation": "api/chat", "models": "api/tags", "embedding": "api/embed"}
	}
	path := dStr(p[kind+"_path"])
	if path == "" {
		path = defaults[kind]
	}
	return strings.TrimRight(dStr(p["base_url"]), "/") + "/" + strings.TrimLeft(path, "/")
}
func providerClient(p Object) *http.Client {
	scope := dStr(p["network_scope"])
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no address")
		}
		for _, v := range ips {
			ip := v.IP
			if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				return nil, fmt.Errorf("address disallowed")
			}
			if scope == "internal" && !ip.IsLoopback() && !ip.IsPrivate() {
				return nil, fmt.Errorf("internal endpoint resolved outside private network")
			}
			if scope == "external" && (ip.IsLoopback() || ip.IsPrivate()) {
				return nil, fmt.Errorf("external endpoint resolved to private network")
			}
		}
		var last error
		for _, v := range ips {
			conn, e := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(v.IP.String(), port))
			if e == nil {
				return conn, nil
			}
			last = e
		}
		return nil, last
	}
	return &http.Client{Timeout: time.Duration(dInt(p["timeout_seconds"])) * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil, DialContext: dial, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: time.Duration(dInt(p["timeout_seconds"])) * time.Second, DisableKeepAlives: true}}
}
func (a *AIService) upstream(ctx context.Context, p Object, kind, method string, body any) (Object, error) {
	var raw []byte
	var e error
	if body != nil {
		raw, e = json.Marshal(body)
		if e != nil {
			return nil, e
		}
	}
	req, e := http.NewRequestWithContext(ctx, method, providerURL(p, kind), bytes.NewReader(raw))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	secret, e := a.repo.decryptSecret(dStr(p["id"]), dStr(p["secret_ciphertext"]))
	if e != nil {
		return nil, e
	}
	if secret != "" {
		if p["protocol"] == "anthropic" {
			req.Header.Set("x-api-key", secret)
		} else {
			req.Header.Set("Authorization", "Bearer "+secret)
		}
	}
	if p["protocol"] == "anthropic" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	resp, e := providerClient(p).Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &upstreamError{0, true, "无法连接模型服务，请检查地址、内网范围、证书或超时设置"}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := fmt.Sprintf("模型服务返回 HTTP %d", resp.StatusCode)
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			message += "：请检查凭据和模型权限"
		}
		if resp.StatusCode == 429 {
			message += "：请求频率或额度受限"
		}
		return nil, &upstreamError{resp.StatusCode, resp.StatusCode == 429 || resp.StatusCode == 408 || resp.StatusCode >= 500, message}
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 4_000_001))
	if e != nil {
		return nil, &upstreamError{0, true, "模型响应读取失败"}
	}
	if len(data) > 4_000_000 {
		return nil, &upstreamError{0, false, "模型响应超过 4 MB 限制"}
	}
	out, e := DecodeJSON(data)
	if e != nil {
		return nil, &upstreamError{0, false, "模型服务未返回有效 JSON 对象"}
	}
	return out, nil
}
func generationBody(p Object, in modelInput) (Object, string) {
	model := dStr(dObj(p["models"])[in.Purpose])
	if model == "" {
		model = dStr(dObj(p["models"])["analysis"])
	}
	imageURL := "data:" + in.Mime + ";base64," + in.Image
	body := Object{"model": model, "stream": false}
	switch p["protocol"] {
	case "anthropic":
		content := []any{Object{"type": "text", "text": in.Text}}
		if in.Image != "" {
			content = append(content, Object{"type": "image", "source": Object{"type": "base64", "media_type": in.Mime, "data": in.Image}})
		}
		body["system"], body["max_tokens"], body["messages"] = in.System, p["max_output_tokens"], []any{Object{"role": "user", "content": content}}
	case "openai_responses":
		content := []any{Object{"type": "input_text", "text": in.Text}}
		if in.Image != "" {
			content = append(content, Object{"type": "input_image", "image_url": imageURL})
		}
		body["instructions"], body["input"], body["max_output_tokens"], body["store"] = in.System, []any{Object{"role": "user", "content": content}}, p["max_output_tokens"], false
	case "ollama":
		user := Object{"role": "user", "content": in.Text}
		if in.Image != "" {
			user["images"] = []any{in.Image}
		}
		body["messages"], body["format"], body["options"] = []any{Object{"role": "system", "content": in.System}, user}, "json", Object{"num_predict": p["max_output_tokens"]}
	default:
		var content any = in.Text
		if in.Image != "" {
			content = []any{Object{"type": "text", "text": in.Text}, Object{"type": "image_url", "image_url": Object{"url": imageURL}}}
		}
		tokenParam := dStr(p["chat_token_parameter"])
		if tokenParam == "" {
			tokenParam = "max_completion_tokens"
		}
		body["messages"], body[tokenParam] = []any{Object{"role": "system", "content": in.System}, Object{"role": "user", "content": content}}, p["max_output_tokens"]
	}
	return body, model
}
func parseGeneration(p, response Object) (modelResult, error) {
	out := modelResult{ProviderID: dStr(p["id"])}
	usage := dObj(response["usage"])
	out.InputTokens = dInt(usage["input_tokens"])
	out.OutputTokens = dInt(usage["output_tokens"])
	switch p["protocol"] {
	case "anthropic":
		for _, v := range dList(response["content"]) {
			b := dObj(v)
			if b["type"] == "text" {
				out.Text += dStr(b["text"])
			}
		}
		if response["stop_reason"] == "max_tokens" {
			return out, &upstreamError{0, false, "输出达到上限，请提高输出限制或缩小分析范围"}
		}
	case "openai_responses":
		for _, v := range dList(response["output"]) {
			for _, c := range dList(dObj(v)["content"]) {
				b := dObj(c)
				if b["type"] == "output_text" {
					out.Text += dStr(b["text"])
				}
			}
		}
		if response["status"] == "incomplete" {
			return out, &upstreamError{0, false, "模型输出不完整，请提高输出限制或缩小分析范围"}
		}
	case "ollama":
		out.Text = dStr(dObj(response["message"])["content"])
		out.InputTokens = dInt(response["prompt_eval_count"])
		out.OutputTokens = dInt(response["eval_count"])
		if response["done_reason"] == "length" {
			return out, &upstreamError{0, false, "模型输出达到长度限制"}
		}
	default:
		rows := dList(response["choices"])
		if len(rows) > 0 {
			row := dObj(rows[0])
			out.Text = dStr(dObj(row["message"])["content"])
			if row["finish_reason"] == "length" {
				return out, &upstreamError{0, false, "模型输出达到长度限制"}
			}
		}
		out.InputTokens = dInt(usage["prompt_tokens"])
		out.OutputTokens = dInt(usage["completion_tokens"])
	}
	if strings.TrimSpace(out.Text) == "" {
		return out, &upstreamError{0, false, "模型未返回可用文本，请核对协议、模型和视觉能力"}
	}
	return out, nil
}
func (a *AIService) providerHealth(id string) Object {
	a.mu.Lock()
	defer a.mu.Unlock()
	h := a.health[id]
	if h == nil {
		return Object{"id": id, "requests": 0, "consecutive_failures": 0, "state": "untested"}
	}
	out := Clone(h)
	out["state"] = "healthy"
	if dInt(h["consecutive_failures"]) > 0 {
		out["state"] = "degraded"
	}
	if until, e := time.Parse(time.RFC3339, dStr(h["open_until"])); e == nil && time.Now().Before(until) {
		out["state"] = "open"
	}
	return out
}
func (a *AIService) recordCall(p Object, result modelResult, elapsed time.Duration, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := dStr(p["id"])
	h := a.health[id]
	if h == nil {
		h = Object{"id": id}
	}
	h["requests"] = dInt(h["requests"]) + 1
	h["last_at"] = Stamp()
	h["latency_ms"] = elapsed.Milliseconds()
	h["input_tokens"] = dInt(h["input_tokens"]) + result.InputTokens
	h["output_tokens"] = dInt(h["output_tokens"]) + result.OutputTokens
	if err == nil {
		h["consecutive_failures"] = 0
		h["open_until"] = ""
		h["last_error"] = ""
	} else {
		h["failures"] = dInt(h["failures"]) + 1
		h["consecutive_failures"] = dInt(h["consecutive_failures"]) + 1
		h["last_error"] = err.Error()
		if dInt(h["consecutive_failures"]) >= 3 {
			h["open_until"] = time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
		}
	}
	h["estimated_cost"] = (float64(dInt(h["input_tokens"]))*numberFloat(p["input_price_per_million"]) + float64(dInt(h["output_tokens"]))*numberFloat(p["output_price_per_million"])) / 1e6
	h["currency"] = p["currency"]
	a.health[id] = h
	raw, _ := repoBody(h)
	a.repo.db.ExecContext(context.Background(), "INSERT INTO platform_resources(kind,id,body) VALUES('provider_health',?,?) ON CONFLICT(kind,id) DO UPDATE SET body=excluded.body", id, raw)
}
func numberFloat(v any) float64 {
	b, _ := json.Marshal(v)
	var f float64
	json.Unmarshal(b, &f)
	return f
}
func (a *AIService) generate(ctx context.Context, ids []string, in modelInput, allowExternal bool) (modelResult, []Object, error) {
	attempts := []Object{}
	var last error = dError("没有可用的模型服务")
	for _, id := range ids {
		p, e := a.taskProvider(ctx, id)
		if e != nil {
			return modelResult{}, attempts, e
		}
		if !dBool(p["enabled"]) {
			return modelResult{}, attempts, dError("所选模型服务已停用")
		}
		if p["network_scope"] == "external" && !allowExternal {
			return modelResult{}, attempts, dError("此任务未允许向外部模型发送数据")
		}
		if dStr(dObj(p["models"])[in.Purpose]) == "" {
			return modelResult{}, attempts, dError("所选服务未配置此次任务需要的模型")
		}
		if a.providerHealth(id)["state"] == "open" {
			attempts = append(attempts, Object{"provider_id": id, "status": "circuit_open"})
			last = dError("模型服务连续失败，熔断冷却中；可测试连接或稍后重试")
			continue
		}
		body, model := generationBody(p, in)
		for n := 0; n <= dInt(p["retries"]); n++ {
			start := time.Now()
			response, err := a.upstream(ctx, p, "generation", "POST", body)
			var result modelResult
			if err == nil {
				result, err = parseGeneration(p, response)
			}
			result.Model = model
			a.recordCall(p, result, time.Since(start), err)
			attempt := Object{"provider_id": id, "model": model, "attempt": n + 1, "latency_ms": time.Since(start).Milliseconds(), "status": "success"}
			if err != nil {
				attempt["status"] = "failed"
				attempt["error"] = err.Error()
			}
			attempts = append(attempts, attempt)
			if err == nil {
				return result, attempts, nil
			}
			last = err
			if ctx.Err() != nil {
				return modelResult{}, attempts, ctx.Err()
			}
			var up *upstreamError
			if !errors.As(err, &up) || !up.Retryable {
				return modelResult{}, attempts, err
			}
			if n < dInt(p["retries"]) {
				select {
				case <-ctx.Done():
					return modelResult{}, attempts, ctx.Err()
				case <-time.After(time.Duration(1<<n) * time.Second):
				}
			}
		}
	}
	return modelResult{}, attempts, last
}
func (a *AIService) modelList(ctx context.Context, p Object) ([]string, error) {
	resp, e := a.upstream(ctx, p, "models", "GET", nil)
	if e != nil {
		return nil, e
	}
	items := dList(resp["data"])
	field := "id"
	if p["protocol"] == "ollama" {
		items = dList(resp["models"])
		field = "name"
	}
	out := []string{}
	for _, v := range items {
		if id := dStr(dObj(v)[field]); id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, nil
}
func (a *AIService) embed(ctx context.Context, p Object, texts []string) ([][]float64, error) {
	model := dStr(dObj(p["models"])["embedding"])
	if model == "" {
		return nil, dError("未配置向量模型")
	}
	body := Object{"model": model, "input": texts}
	if p["protocol"] == "ollama" {
		body["truncate"] = false
	} else {
		body["encoding_format"] = "float"
	}
	response, e := a.upstream(ctx, p, "embedding", "POST", body)
	if e != nil {
		return nil, e
	}
	vectors := [][]float64{}
	if p["protocol"] == "ollama" {
		for _, v := range dList(response["embeddings"]) {
			data, _ := json.Marshal(v)
			var f []float64
			if json.Unmarshal(data, &f) != nil {
				return nil, dError("向量格式无效")
			}
			vectors = append(vectors, f)
		}
	} else {
		rows := dList(response["data"])
		sort.SliceStable(rows, func(i, j int) bool { return dInt(dObj(rows[i])["index"]) < dInt(dObj(rows[j])["index"]) })
		for i, v := range rows {
			if dInt(dObj(v)["index"]) != i {
				return nil, dError("向量响应索引不连续")
			}
			data, _ := json.Marshal(dObj(v)["embedding"])
			var f []float64
			if json.Unmarshal(data, &f) != nil {
				return nil, dError("向量格式无效")
			}
			vectors = append(vectors, f)
		}
	}
	if len(vectors) != len(texts) {
		return nil, dError("向量数量与文本数量不一致")
	}
	dims := 0
	for _, v := range vectors {
		if len(v) < 1 || len(v) > 8192 || (dims != 0 && len(v) != dims) {
			return nil, dError("向量维数无效或不一致")
		}
		dims = len(v)
		norm := 0.0
		for _, x := range v {
			if math.IsNaN(x) || math.IsInf(x, 0) {
				return nil, dError("向量包含非有限数值")
			}
			norm += x * x
		}
		if norm == 0 {
			return nil, dError("向量为零，无法建立语义索引")
		}
	}
	return vectors, nil
}

type pinnedProvidersKey struct{}

func (a *AIService) taskProvider(ctx context.Context, id string) (Object, error) {
	if providers, ok := ctx.Value(pinnedProvidersKey{}).(map[string]Object); ok {
		if p, exists := providers[id]; exists {
			return Clone(p), nil
		}
		return nil, dError("服务不在此次任务确认的路由中")
	}
	return a.repo.platformGet(ctx, "provider", id)
}
