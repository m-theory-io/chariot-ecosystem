package chariot

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type dotHost struct {
	Count   int
	Child   *dotHost
	Fn      *FunctionValue
	private string
}

func (h *dotHost) Add(n int) int      { h.Count += n; return h.Count }
func (h *dotHost) Self() *dotHost     { return h }
func (h *dotHost) Fail() (int, error) { return 0, errors.New("host failure") }
func (h *dotHost) ErrorOnly() error   { return errors.New("error only") }
func (h *dotHost) Void()              { h.Count++ }
func (h *dotHost) Panic()             { panic("host panic") }
func (h *dotHost) Sum(nums ...int) int {
	for _, n := range nums {
		h.Count += n
	}
	return h.Count
}

func dotRuntime() *Runtime {
	rt := NewRuntime()
	RegisterValues(rt)
	RegisterHostFunctions(rt)
	RegisterTypeDispatchedFunctions(rt)
	return rt
}

func dotParse(t *testing.T, source string) *Block {
	t.Helper()
	node, err := NewParserWithFilename(source, "dot.ch").parseProgram()
	if err != nil {
		t.Fatalf("parse %s: %v", source, err)
	}
	return node
}

func dotExec(t *testing.T, rt *Runtime, source string) Value {
	t.Helper()
	value, err := dotParse(t, source).Exec(rt)
	if err != nil {
		t.Fatalf("execute %s: %v", source, err)
	}
	return value
}

func TestDotPostfixAndOrdinaryFunctionFields(t *testing.T) {
	rt := dotRuntime()
	for _, source := range []string{
		`mapValue("answer", 42).answer`,
		`mapValue("nested", mapValue("answer", 42)).nested.answer`,
		`mapValue("f", func(x) { x }).f(42)`,
		`mapValue("f" func(x) { x; }).f(42);`,
		`call(mapValue("f", func(x) { x }).f, 42)`,
		`(func(x) { x })(42)`,
		`(func() { func(x) { x } })()(42)`,
		`setq(f, func(x) { x }) f(42)`,
		`setq(o, mapValue("f", func(x) { x })) setq(extracted, o.f) extracted(42)`,
		"mapValue(\"answer\", 42) // comment\n . answer",
	} {
		t.Run(source, func(t *testing.T) {
			if got := dotExec(t, rt, source); got != Number(42) {
				t.Fatalf("got %#v", got)
			}
		})
	}
	if got := dotExec(t, rt, `mapValue("absent", 1).missing`); got != DBNull {
		t.Fatalf("missing map key = %#v", got)
	}
}

func TestDotHostBindingAndLegacyAPIs(t *testing.T) {
	rt := dotRuntime()
	fn := dotExec(t, rt, `func(x) { x }`).(*FunctionValue)
	host := &dotHost{Child: &dotHost{Count: 17}, Fn: fn}
	rt.RegisterHostObject("host", host, true)
	for _, test := range []struct {
		source string
		want   Value
	}{
		{`host.Add(2)`, Number(2)},
		{`setq(bound, host.Add) bound(3)`, Number(5)},
		{`call(host.Add, 4)`, Number(9)},
		{`host.Self().Add(1)`, Number(10)},
		{`host.Child.Count`, Number(17)},
		{`host.Fn(42)`, Number(42)},
		{`callHostMethod("host", "Add", 2)`, Number(12)},
		{`callMethod(getHostObject("host"), "Add", 3)`, Number(15)},
		{`setProp(getHostObject("host"), "Count", 20)`, Number(20)},
		{`getProp(getHostObject("host"), "Count")`, Number(20)},
		{`setHostProperty("host", "Count", 21)`, Number(21)},
		{`getHostProperty("host", "Count")`, Number(21)},
		{`host.count`, Number(21)}, // legacy host field case-insensitivity
		{`host.Sum(1, 2)`, Number(24)},
		{`host.Void()`, DBNull},
		{`host.Count`, Number(25)},
	} {
		if got := dotExec(t, rt, test.source); got != test.want {
			t.Fatalf("%s = %#v, want %#v", test.source, got, test.want)
		}
	}
	legacy := &FuncCall{Name: "host.Add", Args: []Node{&Literal{Val: Number(1)}}}
	if got, err := legacy.Exec(rt); err != nil || got != Number(26) {
		t.Fatalf("legacy dotted FuncCall = %v, %v", got, err)
	}
	// Bound methods retain their actual receiver even after registry replacement.
	bound := dotExec(t, rt, `host.Add`)
	if GetValueType(bound) != ValueFunction || GetValueTypeSpec(bound) != "F" {
		t.Fatal("bound method is not classified as F")
	}
	if got := dotExec(t, rt, `valueType(host.Add)`); got != Str("F") {
		t.Fatal(got)
	}
	rt.RegisterHostObject("host", &dotHost{}, true)
	if got, err := InvokeCallable(rt, bound, []Value{Number(1)}); err != nil || got != Number(27) {
		t.Fatalf("binding = %v, %v", got, err)
	}
}

