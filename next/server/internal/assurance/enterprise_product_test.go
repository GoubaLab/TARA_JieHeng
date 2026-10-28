package assurance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func enterpriseCall(h http.Handler, method, path, body string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://tara.example.internal/api/v1"+path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:4567"
	r.Header.Set("X-Workspace-Client", "jieheng")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://tara.example.internal")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestEnterprisePermissionsSessionsAndIsolation(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	s := h.(*HTTPServer)
	defer s.Close()
	ctx := context.Background()
	if e := repo.BootstrapAdmin(ctx, "administrator", "Synthetic-Password-123"); e != nil {
		t.Fatal(e)
	}
	s.auth = &EnterpriseAuth{origin: "https://tara.example.internal"}
	if w := enterpriseCall(h, "GET", "/workspaces", "", nil, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	login := func(name string) *http.Cookie {
		w := enterpriseCall(h, "POST", "/auth/sessions", `{"username":"`+name+`","password":"Synthetic-Password-123"}`, nil, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatal("unsafe cookie")
		}
		return cookies[0]
	}
	admin := login("administrator")
	bad := enterpriseCall(h, "POST", "/workspaces", `{"title":"bad origin"}`, admin, map[string]string{"Origin": "https://untrusted.example", "Idempotency-Key": "origin-test-001"})
	if bad.Code != 403 {
		t.Fatal("origin", bad.Code)
	}
	w := enterpriseCall(h, "POST", "/workspaces", `{"title":"Synthetic enterprise project"}`, admin, map[string]string{"Idempotency-Key": "enterprise-project-1"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	wid := dStr(httpObject(t, w)["id"])
	createUser := func(name, role string) Object {
		w := enterpriseCall(h, "POST", "/enterprise/users", `{"username":"`+name+`","name":"`+name+`","role":"`+role+`","enabled":true,"password":"Synthetic-Password-123"}`, admin, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		return httpObject(t, w)
	}
	viewer := createUser("observer", "viewer")
	analyst := createUser("analyst", "analyst")
	observerSession := login("observer")
	analystSession := login("analyst")
	if w = enterpriseCall(h, "GET", "/workspaces/"+wid, "", observerSession, nil); w.Code != 404 {
		t.Fatal("nonmember", w.Code)
	}
	if w = enterpriseCall(h, "GET", "/workspaces", "", observerSession, nil); len(dList(httpObject(t, w)["items"])) != 0 {
		t.Fatal("workspace list disclosed")
	}
	for _, u := range []Object{viewer, analyst} {
		w = enterpriseCall(h, "PUT", "/workspaces/"+wid+"/members/"+dStr(u["id"]), `{"role":"reviewer"}`, admin, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w = enterpriseCall(h, "GET", "/workspaces/"+wid, "", observerSession, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	headers := map[string]string{"If-Match": `"0"`, "Idempotency-Key": "enterprise-checkpoint"}
	if w = enterpriseCall(h, "POST", "/workspaces/"+wid+"/checkpoints", `{"name":"test"}`, observerSession, headers); w.Code != 403 {
		t.Fatal("global viewer escalated", w.Code)
	}
	if w = enterpriseCall(h, "POST", "/workspaces/"+wid+"/restorations", `{}`, analystSession, headers); w.Code != 403 {
		t.Fatal("global analyst escalated", w.Code)
	}
	if w = enterpriseCall(h, "POST", "/model-providers", `{}`, analystSession, nil); w.Code != 403 {
		t.Fatal("analyst edited model", w.Code)
	}
	if w = enterpriseCall(h, "POST", "/workspaces/"+wid+"/checkpoints", `{"name":"Synthetic checkpoint"}`, analystSession, headers); w.Code != 201 {
		t.Fatal("analyst checkpoint", w.Code, w.Body.String())
	}
	body := `{"username":"observer","name":"observer","role":"viewer","enabled":false}`
	if w = enterpriseCall(h, "PUT", "/enterprise/users/"+dStr(viewer["id"]), body, admin, map[string]string{"If-Match": `"0"`}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = enterpriseCall(h, "GET", "/workspaces/"+wid, "", observerSession, nil); w.Code != 401 {
		t.Fatal("disabled session survived", w.Code)
	}
	users := httpObject(t, enterpriseCall(h, "GET", "/enterprise/users", "", admin, nil))
	var adminID string
	for _, raw := range dList(users["items"]) {
		u := dObj(raw)
		if u["role"] == "admin" {
			adminID = dStr(u["id"])
		}
	}
	w = enterpriseCall(h, "PUT", "/enterprise/users/"+adminID, `{"username":"administrator","name":"administrator","role":"viewer","enabled":true}`, admin, map[string]string{"If-Match": `"0"`})
	if w.Code != 422 {
		t.Fatal("last admin removed", w.Code)
	}
	for i := 0; i < 5; i++ {
		w = enterpriseCall(h, "POST", "/auth/sessions", `{"username":"missing-user","password":"bad-password"}`, nil, nil)
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	w = enterpriseCall(h, "POST", "/auth/sessions", `{"username":"missing-user","password":"bad-password"}`, nil, nil)
	if w.Code != 429 {
		t.Fatal("rate limit missing", w.Code)
	}
}
func TestIndependentBackupRestoreAndKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live.sqlite3")
	repo, e := NewRepository(path)
	if e != nil {
		t.Fatal(e)
	}
	defer repo.Close()
	ctx := context.Background()
	ws, _, e := repo.Create(ctx, "Backup fixture", "", "backup-fixture-1")
	if e != nil {
		t.Fatal(e)
	}
	provider, e := repo.saveProvider(ctx, "", -1, providerFixture("http://127.0.0.1:11434", "ollama"), "test")
	if e != nil {
		t.Fatal(e)
	}
	backup := filepath.Join(dir, "backup")
	if e = BackupDatabase(ctx, path, backup); e != nil {
		t.Fatal(e)
	}
	if e = BackupDatabase(ctx, path, backup); e == nil {
		t.Fatal("backup overwrite accepted")
	}
	restored := filepath.Join(dir, "restored")
	if e = RestoreBackup(ctx, backup, restored); e != nil {
		t.Fatal(e)
	}
	r, e := NewRepository(filepath.Join(restored, "assurance.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if _, e = r.Get(ctx, dStr(ws["id"])); e != nil {
		t.Fatal(e)
	}
	p, e := r.platformGet(ctx, "provider", dStr(provider["id"]))
	if e != nil {
		t.Fatal(e)
	}
	secret, e := r.decryptSecret(dStr(p["id"]), dStr(p["secret_ciphertext"]))
	if e != nil || secret != "synthetic-secret-not-for-export" {
		t.Fatal("key not preserved", e)
	}
	if e = RestoreBackup(ctx, backup, restored); e == nil {
		t.Fatal("restore overwrite accepted")
	}
	f, e := os.OpenFile(filepath.Join(backup, "assurance.sqlite3"), os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	f.WriteString("corrupt")
	f.Close()
	if e = VerifyBackup(ctx, backup); e == nil {
		t.Fatal("tampered backup accepted")
	}
}
