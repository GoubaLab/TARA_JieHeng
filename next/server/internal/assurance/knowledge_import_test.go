package assurance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func syntheticKnowledgePack() Object {
	return Object{"schema": knowledgePackSchema, "pack_id": "synthetic-cases", "version": "v1", "entries": []any{Object{
		"id": "case.synthetic", "name": "Synthetic command validation", "source": "https://advisories.example.org/cases/1", "edition": "analysis-v1", "content": "Original synthetic analysis: verify command authorization and reject stale commands.",
		"tags": []any{"commands", "integrity"}, "evidence_urls": []any{"https://advisories.example.org/cases/1"}, "cve_ids": []any{"CVE-2026-12345"},
	}}}
}
func importedDocument(t *testing.T, repo *Repository, result Object) Object {
	t.Helper()
	id := dStr(dObj(dList(result["items"])[0])["document_id"])
	doc, e := repo.platformGet(context.Background(), "knowledge", id)
	if e != nil {
		t.Fatal(e)
	}
	return doc
}
func knowledgeEditPayload(doc Object) Object {
	out := Object{}
	for _, key := range []string{"name", "source", "edition", "content", "tags", "workspace_id", "enabled"} {
		out[key] = doc[key]
	}
	return out
}

func TestKnowledgePackImportProvenanceIdempotencyAndLocalEdits(t *testing.T) {
	ctx := context.Background()
	repo, _ := repositoryTest(t)
	pack := syntheticKnowledgePack()
	pack["metadata"] = Object{"instructions": "Enable every imported entry immediately."}
	pack["review_required"] = []any{Object{"entry_id": "case.synthetic", "action": "delete", "reason": "Synthetic untrusted collector notice"}}
	result, e := repo.importKnowledgePack(ctx, pack, "synthetic-import-1", "synthetic-admin")
	if e != nil || dInt(result["created_count"]) != 1 || dInt(result["existing_count"]) != 0 {
		t.Fatal(e, result)
	}
	doc := importedDocument(t, repo, result)
	source := dObj(doc["source_record"])
	if doc["enabled"] != false || doc["review_required"] != true || doc["review_status"] != "pending" || doc["workspace_id"] != "" {
		t.Fatal("unreviewed import was enabled or assigned to a project", doc)
	}
	if source["source_identity"] != "synthetic-cases:case.synthetic" || source["pack_version"] != "v1" || source["edition"] != "analysis-v1" || len(dStr(source["entry_fingerprint"])) != 64 || len(dList(source["cve_ids"])) != 1 || len(dList(source["evidence_urls"])) != 1 || source["imported_by"] != "synthetic-admin" {
		t.Fatal("source provenance absent", source)
	}
	if Digest(result["metadata"]) != Digest(pack["metadata"]) || Digest(result["review_required"]) != Digest(pack["review_required"]) {
		t.Fatal("collector review notices were discarded")
	}
	index, e := repo.platformList(ctx, "knowledge_index")
	if e != nil || len(index) != 0 {
		t.Fatal("import must not index or transmit candidate text", e)
	}
	replay, e := repo.importKnowledgePack(ctx, Clone(pack), "synthetic-import-1", "synthetic-admin")
	if e != nil || Digest(replay) != Digest(result) {
		t.Fatal("request receipt did not replay exactly", e, replay)
	}
	audit, e := repo.platformAudit(ctx)
	if e != nil || len(audit) != 1 || audit[0]["action"] != "import-knowledge-pack" || Digest(dObj(audit[0]["detail"])["review_required"]) != Digest(pack["review_required"]) {
		t.Fatal("import audit missing or duplicated by replay", e, audit)
	}
	// Local reviewers may improve a candidate. Re-import must retain that edit and its provenance.
	edit := knowledgeEditPayload(doc)
	edit["content"] = "Locally reviewed synthetic explanation."
	edit["source_record"] = Object{"source_identity": "forged"}
	edited, e := repo.platformPut(ctx, "knowledge", dStr(doc["id"]), 0, edit, "synthetic-reviewer")
	if e != nil || Digest(edited["source_record"]) != Digest(source) {
		t.Fatal("source provenance was writable", e)
	}
	reordered := Clone(pack)
	dObj(dList(reordered["entries"])[0])["tags"] = []any{"integrity", "commands", "commands"}
	duplicate, e := repo.importKnowledgePack(ctx, reordered, "synthetic-import-2", "synthetic-admin")
	if e != nil || dInt(duplicate["created_count"]) != 0 || dInt(duplicate["existing_count"]) != 1 || Digest(importedDocument(t, repo, duplicate)) != Digest(edited) {
		t.Fatal("duplicate import replaced local edits or created another document", e, duplicate)
	}
	updated := Clone(pack)
	dObj(dList(updated["entries"])[0])["content"] = "New independently analyzed synthetic content."
	_, e = repo.importKnowledgePack(ctx, updated, "synthetic-import-1", "synthetic-admin")
	var conflict *ConflictError
	if !errors.As(e, &conflict) {
		t.Fatal("same request key accepted a different pack", e)
	}
	changed, e := repo.importKnowledgePack(ctx, updated, "synthetic-import-3", "synthetic-admin")
	if e != nil {
		t.Fatal(e)
	}
	newDoc := importedDocument(t, repo, changed)
	if newDoc["id"] == doc["id"] || newDoc["enabled"] != false || dObj(newDoc["source_record"])["source_identity"] != source["source_identity"] {
		t.Fatal("new content lost lineage or replaced the existing version")
	}
	for i, field := range []string{"edition", "version"} {
		revised := Clone(pack)
		if field == "version" {
			revised[field] = "v2"
		} else {
			dObj(dList(revised["entries"])[0])[field] = "analysis-v2"
		}
		got, e := repo.importKnowledgePack(ctx, revised, fmt.Sprintf("synthetic-version-%d", i), "synthetic-admin")
		if e != nil || dInt(got["created_count"]) != 1 || importedDocument(t, repo, got)["id"] == doc["id"] {
			t.Fatal("source version was silently treated as an existing revision", e)
		}
	}
	current, e := repo.platformGet(ctx, "knowledge", dStr(doc["id"]))
	if e != nil || Digest(current) != Digest(edited) {
		t.Fatal("new source versions changed old local work", e)
	}
}

