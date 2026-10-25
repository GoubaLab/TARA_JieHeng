package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegulationsRetainedAfterCatalogWithdrawal(t *testing.T) {
	r := libraryCatalog(t).Regulations
	total := 0
	for _, doc := range r.documents {
		page, err := r.Clauses(libStr(doc["id"]), "", 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if len(libList(page["items"])) != libInt(doc["clause_count"]) {
			t.Fatal("document clause count mismatch")
		}
		total += libInt(doc["clause_count"])
		index := 0
		for _, artifact := range libRows(doc["artifacts"]) {
			if !strings.HasSuffix(strings.ToLower(libStr(artifact["file"])), ".pdf") {
				continue
			}
			file, err := r.Source(libStr(doc["id"]), index)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) != artifact["sha256"] {
				t.Fatal("public source changed")
			}
			index++
		}
	}
	if libInt(r.Summary()["total_clauses"]) != total {
		t.Fatal("public regulation inventory changed")
	}
}

func TestEmptyRegulationManifestSupportsSourceOnlyClone(t *testing.T) {
	directory := t.TempDir()
	manifest := []byte(`{"schema_version":1,"verified_at":null,"documents":[]}`)
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := NewRegulations(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(libList(r.Summary()["documents"])) != 0 || libInt(r.Summary()["total_clauses"]) != 0 {
		t.Fatal("empty source distribution invented regulations")
	}
	if _, err = r.Get("reg:synthetic:1"); !errors.Is(err, ErrReferenceNotFound) {
		t.Fatalf("unconfigured clause: %v", err)
	}
	if _, err = r.Source("synthetic", 0); !errors.Is(err, ErrReferenceNotFound) {
		t.Fatalf("unconfigured original: %v", err)
	}
	for _, invalid := range []string{`{"schema_version":1}`, `{"schema_version":1,"documents":null}`, `{"schema_version":1,"documents":{}}`} {
		if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewRegulations(directory); err == nil {
			t.Fatal("malformed manifest silently became an empty library")
		}
	}
	if _, err := NewRegulations(t.TempDir()); err == nil {
		t.Fatal("missing manifest silently became an empty library")
	}
}
func TestRegulationIntegrityFailures(t *testing.T) {
	for _, mode := range []string{"valid", "tamper", "wrong_count", "partial_as_complete", "missing_parent", "cycle", "cross_version", "path_escape", "symlink_escape"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			rows := []Object{{"key": "reg:test:1", "document_id": "test", "number": "1", "body": "正式正文", "parent": nil}}
			doc := Object{"id": "test", "coverage": "partial", "clause_count": 1, "clauses_file": "clauses.json", "source_url": "https://example.invalid/reference", "edition": "fixture", "code": "TEST"}
			switch mode {
			case "wrong_count":
				doc["clause_count"] = 2
			case "partial_as_complete":
				doc["coverage"] = "complete_for_edition"
			case "missing_parent":
				rows[0]["parent"] = "reg:test:missing"
			case "cycle":
				rows[0]["parent"] = "reg:test:1"
			case "cross_version":
				rows[0]["document_id"] = "other"
			}
			body, _ := json.Marshal(rows)
			sum := sha256.Sum256(body)
			doc["clauses_sha256"] = hex.EncodeToString(sum[:])
			file := filepath.Join(directory, "clauses.json")
			if e := os.WriteFile(file, body, 0600); e != nil {
				t.Fatal(e)
			}
			if mode == "path_escape" || mode == "symlink_escape" {
				outside := filepath.Join(t.TempDir(), "outside.json")
				if e := os.WriteFile(outside, body, 0600); e != nil {
					t.Fatal(e)
				}
				if mode == "path_escape" {
					relative, _ := filepath.Rel(directory, outside)
					doc["clauses_file"] = relative
				} else {
					link := filepath.Join(directory, "link.json")
					if e := os.Symlink(outside, link); e != nil {
						t.Fatal(e)
					}
					doc["clauses_file"] = "link.json"
				}
			}
			manifest, _ := json.Marshal(Object{"schema_version": 1, "documents": []any{doc}})
			_ = os.WriteFile(filepath.Join(directory, "manifest.json"), manifest, 0600)
			if mode == "tamper" {
				_ = os.WriteFile(file, []byte("[]"), 0600)
			}
			_, e := NewRegulations(directory)
			if (e == nil) != (mode == "valid") {
				t.Fatalf("mode %s: %v", mode, e)
			}
		})
	}
}