func TestDotEvaluationOrder(t *testing.T) {
	rt := dotRuntime()
	var order []string
	fn := dotExec(t, rt, `func(a, b) { b }`)
	m := NewMap()
	m.Set("run", fn)
	rt.Register("receiver", func(...Value) (Value, error) { order = append(order, "receiver"); return m, nil })
	rt.Register("arg", func(args ...Value) (Value, error) {
		order = append(order, string(args[0].(Str)))
		return Number(len(order)), nil
	})
	if got := dotExec(t, rt, `receiver().run(arg("first"), arg("second"))`); got != Number(3) {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(order, []string{"receiver", "first", "second"}) {
		t.Fatalf("order = %v", order)
	}
}

func TestDotNamedPrecedenceAndUDFArguments(t *testing.T) {
	rt := dotRuntime()
	rt.Register("pick", func(...Value) (Value, error) { return Number(1), nil })
	rt.functions["pick"] = dotExec(t, rt, `func() { 2 }`).(*FunctionValue)
	rt.currentScope.Set("pick", dotExec(t, rt, `func() { 3 }`))
	if got := dotExec(t, rt, `pick()`); got != Number(1) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `(pick)()`); got != Number(3) {
		t.Fatal(got)
	}
	delete(rt.funcs, "pick")
	if got := dotExec(t, rt, `pick()`); got != Number(2) {
		t.Fatal(got)
	}
	rt.functions["identity"] = dotExec(t, rt, `func(x) { x }`).(*FunctionValue)
	if got := dotExec(t, rt, `identity(42)`); got != Number(42) {
		t.Fatalf("UDF args = %#v", got)
	}
	// Existing functions remain permissive (no new class-style arity policy).
	if got := dotExec(t, rt, `identity()`); got != DBNull {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `identity(42, 99)`); got != Number(42) {
		t.Fatal(got)
	}
}

func TestDotCanonicalMembers(t *testing.T) {
	rt := dotRuntime()
	mapNode := NewMapNode("m")
	mapNode.Set("answer", Number(42))
	jsonNode := NewJSONNodeFromInterface(map[string]interface{}{"answer": 42}, "j")
	tree := NewTreeNode("t")
	tree.SetAttribute("answer", Number(42))
	values := []Value{mapNode, jsonNode, tree, NewSimpleJSON(map[string]interface{}{"answer": 42}), map[string]Value{"answer": Number(42)}, map[string]interface{}{"answer": 42}}
	for _, value := range values {
		rt.currentScope.Set("obj", value)
		dot := dotExec(t, rt, `obj.answer`)
		prop := dotExec(t, rt, `getProp(obj, "answer")`)
		if !reflect.DeepEqual(dot, prop) {
			t.Fatalf("%T: dot = %#v, getProp = %#v", value, dot, prop)
		}
		_, dotErr := ResolveMember(rt, value, "missing")
		_, propErr := rt.funcs["getProp"](value, Str("missing"))
		if (dotErr == nil) != (propErr == nil) {
			t.Fatalf("%T: missing mismatch", value)
		}
	}
}

func TestDotRuntimeErrorsDoNotPanic(t *testing.T) {
	rt := dotRuntime()
	rt.RegisterHostObject("host", &dotHost{}, true)
	for _, source := range []string{
		`null.member`, `42.member`, `"text".member`, `host.Missing()`,
		`host.add(1)`, `host.Add()`, `host.Add("wrong")`, `host.Count()`,
		`host.Fail()`, `host.ErrorOnly()`, `host.Panic()`, `host.private`,
		`host.Child.Count`, `callMethod(getHostObject("host"), 2)`,
		`setProp(getHostObject("host"), "Count", "wrong")`,
	} {
		t.Run(source, func(t *testing.T) {
			_, err := dotParse(t, source).Exec(rt)
			if err == nil {
				t.Fatalf("expected error: %s", source)
			}
		})
	}
	var nilHost *dotHost
	rt.RegisterHostObject("nilHost", nilHost, true)
	if _, err := dotParse(t, `nilHost.Add(1)`).Exec(rt); err == nil {
		t.Fatal("nil receiver accepted")
	}
}

