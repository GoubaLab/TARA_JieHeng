package assurance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func libraryCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := NewCatalog("", "../../../data/regulations")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func libraryEqual(t *testing.T, got, want any) {
	t.Helper()
	a, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var x, y any
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal(b, &y)
	if !reflect.DeepEqual(x, y) {
		t.Fatalf("mismatch\ngot: %s\nwant: %s", a, b)
	}
}
func TestCatalogWithdrawnAndSnapshotIgnored(t *testing.T) {
	snapshot := t.TempDir()
	if err := os.WriteFile(filepath.Join(snapshot, "manifest.json"), []byte("invalid and must never be parsed"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := NewCatalog(snapshot, "../../../data/regulations")
	if err != nil {
		t.Fatal(err)
	}
	libraryEqual(t, c.Summary(), Object{"categories": []Object{}, "gaps": Object{"relationships": 0, "missing_control_ids": 0, "references": 0}, "origin": "未配置参考资料", "network_sync": false, "withdrawn": true})
	for category := range withdrawnCategories {
		page, err := c.Search(category, "anything", 3, 10)
		if err != nil {
			t.Fatal(err)
		}
		if libInt(page["total"]) != 0 || len(libList(page["items"])) != 0 || !libBool(page["withdrawn"]) {
			t.Fatal("retired category disclosed content")
		}
		key := category + ":retired-fixture-a"
		entry, err := c.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		libraryEqual(t, entry, Object{"key": key, "title": "引用已失效", "description": "", "withdrawn": true, "citable": false})
	}
	if _, err := c.Search("unconfigured", "", 0, 10); !errors.Is(err, ErrReferenceNotFound) {
		t.Fatalf("unexpected category result: %v", err)
	}
	if _, err := c.Get("unknown:synthetic"); !errors.Is(err, ErrReferenceNotFound) {
		t.Fatalf("unexpected reference result: %v", err)
	}
	if _, err := c.Search("systems", "", 0, 0); err == nil {
		t.Fatal("invalid paging accepted")
	}
}
func TestWithdrawnReferencesStayOnExistingRecords(t *testing.T) {
	c := libraryCatalog(t)
	ws := Object{"assets": []any{Object{"id": "asset-a", "reference": "systems:retired-fixture-a"}}, "scenarios": []any{Object{"id": "scenario-a", "references": []any{"patterns:retired-fixture-a"}, "paths": []any{Object{"id": "path-a", "reference": "techniques:retired-fixture-a"}}}}, "requirements": []any{Object{"id": "req-a", "reference": "controls:retired-fixture-a", "standard_refs": []any{"gb:retired-fixture-a"}}}, "architecture": Object{"nodes": []any{Object{"id": "entity-a", "reference": "systems:retired-fixture-a", "children": []any{Object{"id": "child-a", "reference": "systems:retired-fixture-b"}}}, Object{"id": "old-store", "kind": "store", "reference": "systems:retired-fixture-c"}}}}
	cases := []struct {
		action  string
		payload Object
		valid   bool
	}{
		{"record-asset", Object{"id": "asset-a", "reference": "systems:retired-fixture-a"}, true},
		{"record-asset", Object{"id": "asset-a", "reference": nil}, true},
		{"record-asset", Object{"reference": "systems:retired-fixture-a"}, false},
		{"record-asset", Object{"id": "asset-b", "reference": "systems:retired-fixture-a"}, false},
		{"record-asset", Object{"id": "asset-a", "reference": "systems:retired-fixture-b"}, false},
		{"record-scenario", Object{"id": "scenario-a", "references": []any{"patterns:retired-fixture-a"}, "paths": []any{Object{"id": "path-a", "reference": "techniques:retired-fixture-a"}}}, true},
		{"record-scenario", Object{"id": "scenario-a", "paths": []any{Object{"id": "path-b", "reference": "techniques:retired-fixture-a"}}}, false},
		{"record-scenario", Object{"id": "scenario-a", "references": []any{"techniques:retired-fixture-a"}}, false},
		{"specify-requirement", Object{"id": "req-a", "reference": "controls:retired-fixture-a", "standard_refs": []any{"gb:retired-fixture-a"}}, true},
		{"specify-requirement", Object{"id": "req-a", "standard_refs": []any{"gb:retired-fixture-a", "r155:retired-fixture-a"}}, false},
		{"replace-architecture", Object{"nodes": []any{Object{"id": "entity-a", "reference": "systems:retired-fixture-a", "children": []any{Object{"id": "child-a", "reference": "systems:retired-fixture-b"}, Object{"id": "old-store", "kind": "store", "reference": "systems:retired-fixture-c"}}}}}, true},
		{"replace-architecture", Object{"nodes": []any{Object{"id": "entity-a", "children": []any{Object{"id": "child-new", "reference": "systems:retired-fixture-b"}}}}}, false},
		{"select-regulations", Object{"references": []any{"gb:retired-fixture-a"}}, false},
		{"select-regulations", Object{}, true},
	}
	before := Digest(ws)
	for _, test := range cases {
		err := c.ValidateReferencesForWorkspace(test.action, test.payload, ws)
		if (err == nil) != test.valid {
			t.Errorf("%s %#v: %v", test.action, test.payload, err)
		}
	}
	if Digest(ws) != before {
		t.Fatal("reference validation changed aggregate")
	}
	if err := c.ValidateReferences("record-asset", Object{"reference": "systems:retired-fixture-a"}); !errors.Is(err, ErrReferenceWithdrawn) {
		t.Fatalf("new retired reference accepted: %v", err)
	}
}
func TestPublicRegulationsRemainCitable(t *testing.T) {
	c := libraryCatalog(t)
	for _, doc := range c.Regulations.documents {
		page, err := c.Regulations.Clauses(libStr(doc["id"]), "", 0, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, clause := range libRows(page["items"]) {
			key := libStr(clause["key"])
			entry, err := c.Get(key)
			if err != nil {
				t.Fatal(err)
			}
			if !libBool(entry["description"]) || libBool(entry["withdrawn"]) {
				t.Fatal("public regulation withdrawn")
			}
			if err := c.ValidateReferences("select-regulations", Object{"references": []any{key}}); err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateReferences("specify-requirement", Object{"standard_refs": []any{key}}); err != nil {
				t.Fatal(err)
			}
		}
	}
}
