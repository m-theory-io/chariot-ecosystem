package chariot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAuditStatementSourceRoundTrips(t *testing.T) {
	for _, source := range []string{
		`if(true){1; (func(){42})()}`,
		`if(false){0} else {1; (func(){42})()}`,
		`setq(n,0); while(equal(n,0)){setq(n,add(n,1)); (func(){42})()}`,
		`switch(1){case(1){1; (func(){42})()} default(){0}}`,
		`switch(2){case(1){0} default(){1; (func(){42})()}}`,
		`(func(){1; (func(){42})()})()`,
	} {
		t.Run(source, func(t *testing.T) {
			node := dotParse(t, source)
			for _, rendered := range []string{source, node.ToString(), PrettyPrintNode(node, "")} {
				if got := dotExec(t, classRuntime(), rendered); got != Number(42) {
					t.Fatalf("rendered %s: got %v", rendered, got)
				}
			}
		})
	}
	for _, source := range []string{`func(){}`, `(func(){})()`} {
		node := dotParse(t, source)
		for _, rendered := range []string{node.ToString(), PrettyPrintNode(node, "")} {
			if strings.Contains(rendered, "ERROR") || strings.TrimSpace(rendered) == "" {
				t.Fatalf("empty function rendered as %q", rendered)
			}
			dotExec(t, classRuntime(), rendered)
		}
	}
}

func TestAuditDeclarationContractsAndInstanceType(t *testing.T) {
	rt := classRuntime()
	rt.RegisterHostObject("host", &dotHost{}, true)
	dotExec(t, rt, `class(C){} method(C,answer,func(self){42}); setq(c,new(C));
	declare(f,"F",c.answer); declare(h,"F",host.Add); declare(o,"O",c);
	declareGlobal(gf,"F",c.answer); declareGlobal(go,"O",c)`)
	for source, want := range map[string]Value{
		`f()`: Number(42), `gf()`: Number(42), `h(2)`: Number(2), `valueType(c)`: Str("O"), `valueType(f)`: Str("F"),
	} {
		if got := dotExec(t, rt, source); got != want {
			t.Fatalf("%s: got %v, want %v", source, got, want)
		}
	}
	instance, _ := rt.currentScope.Get("c")
	if GetValueType(instance) != ValueObject || ValueTypeToSpec(GetValueType(instance)) != "O" {
		t.Fatal("instance not classified as object")
	}
	classError(t, rt, `declare(bad,"F",1)`, "expected function")
	classError(t, rt, `declare(bad,"O",1)`, "expected object")
}

func TestAuditDefaultClosuresKeepLiveLexicalBindings(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `class(C){field(make,"F",func(){new(C)}) field(help,"F",func(){helper()})}
	setq(first,new(C)); setq(helper,func(){42}); setq(second,new(C));
	class(Private){field(f,"F",(func(){setq(state,mapValue("n",0)); func(){setProp(state,"n",add(state.n,1)); helper(); state.n}})())}
	setq(a,new(Private)); setq(b,new(Private))`)
	for _, source := range []string{`first.help()`, `second.help()`, `first.make().help()`} {
		if got := dotExec(t, rt, source); got != Number(42) {
			t.Fatal(source, got)
		}
	}
	first, _ := rt.currentScope.Get("first")
	created := dotExec(t, rt, `first.make()`).(*InstanceValue)
	if created.Class != first.(*InstanceValue).Class {
		t.Fatal("recursive default lost live class identity")
	}
	dotExec(t, rt, `setq(helper,func(){43})`)
	if got := dotExec(t, rt, `first.help()`); got != Number(43) {
		t.Fatal("default captured a lexical snapshot", got)
	}
	for _, source := range []string{`a.f()`, `b.f()`} {
		if got := dotExec(t, rt, source); got != Number(1) {
			t.Fatal("private mutable closure state leaked", source, got)
		}
	}
	clone := rt.CloneRuntime()
	dotExec(t, clone, `setq(helper,func(){99})`)
	if got := dotExec(t, clone, `first.make().help()`); got != Number(99) {
		t.Fatal("runtime clone did not rebind live default scope", got)
	}
	if got := dotExec(t, rt, `first.help()`); got != Number(43) {
		t.Fatal("runtime clone leaked lexical mutation", got)
	}
	if got := dotExec(t, clone, `a.f()`); got != Number(2) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `a.f()`); got != Number(2) {
		t.Fatal("runtime clone leaked private state", got)
	}
}

