package assurance

import (
	"context"
	"database/sql"
)

// Imported versions retain provenance. Activating one version retires its previous
// active version in the same transaction; a failed review changes neither version.
func prepareKnowledgeReview(ctx context.Context, tx *sql.Tx, id string, old, out Object, actor string) error {
	reviewed := dBool(out["reviewed"])
	delete(out, "reviewed")
	if old == nil || old["source_record"] == nil {
		return nil
	}
	out["source_record"] = old["source_record"]
	out["review_required"] = true
	out["review_status"] = "pending"
	if !dBool(out["enabled"]) {
		return nil
	}
	if !reviewed {
		return dError("公开资料启用前须确认已复核事实、适用条件与证据来源")
	}
	out["review_required"] = false
	out["review_status"] = "reviewed"
	out["reviewed_by"], out["reviewed_at"] = actor, Stamp()
	out["reviewed_content_hash"] = Digest(Object{"name": out["name"], "source": out["source"], "edition": out["edition"], "content": out["content"], "tags": out["tags"]})
	identity := dStr(dObj(old["source_record"])["source_identity"])
	if identity == "" {
		return dError("公开资料缺少来源身份，请重新导入")
	}
	rows, e := tx.QueryContext(ctx, "SELECT body FROM platform_resources WHERE kind='knowledge' AND id<>?", id)
	if e != nil {
		return e
	}
	previous := []Object{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return e
		}
		doc, e := DecodeJSON([]byte(raw))
		if e != nil {
			rows.Close()
			return e
		}
		if dBool(doc["enabled"]) && dObj(doc["source_record"])["source_identity"] == identity && doc["workspace_id"] == out["workspace_id"] {
			previous = append(previous, doc)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, doc := range previous {
		doc["enabled"], doc["review_status"], doc["superseded_by"], doc["updated_at"] = false, "superseded", id, Stamp()
		doc["revision"] = dInt(doc["revision"]) + 1
		raw, e := repoBody(doc)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE platform_resources SET body=? WHERE kind='knowledge' AND id=?", raw, doc["id"]); e != nil {
			return e
		}
		if e = platformEvent(ctx, tx, actor, "supersede-knowledge", dStr(doc["id"]), Object{"new_document_id": id, "source_identity": identity}); e != nil {
			return e
		}
	}
	return nil
}
