package assurance

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func TestWirelessLayoutPersistsWithoutInvalidatingAssessment(t *testing.T) {
	h, repo, _ := httpTestServer(t)
	id := httpCreate(t, h)
	architecture := genericArchitecture(t)
	layout := Object{"source_offset": Object{"x": -40.5, "y": 150}, "target_offset": Object{"x": 0, "y": 140}, "source_side": "bottom", "target_side": "left", "waypoint": Object{"x": 320, "y": 380.5}}
	dObj(dList(architecture["flows"])[0])["wireless_layout"] = layout
	response := httpCall(h, "PUT", "/api/v1/workspaces/"+id+"/architecture", httpGenericBody(t, architecture), map[string]string{"If-Match": `"0"`, "Idempotency-Key": "wireless-layout-persist"})
	if response.Code != 200 {
		t.Fatalf("save: %d %s", response.Code, response.Body.String())
	}
	ws, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	saved := dObj(dList(dObj(ws["architecture"])["flows"])[0])
	if decimalRat(dObj(dObj(saved["wireless_layout"])["source_offset"])["x"]).Cmp(decimalRat(-40.5)) != 0 {
		t.Fatalf("manual layout not retained: %v", saved)
	}
	hash := AssessmentHash(ws, genericScenario())
	delete(saved, "wireless_layout")
	if AssessmentHash(ws, genericScenario()) != hash {
		t.Fatal("visual-only wireless layout invalidated assessment")
	}
	if _, exists := saved["wireless_layout"]; exists {
		t.Fatal("test removal failed")
	}
	persisted, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := dObj(dList(dObj(persisted["architecture"])["flows"])[0])["wireless_layout"]; !exists {
		t.Fatal("read mutated persisted layout")
	}
}
func TestWirelessLayoutRejectsInvalidCoordinatesAndFields(t *testing.T) {
	for i, layout := range []Object{
		{"source_offset": Object{"x": 10001}}, {"target_offset": Object{"y": math.Inf(1)}},
		{"waypoint": Object{"x": math.NaN()}}, {"source_side": "diagonal"},
		{"target_side": "auto"}, {"waypoint": Object{"x": "50"}}, {"unknown": 1},
	} {
		architecture := genericArchitecture(t)
		dObj(dList(architecture["flows"])[0])["wireless_layout"] = layout
		if _, err := ValidatePayload("replace-architecture", architecture); err == nil {
			t.Fatal(fmt.Sprintf("invalid layout %d accepted", i))
		}
	}
}