func TestDotParserErrorsAndNumbers(t *testing.T) {
	for _, source := range []string{
		`a.`, `a..b`, `.a`, `a.(b)`, `a.2`, `a.b = 1`, `a?.b`,
		`f(`, `(f`, `a+b`, `a-b`, `/`, `f(1;2)`, `1.2.3`,
		`1e+`, `"unterminated`, "`unterminated",
	} {
		_, err := NewParserWithFilename(source, "invalid.ch").parseProgram()
		if err == nil {
			t.Errorf("accepted %q", source)
		} else if !strings.Contains(err.Error(), "invalid.ch:") {
			t.Errorf("missing source location: %v", err)
		}
	}
	rt := dotRuntime()
	for source, want := range map[string]Number{"1.25": 1.25, "-1.25": -1.25, "1.": 1, "-0.5": -.5, "1e3": 1000, "2.5e-2": .025} {
		if got := dotExec(t, rt, source); got != want {
			t.Errorf("%s = %#v", source, got)
		}
	}
	lexer := NewLexer(" // comment\n  a.\n b(42)")
	for i, want := range []Token{{Type: TOK_IDENT, Text: "a", Line: 2, Column: 3, NewlineBefore: true}, {Type: TOK_DOT, Text: ".", Line: 2, Column: 4}, {Type: TOK_IDENT, Text: "b", Line: 3, Column: 2, NewlineBefore: true}} {
		if got := lexer.Next(); got != want {
			t.Fatalf("token %d = %+v, want %+v", i, got, want)
		}
	}
}

func TestDotASTAndSourceRoundTrips(t *testing.T) {
	for _, source := range []string{
		`mapValue("f", func(x) { x }).f(42)`,
		`(func() { mapValue("f", func(x) { x }) })().f(42)`,
		`call(mapValue("f", func(x) { x }).f, 42)`,
		`mapValue("a", [1, 2]).a`,
	} {
		original := dotParse(t, source)
		want, err := original.Exec(dotRuntime())
		if err != nil {
			t.Fatal(err)
		}
		for _, encoded := range []interface{}{original.ToMap(), serializeNode(original)} {
			data, err := json.Marshal(encoded)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			strict, err := NodeFromMap(decoded)
			if err != nil {
				t.Fatal(err)
			}
			for _, ast := range []Node{strict, deserializeNode(decoded)} {
				if ast == nil {
					t.Fatal("lost AST")
				}
				got, err := ast.Exec(dotRuntime())
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("roundtrip %s: %#v, %v (want %#v)", source, got, err, want)
				}
				for _, text := range []string{ast.ToString(), PrettyPrintNode(ast, "")} {
					got := dotExec(t, dotRuntime(), text)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("rendered %s = %#v, want %#v", text, got, want)
					}
				}
			}
		}
	}
	node := dotParse(t, "\n  mapValue(\"x\", 1).x").Stmts[0].(*MemberAccessNode)
	round, err := NodeFromMap(node.ToMap())
	if err != nil || round.GetPos() != (SourcePos{File: "dot.ch", Line: 2, Column: 3}) {
		t.Fatalf("position = %v, %v", round, err)
	}
}

func TestDotFunctionValuePersistence(t *testing.T) {
	rt := dotRuntime()
	fn := dotExec(t, rt, `func(x) { mapValue("f", func(y) { y }).f(x) }`)
	data, err := json.Marshal(convertValueToNative(fn))
	if err != nil {
		t.Fatal(err)
	}
	var native interface{}
	if err := json.Unmarshal(data, &native); err != nil {
		t.Fatal(err)
	}
	restored := convertFromNativeValue(native)
	got, err := InvokeCallable(dotRuntime(), restored, []Value{Number(42)})
	if err != nil || got != Number(42) {
		t.Fatalf("persisted function = %#v, %v", got, err)
	}
}

type dotExtension struct{ value Value }

func (d *dotExtension) ResolveMember(name string) (Value, error) {
	if name != "field" {
		return nil, errors.New("missing extension member")
	}
	return d.value, nil
}
func (d *dotExtension) SetMember(name string, value Value) (Value, error) {
	if name != "field" {
		return nil, errors.New("unknown extension field")
	}
	d.value = value
	return value, nil
}

type dotCallable struct{}

func (*dotCallable) Invoke(_ *Runtime, args []Value) (Value, error) { return args[0], nil }

func TestDotExtensionHooks(t *testing.T) {
	rt := dotRuntime()
	rt.currentScope.Set("obj", &dotExtension{value: &dotCallable{}})
	if got := dotExec(t, rt, `obj.field(42)`); got != Number(42) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `call(getProp(obj, "field"), 42)`); got != Number(42) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `setProp(obj, "field", 9) obj.field`); got != Number(9) {
		t.Fatal(got)
	}
}