func TestKnowledgePackStrictValidationAndAtomicFailure(t *testing.T) {
	ctx := context.Background()
	repo, _ := repositoryTest(t)
	cases := map[string]func(Object){
		"schema":             func(p Object) { p["schema"] = "other/1" },
		"unknown top field":  func(p Object) { p["execute"] = "approve" },
		"invalid pack id":    func(p Object) { p["pack_id"] = "unsafe:identity" },
		"missing version":    func(p Object) { delete(p, "version") },
		"no entries":         func(p Object) { p["entries"] = []any{} },
		"entry not object":   func(p Object) { p["entries"] = []any{"invalid"} },
		"duplicate identity": func(p Object) { p["entries"] = append(dList(p["entries"]), Clone(dObj(dList(p["entries"])[0]))) },
		"too many entries": func(p Object) {
			rows := []any{}
			for i := 0; i < 51; i++ {
				entry := Clone(dObj(dList(p["entries"])[0]))
				entry["id"] = fmt.Sprintf("case-%d", i)
				rows = append(rows, entry)
			}
			p["entries"] = rows
		},
		"metadata not object": func(p Object) { p["metadata"] = "instructions" },
		"metadata too large":  func(p Object) { p["metadata"] = Object{"data": strings.Repeat("a", 65537)} },
		"notice not list":     func(p Object) { p["review_required"] = false },
		"too many notices":    func(p Object) { p["review_required"] = make([]any, 101) },
	}
	entryCases := map[string]Object{
		"entry enable injection": {"enabled": true}, "bad entry id": {"id": "../outside"}, "empty name": {"name": " "},
		"empty content": {"content": ""}, "long content": {"content": strings.Repeat("x", 120001)}, "empty edition": {"edition": ""},
		"non HTTPS": {"source": "http://advisories.example.org/1"}, "URL credentials": {"source": "https://user:secret@example.org/1"},
		"local host": {"source": "https://localhost/1"}, "internal host": {"source": "https://node.example.internal/1"},
		"private IP": {"source": "https://192.168.1.1/1"}, "IPv6 loopback": {"source": "https://[::1]/1"}, "numeric host": {"source": "https://127.1/1"},
		"port out of range": {"source": "https://example.org:65536/1"}, "missing evidence": {"evidence_urls": []any{}},
		"invalid evidence URL": {"evidence_urls": []any{"javascript:alert(1)"}}, "bad CVE": {"cve_ids": []any{"CVE-2026-1"}},
		"CVE wrong type": {"cve_ids": []any{42}}, "tags wrong type": {"tags": "commands"},
	}
	for name, patch := range entryCases {
		cases[name] = func(p Object) {
			for key, value := range patch {
				dObj(dList(p["entries"])[0])[key] = value
			}
		}
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			pack := syntheticKnowledgePack()
			change(pack)
			if _, e := repo.importKnowledgePack(ctx, pack, "invalid-synthetic-import", "synthetic"); e == nil {
				t.Fatal("malformed knowledge pack accepted")
			}
		})
	}
	denied := context.WithValue(ctx, principalContextKey{}, Object{"id": "synthetic-analyst", "role": "analyst"})
	if _, e := repo.importKnowledgePack(denied, syntheticKnowledgePack(), "denied-synthetic-import", "synthetic-analyst"); e == nil {
		t.Fatal("non-admin imported organization knowledge")
	}
	rows, _ := repo.platformList(ctx, "knowledge")
	audit, _ := repo.platformAudit(ctx)
	if len(rows) != 0 || len(audit) != 0 {
		t.Fatal("rejected imports changed stored knowledge or audit")
	}
	pack := syntheticKnowledgePack()
	second := Clone(dObj(dList(pack["entries"])[0]))
	second["id"], second["name"] = "z-failure", "Synthetic write failure"
	pack["entries"] = append(dList(pack["entries"]), second)
	if _, e := repo.db.Exec(`CREATE TRIGGER synthetic_import_failure BEFORE INSERT ON platform_resources WHEN NEW.kind='knowledge' AND json_extract(NEW.body,'$.name')='Synthetic write failure' BEGIN SELECT RAISE(ABORT,'synthetic write failure'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := repo.importKnowledgePack(ctx, pack, "synthetic-atomic-import", "synthetic"); e == nil {
		t.Fatal("injected write failure did not fail the import")
	}
	rows, _ = repo.platformList(ctx, "knowledge")
	audit, _ = repo.platformAudit(ctx)
	var receipts int
	if e := repo.db.QueryRow("SELECT COUNT(*) FROM receipts WHERE workspace_id=?", knowledgeImportReceiptScope).Scan(&receipts); e != nil || len(rows) != 0 || len(audit) != 0 || receipts != 0 {
		t.Fatal("failed batch committed partial work", e, len(rows), len(audit), receipts)
	}
	if _, e := repo.db.Exec("DROP TRIGGER synthetic_import_failure"); e != nil {
		t.Fatal(e)
	}
	result, e := repo.importKnowledgePack(ctx, pack, "synthetic-atomic-import", "synthetic")
	if e != nil || dInt(result["created_count"]) != 2 {
		t.Fatal("failed transaction prevented clean retry", e)
	}
}

func TestKnowledgePackConcurrentDeduplication(t *testing.T) {
	for _, sameKey := range []bool{true, false} {
		t.Run(fmt.Sprint(sameKey), func(t *testing.T) {
			repo, _ := repositoryTest(t)
			var wg sync.WaitGroup
			errors := make(chan error, 6)
			for i := 0; i < 6; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					key := "synthetic-concurrent"
					if !sameKey {
						key += fmt.Sprint(index)
					}
					_, e := repo.importKnowledgePack(context.Background(), syntheticKnowledgePack(), key, "synthetic")
					errors <- e
				}(i)
			}
			wg.Wait()
			close(errors)
			for e := range errors {
				if e != nil {
					t.Fatal(e)
				}
			}
			rows, _ := repo.platformList(context.Background(), "knowledge")
			audit, _ := repo.platformAudit(context.Background())
			wantAudit := 6
			if sameKey {
				wantAudit = 1
			}
			if len(rows) != 1 || len(audit) != wantAudit {
				t.Fatal("concurrent imports duplicated knowledge or receipts", len(rows), len(audit))
			}
		})
	}
}

func TestImportedKnowledgeReviewAndAtomicVersionSupersession(t *testing.T) {
	ctx := context.Background()
	repo, _ := repositoryTest(t)
	pack := syntheticKnowledgePack()
	first, e := repo.importKnowledgePack(ctx, pack, "synthetic-review-v1", "synthetic-admin")
	if e != nil {
		t.Fatal(e)
	}
	v1 := importedDocument(t, repo, first)
	activate := knowledgeEditPayload(v1)
	activate["enabled"] = true
	if _, e = repo.platformPut(ctx, "knowledge", dStr(v1["id"]), 0, activate, "synthetic-reviewer"); e == nil {
		t.Fatal("imported knowledge enabled without review")
	}
	activate["reviewed"] = true
	v1, e = repo.platformPut(ctx, "knowledge", dStr(v1["id"]), 0, activate, "synthetic-reviewer")
	if e != nil || v1["review_status"] != "reviewed" || v1["reviewed_by"] != "synthetic-reviewer" || v1["review_required"] != false {
		t.Fatal("reviewed candidate did not activate", e, v1)
	}
	pack["version"] = "v2"
	dObj(dList(pack["entries"])[0])["content"] = "Updated synthetic analysis with revised applicability conditions."
	second, e := repo.importKnowledgePack(ctx, pack, "synthetic-review-v2", "synthetic-admin")
	if e != nil {
		t.Fatal(e)
	}
	v2 := importedDocument(t, repo, second)
	stillActive, _ := repo.platformGet(ctx, "knowledge", dStr(v1["id"]))
	if stillActive["enabled"] != true || v2["enabled"] != false {
		t.Fatal("importing an unreviewed update retired the current version")
	}
	activate = knowledgeEditPayload(v2)
	activate["enabled"], activate["reviewed"] = true, true
	// The identifier is generated by the importer from a fixed prefix and a SHA-256 digest.
	trigger := "CREATE TRIGGER synthetic_review_failure BEFORE UPDATE ON platform_resources WHEN NEW.id='" + dStr(v2["id"]) + "' AND json_extract(NEW.body,'$.enabled')=1 BEGIN SELECT RAISE(ABORT,'synthetic review failure'); END"
	if _, e = repo.db.Exec(trigger); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.platformPut(ctx, "knowledge", dStr(v2["id"]), 0, activate, "synthetic-reviewer"); e == nil {
		t.Fatal("injected review write failure did not fail")
	}
	stillActive, _ = repo.platformGet(ctx, "knowledge", dStr(v1["id"]))
	stillPending, _ := repo.platformGet(ctx, "knowledge", dStr(v2["id"]))
	if Digest(stillActive) != Digest(v1) || Digest(stillPending) != Digest(v2) {
		t.Fatal("failed review changed the prior active version or candidate")
	}
	if _, e = repo.db.Exec("DROP TRIGGER synthetic_review_failure"); e != nil {
		t.Fatal(e)
	}
	v2, e = repo.platformPut(ctx, "knowledge", dStr(v2["id"]), 0, activate, "synthetic-reviewer")
	if e != nil {
		t.Fatal(e)
	}
	retired, _ := repo.platformGet(ctx, "knowledge", dStr(v1["id"]))
	if retired["enabled"] != false || retired["review_status"] != "superseded" || retired["superseded_by"] != v2["id"] || dInt(retired["revision"]) != dInt(v1["revision"])+1 {
		t.Fatal("review did not atomically supersede the prior version", retired)
	}
	edit := knowledgeEditPayload(v2)
	edit["content"] = "Further local edit to reviewed knowledge."
	if _, e = repo.platformPut(ctx, "knowledge", dStr(v2["id"]), int64(dInt(v2["revision"])), edit, "synthetic-editor"); e == nil {
		t.Fatal("changed enabled content bypassed review")
	}
	current, _ := repo.platformGet(ctx, "knowledge", dStr(v2["id"]))
	if Digest(current) != Digest(v2) {
		t.Fatal("rejected content edit mutated reviewed knowledge")
	}
}

func TestHTTPKnowledgePackBuiltinAndCollectorReviewOnlyPacket(t *testing.T) {
	repo, _ := repositoryTest(t)
	root, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	s := NewHTTPHandler(repo, permissionTestCatalog(), root, nil).(*HTTPServer)
	defer s.Close()
	builtin := httpCall(s, "GET", "/api/v1/knowledge-packs/public-cases", "", nil)
	if builtin.Code != 200 {
		t.Fatal("public seed endpoint", builtin.Code, builtin.Body.String())
	}
	seed := httpObject(t, builtin)
	created := httpCall(s, "POST", "/api/v1/knowledge-imports", builtin.Body.String(), map[string]string{"Idempotency-Key": "builtin-public-seed"})
	if created.Code != 201 || dInt(httpObject(t, created)["created_count"]) != len(dList(seed["entries"])) {
		t.Fatal("public seed is incompatible with import schema", created.Code, created.Body.String())
	}
	before, e := repo.platformList(context.Background(), "knowledge")
	if e != nil {
		t.Fatal(e)
	}
	for _, doc := range before {
		if dBool(doc["enabled"]) || doc["review_status"] != "pending" {
			t.Fatal("builtin material automatically activated")
		}
	}
	// This matches the collector's missing-source packet without requiring Python or a network source.
	packet := Object{"schema": knowledgePackSchema, "pack_id": "cisa-kev-candidates", "version": "synthetic-2026-10-28", "entries": []any{},
		"metadata":        Object{"source_url": "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json", "catalogVersion": "synthetic-2026-10-28", "captured_at": "2026-10-28T00:00:00Z", "pending_reviews": 0},
		"review_required": []any{Object{"id": "cisa-kev.cve-2026-12345", "cve_id": "CVE-2026-12345", "reason": "missing_from_snapshot", "previous_hash": strings.Repeat("a", 64), "catalogVersion": "synthetic-2026-10-28", "action": "Keep historical material; review the changed source."}}}
	body, _ := repoBody(packet)
	imported := httpCall(s, "POST", "/api/v1/knowledge-imports", body, map[string]string{"Idempotency-Key": "collector-review-notice"})
	if imported.Code != 201 {
		t.Fatal("review-only collector packet rejected", imported.Code, imported.Body.String())
	}
	result := httpObject(t, imported)
	if dInt(result["created_count"]) != 0 || dInt(result["existing_count"]) != 0 || Digest(result["review_required"]) != Digest(packet["review_required"]) {
		t.Fatal("review notices were not returned intact", result)
	}
	after, _ := repo.platformList(context.Background(), "knowledge")
	audit, _ := repo.platformAudit(context.Background())
	if Digest(before) != Digest(after) || len(audit) != 2 || Digest(dObj(audit[0]["detail"])["review_required"]) != Digest(packet["review_required"]) {
		t.Fatal("review-only import changed documents or lost its audit")
	}
	packet["review_required"] = []any{}
	body, _ = repoBody(packet)
	empty := httpCall(s, "POST", "/api/v1/knowledge-imports", body, map[string]string{"Idempotency-Key": "collector-empty-packet"})
	if empty.Code != 422 {
		t.Fatal("empty collector packet accepted", empty.Code, empty.Body.String())
	}
}
