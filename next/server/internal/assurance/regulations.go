package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Regulations struct {
	Directory  string
	documents  []Object
	byDocument map[string]Object
	rows       map[string][]Object
	byKey      map[string]Object
	verifiedAt any
}

func checkedLibraryFile(directory, name, digest string) ([]byte, string, error) {
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	if filepath.IsAbs(name) {
		return nil, "", fmt.Errorf("法规文件路径无效: %s", name)
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil {
		return nil, "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("法规文件路径无效: %s", name)
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("法规文件路径无效: %s", name)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	if len(digest) != 64 || hex.EncodeToString(sum[:]) != digest {
		return nil, "", fmt.Errorf("法规文件校验失败: %s", name)
	}
	return data, path, nil
}

func NewRegulations(directory string) (*Regulations, error) {
	raw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var manifest Object
	if err = libDecodeJSON(raw, &manifest); err != nil {
		return nil, err
	}
	if libInt(manifest["schema_version"]) != 1 {
		return nil, fmt.Errorf("不支持的法规数据库格式")
	}
	if !libIsList(manifest["documents"]) {
		return nil, fmt.Errorf("法规版本目录必须是列表；未配置法规时使用空列表")
	}
	r := &Regulations{Directory: directory, documents: []Object{}, byDocument: map[string]Object{}, rows: map[string][]Object{}, byKey: map[string]Object{}, verifiedAt: manifest["verified_at"]}
	identifier := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	for _, doc := range libRows(manifest["documents"]) {
		id := libStr(doc["id"])
		if !identifier.MatchString(id) || r.byDocument[id] != nil {
			return nil, fmt.Errorf("法规版本标识无效或重复")
		}
		coverage := libStr(doc["coverage"])
		if coverage != "metadata_only" && coverage != "partial" && coverage != "complete_for_edition" {
			return nil, fmt.Errorf("法规覆盖状态无效")
		}
		for _, a := range libRows(doc["artifacts"]) {
			if _, _, e := checkedLibraryFile(directory, libStr(a["file"]), libStr(a["sha256"])); e != nil {
				return nil, e
			}
		}
		rows := []Object{}
		if libBool(doc["clauses_file"]) {
			data, _, e := checkedLibraryFile(directory, libStr(doc["clauses_file"]), libStr(doc["clauses_sha256"]))
			if e != nil {
				return nil, e
			}
			if e = libDecodeJSON(data, &rows); e != nil || rows == nil {
				return nil, fmt.Errorf("法规条款文件不是列表")
			}
		}
		if libInt(doc["clause_count"]) != len(rows) {
			return nil, fmt.Errorf("法规条款数量与清单不符")
		}
		if coverage == "metadata_only" && len(rows) > 0 {
			return nil, fmt.Errorf("仅元数据的法规不能包含可引用条款")
		}
		if coverage == "complete_for_edition" && (len(rows) == 0 || !libBool(doc["coverage_evidence"])) {
			return nil, fmt.Errorf("完整版本必须包含条款和覆盖核对说明")
		}
		for _, row := range rows {
			key := libStr(row["key"])
			if !strings.HasPrefix(key, "reg:"+id+":") || r.byKey[key] != nil {
				return nil, fmt.Errorf("法规条款标识无效或重复")
			}
			if libStr(row["document_id"]) != id || !libBool(row["number"]) || !libBool(row["body"]) {
				return nil, fmt.Errorf("法规条款缺少版本、条号或正文")
			}
			libSetDefault(row, "description", row["body"])
			libSetDefault(row, "source_url", doc["source_url"])
			libSetDefault(row, "edition", doc["edition"])
			libSetDefault(row, "code", doc["code"])
			libSetDefault(row, "language", libFirst(doc["language"], "zh-CN"))
			libSetDefault(row, "provenance", Object{"document_id": id, "sha256": doc["clauses_sha256"], "source_url": doc["source_url"], "edition": doc["edition"]})
			r.byKey[key] = row
		}
		r.documents = append(r.documents, doc)
		r.byDocument[id] = doc
		r.rows[id] = rows
	}
	for key, row := range r.byKey {
		seen := map[string]bool{key: true}
		parent := libStr(row["parent"])
		for parent != "" {
			p, ok := r.byKey[parent]
			if !ok || seen[parent] {
				return nil, fmt.Errorf("法规条款父级缺失或存在循环")
			}
			if p["document_id"] != row["document_id"] {
				return nil, fmt.Errorf("法规条款父级不能跨版本")
			}
			seen[parent] = true
			parent = libStr(p["parent"])
		}
	}
	return r, nil
}
func (r *Regulations) Summary() Object {
	return libCopy(Object{"documents": r.documents, "verified_at": r.verifiedAt, "network_sync": false, "total_clauses": len(r.byKey), "coverage_note": "完整仅指列明版本；后续修改单及未取得正文的标准单独列示。"})
}
func (r *Regulations) Clauses(doc, q string, offset, limit int) (Object, error) {
	d, ok := r.byDocument[doc]
	if !ok {
		return nil, libMissing(doc)
	}
	if offset < 0 || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("offset 必须非负，limit 必须在 1 到 1000 之间")
	}
	filtered := []Object{}
	query := dCasefold(strings.TrimSpace(q))
	for _, row := range r.rows[doc] {
		hay := libStr(row["number"]) + " " + libStr(row["title"]) + " " + libStr(row["body"]) + " " + libStr(row["key"])
		if query == "" || strings.Contains(dCasefold(hay), query) {
			filtered = append(filtered, row)
		}
	}
	start := min(offset, len(filtered))
	end := min(start+limit, len(filtered))
	return libCopy(Object{"document": d, "document_id": doc, "items": filtered[start:end], "total": len(filtered), "offset": offset, "limit": limit}), nil
}
func (r *Regulations) Get(key string) (Object, error) {
	row, ok := r.byKey[key]
	if !ok {
		return nil, libMissing(key)
	}
	return libCopy(row), nil
}
func (r *Regulations) Source(doc string, index int) (string, error) {
	d, ok := r.byDocument[doc]
	if !ok {
		return "", libMissing(doc)
	}
	pdfs := []Object{}
	for _, a := range libRows(d["artifacts"]) {
		if strings.HasSuffix(strings.ToLower(libStr(a["file"])), ".pdf") {
			pdfs = append(pdfs, a)
		}
	}
	if index < 0 || index >= len(pdfs) {
		return "", libMissing("该版本正式原文尚未入库")
	}
	a := pdfs[index]
	_, path, e := checkedLibraryFile(r.Directory, libStr(a["file"]), libStr(a["sha256"]))
	return path, e
}
