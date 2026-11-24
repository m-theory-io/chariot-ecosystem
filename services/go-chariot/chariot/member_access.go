package chariot

import (
	"fmt"
	"reflect"
)

// MemberReader is the extension point for values with their own member model.
// Implementations return bound callables only for declared methods; ordinary
// function-valued fields must be returned unchanged. Names are case-sensitive.
type MemberReader interface {
	ResolveMember(name string) (Value, error)
}

// MemberWriter is used by setProp, not by assignment syntax (which is unsupported).
type MemberWriter interface {
	SetMember(name string, value Value) (Value, error)
}

// Callable is implemented by bound methods and other runtime-aware callables.
// Invoke runs inside the caller's execution boundary; it must not acquire the
// non-reentrant Runtime.WithExecution guard again.
type Callable interface {
	Invoke(rt *Runtime, args []Value) (Value, error)
}

// InvokeCallable is shared by expression calls, named callable variables and call.
// Ordinary FunctionValues retain their existing permissive argument binding.
func InvokeCallable(rt *Runtime, value Value, args []Value) (Value, error) {
	if entry, ok := value.(ScopeEntry); ok {
		value = entry.Value
	}
	if isNilMemberReceiver(value) {
		return nil, fmt.Errorf("null is not callable")
	}
	switch fn := value.(type) {
	case *FunctionValue:
		if fn == nil || fn.Body == nil {
			return nil, fmt.Errorf("function has no executable body")
		}
		return executeFunctionValue(rt, fn, args)
	case Callable:
		return fn.Invoke(rt, args)
	default:
		return nil, fmt.Errorf("value of type %T is not callable", value)
	}
}

// ResolveMember uses the same property helpers as getProp. Consequently missing
// MapValue keys yield DBNull, while map/node helpers retain their own errors.
// Host properties retain the legacy case-insensitive field lookup; host methods
// use exact exported Go names and the actual receiver (including pointers).
func ResolveMember(rt *Runtime, receiver Value, name string) (Value, error) {
	if entry, ok := receiver.(ScopeEntry); ok {
		receiver = entry.Value
	}
	if isNilMemberReceiver(receiver) {
		return nil, fmt.Errorf("cannot read member %q on null", name)
	}
	if reader, ok := receiver.(MemberReader); ok {
		return reader.ResolveMember(name)
	}
	args := []Value{receiver, Str(name)}
	switch obj := receiver.(type) {
	case *HostObjectValue:
		return resolveHostMember(rt, obj.Value, name)
	case *MapValue:
		return getPropMapValue(args...)
	case *MapNode:
		return getPropMapNode(args...)
	case *SimpleJSON:
		return getPropSimpleJSON(args...)
	case *JSONNode:
		return getPropJSON(args...)
	case *TreeNodeImpl:
		return getPropTreeNode(args...)
	case *XMLNode:
		return getPropXML(args...)
	case *Plan:
		return getPropPlan(args...)
	case map[string]Value:
		return getPropMap(args...)
	case map[string]interface{}:
		return getPropMapInterface(args...)
	default:
		// VarRef historically returns raw registered Go objects. Keep that API
		// while allowing their exported members through the host path.
		rv := reflect.ValueOf(receiver)
		if (rv.Kind() == reflect.Struct || rv.Kind() == reflect.Ptr) && rv.Type().Comparable() {
			for _, host := range rt.objects {
				if reflect.TypeOf(host) == rv.Type() && host == receiver {
					return resolveHostMember(rt, receiver, name)
				}
			}
		}
		return nil, fmt.Errorf("member access not supported for %T", receiver)
	}
}

func isNilMemberReceiver(v Value) bool {
	if v == nil {
		return true
	}
	if _, ok := v.(*dbNullType); ok {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Interface, reflect.Func, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

type boundHostMethod struct {
	receiver interface{}
	name     string
}

func (m *boundHostMethod) String() string { return "bound host method " + m.name }

func (m *boundHostMethod) Invoke(rt *Runtime, args []Value) (Value, error) {
	return rt.CallHostMethod(m.receiver, m.name, args)
}

func resolveHostMember(rt *Runtime, obj interface{}, name string) (Value, error) {
	if isNilMemberReceiver(obj) {
		return nil, fmt.Errorf("cannot read member %q on null host object", name)
	}
	if reflect.ValueOf(obj).MethodByName(name).IsValid() {
		return &boundHostMethod{receiver: obj, name: name}, nil
	}
	value, err := getObjectProperty(obj, name)
	if err != nil {
		return nil, err
	}
	// Preserve Chariot functions/values in fields without binding a receiver.
	return hostPropertyValue(rt, value)
}

func hostPropertyValue(rt *Runtime, value interface{}) (Value, error) {
	switch value.(type) {
	case *ClassValue, *InterfaceValue, *Signature, *FunctionValue, Callable, MemberReader, *MapValue, *MapNode, *SimpleJSON, *JSONNode, *TreeNodeImpl, *HostObjectValue, map[string]Value, map[string]interface{}, *ArrayValue:
		return value, nil
	}
	if value == nil {
		return DBNull, nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Struct {
		return &HostObjectValue{Value: value}, nil
	}
	return rt.convertToChariotValue(rv)
}

// MemberAccessNode is a postfix property read, separate from a named FuncCall.
type MemberAccessNode struct {
	Receiver Node
	Name     string
	Pos      SourcePos
}

func (m *MemberAccessNode) GetPos() SourcePos    { return m.Pos }
func (m *MemberAccessNode) SetPos(pos SourcePos) { m.Pos = pos }
func (m *MemberAccessNode) Exec(rt *Runtime) (Value, error) {
	receiver, err := m.Receiver.Exec(rt)
	if err != nil {
		return nil, err
	}
	value, err := ResolveMember(rt, receiver, m.Name)
	if err != nil {
		return nil, fmt.Errorf("%s:%d:%d: member %s: %w", m.Pos.File, m.Pos.Line, m.Pos.Column, m.Name, err)
	}
	return value, nil
}
func (m *MemberAccessNode) ToMap() map[string]interface{} {
	return map[string]interface{}{"_node_type": "MemberAccessNode", "receiver": m.Receiver.ToMap(), "name": m.Name, "pos": m.Pos}
}
func (m *MemberAccessNode) ToString() string {
	return "(" + m.Receiver.ToString() + ")." + m.Name
}
