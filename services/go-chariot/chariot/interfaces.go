package chariot

import (
	"fmt"
	"slices"
)

// Signature metadata is immutable to scripts. Methods include V for self in the
// stored function signature; structural requirements contain public args only.
type Signature struct {
	Parameters []string
	Return     string
}
type InterfaceValue struct {
	Name         string
	requirements map[string]*Signature
}

func (s *Signature) String() string      { return fmt.Sprintf("signature(%v, %s)", s.Parameters, s.Return) }
func (i *InterfaceValue) String() string { return "interface(" + i.Name + ")" }
func copySignature(s *Signature) *Signature {
	if s == nil {
		return nil
	}
	return &Signature{append([]string{}, s.Parameters...), s.Return}
}
func validContract(s string) bool {
	switch s {
	case "V", "N", "S", "L", "D", "A", "M", "F", "O", "J", "X", "T", "P", "E":
		return true
	}
	return false
}
func checkContract(typ string, v Value) error {
	if !validContract(typ) {
		return fmt.Errorf("invalid contract %q", typ)
	}
	if typ == "V" {
		return nil
	}
	if typ == "O" {
		if i, ok := v.(*InstanceValue); ok && i != nil {
			return nil
		}
	}
	if typ == "F" {
		if _, ok := v.(Callable); ok {
			return nil
		}
	}
	if typ == "D" {
		if s, ok := v.(Str); !ok || !IsDateString(string(s)) {
			return fmt.Errorf("expected date")
		}
		return nil
	}
	if isNilMemberReceiver(v) {
		return fmt.Errorf("expected %s, got null", typ)
	}
	return validateTypeCompatibility(typ, v)
}
func (s *Signature) checkArgs(args []Value) error {
	if len(args) != len(s.Parameters) {
		return fmt.Errorf("signature expects %d arguments, got %d", len(s.Parameters), len(args))
	}
	for i, typ := range s.Parameters {
		if err := checkContract(typ, args[i]); err != nil {
			return fmt.Errorf("argument %d: %w", i+1, err)
		}
	}
	return nil
}
func conformance(v Value, in *InterfaceValue) error {
	i, ok := v.(*InstanceValue)
	if !ok || i == nil {
		return fmt.Errorf("%s requires a script instance (host/map conformance unsupported)", in.Name)
	}
	for name, want := range in.requirements {
		fn, ok := i.Class.methods[name]
		if !ok {
			return fmt.Errorf("%s: missing method %s", in.Name, name)
		}
		if err := validateMethod(fn); err != nil {
			return fmt.Errorf("%s.%s: %w", i.Class.Name, name, err)
		}
		if fn.Signature == nil {
			return fmt.Errorf("%s.%s: signature metadata required", i.Class.Name, name)
		}
		got := fn.Signature
		if len(got.Parameters) == 0 || !slices.Equal(got.Parameters[1:], want.Parameters) || got.Return != want.Return {
			return fmt.Errorf("%s.%s: signature mismatch", i.Class.Name, name)
		}
	}
	return nil
}
func RegisterInterfaces(rt *Runtime) {
	rt.Register("signature", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("signature requires array of type codes and return code")
		}
		params, ok := args[0].(*ArrayValue)
		if !ok || params == nil {
			return nil, fmt.Errorf("signature parameters must be array")
		}
		ret, ok := args[1].(Str)
		if !ok || !validContract(string(ret)) {
			return nil, fmt.Errorf("invalid return contract")
		}
		s := &Signature{Parameters: make([]string, len(params.Elements)), Return: string(ret)}
		for i, p := range params.Elements {
			typ, ok := p.(Str)
			if !ok || !validContract(string(typ)) {
				return nil, fmt.Errorf("invalid parameter contract %v", p)
			}
			s.Parameters[i] = string(typ)
		}
		return s, nil
	})
	rt.Register("withSignature", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("withSignature requires function and signature")
		}
		fn, ok := args[0].(*FunctionValue)
		if !ok || fn == nil {
			return nil, fmt.Errorf("withSignature requires function")
		}
		s, ok := args[1].(*Signature)
		if !ok || s == nil || len(s.Parameters) != len(fn.Parameters) {
			return nil, fmt.Errorf("signature parameter count must match function")
		}
		copy := *fn
		copy.Signature = copySignature(s)
		return &copy, nil
	})
	for _, name := range []string{"satisfies", "assertSatisfies"} {
		assert := name == "assertSatisfies"
		rt.Register(name, func(args ...Value) (Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("conformance requires value and interface")
			}
			in, ok := args[1].(*InterfaceValue)
			if !ok || in == nil {
				return nil, fmt.Errorf("expected interface")
			}
			err := conformance(args[0], in)
			if assert {
				if err != nil {
					return nil, err
				}
				return args[0], nil
			}
			return Bool(err == nil), nil
		})
	}
}
