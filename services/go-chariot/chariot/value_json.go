package chariot

import (
	"fmt"
	"reflect"
)

// ValueToJSONChecked projects instance fields as plain JSON objects, excluding
// methods, class definitions and lexical scopes. It is a wire projection, not a
// persistence format. Repeated aliases are allowed; active-path cycles fail.
func ValueToJSONChecked(value Value) (interface{}, error) {
	return projectValueJSON(value, make(map[jsonValueIdentity]bool))
}

type jsonValueIdentity struct {
	typ reflect.Type
	ptr uintptr
}

func projectValueJSON(value Value, active map[jsonValueIdentity]bool) (interface{}, error) {
	if isNilMemberReceiver(value) {
		return nil, nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice {
		key := jsonValueIdentity{rv.Type(), rv.Pointer()}
		if active[key] {
			return nil, fmt.Errorf("JSON projection: cyclic value of type %T", value)
		}
		active[key] = true
		defer delete(active, key)
	}
	project := func(v Value) (interface{}, error) { return projectValueJSON(v, active) }
	projectFields := func(fields map[string]Value) (interface{}, error) {
		out := make(map[string]interface{}, len(fields))
		for name, v := range fields {
			var err error
			out[name], err = project(v)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", name, err)
			}
		}
		return out, nil
	}
	switch v := value.(type) {
	case Number:
		return float64(v), nil
	case Str:
		return string(v), nil
	case Bool:
		return bool(v), nil
	case string, bool, float64, float32, int, int64, int32, uint, uint64, uint32:
		return v, nil
	case *InstanceValue:
		return projectFields(v.fields)
	case *MapValue:
		return projectFields(v.Values)
	case map[string]Value:
		return projectFields(v)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for name, item := range v {
			var err error
			out[name], err = project(item)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *ArrayValue:
		out := make([]interface{}, len(v.Elements))
		for i, item := range v.Elements {
			var err error
			out[i], err = project(item)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case []Value:
		out := make([]interface{}, len(v))
		for i, item := range v {
			var err error
			out[i], err = project(item)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			var err error
			out[i], err = project(item)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *Plan:
		return map[string]interface{}{"type": "plan", "name": v.Name}, nil
	case *FunctionValue:
		name := v.SourceCode
		if name == "" {
			name = "<anonymous>"
		}
		return map[string]interface{}{"type": "function", "name": name}, nil
	case TreeNode:
		return map[string]interface{}{"type": "treenode", "name": v.Name()}, nil
	default:
		return nil, fmt.Errorf("JSON projection: unsupported value %T", value)
	}
}
