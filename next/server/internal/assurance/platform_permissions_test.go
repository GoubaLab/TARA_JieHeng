package assurance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// This catalog is entirely synthetic; these tests never load historic or published libraries.
func permissionTestCatalog() *Catalog {
	return &Catalog{Regulations: &Regulations{
		byDocument: map[string]Object{"synthetic": {"id": "synthetic", "code": "SYNTHETIC", "edition": "v1", "source_url": "https://example.invalid/synthetic-v1"}},
		byKey:      map[string]Object{"reg:synthetic:1": {"key": "reg:synthetic:1", "document_id": "synthetic", "number": "1", "body": "Synthetic authenticated commands must be verified."}},
	}}
}

func TestEnterpriseKnowledgeIndexOwnershipAndQueuedScope(t *testing.T) {
	ctx := context.Background()
	repo, _ := repositoryTest(t)
	if e := repo.BootstrapAdmin(ctx, "administrator", "Synthetic-Password-123"); e != nil {
		t.Fatal(e)
	}
	h, e := NewEnterpriseHTTPHandler(repo, permissionTestCatalog(), t.TempDir(), nil, "https://tara.example.internal")
	if e != nil {
		t.Fatal(e)
	}
	s := h.(*HTTPServer)
	defer s.Close()
	login := func(name string) *http.Cookie {
		w := enterpriseCall(h, "POST", "/auth/sessions", `{"username":"`+name+`","password":"Synthetic-Password-123"}`, nil, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		return w.Result().Cookies()[0]
	}
	admin := login("administrator")
	w := enterpriseCall(h, "POST", "/enterprise/users", `{"username":"analyst","name":"Synthetic analyst","role":"analyst","enabled":true,"password":"Synthetic-Password-123"}`, admin, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	analystID := dStr(httpObject(t, w)["id"])
	analyst := login("analyst")
	w = enterpriseCall(h, "POST", "/workspaces", `{"title":"Synthetic index permissions"}`, admin, map[string]string{"Idempotency-Key": "index-permission-workspace"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	wid := dStr(httpObject(t, w)["id"])
	w = enterpriseCall(h, "PUT", "/workspaces/"+wid+"/members/"+analystID, `{"role":"analyst"}`, admin, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}))
	defer upstream.Close()
	provider, e := repo.saveProvider(ctx, "", -1, providerFixture(upstream.URL, "openai_chat"), "synthetic")
	if e != nil {
		t.Fatal(e)
	}
	createDocument := func(scope, key string) Object {
		body, _ := repoBody(Object{"name": "Synthetic document", "source": "Original synthetic test", "edition": "v1", "content": "Synthetic commands require authentication.", "workspace_id": scope, "enabled": true})
		w := enterpriseCall(h, "POST", "/knowledge-documents", body, admin, map[string]string{"Idempotency-Key": key})
		if w.Code != 201 {
			t.Fatal(w.Code, w.Body.String())
		}
		return httpObject(t, w)
	}
	organizationDoc := createDocument("", "synthetic-organization-doc")
	projectDoc := createDocument(wid, "synthetic-project-doc")
	createIndex := func(cookie *http.Cookie, doc Object, key string, status int) Object {
		body, _ := repoBody(Object{"kind": "knowledge_index", "provider_ids": []any{provider["id"]}, "document_ids": []any{doc["id"]}})
		w := enterpriseCall(h, "POST", "/workspaces/"+wid+"/ai-tasks", body, cookie, map[string]string{"If-Match": `"0"`, "Idempotency-Key": key})
		if w.Code != status {
			t.Fatalf("index creation: want %d, got %d: %s", status, w.Code, w.Body.String())
		}
		return httpObject(t, w)
	}
	createIndex(analyst, organizationDoc, "denied-organization-index", 403)
	if calls.Load() != 0 {
		t.Fatal("denied organization indexing transmitted document text")
	}
	adminJob := createIndex(admin, organizationDoc, "admin-organization-index", 202)
	if job := awaitJob(t, s.ai, wid, dStr(adminJob["id"])); job["status"] != "completed" {
		t.Fatal(job["status"], job["error"])
	}
	w = enterpriseCall(h, "POST", "/workspaces/"+wid+"/ai-tasks/"+dStr(adminJob["id"])+"/retries", `{}`, analyst, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "denied-organization-retry"})
	if w.Code != 403 || calls.Load() != 1 {
		t.Fatal("analyst retried organization indexing", w.Code, w.Body.String(), calls.Load())
	}
	projectJob := createIndex(analyst, projectDoc, "allowed-project-index", 202)
	if job := awaitJob(t, s.ai, wid, dStr(projectJob["id"])); job["status"] != "completed" {
		t.Fatal(job["status"], job["error"])
	}
	if calls.Load() != 2 {
		t.Fatal("project analyst could not index project knowledge")
	}
	// Occupy the worker slots deterministically, then simulate a future administrative scope migration.
	for i := 0; i < cap(s.ai.slots); i++ {
		s.ai.slots <- struct{}{}
	}
	queued := createIndex(analyst, projectDoc, "queued-project-index", 202)
	if dObj(queued["document_scopes"])[dStr(projectDoc["id"])] != wid {
		t.Fatal("index task did not pin document ownership")
	}
	moved := Clone(projectDoc)
	moved["workspace_id"] = ""
	if _, e = repo.platformPut(ctx, "knowledge", dStr(projectDoc["id"]), int64(dInt(projectDoc["revision"])), moved, "synthetic migration"); e != nil {
		t.Fatal(e)
	}
	<-s.ai.slots
	job := awaitJob(t, s.ai, wid, dStr(queued["id"]))
	if job["status"] != "failed" || !strings.Contains(dStr(job["error"]), "所属范围") || calls.Load() != 2 {
		t.Fatal("queued task indexed a changed document scope", job["status"], job["error"], calls.Load())
	}
	for i := 1; i < cap(s.ai.slots); i++ {
		<-s.ai.slots
	}
}

func TestAIAdoptionRevalidatesRegulationSources(t *testing.T) {
	for _, change := range []string{"unchanged", "body", "edition", "source", "withdrawn", "removed", "catalog_unavailable"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			repo, _ := repositoryTest(t)
			catalog := permissionTestCatalog()
			a := NewAIService(repo, catalog)
			defer a.Close()
			ws, _, e := repo.Create(ctx, "Synthetic regulation analysis", "", "synthetic-regulation-project")
			if e != nil {
				t.Fatal(e)
			}
			wid := dStr(ws["id"])
			ws["regulation_refs"] = []any{"reg:synthetic:1"}
			raw, _ := repoBody(ws)
			if _, e = repo.db.ExecContext(ctx, "UPDATE workspaces SET body=? WHERE id=?", raw, wid); e != nil {
				t.Fatal(e)
			}
			retrieved, e := a.searchKnowledge(ctx, wid, "Synthetic authenticated commands", nil, "", false, 8)
			if e != nil || len(dList(retrieved["items"])) != 1 {
				t.Fatal("synthetic regulation retrieval", e, retrieved)
			}
			source := dObj(dList(retrieved["items"])[0])
			proposal := Object{"summary": "Synthetic proposal", "assumptions": []any{}, "questions": []any{}, "citations": []any{source["id"]}, "assets": []any{Object{"id": "new-asset", "name": "Synthetic commands", "properties": []any{"integrity"}}}}
			job := Object{"id": "synthetic-ready-job", "workspace_id": wid, "status": "ready", "kind": "analysis", "base_revision": 0, "sources": []any{source}, "proposal": proposal}
			raw, _ = repoBody(job)
			if _, e = repo.db.ExecContext(ctx, "INSERT INTO ai_jobs(id,workspace_id,request_key,fingerprint,body) VALUES(?,?,?,?,?)", job["id"], wid, "synthetic-job-request", "synthetic", raw); e != nil {
				t.Fatal(e)
			}
			switch change {
			case "body":
				catalog.Regulations.byKey["reg:synthetic:1"]["body"] = "Updated synthetic requirement."
			case "edition":
				catalog.Regulations.byDocument["synthetic"]["edition"] = "v2"
			case "source":
				catalog.Regulations.byDocument["synthetic"]["source_url"] = "https://example.invalid/synthetic-v2"
			case "withdrawn":
				catalog.Regulations.byKey["reg:synthetic:1"]["withdrawn"] = true
			case "removed":
				delete(catalog.Regulations.byKey, "reg:synthetic:1")
			case "catalog_unavailable":
				a.catalog = nil
			}
			_, _, e = a.adoptJob(ctx, wid, dStr(job["id"]), "synthetic-adoption", "Synthetic owner", "synthetic", 0)
			current, readErr := repo.Get(ctx, wid)
			if readErr != nil {
				t.Fatal(readErr)
			}
			checkpoints, readErr := repo.checkpoints(ctx, wid)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if change == "unchanged" {
				if e != nil || len(dList(current["assets"])) != 1 || len(checkpoints) != 1 {
					t.Fatal("unchanged regulation prevented adoption", e)
				}
				return
			}
			if e == nil || !strings.Contains(e.Error(), "引用法规") {
				t.Fatal("changed or withdrawn regulation accepted", e)
			}
			storedJob, readErr := a.job(ctx, wid, dStr(job["id"]))
			if readErr != nil || storedJob["status"] != "ready" || dInt(current["revision"]) != 0 || len(dList(current["assets"])) != 0 || len(checkpoints) != 0 {
				t.Fatal("rejected adoption mutated work products or task", readErr, current, storedJob)
			}
		})
	}
}
