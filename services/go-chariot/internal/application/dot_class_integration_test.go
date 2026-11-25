package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-theory-io/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/m-theory-io/chariot-ecosystem/services/go-chariot/configs"
	"github.com/m-theory-io/chariot-ecosystem/services/go-chariot/logs"
)

type sessionDotHost struct{ total atomic.Int64 }

func (h *sessionDotHost) Add(n int) int { return int(h.total.Add(int64(n))) }

func TestHostRESTDotClassSessionsIsolateScriptStateAndBorrowHost(t *testing.T) {
	old := cfg.ChariotConfig
	cfg.ChariotConfig.DataPath = t.TempDir()
	cfg.ChariotConfig.ListenersFile = "listeners.json"
	cfg.ChariotConfig.ListenerScriptsDir = ""
	cfg.ChariotConfig.FunctionLib = ""
	t.Cleanup(func() { cfg.ChariotConfig = old })
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	fixture := &sessionDotHost{}
	rt.RegisterHostObject("dotHost", fixture, true)
	if _, err := rt.ExecProgram(`
interface(Greeter){requires(hello,signature(array(),"S"))}
class(Person){field(name,"S","bootstrap") field(data,"M",mapValue("label","base"))}
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
	sm := chariot.NewSessionManager(time.Minute, time.Minute)
	logger := logs.NewZapLogger()
	t.Cleanup(func() { _ = logger.Sync() })
	host, err := NewHost(HostOptions{
		Runtime:        &ApplicationRuntime{Runtime: rt, BootstrapLoaded: true},
		SessionManager: sm, Logger: logger, DevRESTEnabled: true,
	})
	if err != nil {
		sm.Stop()
		t.Fatal(err)
	}
	shutdown := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := host.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}
	t.Cleanup(shutdown)
	first := sm.NewSession("first", logger, "dot-first")
	second := sm.NewSession("second", logger, "dot-second")
	if first == nil || second == nil {
		t.Fatal("host did not create sessions")
	}
	// Execute through the real router and bearer authentication so this checks
	// session selection and response conversion as well as runtime behavior.
	execute := func(token, source string) any {
		t.Helper()
		payload, err := json.Marshal(map[string]string{"program": source})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/execute", strings.NewReader(string(payload)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		host.Echo().ServeHTTP(response, req)
		var result struct {
			Result string `json:"result"`
			Data   any    `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode execute: %v: %s", err, response.Body.String())
		}
		if response.Code != http.StatusOK || result.Result != "OK" {
			t.Fatalf("execute %s: HTTP %d: %s", source, response.Code, response.Body.String())
		}
		return result.Data
	}
	if got := execute("dot-first", `renamePerson("first") setProp(p.data,"label","first-map") envelope.echo(assertSatisfies(envelope.person,Greeting).hello())`); got != "first" {
		t.Fatalf("first session result: %#v", got)
	}
	if got := execute("dot-second", `assertSatisfies(p,Greeting).hello()`); got != "bootstrap" {
		t.Fatalf("second session inherited first receiver mutation: %#v", got)
	}
	if got := execute("dot-second", `p.data.label`); got != "base" {
		t.Fatalf("second session inherited map mutation: %#v", got)
	}
	if got := execute("dot-second", `renamePerson("second") new(PersonClass).hello()`); got != "bootstrap" {
		t.Fatalf("class default mutated: %#v", got)
	}
	for i, token := range []string{"dot-first", "dot-second"} {
		if got := execute(token, `mapValue("host",dotHost).host.Add(1)`); got != float64(i+1) {
			t.Fatalf("borrowed host result: %#v", got)
		}
	}
	port := availablePort(t)
	execute("dot-first", fmt.Sprintf(`listen(%d,func(req){return(mapValue("field",p.name,"method",assertSatisfies(p,Greeting).hello(),"map",p.data.label,"host",dotHost.Add(1)))},map("bind_address","127.0.0.1"))`, port))
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	t.Cleanup(client.CloseIdleConnections)
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil || readErr != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("listener response: %s; decode=%v read=%v HTTP=%d", body, err, readErr, response.StatusCode)
	}
	if result["field"] != "first" || result["method"] != "first" || result["map"] != "first-map" || result["host"] != float64(3) {
		t.Fatalf("listener lost session state: %v", result)
	}
	if got := execute("dot-second", `p.hello()`); got != "second" {
		t.Fatalf("session receiver alias leaked: %#v", got)
	}
	if got, err := rt.ExecProgram(`p.hello()`); err != nil || got != chariot.Str("bootstrap") {
		t.Fatalf("bootstrap mutated: %v, %v", got, err)
	}
	if got, err := rt.ExecProgram(`p.data.label`); err != nil || got != chariot.Str("base") {
		t.Fatalf("bootstrap map mutated: %v, %v", got, err)
	}
	if fixture.total.Load() != 3 || len(rt.ListListeners()) != 0 || len(first.Runtime.ListListeners()) != 1 || len(second.Runtime.ListListeners()) != 0 {
		t.Fatal("host effects or session listener ownership incorrect")
	}
	shutdown()
	if sm.GetActiveSessions() != 0 || len(first.Runtime.ListListeners()) != 0 {
		t.Fatal("host shutdown retained session resources")
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("host shutdown leaked session listener: %v", err)
	}
	_ = l.Close()
}
