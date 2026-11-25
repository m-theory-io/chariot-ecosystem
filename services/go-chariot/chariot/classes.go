package chariot

import "fmt"

// ClassValue owns a shared, additive-only method table. Script fields are private
// to InstanceValue so all writes pass through their declared contracts.
type ClassValue struct {
	Name         string
	fields       map[string]FieldDefinition
	methods      map[string]*FunctionValue
	lexicalScope *Scope
}
type FieldDefinition struct {
	Type     string
	Default  Value
	Required bool
}
type InstanceValue struct {
	Class  *ClassValue
	fields map[string]Value
}
type boundScriptMethod struct {
	receiver *InstanceValue
	function *FunctionValue
}

func (c *ClassValue) String() string        { return "class(" + c.Name + ")" }
func (i *InstanceValue) String() string     { return "instance(" + i.Class.Name + ")" }
func (m *boundScriptMethod) String() string { return "bound script method" }
func (i *InstanceValue) ResolveMember(name string) (Value, error) {
	if v, ok := i.fields[name]; ok {
		return v, nil
	}
	if fn, ok := i.Class.methods[name]; ok {
		return &boundScriptMethod{i, fn}, nil
	}
	return nil, fmt.Errorf("%s: unknown member %q", i.Class.Name, name)
}
func (i *InstanceValue) SetMember(name string, value Value) (Value, error) {
	f, ok := i.Class.fields[name]
	if !ok {
		return nil, fmt.Errorf("%s: unknown field %q", i.Class.Name, name)
	}
	if err := checkContract(f.Type, value); err != nil {
		return nil, fmt.Errorf("field %s: %w", name, err)
	}
	i.fields[name] = value
	return value, nil
}
func (m *boundScriptMethod) Invoke(rt *Runtime, args []Value) (Value, error) {
	return executeFunctionValue(rt, m.function, append([]Value{m.receiver}, args...))
}

// Declarations deliberately reuse FuncCall/Block AST encoding. Their bodies are
// declarative, never evaluated as ordinary executable blocks.
func executeTypeDeclaration(rt *Runtime, call *FuncCall) (Value, error) {
	if len(call.Args) != 2 {
		return nil, fmt.Errorf("%s requires (Name) { declarations }", call.Name)
	}
	name, ok := call.Args[0].(*VarRef)
	if !ok {
		return nil, fmt.Errorf("%s name must be an identifier", call.Name)
	}
	body, ok := call.Args[1].(*Block)
	if !ok {
		return nil, fmt.Errorf("%s requires a declaration block", call.Name)
	}
	if _, exists := rt.currentScope.Get(name.Name); exists {
		return nil, fmt.Errorf("name %s already declared", name.Name)
	}
	c := &ClassValue{Name: name.Name, fields: map[string]FieldDefinition{}, methods: map[string]*FunctionValue{}, lexicalScope: rt.currentScope}
	in := &InterfaceValue{Name: name.Name, requirements: map[string]*Signature{}}
	for _, stmt := range body.Stmts {
		decl, ok := stmt.(*FuncCall)
		if !ok {
			return nil, fmt.Errorf("%s: expected declaration", name.Name)
		}
		if call.Name == "class" && decl.Name != "field" || call.Name == "interface" && decl.Name != "requires" {
			return nil, fmt.Errorf("%s: invalid declaration %s", name.Name, decl.Name)
		}
		if len(decl.Args) < 2 || len(decl.Args) > 3 {
			return nil, fmt.Errorf("%s requires name and contract (field default optional)", decl.Name)
		}
		lit, ok := decl.Args[0].(*Literal)
		if !ok {
			return nil, fmt.Errorf("%s name must be a symbol or string", decl.Name)
		}
		member, ok := lit.Val.(Str)
		if !ok || member == "" {
			return nil, fmt.Errorf("invalid member name")
		}
		contract, err := decl.Args[1].Exec(rt)
		if err != nil {
			return nil, err
		}
		if call.Name == "interface" {
			sig, ok := contract.(*Signature)
			if !ok || len(decl.Args) != 2 {
				return nil, fmt.Errorf("requires expects name and signature")
			}
			if _, exists := in.requirements[string(member)]; exists {
				return nil, fmt.Errorf("duplicate requirement %s", member)
			}
			in.requirements[string(member)] = copySignature(sig)
			continue
		}
		typ, ok := contract.(Str)
		if !ok || !validContract(string(typ)) {
			return nil, fmt.Errorf("invalid field type %v", contract)
		}
		if _, exists := c.fields[string(member)]; exists {
			return nil, fmt.Errorf("duplicate field %s", member)
		}
		field := FieldDefinition{Type: string(typ), Required: len(decl.Args) == 2}
		if !field.Required {
			field.Default, err = decl.Args[2].Exec(rt)
			if err != nil {
				return nil, err
			}
			if err := checkContract(field.Type, field.Default); err != nil {
				return nil, err
			}
			// Reject resources that cannot be independently copied at declaration time.
			field.Default, err = copyDefault(field.Default, c.lexicalScope)
			if err != nil {
				return nil, err
			}
		}
		c.fields[string(member)] = field
	}
	var value Value = c
	if call.Name == "interface" {
		value = in
	}
	rt.currentScope.Set(name.Name, value)
	return value, nil
}