func TestAuditPersistedMethodReceiverContracts(t *testing.T) {
	want := &InterfaceValue{Name: "Answer", requirements: map[string]*Signature{"answer": {Return: "N"}}}
	for _, test := range []struct {
		name   string
		params []string
		sig    *Signature
		valid  bool
	}{
		{"valid", []string{"self"}, &Signature{[]string{"V"}, "N"}, true},
		{"wrong self contract", []string{"self"}, &Signature{[]string{"N"}, "N"}, false},
		{"missing self contract", []string{"self"}, &Signature{nil, "N"}, false},
		{"extra contract", []string{"self"}, &Signature{[]string{"V", "N"}, "N"}, false},
		{"wrong receiver name", []string{"other"}, &Signature{[]string{"V"}, "N"}, false},
		{"missing receiver", nil, &Signature{nil, "N"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn := &FunctionValue{Parameters: test.params, Signature: test.sig, Body: &Block{Stmts: []Node{&Literal{Val: Number(42)}}}}
			c := &ClassValue{Name: "C", fields: map[string]FieldDefinition{}, methods: map[string]*FunctionValue{"answer": fn}}
			raw, err := ScriptValueToNative(c)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			var wire interface{}
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			decoded := convertFromNativeValue(wire)
			_, failed := decoded.(error)
			if failed == test.valid {
				t.Fatalf("decoded %T: %v", decoded, decoded)
			}
			if err := conformance(&InstanceValue{Class: c}, want); (err == nil) != test.valid {
				t.Fatalf("conformance disagreed with decoder: %v", err)
			}
			rt := classRuntime()
			_, err = rt.funcs["method"](&ClassValue{Name: "Registered", fields: map[string]FieldDefinition{}, methods: map[string]*FunctionValue{}}, Str("answer"), fn)
			if (err == nil) != test.valid {
				t.Fatalf("registration disagreed: %v", err)
			}
			if test.valid {
				instance := &InstanceValue{Class: decoded.(*ClassValue), fields: map[string]Value{}}
				bound, err := instance.ResolveMember("answer")
				if err != nil {
					t.Fatal(err)
				}
				if got, err := InvokeCallable(rt, bound, nil); err != nil || got != Number(42) {
					t.Fatalf("decoded method: %v %v", got, err)
				}
			}
		})
	}
}

func TestAuditHostPropertiesPreserveScriptTypes(t *testing.T) {
	rt := classRuntime()
	c := dotExec(t, rt, `class(C){}`).(*ClassValue)
	i := dotExec(t, rt, `interface(I){}`).(*InterfaceValue)
	s := &Signature{Return: "V"}
	host := &struct {
		Class     *ClassValue
		Interface *InterfaceValue
		Signature *Signature
	}{c, i, s}
	rt.RegisterHostObject("types", host, true)
	for name, want := range map[string]Value{"Class": c, "Interface": i, "Signature": s} {
		if got := dotExec(t, rt, "types."+name); got != want {
			t.Fatalf("%s wrapped as %T", name, got)
		}
	}
	if got := dotExec(t, rt, `satisfies(new(types.Class),types.Interface)`); got != Bool(true) {
		t.Fatal(got)
	}
	dotExec(t, rt, `withSignature(func(){},types.Signature)`)
}

func TestAuditInstanceJSONProjectionAndCycles(t *testing.T) {
	rt := classRuntime()
	instance := dotExec(t, rt, `class(C){field(name,"S","Ada") field(next,"V",0)} new(C)`).(*InstanceValue)
	root := NewMap()
	root.Set("instances", &ArrayValue{Elements: []Value{instance, instance}})
	projected, err := ValueToJSONChecked(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"instances":[{"name":"Ada","next":0},{"name":"Ada","next":0}]}` {
		t.Fatal(string(data))
	}
	before := ValueToJSON(instance)
	instance.fields["name"] = Str("Grace")
	if reflect.DeepEqual(before, ValueToJSON(instance)) {
		t.Fatal("field change lost in JSON projection")
	}
	instance.fields["next"] = root
	array := &ArrayValue{}
	array.Elements = []Value{array}
	native := map[string]Value{}
	native["self"] = native
	for _, cyclic := range []Value{instance, root, array, native} {
		if _, err := ValueToJSONChecked(cyclic); err == nil || !strings.Contains(err.Error(), "cyclic") {
			t.Fatalf("cycle not rejected: %v", err)
		}
		if _, err := json.Marshal(ValueToJSON(cyclic)); err != nil {
			t.Fatal("legacy error projection not JSON safe", err)
		}
	}
	var typedNil *InstanceValue
	if got, err := ValueToJSONChecked(typedNil); err != nil || got != nil {
		t.Fatalf("typed nil: %v %v", got, err)
	}
}

func TestAuditHTTPInstanceProjectionAndCycleErrors(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"instance", `new(C)`, http.StatusOK},
		{"envelope", `httpResponse(201,new(C))`, http.StatusCreated},
		{"cyclic", `setq(c,new(C)); setProp(c,"next",c); c`, http.StatusInternalServerError},
		{"cyclic envelope", `setq(c,new(C)); setProp(c,"next",c); httpResponse(201,c)`, http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			rt := newListenerTestRuntime(t)
			port := availableListenerTestPort(t)
			_, err := rt.ExecProgram(fmt.Sprintf(`class(C){field(name,"S","Ada") field(next,"V",0)}; setq(handler,func(req){%s}); listen(%d,handler,mapValue("bind_address","127.0.0.1"))`, test.body, port))
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Timeout: 5 * time.Second}
			response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var payload map[string]interface{}
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status {
				t.Fatalf("status %d: %v", response.StatusCode, payload)
			}
			if test.status == http.StatusInternalServerError {
				if message, _ := payload["error"].(string); !strings.Contains(message, "cyclic") {
					t.Fatal(payload)
				}
			} else if payload["name"] != "Ada" || payload["next"] != float64(0) {
				t.Fatal(payload)
			}
		})
	}
}
