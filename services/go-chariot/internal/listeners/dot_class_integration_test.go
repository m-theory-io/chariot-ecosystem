package listeners

import (
	"encoding/json"
	"sync/atomic"
	"testing"

	ch "github.com/m-theory-io/chariot-ecosystem/services/go-chariot/chariot"
)

// This is a borrowed Go resource, not a mocked builtin. Its effects deliberately
// survive DevTest disposal, unlike script instances, maps, and bound receivers.
type devTestDotHost struct{ calls atomic.Int64 }

func (h *devTestDotHost) Add(n int) int { return int(h.calls.Add(int64(n))) }

func TestDevTestDotClassInterfaceAndHostEndToEnd(t *testing.T) {
	m, bootstrap := testManager(t)
	host := &devTestDotHost{}
	bootstrap.RegisterHostObject("dotHost", host, true)
	const setup = `
interface(Greeter){requires(hello,signature(array(),"S"))}
class(Person){field(name,"S","bootstrap") field(data,"M",mapValue("label","base"))}
method(Person,hello,withSignature(func(self){return(self.name)},signature(array("V"),"S")))
method(Person,rename,func(self,name){setProp(self,"name",name) return(self.hello())})
declareGlobal(PersonClass,"V",Person)
declareGlobal(Greeting,"V",Greeter)
declareGlobal(p,"V",new(Person))
declareGlobal(alias,"V",p)
declareGlobal(renamePerson,"F",p.rename)
declareGlobal(envelope,"M",mapValue("person",p,"echo",func(x){return(x)}))
`
	if _, err := bootstrap.ExecProgram(setup); err != nil {
		t.Fatal(err)
	}
	define(t, m, "dot-class", `
renamePerson("devtest")
setProp(p.data,"label","cloned-map")
method(PersonClass,cloneOnly,func(self){return(self.data.label)})
listen(0,func(req){
 return(mapValue("field",alias.name,"method",assertSatisfies(envelope.person,Greeting).hello(),
  "nested",p.cloneOnly(),"mapCall",envelope.echo(p.hello()),"host",dotHost.Add(1),
  "classGlobal",new(PersonClass).name,"conforms",satisfies(p,Greeting)))
},map("bind_address","127.0.0.1"))
`)
	// Restart must clone the bootstrap afresh, not reuse the prior mutated class.
	for run := 0; run < 2; run++ {
		listener, err := m.Start("dot-class")
		if err != nil {
			t.Fatal(err)
		}
		if listener.Status != "running" || !listener.IsHealthy || len(listener.Resources) != 1 {
			t.Fatalf("start: %+v", listener)
		}
		address := listener.Resources[0].Address
		for i := 1; i <= 2; i++ {
			var body map[string]any
			if err := json.Unmarshal([]byte(request(t, address)), &body); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]any{
				"field": "devtest", "method": "devtest", "nested": "cloned-map",
				"mapCall": "devtest", "host": float64(run*2 + i),
				"classGlobal": "bootstrap", "conforms": true,
			} {
				if body[key] != want {
					t.Errorf("response %s = %#v, want %#v; body=%v", key, body[key], want, body)
				}
			}
		}
		stop(t, m, "dot-class")
		assertClosed(t, address)
		if got := bootstrap.ListListeners(); len(got) != 0 {
			t.Fatalf("DevTest transferred listener ownership to bootstrap: %+v", got)
		}
		for _, source := range []string{`p.name`, `alias.hello()`, `envelope.person.name`, `new(PersonClass).hello()`} {
			if got, err := bootstrap.ExecProgram(source); err != nil || got != ch.Str("bootstrap") {
				t.Fatalf("bootstrap changed: %s = %v, %v", source, got, err)
			}
		}
		if got, err := bootstrap.ExecProgram(`p.data.label`); err != nil || got != ch.Str("base") {
			t.Fatalf("bootstrap map changed: %v, %v", got, err)
		}
		if _, err := bootstrap.ExecProgram(`p.cloneOnly()`); err == nil {
			t.Fatal("DevTest method registration mutated bootstrap class")
		}
	}
	if got := host.calls.Load(); got != 4 {
		t.Fatalf("borrowed Go host calls = %d, want 4", got)
	}
}