func RegisterClasses(rt *Runtime) {
	rt.Register("new", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("new requires class and optional mapValue attributes")
		}
		c, ok := args[0].(*ClassValue)
		if !ok || c == nil {
			return nil, fmt.Errorf("new requires a class")
		}
		attrs := NewMap()
		if len(args) == 2 {
			attrs, ok = args[1].(*MapValue)
			if !ok || attrs == nil {
				return nil, fmt.Errorf("new attributes must be mapValue")
			}
		}
		for name := range attrs.Values {
			if _, ok := c.fields[name]; !ok {
				return nil, fmt.Errorf("%s: unknown field %s", c.Name, name)
			}
		}
		i := &InstanceValue{Class: c, fields: map[string]Value{}}
		copier := newScriptCopier(true)
		copier.lexicalScope = c.lexicalScope
		for name, f := range c.fields {
			v, exists := attrs.Values[name]
			if !exists {
				if f.Required {
					return nil, fmt.Errorf("%s: required field %s", c.Name, name)
				}
				var err error
				v, err = copier.value(f.Default)
				if err != nil {
					return nil, err
				}
			}
			if _, err := i.SetMember(name, v); err != nil {
				return nil, err
			}
		}
		return i, nil
	})
	rt.Register("method", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("method requires class, name, func(self, ...)")
		}
		c, ok := args[0].(*ClassValue)
		if !ok || c == nil {
			return nil, fmt.Errorf("method requires class")
		}
		name, ok := args[1].(Str)
		if !ok || name == "" {
			return nil, fmt.Errorf("method requires name")
		}
		fn, ok := args[2].(*FunctionValue)
		if !ok {
			return nil, fmt.Errorf("method requires function")
		}
		if err := validateMethod(fn); err != nil {
			return nil, err
		}
		if _, exists := c.fields[string(name)]; exists {
			return nil, fmt.Errorf("method %s collides with field", name)
		}
		if _, exists := c.methods[string(name)]; exists {
			return nil, fmt.Errorf("method %s already declared", name)
		}
		copy := *fn
		copy.Parameters = append([]string(nil), fn.Parameters...)
		copy.Signature = copySignature(fn.Signature)
		c.methods[string(name)] = &copy
		return c, nil
	})
	RegisterInterfaces(rt)
}

// Registration, persistence, and conformance must enforce the same receiver
// contract, including the annotation's implicit-to-callers self argument.
func validateMethod(fn *FunctionValue) error {
	if fn == nil || fn.Body == nil || len(fn.Parameters) == 0 || fn.Parameters[0] != "self" {
		return fmt.Errorf("method function must start with explicit self parameter")
	}
	if fn.Signature != nil && (len(fn.Signature.Parameters) != len(fn.Parameters) || fn.Signature.Parameters[0] != "V") {
		return fmt.Errorf("method signature must include V for self")
	}
	return nil
}
