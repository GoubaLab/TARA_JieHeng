package assurance

// Presets are project-local templates. Flows retain independent value snapshots;
// editing these templates must not reprocess or invalidate existing assessments.
func applyCommunicationSettings(original, payload Object) (Object, string, error) {
	settings, err := ValidatePayload("configure-communication", payload)
	if err != nil {
		return nil, "", err
	}
	ws := Clone(original)
	ws["communication_settings"] = settings
	ws["revision"] = dInt(ws["revision"]) + 1
	ws["updated_at"] = domainStamp()
	return ws, dStr(ws["id"]), nil
}
