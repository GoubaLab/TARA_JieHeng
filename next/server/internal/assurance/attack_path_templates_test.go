package assurance

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func attackTemplateFind(t *testing.T, items []any, id string) Object {
	t.Helper()
	for _, raw := range items {
		item := dObj(raw)
		if item["id"] == id {
			return item
		}
	}
	t.Fatal("missing template", id)
	return nil
}
func attackTemplateAsset(t *testing.T, ws Object, kind, id string) Object {
	t.Helper()
	for _, raw := range dList(ws["assets"]) {
		a := dObj(raw)
		source := assetSource(a)
		if source["kind"] == kind && source["id"] == id {
			return a
		}
	}
	t.Fatal("missing asset", kind, id)
	return nil
}
func TestAttackPathTemplatesOriginalCatalogNativePathAndStableIdentity(t *testing.T) {
	items, e := attackPathTemplateItems(nil, "", "")
	if e != nil {
		t.Fatal(e)
	}
	if len(items) != 16 {
		t.Fatalf("got %d templates", len(items))
	}
	seen := map[string]bool{}
	for _, raw := range items {
		item := dObj(raw)
		id := dStr(item["id"])
		if seen[id] || !strings.HasPrefix(id, "AP-") || item["source_code"] != "TPL:"+id+"@1" {
			t.Fatal("invalid stable identity", item)
		}
		seen[id] = true
		for _, k := range []string{"name", "category", "summary", "entry_point", "actor", "objective"} {
			if !dBool(item[k]) {
				t.Fatal("missing text", k, item)
			}
		}
		for _, k := range []string{"prerequisites", "steps", "controls_to_check", "properties", "tags"} {
			if len(dList(item[k])) == 0 {
				t.Fatal("missing review data", k, item)
			}
		}
		if dInt(dObj(item["match"])["score"]) != 0 || len(dList(dObj(item["match"])["reasons"])) != 0 {
			t.Fatal("uncontextualized finding", item)
		}
		for _, p := range dList(item["properties"]) {
			if !automationIn(dStr(p), dProperties...) {
				t.Fatal("unknown property")
			}
		}
		steps := []string{}
		for _, key := range []string{"prerequisites", "steps", "controls_to_check"} {
			for _, v := range dList(item[key]) {
				steps = append(steps, dStr(v))
			}
		}
		path, e := dValidateModel("AttackPath", Object{"name": item["name"], "source_code": item["source_code"], "actor": item["actor"], "vector": item["entry_point"], "steps": strings.Join(steps, "\n"), "feasibility": nil, "rationale": ""})
		if e != nil {
			t.Fatal("template does not fit native path", id, e)
		}
		if path["feasibility"] != nil || path["potential"] != nil || path["reference"] != nil || path["source_code"] != item["source_code"] {
			t.Fatal("invented rating or reference", path)
		}
	}
	dObj(items[0])["name"] = "mutated caller copy"
	again, _ := attackPathTemplateItems(nil, "", "")
	if dObj(again[0])["name"] == "mutated caller copy" {
		t.Fatal("shared mutable builtin catalog")
	}
}
func TestAttackPathTemplateRecommendationsUseExactRecordedFacts(t *testing.T) {
	ws := automationFixture(t)
	before := Digest(ws)
	asset := attackTemplateAsset(t, ws, "flow", "update")
	items, e := attackPathTemplateItems(ws, dStr(asset["id"]), "integrity")
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"AP-WIRELESS-PEER", "AP-UPDATE-CONTENT", "AP-UPDATE-ROLLBACK", "AP-CROSS-DOMAIN-INPUT"} {
		item := attackTemplateFind(t, items, id)
		if dInt(dObj(item["match"])["score"]) <= 10 || len(dList(dObj(item["match"])["reasons"])) == 0 {
			t.Fatal("missing precise context", id, item)
		}
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-DEBUG-WRITE")["match"])["score"]) > 10 {
		t.Fatal("another flow leaked into recommendations")
	}
	if len(items) != 16 || Digest(ws) != before {
		t.Fatal("recommendation filtered catalog or mutated project")
	}
	repeat, _ := attackPathTemplateItems(ws, dStr(asset["id"]), "integrity")
	if Digest(items) != Digest(repeat) {
		t.Fatal("unstable recommendation")
	}
	moved := Clone(ws)
	for _, raw := range dList(dObj(moved["architecture"])["nodes"]) {
		n := dObj(raw)
		n["x"], n["y"] = 9000, -9000
	}
	movedItems, _ := attackPathTemplateItems(moved, dStr(asset["id"]), "integrity")
	if Digest(items) != Digest(movedItems) {
		t.Fatal("geometry inferred security facts")
	}
}
func TestAttackPathTemplatesDoNotInferStoreFlowOrParentNetwork(t *testing.T) {
	ws := automationFixture(t)
	// A manual asset linked to an entity is not the store inside that entity.
	ws["assets"] = append(dList(ws["assets"]), Object{"id": "manual", "name": "Other manual data", "node_ids": []any{"device"}, "properties": []any{"confidentiality"}})
	items, e := attackPathTemplateItems(ws, "manual", "")
	if e != nil {
		t.Fatal(e)
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-STORAGE-READ")["match"])["score"]) != 5 {
		t.Fatal("inferred a particular store from node links")
	}
	store := attackTemplateAsset(t, ws, "store", "store")
	items, e = attackPathTemplateItems(ws, dStr(store["id"]), "")
	if e != nil {
		t.Fatal(e)
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-STORAGE-READ")["match"])["score"]) <= 5 {
		t.Fatal("exact store source not used")
	}
	if dInt(dObj(attackTemplateFind(t, items, "AP-NETWORK-SERVICE")["match"])["score"]) > 5 {
		t.Fatal("parent OS or network treated as store entry")
	}
	assetSource(store)["status"] = "missing"
	items, _ = attackPathTemplateItems(ws, dStr(store["id"]), "")
	if dInt(dObj(attackTemplateFind(t, items, "AP-STORAGE-READ")["match"])["score"]) > 5 {
		t.Fatal("retired source used")
	}
	flow := attackTemplateAsset(t, ws, "flow", "update")
	for _, raw := range dList(dObj(ws["architecture"])["flows"]) {
		f := dObj(raw)
		if f["id"] == "update" {
			f["channel_kind"] = "energy"
		}
	}
	items, _ = attackPathTemplateItems(ws, dStr(flow["id"]), "")
	if dInt(dObj(attackTemplateFind(t, items, "AP-WIRELESS-PEER")["match"])["score"]) > 5 {
		t.Fatal("energy flow used as a data entry")
	}
}
func TestAttackPathTemplateHTTPValidationAndReadOnlyContract(t *testing.T) {
	repo, _ := repositoryTest(t)
	ws := repoTestCreate(t, repo)
	s := &HTTPServer{repository: repo, mux: http.NewServeMux(), logger: log.New(io.Discard, "", 0), static: http.NotFoundHandler()}
	s.route("GET "+apiPrefix+"/attack-path-templates", s.attackPathTemplates)
	s.route("GET "+apiPrefix+"/workspaces/{workspace_id}/attack-path-templates", s.workspaceAttackPathTemplates)
	global := httpCall(s, "GET", apiPrefix+"/attack-path-templates", "", nil)
	if global.Code != 200 || len(dList(httpObject(t, global)["items"])) != 16 {
		t.Fatal(global.Code, global.Body.String())
	}
	path := apiPrefix + "/workspaces/" + dStr(ws["id"]) + "/attack-path-templates"
	for _, c := range []struct {
		suffix string
		status int
	}{{"", 200}, {"?asset_id=unrelated", 404}, {"?property=unknown", 422}, {"?property=integrity&property=availability", 400}, {"?unknown=x", 400}, {"?asset_id=x&asset_id=y", 400}} {
		response := httpCall(s, "GET", path+c.suffix, "", nil)
		if response.Code != c.status {
			t.Fatal(c.suffix, response.Code, response.Body.String())
		}
	}
	response := httpCall(s, "GET", path, "", nil)
	if response.Header().Get("ETag") != `"0"` || dInt(httpObject(t, response)["base_revision"]) != 0 {
		t.Fatal("missing revision context")
	}
	if r := httpCall(s, "POST", apiPrefix+"/attack-path-templates", "{}", nil); r.Code != 405 {
		t.Fatal("catalog write exposed", r.Code)
	}
	if r := httpCall(s, "GET", apiPrefix+"/attack-path-templates?property=integrity", "", nil); r.Code != 400 {
		t.Fatal("global query accepted", r.Code)
	}
	after, e := repo.Get(context.Background(), dStr(ws["id"]))
	if e != nil || Digest(after) != Digest(ws) {
		t.Fatal("template read mutated project", e)
	}
}
func TestAttackPathTemplateEndpointChecksProjectMembership(t *testing.T) {
	repo, _ := repositoryTest(t)
	ws := repoTestCreate(t, repo)
	s := &HTTPServer{repository: repo, auth: &EnterpriseAuth{origin: "https://tara.example.internal"}}
	req := httptest.NewRequest("GET", "https://tara.example.internal/api/v1/workspaces/"+dStr(ws["id"])+"/attack-path-templates", nil)
	req.SetPathValue("workspace_id", dStr(ws["id"]))
	req = req.WithContext(context.WithValue(req.Context(), principalContextKey{}, Object{"id": "viewer-synthetic", "role": "viewer"}))
	if e := s.workspaceAttackPathTemplates(httptest.NewRecorder(), req); e != ErrNotFound {
		t.Fatal("non-member can read project context", e)
	}
	if _, e := repo.db.Exec("INSERT INTO enterprise_members(workspace_id,user_id,role) VALUES(?,?,?)", ws["id"], "viewer-synthetic", "viewer"); e != nil {
		t.Fatal(e)
	}
	response := httptest.NewRecorder()
	if e := s.workspaceAttackPathTemplates(response, req); e != nil || response.Code != 200 {
		t.Fatal("read-only member cannot browse", e, response.Code)
	}
}
