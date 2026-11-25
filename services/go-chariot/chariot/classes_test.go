package chariot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/m-theory-io/chariot-ecosystem/services/go-chariot/logs"
)

func classRuntime() *Runtime {
	rt := dotRuntime()
	RegisterArray(rt)
	RegisterMath(rt)
	RegisterCompares(rt)
	RegisterFlow(rt)
	return rt
}
func classError(t *testing.T, rt *Runtime, source, contains string) {
	t.Helper()
	_, err := dotParse(t, source).Exec(rt)
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Fatalf("%s: got %v, want %q", source, err, contains)
	}
}

func TestClassesBehavior(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `class(Person) { field(name,"S","") field(items,"A",array()) field(next,"V") field(callback,"F",func(x){x}) }
	method(Person,hello,withSignature(func(self){self.name},signature(array("V"),"S")))
	method(Person,rename,func(self,name){setProp(self,"name",name) self.hello()})
	setq(a,new(Person,mapValue("name","Ada","next",0)))
	setq(b,new(Person,mapValue("next",0))) setq(alias,a) setq(extracted,a.rename)`)
	if got := dotExec(t, rt, `extracted("Grace")`); got != Str("Grace") {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `alias.name`); got != Str("Grace") {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `b.name`); got != Str("") {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `a.callback(42)`); got != Number(42) {
		t.Fatal(got)
	}
	a, _ := rt.currentScope.Get("a")
	b, _ := rt.currentScope.Get("b")
	ai, bi := a.(*InstanceValue), b.(*InstanceValue)
	if ai.Class != bi.Class {
		t.Fatal("methods must be shared per class")
	}
	ai.fields["items"].(*ArrayValue).Elements = append(ai.fields["items"].(*ArrayValue).Elements, Number(1))
	if len(bi.fields["items"].(*ArrayValue).Elements) != 0 {
		t.Fatal("shared mutable default")
	}
	for _, test := range []struct{ src, err string }{
		{`new(Person)`, "required field"},
		{`new(Person,mapValue("other",1))`, "unknown field"},
		{`setProp(a,"name",2)`, "expected string"},
		{`setProp(a,"other",2)`, "unknown field"},
		{`a.other`, "unknown member"},
		{`method(Person,name,func(self){1})`, "collides"},
		{`method(Person,hello,func(self){1})`, "already declared"},
		{`method(Person,bad,func(x){x})`, "self"},
		{`class(Bad){field(a,"S",1)}`, "expected string"},
		{`class(Bad){field(a,"S","") field(a,"S","")}`, "duplicate field"},
		{`class(Bad){field(a,"invalid",1)}`, "invalid field type"},
		{`class(Bad){setq(x,1)}`, "invalid declaration"},
		{`class(Person){}`, "already declared"},
	} {
		classError(t, rt, test.src, test.err)
	}
	// Receiver mutations are references, including cycles; displays never recurse.
	dotExec(t, rt, `setProp(a,"next",a)`)
	if ai.fields["next"] != ai {
		t.Fatal("lost reference semantics")
	}
	if ai.String() != "instance(Person)" {
		t.Fatal(ai.String())
	}
}

func TestSignaturesAndInterfaces(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `interface(Greeter){requires(hello,signature(array(),"S"))}
	class(Person){field(name,"S","Ada")}
	method(Person,hello,withSignature(func(self){return(self.name)},signature(array("V"),"S")))
	setq(p,new(Person))`)
	if got := dotExec(t, rt, `satisfies(p,Greeter)`); got != Bool(true) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `assertSatisfies(p,Greeter).hello()`); got != Str("Ada") {
		t.Fatal(got)
	}
	dotExec(t, rt, `class(Unannotated){} method(Unannotated,hello,func(self){"hi"})`)
	classError(t, rt, `assertSatisfies(new(Unannotated),Greeter)`, "metadata required")
	classError(t, rt, `assertSatisfies(mapValue(),Greeter)`, "script instance")
	dotExec(t, rt, `class(Missing){}`)
	classError(t, rt, `assertSatisfies(new(Missing),Greeter)`, "missing method")
	dotExec(t, rt, `class(Wrong){} method(Wrong,hello,withSignature(func(self,x){x},signature(array("V","N"),"N")))`)
	if got := dotExec(t, rt, `satisfies(new(Wrong),Greeter)`); got != Bool(false) {
		t.Fatal(got)
	}
	classError(t, rt, `assertSatisfies(new(Wrong),Greeter)`, "signature mismatch")
	// Ordinary functions remain permissive; annotations opt into strict binding.
	if got := dotExec(t, rt, `(func(x){x})()`); got != DBNull {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `(func(x){x})(1,2)`); got != Number(1) {
		t.Fatal(got)
	}
	dotExec(t, rt, `setq(f,withSignature(func(x){x},signature(array("N"),"N")))`)
	classError(t, rt, `f()`, "expects 1 arguments")
	classError(t, rt, `call(f,"x")`, "argument 1")
	classError(t, rt, `(withSignature(func(){1},signature(array(),"S")))()`, "expected string")
	classError(t, rt, `p.hello(1)`, "expects 1 arguments")
	classError(t, rt, `withSignature(func(x){x},signature(array(),"V"))`, "parameter count")
	// One guard only: method invocation stays inside the caller boundary.
	if err := rt.WithExecution(func() error { _, err := dotParse(t, `p.hello()`).Exec(rt); return err }); err != nil {
		t.Fatal(err)
	}
}

