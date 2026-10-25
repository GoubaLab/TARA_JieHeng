package assurance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func repositoryTest(t *testing.T) (*Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "assurance.sqlite3")
	r, e := NewRepository(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, path
}
func repoTestCreate(t *testing.T, r *Repository) Object {
	t.Helper()
	ws, _, e := r.Create(context.Background(), "仓储验证", "", "create-test-0001")
	if e != nil {
		t.Fatal(e)
	}
	return ws
}
func TestRepositoryConcurrentReplayAndRevisionAcrossConnections(t *testing.T) {
	r, path := repositoryTest(t)
	other, e := NewRepository(path)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	repos := []*Repository{r, other}
	ctx := context.Background()
	var group sync.WaitGroup
	results := make(chan Object, 16)
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			ws, _, e := repos[i%2].Create(ctx, "并发创建", "", "concurrent-create")
			results <- ws
			errs <- e
		}(i)
	}
	group.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	key := ""
	for ws := range results {
		if key == "" {
			key = fmt.Sprint(ws["id"])
		}
		if ws["id"] != key {
			t.Fatal("duplicate workspace created")
		}
	}
	list, total, e := r.List(ctx, 0, 100)
	if e != nil || total != 1 || len(list) != 1 {
		t.Fatalf("create not atomic: %d %v", total, e)
	}
	errs = make(chan error, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			result, _, e := repos[i%2].Change(ctx, key, 0, "concurrent-change", "define-item", Object{"purpose": "one change"}, nil)
			if e == nil {
				n, _ := repoNumber(result["revision"])
				if n != 1 {
					e = fmt.Errorf("unexpected revision %d", n)
				}
			}
			errs <- e
		}(i)
	}
	group.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	errs = make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			_, _, e := repos[i].Change(ctx, key, 1, fmt.Sprintf("unique-change-%d", i), "define-item", Object{"purpose": fmt.Sprint(i)}, nil)
			errs <- e
		}(i)
	}
	group.Wait()
	close(errs)
	success, stale := 0, 0
	for e := range errs {
		var rev *RevisionError
		if e == nil {
			success++
		} else if errors.As(e, &rev) {
			stale++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("concurrent revision outcomes: %d %d", success, stale)
	}
	_, count, e := r.Events(ctx, key, 0, 100)
	if e != nil || count != 3 {
		t.Fatalf("duplicate event or lost mutation: %d %v", count, e)
	}
}
func TestRepositoryLegacyPythonReceiptReplayBeforeRevision(t *testing.T) {
	r, _ := repositoryTest(t)
	ws := repoTestCreate(t, r)
	ctx := context.Background()
	key := fmt.Sprint(ws["id"])
	result := Object{"workspace_id": key, "revision": 1, "entity_id": key}
	body, _ := repoBody(result)
	// Fixed Python json.dumps(sort_keys=True, ensure_ascii=False) oracle for this command.
	fp := "beed2562d6f1e0b3504f1b5ffbd79ae24d0452bf9a87fb328881f1bb14663fa4"
	if _, e := r.db.Exec("INSERT INTO receipts VALUES(?,?,?,?)", key, "legacy-request-001", fp, body); e != nil {
		t.Fatal(e)
	}
	called := false
	got, replay, e := r.Change(ctx, key, 999, "legacy-request-001", "define-item", Object{"purpose": "旧项目"}, func(string, Object, Object) (Object, error) {
		called = true
		return nil, errors.New("must not prepare")
	})
	if e != nil || !replay || called || Digest(got) != Digest(result) {
		t.Fatalf("legacy replay failed: %v %v %v", replay, called, e)
	}
	_, _, e = r.Change(ctx, key, 0, "legacy-request-001", "define-item", Object{"purpose": "different"}, nil)
	var conflict *ConflictError
	if !errors.As(e, &conflict) {
		t.Fatalf("expected conflict: %v", e)
	}
}
func TestRepositoryRollbackAfterLateWriteFailure(t *testing.T) {
	r, _ := repositoryTest(t)
	ws := repoTestCreate(t, r)
	ctx := context.Background()
	key := fmt.Sprint(ws["id"])
	before := Digest(ws)
	if _, e := r.db.Exec(`CREATE TRIGGER reject_receipt BEFORE INSERT ON receipts BEGIN SELECT RAISE(ABORT,'test failure'); END`); e != nil {
		t.Fatal(e)
	}
	_, _, e := r.Change(ctx, key, 0, "rollback-request", "define-item", Object{"purpose": "must roll back"}, nil)
	if e == nil {
		t.Fatal("accepted failure")
	}
	after, e := r.Get(ctx, key)
	if e != nil || Digest(after) != before {
		t.Fatal("partial workspace committed")
	}
	_, total, e := r.Events(ctx, key, 0, 100)
	if e != nil || total != 1 {
		t.Fatal("partial event committed")
	}
	var receipts int
	_ = r.db.QueryRow("SELECT count(*) FROM receipts").Scan(&receipts)
	if receipts != 0 {
		t.Fatal("partial receipt committed")
	}
}
func TestRepositoryFloatIdentityAndImmutableBaseline(t *testing.T) {
	r, path := repositoryTest(t)
	ctx := context.Background()
	fixture, e := DecodeJSON([]byte(domainGoldenFixture))
	if e != nil {
		t.Fatal(e)
	}
	ws := fixture["initial"].(map[string]any)
	var seal Object
	for _, raw := range fixture["steps"].([]any) {
		step := raw.(map[string]any)
		if step["action"] == "seal-baseline" {
			seal = step["payload"].(map[string]any)
			break
		}
		ws = step["expected"].(map[string]any)
	}
	if seal == nil {
		t.Fatal("baseline fixture missing")
	}
	before := Digest(ws)
	body, _ := repoBody(ws)
	key := fmt.Sprint(ws["id"])
	if _, e = r.db.Exec("INSERT INTO workspaces VALUES(?,?)", key, body); e != nil {
		t.Fatal(e)
	}
	revision, _ := repoNumber(ws["revision"])
	result, _, e := r.Change(ctx, key, revision, "seal-baseline-001", "seal-baseline", seal, nil)
	if e != nil {
		t.Fatal(e)
	}
	id := fmt.Sprint(result["entity_id"])
	snapshot, e := r.Baseline(ctx, key, id)
	if e != nil || Digest(snapshot) != before {
		t.Fatalf("snapshot changed: %v", e)
	}
	if _, _, e = r.Change(ctx, key, revision+1, "after-baseline-01", "define-item", Object{"purpose": "changed later"}, nil); e != nil {
		t.Fatal(e)
	}
	snapshot, e = r.Baseline(ctx, key, id)
	if e != nil || Digest(snapshot) != before {
		t.Fatal("baseline overwritten")
	}
	other, e := NewRepository(path)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	snapshot, e = other.Baseline(ctx, key, id)
	if e != nil || Digest(snapshot) != before {
		t.Fatal("numeric identity changed after reopen")
	}
	if !strings.Contains(body, ".0") {
		t.Fatal("fixture lacks floating-point coordinates")
	}
}
func TestRepositoryNewReplayIgnoresLaterIfMatchAndPrepare(t *testing.T) {
	r, _ := repositoryTest(t)
	ws := repoTestCreate(t, r)
	key := fmt.Sprint(ws["id"])
	ctx := context.Background()
	p := Object{"purpose": "unchanged"}
	one, _, e := r.Change(ctx, key, 0, "modern-replay-001", "define-item", p, nil)
	if e != nil {
		t.Fatal(e)
	}
	two, replay, e := r.Change(ctx, key, 100, "modern-replay-001", "define-item", p, func(string, Object, Object) (Object, error) { return nil, fmt.Errorf("must not run") })
	if e != nil || !replay || Digest(one) != Digest(two) {
		t.Fatal("replay did not precede revision/validation", e)
	}
	got, _ := r.Get(ctx, key)
	if got["revision"] != json.Number("1") {
		t.Fatalf("unexpected revision %#v", got["revision"])
	}
}
