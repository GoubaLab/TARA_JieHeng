package assurance

import (
	"errors"
	"testing"
)

func TestNoReferenceContentCanBeGenerated(t *testing.T) {
	c := libraryCatalog(t)
	ws := Object{"id": "synthetic-project", "assets": []any{Object{"id": "synthetic-asset", "name": "原创示例资产"}}}
	before := Digest(ws)
	for _, reference := range []string{"", "systems:retired-fixture-a", "future:synthetic"} {
		result, err := Recommendations(ws, c, "synthetic-asset", reference)
		if !errors.Is(err, ErrReferenceDataUnavailable) || result != nil {
			t.Fatalf("recommendation enabled: %v", err)
		}
	}
	for _, payload := range []Object{{}, {"asset_id": "synthetic-asset", "reference": "systems:retired-fixture-a", "relationship_keys": []any{"relationships:retired-fixture-a"}}, {"resolved_candidates": []any{Object{"title": "客户端合成内容"}}}} {
		result, err := ResolveSelection(c, payload)
		if !errors.Is(err, ErrReferenceDataUnavailable) || result != nil {
			t.Fatalf("reference resolution enabled: %v", err)
		}
	}
	view := DisplayCandidate(Object{"key": "relationships:retired-fixture-a", "damage": "不得通过边界的合成正文", "paths": []any{Object{"steps": "不应呈现"}}}, Object{}, nil)
	if libBool(view["selectable"]) || view["damage"] != "" || len(libList(view["paths"])) != 0 || len(libList(view["provenance"])) != 0 {
		t.Fatal("candidate disclosed source content")
	}
	if Digest(ws) != before {
		t.Fatal("disabled recommendation mutated project")
	}
}