func TestClassDeclarationRoundTrips(t *testing.T) {
	source := `interface(G){requires(hello,signature(array(),"S"))} class(P){field(name,"S","Ada")}
	method(P,hello,withSignature(func(self){self.name},signature(array("V"),"S")))
	assertSatisfies(new(P),G).hello()`
	node := dotParse(t, source)
	sources := []string{node.ToString(), PrettyPrintNode(node, "")}
	for _, raw := range []interface{}{node.ToMap(), serializeNode(node)} {
		data, err := json.Marshal(raw)
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
		for _, n := range []Node{strict, deserializeNode(decoded)} {
			got, err := n.Exec(classRuntime())
			if err != nil || got != Str("Ada") {
				t.Fatalf("round trip: %v %v", got, err)
			}
			sources = append(sources, n.ToString(), PrettyPrintNode(n, ""))
		}
	}
	for _, source := range sources {
		if got := dotExec(t, classRuntime(), source); got != Str("Ada") {
			t.Fatal(got)
		}
	}
}

func TestClassCloneGraphAndSessionIsolation(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `class(P){field(name,"S","base") field(next,"V",0)}
	method(P,rename,func(self,n){setProp(self,"name",n)})
	declareGlobal(Public,"V",P) declareGlobal(p,"V",new(P))
	declareGlobal(alias,"V",p) declareGlobal(bound,"V",p.rename)
	setProp(p,"next",p)
	declareGlobal(closure,"F",(func(){setq(local,p) func(n){local.rename(n)}})())`)
	clone := rt.CloneRuntime()
	dotExec(t, clone, `bound("clone") closure("closed") method(Public,extra,func(self){42})`)
	original, _ := rt.globalScope.Get("p")
	copied, _ := clone.globalScope.Get("p")
	alias, _ := clone.globalScope.Get("alias")
	if copied == original || copied != alias {
		t.Fatal("clone identity/alias failure")
	}
	ci := copied.(*InstanceValue)
	if ci.fields["next"] != ci || ci.fields["name"] != Str("closed") {
		t.Fatal("cycle or closure not rebound")
	}
	if original.(*InstanceValue).fields["name"] != Str("base") {
		t.Fatal("receiver leaked")
	}
	if _, ok := original.(*InstanceValue).Class.methods["extra"]; ok {
		t.Fatal("definition leaked")
	}
	// A second clone must preserve captured lexical state, not flatten to globals.
	second := clone.CloneRuntime()
	dotExec(t, second, `closure("second")`)
	if ci.fields["name"] != Str("closed") {
		t.Fatal("second clone closure leaked")
	}
	sm := NewSessionManager(time.Minute, time.Minute)
	defer sm.Stop()
	sm.SetBootstrapRuntime(rt)
	// Nil loggers aren't supported by session creation; use the project's logger.
	logger := logs.NewZapLogger()
	defer logger.Sync()
	session := sm.NewSession("test", logger, "class-test")
	if session == nil {
		t.Fatal("no session")
	}
	dotExec(t, session.Runtime, `bound("session")`)
	if original.(*InstanceValue).fields["name"] != Str("base") {
		t.Fatal("session leaked receiver")
	}
}

