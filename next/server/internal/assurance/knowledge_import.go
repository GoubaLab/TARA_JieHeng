package assurance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const knowledgePackSchema = "jieheng.knowledge-pack/1"
const knowledgeImportReceiptScope = "platform:knowledge-imports"

var knowledgeSourceID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
var knowledgeCVEID = regexp.MustCompile(`^CVE-[0-9]{4}-[0-9]{4,19}$`)
var publicSourceHost = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

// This validates a citation's public HTTPS format only. Import never resolves or fetches it.
func validatePublicKnowledgeURL(value string) error {
	u, e := url.Parse(value)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Opaque != "" || strings.ContainsAny(value, "\\ \t\r\n") {
		return dError("公开出处必须是无账号信息的 HTTPS 链接")
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return dError("公开出处端口无效")
		}
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return dError("公开出处不能使用本机或内网地址")
		}
		return nil
	}
	if len(host) > 253 || !publicSourceHost.MatchString(host) {
		return dError("公开出处必须使用完整公开域名或公开 IP 地址")
	}
	last := host[strings.LastIndex(host, ".")+1:]
	if strings.Trim(last, "0123456789") == "" {
		return dError("公开出处域名格式无效")
	}
	for _, suffix := range []string{"localhost", "local", "internal", "lan", "home", "test", "invalid"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return dError("公开出处不能使用本机、内网或无效域名")
		}
	}
	return nil
}

func knowledgePackFields(value Object, fields map[string]dField) (Object, error) {
	for key := range value {
		if _, exists := fields[key]; !exists {
			return nil, dError("知识包包含未知字段：" + key)
		}
	}
	out := Object{}
	for key, field := range fields {
		v, exists := value[key]
		if !exists {
			v = resolveDefault(field.fallback)
		}
		checked, e := dValidateField(field, v)
		if e != nil {
			return nil, dError("知识包字段 " + key + " 无效：" + e.Error())
		}
		out[key] = checked
	}
	return out, nil
}

func validateKnowledgePack(pack Object) (Object, error) {
	allowed := map[string]bool{"schema": true, "pack_id": true, "version": true, "entries": true, "metadata": true, "review_required": true}
	for key := range pack {
		if !allowed[key] {
			return nil, dError("知识包包含未知字段：" + key)
		}
	}
	if pack["schema"] != knowledgePackSchema {
		return nil, dError("知识包格式必须为 " + knowledgePackSchema)
	}
	out, e := knowledgePackFields(Object{"pack_id": pack["pack_id"], "version": pack["version"]}, map[string]dField{"pack_id": ds(1, 100), "version": ds(1, 200)})
	if e != nil {
		return nil, e
	}
	if !knowledgeSourceID.MatchString(dStr(out["pack_id"])) {
		return nil, dError("pack_id 仅允许字母、数字、点、下划线与连字符")
	}
	out["schema"] = knowledgePackSchema
	entries := dList(pack["entries"])
	if !libIsList(pack["entries"]) || len(entries) > 50 {
		return nil, dError("知识包 entries 必须为不超过 50 个条目的列表")
	}
	schema := map[string]dField{"id": ds(1, 100), "name": ds(1, 200), "source": ds(1, 1000), "edition": ds(1, 200), "content": ds(1, 120000), "tags": dl(ds(1, 80), 0, 30, false), "evidence_urls": dl(ds(1, 1000), 1, 30, true), "cve_ids": dl(ds(1, 28), 0, 50, false)}
	validated := []any{}
	seen := map[string]bool{}
	for i, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, dError(fmt.Sprintf("知识包条目 %d 必须为对象", i+1))
		}
		entry, e = knowledgePackFields(entry, schema)
		if e != nil {
			return nil, dError(fmt.Sprintf("知识包条目 %d：%s", i+1, e.Error()))
		}
		id := dStr(entry["id"])
		if !knowledgeSourceID.MatchString(id) || seen[id] {
			return nil, dError("条目标识无效或重复：" + id)
		}
		seen[id] = true
		if e = validatePublicKnowledgeURL(dStr(entry["source"])); e != nil {
			return nil, e
		}
		for _, v := range dList(entry["evidence_urls"]) {
			if e = validatePublicKnowledgeURL(dStr(v)); e != nil {
				return nil, e
			}
		}
		for _, v := range dList(entry["cve_ids"]) {
			if !knowledgeCVEID.MatchString(dStr(v)) {
				return nil, dError("CVE 标识格式无效：" + dStr(v))
			}
		}
		// These fields are sets: rearranging them should not create an artificial revision.
		for _, field := range []string{"tags", "evidence_urls", "cve_ids"} {
			items := dUnique(dList(entry[field]))
			sort.Slice(items, func(i, j int) bool { return dStr(items[i]) < dStr(items[j]) })
			entry[field] = items
		}
		validated = append(validated, entry)
	}
	sort.Slice(validated, func(i, j int) bool { return dStr(dObj(validated[i])["id"]) < dStr(dObj(validated[j])["id"]) })
	out["entries"] = validated
	metadata, notices := Object{}, []any{}
	if raw, exists := pack["metadata"]; exists {
		var ok bool
		metadata, ok = raw.(map[string]any)
		if !ok || metadata == nil {
			return nil, dError("知识包 metadata 必须为对象")
		}
	}
	if raw, exists := pack["review_required"]; exists {
		if !libIsList(raw) || len(dList(raw)) > 100 {
			return nil, dError("知识包 review_required 必须为不超过 100 项的列表")
		}
		notices = dList(raw)
	}
	if len(entries) == 0 && len(notices) == 0 {
		return nil, dError("知识包至少须包含一个资料条目或待复核通知")
	}
	// Collector notices are untrusted data, never deletion, activation or execution instructions.
	for _, extra := range []any{metadata, notices} {
		raw, e := CanonicalJSON(extra)
		if e != nil || len(raw) > 65536 {
			return nil, dError("知识包附加信息须为有效 JSON，且每部分不超过 64 KB")
		}
	}
	out["metadata"], out["review_required"] = metadata, notices
	return out, nil
}

