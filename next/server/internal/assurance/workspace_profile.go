package assurance

// Profile edits change descriptive metadata without reprocessing work products.
func applyWorkspaceProfile(original, payload Object) (Object, string, error) {
	profile, err := ValidatePayload("update-workspace-profile", payload)
	if err != nil {
		return nil, "", err
	}
	ws := Clone(original)
	ws["title"] = profile["title"]
	ws["description"] = profile["description"]
	ws["revision"] = dInt(ws["revision"]) + 1
	ws["updated_at"] = domainStamp()
	return ws, dStr(ws["id"]), nil
}
