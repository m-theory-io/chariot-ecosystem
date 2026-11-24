package mcp

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/m-theory-io/chariot-ecosystem/services/go-chariot/chariot"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type executeDotHost struct{ total atomic.Int64 }

func (h *executeDotHost) Add(n int) int { return int(h.total.Add(int64(n))) }

func TestMCPExecuteDotClassGlobalsInterfacesAndHost(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	host := &executeDotHost{}
	rt.RegisterHostObject("dotHost", host, true)
	if _, err := rt.ExecProgram(`
interface(Greeter){requires(hello,signature(array(),"S"))}
class(Person){field(name,"S","bootstrap")}
method(Person,hello,withSignature(func(self){return(self.name)},signature(array("V"),"S")))
method(Person,rename,func(self,name){setProp(self,"name",name) return(self.hello())})
declareGlobal(PersonClass,"V",Person)
declareGlobal(Greeting,"V",Greeter)
declareGlobal(p,"V",new(Person))
declareGlobal(renamePerson,"F",p.rename)
declareGlobal(envelope,"M",mapValue("person",p,"echo",func(x){return(x)}))
`); err != nil {
		t.Fatal(err)
	}
	session := connectTestMCP(t, ServerOptions{RuntimeProvider: func() *chariot.Runtime { return rt }})
	check := func(code, want string) {
		t.Helper()
		result := executeTestMCP(t, session, code)
		if result.IsError || len(result.Content) != 1 {
			t.Fatalf("%s: error=%t content=%#v", code, result.IsError, result.Content)
		}
		text, ok := result.Content[0].(*sdkmcp.TextContent)
		if !ok || text.Text != want {
			t.Fatalf("%s: content=%#v, want %q", code, result.Content, want)
		}
	}
	check(`new(PersonClass,mapValue("name","constructed")).hello()`, "constructed")
	check(`assertSatisfies(p,Greeting).hello()`, "bootstrap")
	check(`renamePerson("mcp")`, "mcp")
	check(`envelope.echo(assertSatisfies(envelope.person,Greeting).hello())`, "mcp")
	check(`setq(bound,dotHost.Add) bound(3)`, "3")
	check(`mapValue("host",dotHost).host.Add(4)`, "7")
	check(`dotHost.Add(0)`, "7")
	// The real execute tool must report signature/conformance failures, and a
	// failed request must not make the provider runtime unusable on the next call.
	for _, test := range []struct{ code, message string }{
		{`p.hello(1)`, "expects"},
		{`class(Missing){} assertSatisfies(new(Missing),Greeting)`, "missing method"},
		{`class(WrongReturn){} method(WrongReturn,hello,withSignature(func(self){return(42)},signature(array("V"),"S"))) assertSatisfies(new(WrongReturn),Greeting).hello()`, "expected string"},
	} {
		result := executeTestMCP(t, session, test.code)
		if !result.IsError || len(result.Content) != 1 {
			t.Fatalf("expected tool error for %s: %#v", test.code, result)
		}
		text, ok := result.Content[0].(*sdkmcp.TextContent)
		if !ok || !strings.Contains(text.Text, test.message) {
			t.Fatalf("expected %q: %#v", test.message, result.Content)
		}
		check(`p.hello()`, "mcp")
	}
	if host.total.Load() != 7 {
		t.Fatalf("Go host effects missing: %d", host.total.Load())
	}
	if got, err := rt.ExecProgram(`p.name`); err != nil || got != chariot.Str("mcp") {
		t.Fatalf("execute did not mutate the provider's actual instance: %v, %v", got, err)
	}
}