// importKnowledgePack only creates disabled review candidates. Existing local edits and
// approved versions remain untouched; activation and supersession happen in the review transaction.
func (r *Repository) importKnowledgePack(ctx context.Context, pack Object, key, actor string) (Object, error) {
	if user, ok := ctx.Value(principalContextKey{}).(map[string]any); ok && user["role"] != "admin" {
		return nil, fail(403, "需要管理员权限", "组织级公开知识包由管理员导入")
	}
	if e := validateIdempotency(key); e != nil {
		return nil, e
	}
	checked, e := validateKnowledgePack(pack)
	if e != nil {
		return nil, e
	}
	fingerprint := Digest(checked)
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	receipt, e := readReceipt(ctx, tx, knowledgeImportReceiptScope, key, fingerprint)
	if e != nil || receipt != nil {
		return receipt, e
	}
	items := []any{}
	created, existing := 0, 0
	for _, rawEntry := range dList(checked["entries"]) {
		entry := dObj(rawEntry)
		identity := dStr(checked["pack_id"]) + ":" + dStr(entry["id"])
		entryFingerprint := Digest(entry)
		id := "knowledge_" + Digest(Object{"source_identity": identity, "pack_version": checked["version"], "entry_fingerprint": entryFingerprint})
		old, readErr := readObject(tx.QueryRowContext(ctx, "SELECT body FROM platform_resources WHERE kind='knowledge' AND id=?", id))
		wasCreated := false
		if readErr == nil {
			source := dObj(old["source_record"])
			if source["source_identity"] != identity || source["pack_version"] != checked["version"] || source["entry_fingerprint"] != entryFingerprint {
				return nil, &ConflictError{Detail: "已有知识记录与导入来源身份不一致，未覆盖现有资料"}
			}
			existing++
		} else if errors.Is(readErr, ErrNotFound) {
			now := Stamp()
			doc := Object{"id": id, "name": entry["name"], "source": entry["source"], "edition": entry["edition"], "content": entry["content"], "tags": entry["tags"], "workspace_id": "", "enabled": false, "revision": 0, "created_at": now, "updated_at": now, "review_required": true, "review_status": "pending"}
			doc["source_record"] = Object{"schema": knowledgePackSchema, "pack_id": checked["pack_id"], "pack_version": checked["version"], "entry_id": entry["id"], "source_identity": identity, "entry_fingerprint": entryFingerprint, "pack_fingerprint": fingerprint, "edition": entry["edition"], "source_url": entry["source"], "evidence_urls": entry["evidence_urls"], "cve_ids": entry["cve_ids"], "imported_at": now, "imported_by": actor}
			body, e := repoBody(doc)
			if e != nil {
				return nil, e
			}
			if _, e = tx.ExecContext(ctx, "INSERT INTO platform_resources(kind,id,body) VALUES('knowledge',?,?)", id, body); e != nil {
				return nil, e
			}
			created++
			wasCreated = true
		} else {
			return nil, readErr
		}
		items = append(items, Object{"document_id": id, "entry_id": entry["id"], "source_identity": identity, "created": wasCreated})
	}
	out := Object{"pack_id": checked["pack_id"], "version": checked["version"], "created_count": created, "existing_count": existing, "items": items, "metadata": checked["metadata"], "review_required": checked["review_required"]}
	if e = platformEvent(ctx, tx, actor, "import-knowledge-pack", dStr(checked["pack_id"]), Object{"version": checked["version"], "fingerprint": fingerprint, "created_count": created, "existing_count": existing, "items": items, "metadata": checked["metadata"], "review_required": checked["review_required"]}); e != nil {
		return nil, e
	}
	body, e := repoBody(out)
	if e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO receipts(workspace_id,request_id,fingerprint,body) VALUES(?,?,?,?)", knowledgeImportReceiptScope, key, fingerprint, body); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return out, nil
}
