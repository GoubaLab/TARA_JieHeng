package assurance

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// These namespaces support identifiers retained by existing project records.
// Catalog entries are unavailable and cannot be assigned to new records.
var withdrawnCategories = map[string]bool{"systems": true, "losses": true, "patterns": true, "techniques": true, "controls": true, "relationships": true, "checks": true, "methods": true, "criteria": true, "collections": true, "gb": true, "r155": true}
var ErrReferenceWithdrawn = errors.New("引用已失效，请选择可用资料")
var ErrReferenceDataUnavailable = errors.New("参考目录未配置，当前操作不可用")

type Catalog struct{ Regulations *Regulations }

// The first argument remains for source compatibility; it is never opened or inspected.
func NewCatalog(_ string, regulationsDir string) (*Catalog, error) {
	reg, err := NewRegulations(regulationsDir)
	if err != nil {
		return nil, err
	}
	return &Catalog{Regulations: reg}, nil
}
func (c *Catalog) Summary() Object {
	return Object{"categories": []Object{}, "gaps": Object{"relationships": 0, "missing_control_ids": 0, "references": 0}, "origin": "未配置参考资料", "network_sync": false, "withdrawn": true}
}
func (c *Catalog) Search(category, q string, offset, limit int) (Object, error) {
	if offset < 0 || limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("分页参数无效")
	}
	if !withdrawnCategories[category] {
		return nil, libMissing(category)
	}
	return Object{"items": []Object{}, "total": 0, "offset": offset, "limit": limit, "withdrawn": true}, nil
}
func withdrawnKey(key string) bool {
	category, id, found := strings.Cut(key, ":")
	return found && id != "" && withdrawnCategories[category]
}
func (c *Catalog) Get(key string) (Object, error) {
	if strings.HasPrefix(key, "reg:") {
		clause, err := c.Regulations.Get(key)
		if err != nil {
			return nil, err
		}
		doc := c.Regulations.byDocument[libStr(clause["document_id"])]
		var parent any
		if libBool(clause["parent"]) {
			p, e := c.Regulations.Get(libStr(clause["parent"]))
			if e != nil {
				return nil, e
			}
			parent = p["number"]
		}
		body := libFirst(clause["body"], clause["description"])
		clause["title"] = libStr(doc["code"]) + " · " + libStr(doc["edition"]) + " · " + libStr(clause["number"])
		clause["description"] = body
		clause["document_coverage"] = doc["coverage"]
		clause["facts"] = Object{"文档": doc["title"], "版本": doc["edition"], "条号": clause["number"], "上级章节": parent, "原文": body, "正文来源": libFirst(clause["source_url"], doc["source_url"]), "原件说明": doc["source_note"], "原文核对说明": clause["source_note"], "版本收录说明": doc["coverage_note"]}
		return clause, nil
	}
	if withdrawnKey(key) {
		return Object{"key": key, "title": "引用已失效", "description": "", "withdrawn": true, "citable": false}, nil
	}
	return nil, libMissing(key)
}
func libIsList(value any) bool {
	if value == nil {
		return false
	}
	kind := reflect.TypeOf(value).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}
func catalogRecord(rows any, id any) Object {
	key, ok := id.(string)
	if !ok || key == "" {
		return Object{}
	}
	for _, row := range libRows(rows) {
		if row["id"] == key {
			return row
		}
	}
	return Object{}
}
func catalogRefValues(value any) []any {
	if value == nil {
		return []any{}
	}
	if text, ok := value.(string); ok && text == "" {
		return []any{}
	}
	if libIsList(value) {
		return libList(value)
	}
	return []any{value}
}
func (c *Catalog) validateReferenceField(value, previous any, standards bool) error {
	allowed := map[string]bool{}
	for _, v := range catalogRefValues(previous) {
		if key, ok := v.(string); ok {
			allowed[key] = true
		}
	}
	for _, v := range catalogRefValues(value) {
		key, ok := v.(string)
		if !ok {
			return fmt.Errorf("引用标识必须为文本")
		}
		if standards && !strings.HasPrefix(key, "reg:") && !strings.HasPrefix(key, "gb:") && !strings.HasPrefix(key, "r155:") {
			return fmt.Errorf("标准引用必须选择法规数据库中的条款")
		}
		if withdrawnKey(key) {
			if allowed[key] {
				continue
			}
			return fmt.Errorf("%w: %s", ErrReferenceWithdrawn, key)
		}
		if _, err := c.Get(key); err != nil {
			return err
		}
	}
	return nil
}

// ValidateReferences checks references without an existing record context.
func (c *Catalog) ValidateReferences(action string, payload Object) error {
	return c.ValidateReferencesForWorkspace(action, payload, nil)
}

// ValidateReferencesForWorkspace must receive the aggregate read inside the write transaction.
// An unavailable key may remain only on its existing object and reference field, or be removed.
func (c *Catalog) ValidateReferencesForWorkspace(action string, payload, current Object) error {
	if action == "select-regulations" {
		refs, supplied := payload["references"]
		if !supplied {
			refs = []any{}
		}
		if !libIsList(refs) {
			return fmt.Errorf("法规引用必须是条款列表")
		}
		for _, v := range libList(refs) {
			key, ok := v.(string)
			if !ok || !strings.HasPrefix(key, "reg:") {
				return fmt.Errorf("请从法规数据库选择已核对正文的条款")
			}
			if _, err := c.Get(key); err != nil {
				return err
			}
		}
	}
	collection := map[string]string{"record-asset": "assets", "record-scenario": "scenarios", "specify-requirement": "requirements"}[action]
	previous := catalogRecord(current[collection], payload["id"])
	fields := map[string][]string{"record-asset": {"reference"}, "record-scenario": {"references"}, "specify-requirement": {"reference", "standard_refs"}}[action]
	for _, field := range fields {
		if err := c.validateReferenceField(payload[field], previous[field], field == "standard_refs"); err != nil {
			return err
		}
	}
	if action == "record-scenario" {
		for _, path := range libRows(payload["paths"]) {
			old := catalogRecord(previous["paths"], path["id"])
			if err := c.validateReferenceField(path["reference"], old["reference"], false); err != nil {
				return err
			}
		}
	}
	if action == "replace-architecture" {
		oldByID := map[string]Object{}
		for _, node := range libRows(libObj(current["architecture"])["nodes"]) {
			oldByID[libStr(node["id"])] = node
			for _, child := range libRows(node["children"]) {
				oldByID[libStr(child["id"])] = child
			}
		}
		for _, value := range libList(payload["nodes"]) {
			node, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("实体格式无效")
			}
			old := oldByID[libStr(node["id"])]
			if err := c.validateReferenceField(node["reference"], old["reference"], false); err != nil {
				return err
			}
			children, exists := node["children"]
			if exists && !libIsList(children) {
				return fmt.Errorf("实体内部组成必须是列表")
			}
			for _, value := range libList(children) {
				child, ok := value.(map[string]any)
				if !ok {
					return fmt.Errorf("实体内部组成格式无效")
				}
				old := oldByID[libStr(child["id"])]
				if err := c.validateReferenceField(child["reference"], old["reference"], false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