func TestClassPersistenceBoundaries(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `class(P){field(name,"S","Ada")} setq(p,new(P))`)
	p, _ := rt.currentScope.Get("p")
	raw, err := ScriptValueToNative(p)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	copy, ok := convertFromNativeValue(decoded).(*InstanceValue)
	if !ok {
		t.Fatalf("decode: %v", convertFromNativeValue(decoded))
	}
	if copy.fields["name"] != Str("Ada") {
		t.Fatal(copy.fields)
	}
	dotExec(t, rt, `method(P,hello,func(self){self.name})`)
	if _, err := ScriptValueToNative(p); err == nil || !strings.Contains(err.Error(), "captured") {
		t.Fatalf("expected closure error: %v", err)
	}
	if _, ok := convertFromNativeValue(convertValueToNative(p)).(error); !ok {
		t.Fatal("legacy hooks hid persistence error")
	}
	// Scope-free methods supplied by loaders can be saved and actually invoked.
	p.(*InstanceValue).Class.methods["hello"].Scope = nil
	raw, err = ScriptValueToNative(p)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, ok := convertFromNativeValue(raw).(*InstanceValue)
	if !ok {
		t.Fatal("lost saved instance")
	}
	bound, err := reloaded.ResolveMember("hello")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := InvokeCallable(rt, bound, nil); err != nil || got != Str("Ada") {
		t.Fatalf("persisted method: %v %v", got, err)
	}
	if _, err := ScriptValueToNative(bound); err == nil {
		t.Fatal("serialized bound receiver")
	}
	cyclic := NewMap()
	cyclic.Set("self", cyclic)
	if _, err := ScriptValueToNative(cyclic); err == nil {
		t.Fatal("cycle silently serialized")
	}
	if _, err := ScriptValueToNative(&HostObjectValue{Value: new(int)}); err == nil {
		t.Fatal("serialized host resource")
	}
	if _, err := copyDefault(&HostObjectValue{Value: new(int)}, nil); err == nil {
		t.Fatal("copied host default")
	}
	fn := dotExec(t, rt, `withSignature(func(x){x},signature(array("S"),"S"))`).(*FunctionValue)
	data, err = json.Marshal(convertValueToNative(fn))
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(data, &decoded)
	restored := convertFromNativeValue(decoded).(*FunctionValue)
	if _, err := InvokeCallable(rt, restored, []Value{Number(1)}); err == nil {
		t.Fatal("lost signature persistence")
	}
	data, err = json.Marshal(FunctionValueToMap(fn))
	if err != nil {
		t.Fatal(err)
	}
	var library map[string]interface{}
	if err := json.Unmarshal(data, &library); err != nil {
		t.Fatal(err)
	}
	loaded, err := MapToFunctionValue(library)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InvokeCallable(rt, loaded, []Value{Number(1)}); err == nil {
		t.Fatal("library lost signature")
	}
	for _, source := range []string{`signature(array(),"S")`, `interface(Empty){}`} {
		value := dotExec(t, rt, source)
		raw, err := ScriptValueToNative(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := convertFromNativeValue(raw).(error); ok {
			t.Fatal("metadata round trip failed")
		}
	}
}

func TestClassRecursiveMethodsAndDefaultGraphs(t *testing.T) {
	rt := classRuntime()
	dotExec(t, rt, `class(Counter){field(n,"N",0)}
	method(Counter,count,func(self,n){if(equal(n,0)){return(self.n)} setProp(self,"n",add(self.n,1)) self.count(sub(n,1))})
	setq(counter,new(Counter))`)
	if got := dotExec(t, rt, `counter.count(4)`); got != Number(4) {
		t.Fatal(got)
	}
	// Caller-provided attributes intentionally retain aliases, unlike defaults.
	dotExec(t, rt, `class(Box){field(value,"V")} setq(shared,array(1)) setq(a,new(Box,mapValue("value",shared))) setq(b,new(Box,mapValue("value",shared)))`)
	a, _ := rt.currentScope.Get("a")
	b, _ := rt.currentScope.Get("b")
	if a.(*InstanceValue).fields["value"] != b.(*InstanceValue).fields["value"] {
		t.Fatal("attributes unexpectedly copied")
	}
	cyclic := NewMap()
	cyclic.Set("self", cyclic)
	rt.currentScope.Set("seed", cyclic)
	dotExec(t, rt, `class(Cyclic){field(value,"M",seed)} setq(one,new(Cyclic)) setq(two,new(Cyclic))`)
	one, _ := rt.currentScope.Get("one")
	two, _ := rt.currentScope.Get("two")
	m1 := one.(*InstanceValue).fields["value"].(*MapValue)
	m2 := two.(*InstanceValue).fields["value"].(*MapValue)
	if m1 == m2 || m1 == cyclic || m1.Values["self"] != m1 || m2.Values["self"] != m2 {
		t.Fatal("default cycle copy failure")
	}
	// Function-valued defaults also isolate mutable lexical state.
	dotExec(t, rt, `class(Closures){field(f,"F",(func(){setq(state,mapValue("n",0)) func(){setProp(state,"n",add(state.n,1)) state.n}})())}
	setq(c1,new(Closures)) setq(c2,new(Closures)) c1.f()`)
	if got := dotExec(t, rt, `c1.f()`); got != Number(2) {
		t.Fatal(got)
	}
	if got := dotExec(t, rt, `c2.f()`); got != Number(1) {
		t.Fatal("default closure state shared", got)
	}
}
