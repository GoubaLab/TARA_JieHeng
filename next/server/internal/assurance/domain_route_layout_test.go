package assurance

import (
	"context"
	"math"
	"testing"
)

func TestRouteLayoutPersistsForAllMediaWithoutChangingAssessment(t *testing.T) {
	for _, medium := range []string{"wired", "wireless", "logical"} {
		t.Run(medium, func(t *testing.T) {
			h, repo, _ := httpTestServer(t)
			id := httpCreate(t, h)
			architecture := genericArchitecture(t)
			flow := dObj(dList(architecture["flows"])[0])
			flow["communication"] = Object{"medium": medium, "technology": "test"}
			baseline := NewWorkspace("layout", "")
			baseline["architecture"] = Clone(architecture)
			flow["route_layout"] = Object{"waypoint": Object{"x": 350, "y": 140}}
			flow["sourceSide"] = "bottom"
			flow["targetSide"] = "left"
			response := httpCall(h, "PUT", "/api/v1/workspaces/"+id+"/architecture", httpGenericBody(t, architecture), map[string]string{"If-Match": `"0"`, "Idempotency-Key": "route-layout"})
			if response.Code != 200 {
				t.Fatalf("save %d %s", response.Code, response.Body.String())
			}
			ws, err := repo.Get(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			saved := dObj(dList(dObj(ws["architecture"])["flows"])[0])
			if decimalRat(dObj(dObj(saved["route_layout"])["waypoint"])["y"]).Cmp(decimalRat(140)) != 0 {
				t.Fatal("route lost")
			}
			baseline["architecture"] = dObj(ws["architecture"])
			hash := AssessmentHash(baseline, genericScenario())
			delete(saved, "route_layout")
			saved["sourceSide"] = nil
			saved["targetSide"] = nil
			if AssessmentHash(baseline, genericScenario()) != hash {
				t.Fatal("geometry invalidated assessment")
			}
		})
	}
}
func TestRouteLayoutValidationAndLegacyCompatibility(t *testing.T) {
	for _, layout := range []Object{{"waypoint": Object{"x": 10001}}, {"waypoint": Object{"y": math.Inf(1)}}, {"waypoint": Object{"x": math.NaN()}}, {"waypoint": Object{"x": "20"}}, {"unknown": 1}} {
		a := genericArchitecture(t)
		dObj(dList(a["flows"])[0])["route_layout"] = layout
		if _, err := ValidatePayload("replace-architecture", a); err == nil {
			t.Fatalf("accepted %v", layout)
		}
	}
	a := genericArchitecture(t)
	f := dObj(dList(a["flows"])[0])
	f["route_layout"] = Object{}
	f["wireless_layout"] = Object{"source_offset": Object{"x": 30, "y": 140}}
	valid, err := ValidatePayload("replace-architecture", a)
	if err != nil {
		t.Fatal(err)
	}
	got := dObj(dList(valid["flows"])[0])
	if _, exists := got["route_layout"]; !exists {
		t.Fatal("empty layout omitted")
	}
	if _, exists := got["wireless_layout"]; !exists {
		t.Fatal("legacy data lost")
	}
}
