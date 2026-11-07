package llamacpp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

const MaxRequestBytes = 1 << 20
const MaxResponseBytes = 4 << 20

var errSchema = errors.New("llamacpp: unsupported or invalid phase schema")
var errOutput = errors.New("llamacpp: invalid phase JSON or schema mismatch")

// phaseSchema supports exactly the vocabulary emitted by the three-phase
// planner. Unknown keywords are errors before any generation request. This
// validator supplements, and never substitutes for, LLGuidance qualification.
type phaseSchema struct {
	kind       string
	properties map[string]*phaseSchema
	required   map[string]bool
	enum       map[string]bool
	min, max   int
}

func compileSchema(raw json.RawMessage) (*phaseSchema, error) {
	if len(raw) > MaxRequestBytes {
		return nil, errSchema
	}
	value, err := strictJSON(raw)
	if err != nil {
		return nil, errSchema
	}
	return compileSchemaValue(value)
}

func compileSchemaValue(value any) (*phaseSchema, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errSchema
	}
	kind, ok := object["type"].(string)
	if !ok {
		return nil, errSchema
	}
	s := &phaseSchema{kind: kind, max: -1}
	allowed := map[string]bool{"type": true}
	switch kind {
	case "object":
		for _, key := range []string{"properties", "required", "additionalProperties"} {
			allowed[key] = true
		}
		if additional, ok := object["additionalProperties"].(bool); !ok || additional {
			return nil, errSchema
		}
		properties, ok := object["properties"].(map[string]any)
		if !ok {
			return nil, errSchema
		}
		s.properties = map[string]*phaseSchema{}
		for name, child := range properties {
			compiled, err := compileSchemaValue(child)
			if err != nil {
				return nil, err
			}
			s.properties[name] = compiled
		}
		required, ok := object["required"].([]any)
		if !ok {
			return nil, errSchema
		}
		s.required = map[string]bool{}
		for _, member := range required {
			name, ok := member.(string)
			if !ok || s.required[name] || s.properties[name] == nil {
				return nil, errSchema
			}
			s.required[name] = true
		}
	case "string":
		for _, key := range []string{"enum", "minLength", "maxLength"} {
			allowed[key] = true
		}
		if raw, exists := object["enum"]; exists {
			members, ok := raw.([]any)
			if !ok || len(members) == 0 {
				return nil, errSchema
			}
			s.enum = map[string]bool{}
			for _, member := range members {
				text, ok := member.(string)
				if !ok || s.enum[text] {
					return nil, errSchema
				}
				s.enum[text] = true
			}
		}
		for _, key := range []string{"minLength", "maxLength"} {
			if raw, exists := object[key]; exists {
				number, ok := raw.(json.Number)
				if !ok {
					return nil, errSchema
				}
				n, err := number.Int64()
				if err != nil || n < 0 || n > MaxResponseBytes {
					return nil, errSchema
				}
				if key == "minLength" {
					s.min = int(n)
				} else {
					s.max = int(n)
				}
			}
		}
		if s.max >= 0 && s.min > s.max {
			return nil, errSchema
		}
	default:
		return nil, errSchema
	}
	for key := range object {
		if !allowed[key] {
			return nil, errSchema
		}
	}
	return s, nil
}

func (s *phaseSchema) validate(raw []byte) error {
	if len(raw) > MaxResponseBytes {
		return errOutput
	}
	value, err := strictJSON(raw)
	if err != nil || !s.accepts(value) {
		return errOutput
	}
	return nil
}

func (s *phaseSchema) accepts(value any) bool {
	if s.kind == "string" {
		text, ok := value.(string)
		if !ok {
			return false
		}
		n := utf8.RuneCountInString(text)
		return n >= s.min && (s.max < 0 || n <= s.max) && (s.enum == nil || s.enum[text])
	}
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for key := range s.required {
		if _, exists := object[key]; !exists {
			return false
		}
	}
	for key, child := range object {
		property := s.properties[key]
		if property == nil || !property.accepts(child) {
			return false
		}
	}
	return true
}

// strictJSON rejects duplicate decoded keys at every depth and trailing JSON.
// Neither raw content nor decoder errors are exposed to callers.
func strictJSON(raw []byte) (any, error) {
	if !utf8.Valid(raw) {
		return nil, errOutput
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	value, err := readJSONValue(d, 0)
	if err != nil {
		return nil, errOutput
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errOutput
	}
	return value, nil
}

func readJSONValue(d *json.Decoder, depth int) (any, error) {
	if depth > 64 {
		return nil, errOutput
	}
	token, err := d.Token()
	if err != nil {
		return nil, errOutput
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return token, nil
	}
	switch delimiter {
	case '{':
		object := map[string]any{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return nil, errOutput
			}
			key, ok := token.(string)
			if !ok {
				return nil, errOutput
			}
			if _, exists := object[key]; exists {
				return nil, errOutput
			}
			value, err := readJSONValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		if end, err := d.Token(); err != nil || end != json.Delim('}') {
			return nil, errOutput
		}
		return object, nil
	case '[':
		array := []any{}
		for d.More() {
			value, err := readJSONValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		if end, err := d.Token(); err != nil || end != json.Delim(']') {
			return nil, errOutput
		}
		return array, nil
	default:
		return nil, errOutput
	}
}
