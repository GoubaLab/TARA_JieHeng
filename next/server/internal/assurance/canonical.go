package assurance

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Object = map[string]any
type ValidationError struct{ Detail string }

func (e ValidationError) Error() string { return e.Detail }
func dError(s string) error             { return ValidationError{Detail: s} }

// DecodeJSON retains Python's distinction between integer and float JSON literals.
func DecodeJSON(data []byte) (Object, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	raw, err := dDecodeValue(dec)
	if err != nil {
		return nil, dError("JSON 格式无效: " + err.Error())
	}
	value, ok := raw.(map[string]any)
	if !ok || value == nil {
		return nil, dError("JSON 必须为对象")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, dError("JSON 后存在额外内容")
	}
	return value, nil
}

func dDecodeValue(dec *json.Decoder) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, composite := token.(json.Delim)
	if !composite {
		if number, ok := token.(json.Number); ok && strings.ContainsAny(string(number), ".eE") {
			value, err := strconv.ParseFloat(string(number), 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, dError("JSON 数值超出有限数范围")
			}
		}
		return token, nil
	}
	switch delim {
	case '{':
		value := Object{}
		for dec.More() {
			raw, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := raw.(string)
			if !ok {
				return nil, dError("JSON 对象键必须为字符串")
			}
			if _, exists := value[key]; exists {
				return nil, dError("JSON 对象包含重复键: " + key)
			}
			entry, err := dDecodeValue(dec)
			if err != nil {
				return nil, err
			}
			value[key] = entry
		}
		if end, err := dec.Token(); err != nil || end != json.Delim('}') {
			return nil, dError("JSON 对象未正确结束")
		}
		return value, nil
	case '[':
		value := []any{}
		for dec.More() {
			entry, err := dDecodeValue(dec)
			if err != nil {
				return nil, err
			}
			value = append(value, entry)
		}
		if end, err := dec.Token(); err != nil || end != json.Delim(']') {
			return nil, dError("JSON 列表未正确结束")
		}
		return value, nil
	default:
		return nil, dError("JSON 分隔符无效")
	}
}
func Stamp() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00") }
func Identity(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
func Clone(value Object) Object {
	if value == nil {
		return nil
	}
	return dClone(value).(map[string]any)
}
func dClone(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := Object{}
		for k, v := range x {
			out[k] = dClone(v)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = dClone(v)
		}
		return out
	case []string:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = v
		}
		return out
	case []Object:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = Clone(v)
		}
		return out
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			out := make([]any, rv.Len())
			for i := range out {
				out[i] = dClone(rv.Index(i).Interface())
			}
			return out
		}
		return v
	}
}

// CanonicalJSON intentionally matches Python json.dumps(sort_keys=True,
// ensure_ascii=False), including separator spaces and Python float notation.
func CanonicalJSON(value any) ([]byte, error) {
	var b bytes.Buffer
	err := dEncode(&b, value)
	return b.Bytes(), err
}
func Digest(value any) string {
	b, err := CanonicalJSON(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func dQuote(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
func dFloat(v float64) (string, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "", dError("数值必须有限")
	}
	scientific := strconv.FormatFloat(v, 'e', -1, 64)
	at := strings.LastIndexByte(scientific, 'e')
	exponent, _ := strconv.Atoi(scientific[at+1:])
	if exponent >= -4 && exponent < 16 {
		s := strconv.FormatFloat(v, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s, nil
	}
	return scientific, nil
}
func dEncode(b *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if v {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		dQuote(b, v)
	case json.Number:
		s := string(v)
		if !strings.ContainsAny(s, ".eE") {
			n, ok := new(big.Int).SetString(s, 10)
			if !ok {
				return dError("无效整数")
			}
			b.WriteString(n.String())
		} else {
			f, e := strconv.ParseFloat(s, 64)
			if e != nil {
				return dError("无效数值")
			}
			s, e = dFloat(f)
			if e != nil {
				return e
			}
			b.WriteString(s)
		}
	case float64:
		s, e := dFloat(v)
		if e != nil {
			return e
		}
		b.WriteString(s)
	case float32:
		s, e := dFloat(float64(v))
		if e != nil {
			return e
		}
		b.WriteString(s)
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			dQuote(b, k)
			b.WriteString(": ")
			if err := dEncode(b, v[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			b.WriteString(strconv.FormatInt(rv.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			b.WriteString(strconv.FormatUint(rv.Uint(), 10))
		case reflect.Slice, reflect.Array:
			b.WriteByte('[')
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					b.WriteString(", ")
				}
				if err := dEncode(b, rv.Index(i).Interface()); err != nil {
					return err
				}
			}
			b.WriteByte(']')
		default:
			return dError(fmt.Sprintf("不支持的 JSON 类型 %T", value))
		}
	}
	return nil
}
func dObj(v any) Object {
	if x, ok := v.(map[string]any); ok {
		return x
	}
	return Object{}
}
func dList(v any) []any {
	if x, ok := v.([]any); ok {
		return x
	}
	if v == nil {
		return []any{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return []any{}
}
func dStr(v any) string { s, _ := v.(string); return s }
func dBool(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case json.Number:
		f, _ := x.Float64()
		return f != 0
	case int:
		return x != 0
	case float64:
		return x != 0
	}
	return v != nil
}
func dInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		f, _ := x.Float64()
		return int(f)
	}
	return 0
}
func dHas(values any, needle any) bool {
	for _, v := range dList(values) {
		if reflect.DeepEqual(v, needle) {
			return true
		}
	}
	return false
}
func dFind(rows []any, key any) (Object, error) {
	for _, v := range rows {
		row := dObj(v)
		if row["id"] == key {
			return row, nil
		}
	}
	return nil, dError("引用的记录不存在")
}
