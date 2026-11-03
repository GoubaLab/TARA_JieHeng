package assurance

// Drafts may be discarded before risk assessment is complete. Accepted decisions
// and their evidence are independent records and must not be altered here.
func discardTreatmentDraft(ws Object, scenarioID string) error {
	if _, err := dFind(dList(ws["scenarios"]), scenarioID); err != nil {
		return ErrNotFound
	}
	kept := []any{}
	found := false
	for _, raw := range dList(ws["treatment_drafts"]) {
		if dObj(raw)["scenario_id"] == scenarioID {
			found = true
		} else {
			kept = append(kept, raw)
		}
	}
	if !found {
		return ErrNotFound
	}
	ws["treatment_drafts"] = kept
	return nil
}
