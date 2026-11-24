package chariot

import (
	"encoding/json"
	"fmt"
)

// ScriptValueToNative is the checked persistence boundary. Cycles and lexical
// closures cannot be represented by the legacy value format and fail explicitly.
// Runtime cloning, in contrast, preserves both. This format never saves hosts.
func ScriptValueToNative(value Value) (interface{}, error) {
	return encodeScriptValue(value, map[Value]bool{})
}
func encodeScriptValue(value Value, active map[Value]bool) (interface{}, error) {
	if value == nil || isNilMemberReceiver(value) {
		return nil, nil
	}
	switch value.(type) {
	case *ClassValue, *InstanceValue, *boundScriptMethod, *MapValue, *ArrayValue, *FunctionValue:
		if active[value] {
			return nil, fmt.Errorf("script persistence: aliased/cyclic value graph is unsupported; persist source declarations instead")
		}
		active[value] = true
	}
	encode := func(v Value) (interface{}, error) { return encodeScriptValue(v, active) }
	obj := map[string]interface{}{}
	switch x := value.(type) {
	case Number:
		return float64(x), nil
	case Str:
		return string(x), nil
	case Bool:
		return bool(x), nil
	case *Signature:
		return map[string]interface{}{"_value_type": "signature", "parameters": x.Parameters, "return": x.Return}, nil
	case *InterfaceValue:
		obj["_value_type"], obj["name"] = "interface", x.Name
		reqs := map[string]interface{}{}
		for name, sig := range x.requirements {
			reqs[name], _ = encode(sig)
		}
		obj["requirements"] = reqs
	case *FunctionValue:
		if x.Scope != nil {
			return nil, fmt.Errorf("script persistence: captured function scope unsupported; persist source declarations instead")
		}
		if x.Body == nil {
			return nil, fmt.Errorf("script persistence: function has no executable body")
		}
		obj["_value_type"], obj["parameters"], obj["source"], obj["body"] = "function", x.Parameters, x.SourceCode, serializeNode(x.Body)
		if x.Signature != nil {
			obj["signature"], _ = encode(x.Signature)
		}
	case *ClassValue:
		obj["_value_type"], obj["name"] = "class", x.Name
		fields := map[string]interface{}{}
		methods := map[string]interface{}{}
		for name, f := range x.fields {
			def, err := encode(f.Default)
			if err != nil {
				return nil, err
			}
			fields[name] = map[string]interface{}{"type": f.Type, "required": f.Required, "default": def}
		}
		for name, fn := range x.methods {
			v, err := encode(fn)
			if err != nil {
				return nil, fmt.Errorf("method %s: %w", name, err)
			}
			methods[name] = v
		}
		obj["fields"], obj["methods"] = fields, methods
	case *InstanceValue:
		cl, err := encode(x.Class)
		if err != nil {
			return nil, err
		}
		fields := map[string]interface{}{}
		for name, value := range x.fields {
			fields[name], err = encode(value)
			if err != nil {
				return nil, err
			}
		}
		obj["_value_type"], obj["class"], obj["fields"] = "instance", cl, fields
	case *boundScriptMethod:
		return nil, fmt.Errorf("script persistence: bound receiver unsupported; persist source declarations instead")
	case *ArrayValue:
		out := make([]interface{}, len(x.Elements))
		for i, value := range x.Elements {
			var err error
			out[i], err = encode(value)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *MapValue:
		if x.meta != nil {
			return nil, fmt.Errorf("script persistence: map metadata unsupported")
		}
		out := map[string]interface{}{}
		for name, value := range x.Values {
			var err error
			out[name], err = encode(value)
			if err != nil {
				return nil, err
			}
		}
		obj["_value_type"], obj["entries"] = "script_map", out
	default:
		return nil, fmt.Errorf("script persistence: unsupported value %T", value)
	}
	return obj, nil
}

// Persistence errors remain errors through the old non-error-returning hooks.
// Callers that need a conventional error result should use ScriptValueToNative.
func scriptNativeHook(value Value) interface{} {
	native, err := ScriptValueToNative(value)
	if err != nil {
		return map[string]interface{}{"_value_type": "script_error", "error": err.Error()}
	}
	return native
}
func decodeScriptNative(obj map[string]interface{}) (Value, error) {
	// Normalize direct native calls as well as JSON-decoded data.
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var normalized map[string]interface{}
	if err = json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	return decodeScriptObject(normalized)
}
func decodeScriptObject(obj map[string]interface{}) (Value, error) {
	kind, _ := obj["_value_type"].(string)
	name, _ := obj["name"].(string)
	switch kind {
	case "script_error":
		return nil, fmt.Errorf("%v", obj["error"])
	case "signature":
		params, ok := obj["parameters"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid persisted signature parameters")
		}
		ret, _ := obj["return"].(string)
		if !validContract(ret) {
			return nil, fmt.Errorf("invalid persisted return type")
		}
		s := &Signature{Parameters: make([]string, len(params)), Return: ret}
		for i, p := range params {
			typ, ok := p.(string)
			if !ok || !validContract(typ) {
				return nil, fmt.Errorf("invalid persisted parameter")
			}
			s.Parameters[i] = typ
		}
		return s, nil
	case "interface":
		i := &InterfaceValue{Name: name, requirements: map[string]*Signature{}}
		reqs, ok := obj["requirements"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid interface requirements")
		}
		for name, raw := range reqs {
			sig, ok := convertFromNativeValue(raw).(*Signature)
			if !ok {
				return nil, fmt.Errorf("invalid requirement %s", name)
			}
			i.requirements[name] = sig
		}
		return i, nil
	case "class":
		c := &ClassValue{Name: name, fields: map[string]FieldDefinition{}, methods: map[string]*FunctionValue{}}
		fields, ok := obj["fields"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid class fields")
		}
		for name, raw := range fields {
			f, ok := raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid field %s", name)
			}
			typ, _ := f["type"].(string)
			required, _ := f["required"].(bool)
			if !validContract(typ) {
				return nil, fmt.Errorf("invalid field type")
			}
			v := convertFromNativeValue(f["default"])
			if err, ok := v.(error); ok {
				return nil, err
			}
			if !required {
				if err := checkContract(typ, v); err != nil {
					return nil, err
				}
			}
			c.fields[name] = FieldDefinition{Type: typ, Required: required, Default: v}
		}
		methods, ok := obj["methods"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid class methods")
		}
		for name, raw := range methods {
			fn, ok := convertFromNativeValue(raw).(*FunctionValue)
			if !ok {
				return nil, fmt.Errorf("invalid method %s", name)
			}
			if err := validateMethod(fn); err != nil {
				return nil, fmt.Errorf("invalid method %s: %w", name, err)
			}
			if _, exists := c.fields[name]; exists {
				return nil, fmt.Errorf("method collision %s", name)
			}
			c.methods[name] = fn
		}
		return c, nil
	case "instance":
		c, ok := convertFromNativeValue(obj["class"]).(*ClassValue)
		if !ok {
			return nil, fmt.Errorf("invalid instance class")
		}
		i := &InstanceValue{Class: c, fields: map[string]Value{}}
		fields, ok := obj["fields"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid instance fields")
		}
		if len(fields) != len(c.fields) {
			return nil, fmt.Errorf("instance field count mismatch")
		}
		for name, raw := range fields {
			value := convertFromNativeValue(raw)
			if err, ok := value.(error); ok {
				return nil, err
			}
			if _, err := i.SetMember(name, value); err != nil {
				return nil, err
			}
		}
		return i, nil
	case "script_map":
		entries, ok := obj["entries"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid script map")
		}
		m := NewMap()
		for name, raw := range entries {
			value := convertFromNativeValue(raw)
			if err, ok := value.(error); ok {
				return nil, err
			}
			m.Set(name, value)
		}
		return m, nil
	}
	return nil, fmt.Errorf("unknown script value kind %s", kind)
}
