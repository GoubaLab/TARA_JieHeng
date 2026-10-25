package assurance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var ErrReferenceNotFound = errors.New("引用条目不存在")

func libObj(v any) Object {
	if m, ok := v.(map[string]any); ok && m != nil {
		return m
	}
	return Object{}
}
func libList(v any) []any {
	if v == nil {
		return []any{}
	}
	if a, ok := v.([]any); ok {
		return a
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{}
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out
}
func libRows(v any) []Object {
	out := []Object{}
	for _, x := range libList(v) {
		out = append(out, libObj(x))
	}
	return out
}
func libStr(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "True"
		}
		return "False"
	}
	return fmt.Sprint(v)
}
func libInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	case json.Number:
		n, _ := strconv.Atoi(x.String())
		return n
	}
	n, _ := strconv.Atoi(libStr(v))
	return n
}
func libBool(v any) bool {
	if v == nil {
		return false
	}
	if n, ok := v.(json.Number); ok {
		value, err := n.Float64()
		return err == nil && value != 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() > 0
	case reflect.Int, reflect.Int64:
		return rv.Int() != 0
	case reflect.Float64:
		return rv.Float() != 0
	}
	return true
}
func libClone(v any) any { return dClone(v) }
func libDecodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}
func libCopy(v Object) Object { return libObj(libClone(v)) }
func libFirst(values ...any) any {
	for _, v := range values {
		if libBool(v) {
			return v
		}
	}
	return ""
}
func libSetDefault(m Object, k string, v any) {
	if _, ok := m[k]; !ok {
		m[k] = v
	}
}
func libContains(list any, value any) bool {
	for _, x := range libList(list) {
		if libStr(x) == libStr(value) {
			return true
		}
	}
	return false
}
func libJoin(list any, separator string) string {
	out := []string{}
	for _, v := range libList(list) {
		out = append(out, libStr(v))
	}
	return strings.Join(out, separator)
}
func libFind(rows any, id any) (Object, error) {
	for _, r := range libRows(rows) {
		if libStr(r["id"]) == libStr(id) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("引用的记录不存在")
}
func libTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func libMissing(key string) error { return fmt.Errorf("%w: %s", ErrReferenceNotFound, key) }
