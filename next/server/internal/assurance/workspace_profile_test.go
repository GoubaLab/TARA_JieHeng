package assurance

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestWorkspaceProfilePreservesAssessmentAndFrozenProducts(t *testing.T) {
	steps := dList(dFixture(t)["steps"])
	original := Clone(dObj(dObj(steps[len(steps)-1])["expected"]))
	before := Digest(original)
	updated, id, err := Apply(original, "update-workspace-profile", Object{"title": "  控制系统安全分析  ", "description": "  产品范围说明  "})
	if err != nil {
		t.Fatal(err)
	}
	if id != original["id"] || updated["title"] != "控制系统安全分析" || updated["description"] != "产品范围说明" || dInt(updated["revision"]) != dInt(original["revision"])+1 {
		t.Fatal("profile fields or revision incorrect", updated)
	}
	if Digest(original) != before {
		t.Fatal("profile edit mutated its input")
	}
	for key, value := range original {
		if key != "title" && key != "description" && key != "revision" && key != "updated_at" {
			dEqual(t, updated[key], value)
		}
	}
	if len(updated) != len(original) {
		t.Fatal("profile edit initialized unrelated workspace fields")
	}
	for _, scenario := range libRows(original["scenarios"]) {
		if AssessmentHash(original, scenario) != AssessmentHash(updated, scenario) {
			t.Fatal("profile edit invalidated assessment")
		}
	}
	dEqual(t, Review(updated), Review(original))
	_, _, err = Apply(original, "update-workspace-profile", Object{"title": strings.Repeat("界", 200), "description": strings.Repeat("文", 4000)})
	if err != nil {
		t.Fatal("valid Unicode length bounds rejected", err)
	}
}

func TestWorkspaceProfileHTTPRevisionReplayAndValidation(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	base := apiPrefix + "/workspaces/" + id
	path := base + "/profile"
	original := httpObject(t, httpCall(h, "GET", base, "", nil))
	body := `{"title":"车辆控制系统","description":"控制与诊断接口的威胁分析"}`
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "workspace-profile-001"}
	first := httpCall(h, "PUT", path, body, headers)
	if first.Code != 200 || first.Header().Get("ETag") != `"1"` {
		t.Fatal(first.Code, first.Body.String())
	}
	replay := httpCall(h, "PUT", path, body, headers)
	if replay.Code != 200 || replay.Body.String() != first.Body.String() || replay.Header().Get("ETag") != `"1"` {
		t.Fatal("idempotent profile replay changed result")
	}
	if r := httpCall(h, "PUT", path, `{"title":"不同内容","description":""}`, headers); r.Code != 409 {
		t.Fatal("same key with different payload accepted", r.Code)
	}
	if r := httpCall(h, "PUT", path, body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": "workspace-profile-stale"}); r.Code != 412 || r.Header().Get("ETag") != `"1"` {
		t.Fatal("stale profile update accepted", r.Code)
	}
	if r := httpCall(h, "PUT", path, body, map[string]string{"Idempotency-Key": "workspace-profile-noetag"}); r.Code != 428 {
		t.Fatal("missing precondition accepted", r.Code)
	}
	invalid := []Object{
		{"title": "", "description": ""},
		{"title": "   ", "description": ""},
		{"title": "有效名称"},
		{"description": "缺少名称"},
		{"title": true, "description": ""},
		{"title": "有效名称", "description": nil},
		{"title": strings.Repeat("界", 201), "description": ""},
		{"title": "有效名称", "description": strings.Repeat("文", 4001)},
		{"title": "有效名称", "description": "", "item": Object{}},
		{"title": "有效名称", "description": "", "id": "different"},
	}
	for i, payload := range invalid {
		r := httpCall(h, "PUT", path, httpGenericBody(t, payload), map[string]string{"If-Match": `"1"`, "Idempotency-Key": fmt.Sprintf("workspace-profile-invalid-%02d", i)})
		if r.Code != 422 {
			t.Fatal("invalid profile accepted", i, r.Code, r.Body.String())
		}
	}
	stored := httpObject(t, httpCall(h, "GET", base, "", nil))
	if stored["title"] != "车辆控制系统" || stored["description"] != "控制与诊断接口的威胁分析" || dInt(stored["revision"]) != 1 {
		t.Fatal("invalid update altered stored profile")
	}
	for key, value := range original {
		if key != "title" && key != "description" && key != "revision" && key != "updated_at" {
			dEqual(t, stored[key], value)
		}
	}
	events, count, err := repo.Events(context.Background(), id, 0, 100)
	if err != nil || count != 2 || events[0]["action"] != "update-workspace-profile" {
		t.Fatal("profile event missing or duplicated", count, err)
	}
	dEqual(t, dObj(events[0]["body"])["payload"], Object{"title": "车辆控制系统", "description": "控制与诊断接口的威胁分析"})
	listed := httpObject(t, httpCall(h, "GET", apiPrefix+"/workspaces", "", nil))
	if dObj(dList(listed["items"])[0])["title"] != stored["title"] {
		t.Fatal("project list did not reflect updated profile")
	}
}

