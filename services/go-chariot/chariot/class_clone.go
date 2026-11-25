package chariot

import (
	"fmt"
	"reflect"
)

// One memo spans the entire runtime, including scopes and extracted receivers.
// Host resources remain borrowed for runtime clones, but are forbidden defaults.
type scriptCopier struct {
	seen   map[Value]Value
	strict bool
	err    error
	// Default copies isolate captured private frames, but keep the declaration
	// environment live so recursive classes and later lexical bindings resolve.
	lexicalScope *Scope
}

func newScriptCopier(strict bool) *scriptCopier {
	return &scriptCopier{seen: map[Value]Value{}, strict: strict}
}
func copyDefault(v Value, lexicalScope *Scope) (Value, error) {
	c := newScriptCopier(true)
	c.lexicalScope = lexicalScope
	out, err := c.value(v)
	if err == nil {
		err = c.err
	}
	return out, err
}
func (c *scriptCopier) scope(s *Scope) *Scope {
	if s == nil {
		return nil
	}
	if c.strict && s == c.lexicalScope {
		return s
	}
	if v, ok := c.seen[s]; ok {
		return v.(*Scope)
	}
	out := NewScope(nil)
	c.seen[s] = out
	out.parent = c.scope(s.parent)
	for name, entry := range s.vars {
		var err error
		entry.Value, err = c.value(entry.Value)
		if err != nil {
			c.err = err
		}
		out.vars[name] = entry
	}
	return out
}
func (c *scriptCopier) value(v Value) (Value, error) {
	if v != nil {
		rv := reflect.ValueOf(v)
		if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Map) && rv.IsNil() {
			return v, nil
		}
	}
	switch x := v.(type) {
	case nil, Number, Str, Bool, *dbNullType:
		return v, nil
	case *ClassValue:
		if c.strict {
			// Class references are live definitions, not instance-owned state.
			return x, nil
		}
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := &ClassValue{Name: x.Name, fields: map[string]FieldDefinition{}, methods: map[string]*FunctionValue{}}
		c.seen[x] = out
		out.lexicalScope = c.scope(x.lexicalScope)
		for name, f := range x.fields {
			var err error
			f.Default, err = c.value(f.Default)
			if err != nil {
				return nil, err
			}
			out.fields[name] = f
		}
		for name, fn := range x.methods {
			value, err := c.value(fn)
			if err != nil {
				return nil, err
			}
			out.methods[name] = value.(*FunctionValue)
		}
		return out, nil
	case *InstanceValue:
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := &InstanceValue{fields: map[string]Value{}}
		c.seen[x] = out
		cl, err := c.value(x.Class)
		if err != nil {
			return nil, err
		}
		out.Class = cl.(*ClassValue)
		for name, value := range x.fields {
			out.fields[name], err = c.value(value)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *boundScriptMethod:
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := &boundScriptMethod{}
		c.seen[x] = out
		r, err := c.value(x.receiver)
		if err != nil {
			return nil, err
		}
		f, err := c.value(x.function)
		if err != nil {
			return nil, err
		}
		out.receiver, out.function = r.(*InstanceValue), f.(*FunctionValue)
		return out, nil
	case *FunctionValue:
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := *x
		c.seen[x] = &out
		out.Parameters = append([]string(nil), x.Parameters...)
		out.Signature = copySignature(x.Signature)
		out.Scope = c.scope(x.Scope)
		return &out, nil
	case *MapValue:
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := NewMap()
		c.seen[x] = out
		for name, value := range x.Values {
			var err error
			out.Values[name], err = c.value(value)
			if err != nil {
				return nil, err
			}
		}
		if x.meta != nil {
			meta, err := c.value(x.meta)
			if err != nil {
				return nil, err
			}
			out.meta = meta.(*MapValue)
		}
		return out, nil
	case *ArrayValue:
		if out, ok := c.seen[x]; ok {
			return out, nil
		}
		out := &ArrayValue{Elements: make([]Value, len(x.Elements))}
		c.seen[x] = out
		for i, value := range x.Elements {
			var err error
			out.Elements[i], err = c.value(value)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *Signature:
		return copySignature(x), nil
	case *InterfaceValue:
		out := &InterfaceValue{Name: x.Name, requirements: map[string]*Signature{}}
		for name, sig := range x.requirements {
			out.requirements[name] = copySignature(sig)
		}
		return out, nil
	case map[string]Value:
		// Native maps can contain script objects; memoize by map identity.
		return c.nativeMap(x)
	default:
		if c.strict {
			return nil, fmt.Errorf("mutable default of type %T cannot be safely cloned; use named initialization", v)
		}
		return v, nil
	}
}

type nativeMapIdentity uintptr

func (c *scriptCopier) nativeMap(x map[string]Value) (Value, error) {
	key := nativeMapIdentity(reflect.ValueOf(x).Pointer())
	if out, ok := c.seen[key]; ok {
		return out, nil
	}
	out := map[string]Value{}
	c.seen[key] = out
	for name, value := range x {
		var err error
		out[name], err = c.value(value)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func cloneScriptState(src, dst *Runtime) {
	c := newScriptCopier(false)
	dst.globalScope = c.scope(src.globalScope)
	dst.currentScope = c.scope(src.currentScope)
	for name, fn := range src.functions {
		value, _ := c.value(fn)
		dst.functions[name] = value.(*FunctionValue)
	}
	for name, value := range src.namespaces {
		dst.namespaces[name], _ = c.value(value)
	}
	for name, list := range src.lists {
		value, _ := c.value(list)
		dst.lists[name] = value.(map[string]Value)
	}
	for name, table := range src.tables {
		rows := make([]map[string]Value, len(table))
		for i, row := range table {
			value, _ := c.value(row)
			rows[i] = value.(map[string]Value)
		}
		dst.tables[name] = rows
	}
}
