package assurance

// Reference recommendations require a configured catalog adapter.
func Recommendations(ws Object, catalog *Catalog, assetID, reference string) (Object, error) {
	return nil, ErrReferenceDataUnavailable
}
func ResolveSelection(catalog *Catalog, payload Object) (Object, error) {
	return nil, ErrReferenceDataUnavailable
}

// DisplayCandidate returns an unavailable state for unresolved catalog entries.
func DisplayCandidate(candidate, asset, imported Object) Object {
	return Object{"key": candidate["key"], "title": "引用已失效", "damage": "", "threat": "", "property": nil, "property_label": "", "paths": []Object{}, "controls": []Object{}, "provenance": []Object{}, "warnings": []string{ErrReferenceDataUnavailable.Error()}, "already_applied": false, "scenario_id": nil, "selectable": false, "withdrawn": true, "citable": false}
}