func TestWorkspaceProfileEnterpriseProjectPermissions(t *testing.T) {
	for _, test := range []struct {
		name, accountRole, projectRole string
		status                         int
	}{
		{"unauthenticated", "", "", 401},
		{"nonmember", "analyst", "", 404},
		{"project-viewer", "analyst", "viewer", 403},
		{"account-viewer", "viewer", "analyst", 403},
		{"analyst", "analyst", "analyst", 200},
		{"admin", "admin", "", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			h, repo, _ := httpTestServer(t)
			ws := repoTestCreate(t, repo)
			s := h.(*HTTPServer)
			s.auth = &EnterpriseAuth{origin: "https://tara.example.internal"}
			userID := "profile-permission-user"
			if test.projectRole != "" {
				if _, err := repo.db.Exec("INSERT INTO enterprise_members(workspace_id,user_id,role) VALUES(?,?,?)", ws["id"], userID, test.projectRole); err != nil {
					t.Fatal(err)
				}
			}
			r := httptest.NewRequest("PUT", "https://tara.example.internal/api/v1/workspaces/"+dStr(ws["id"])+"/profile", strings.NewReader(`{"title":"更新项目","description":""}`))
			r.RemoteAddr = "127.0.0.1:4567"
			r.Header.Set("X-Workspace-Client", "jieheng")
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", "https://tara.example.internal")
			r.Header.Set("If-Match", `"0"`)
			r.Header.Set("Idempotency-Key", "profile-permission-001")
			if test.accountRole != "" {
				r = r.WithContext(context.WithValue(r.Context(), principalContextKey{}, Object{"id": userID, "role": test.accountRole}))
			}
			response := httptest.NewRecorder()
			h.ServeHTTP(response, r)
			if response.Code != test.status {
				t.Fatal(response.Code, response.Body.String())
			}
			stored, err := repo.Get(context.Background(), dStr(ws["id"]))
			if err != nil {
				t.Fatal(err)
			}
			if test.status != 200 && Digest(stored) != Digest(ws) {
				t.Fatal("denied request mutated workspace")
			}
		})
	}
}

func TestWorkspaceProfileConcurrentEditorsCannotOverwrite(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	path := apiPrefix + "/workspaces/" + id + "/profile"
	start := make(chan struct{})
	codes := make(chan int, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-start
			body := fmt.Sprintf(`{"title":"Project %d","description":""}`, i)
			response := httpCall(h, "PUT", path, body, map[string]string{"If-Match": `"0"`, "Idempotency-Key": fmt.Sprintf("concurrent-profile-%d", i)})
			codes <- response.Code
		}(i)
	}
	close(start)
	group.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[412] != 1 {
		t.Fatal("concurrent edits were not serialized by revision", counts)
	}
	stored, err := repo.Get(context.Background(), id)
	if err != nil || dInt(stored["revision"]) != 1 {
		t.Fatal("unexpected profile revision", err)
	}
	_, count, err := repo.Events(context.Background(), id, 0, 100)
	if err != nil || count != 2 {
		t.Fatal("conflicting edit created an audit event", count, err)
	}
}
