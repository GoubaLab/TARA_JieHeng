package assurance

import (
	"encoding/base64"
	"net/http"
)

func (s *HTTPServer) taraPreview(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	if _, e := s.repository.Get(r.Context(), r.PathValue("workspace_id")); e != nil {
		return e
	}
	payload, e := readPayload(w, r)
	if e != nil {
		return e
	}
	ws, e := buildTaraImport(payload)
	if e != nil {
		return e
	}
	result := taraSummary(ws)
	result["title"], result["method"] = ws["title"], ws["method"]
	return writeJSON(w, r, 200, result)
}
func (s *HTTPServer) taraSource(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	source := dObj(ws["tara_import"])
	if source["id"] != r.PathValue("resource_id") {
		return ErrNotFound
	}
	data, e := base64.StdEncoding.DecodeString(dStr(source["original_base64"]))
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="tara-source.xlsx"`)
	w.WriteHeader(200)
	_, e = w.Write(data)
	return e
}
func (s *HTTPServer) taraImportResource(w http.ResponseWriter, r *http.Request) error {
	if _, e := httpQuery(r); e != nil {
		return e
	}
	ws, e := s.repository.Get(r.Context(), r.PathValue("workspace_id"))
	if e != nil {
		return e
	}
	source := dObj(ws["tara_import"])
	if source["id"] != r.PathValue("resource_id") {
		return ErrNotFound
	}
	result := Clone(source)
	delete(result, "original_base64")
	setETag(w, ws)
	return writeJSON(w, r, 200, result)
}
