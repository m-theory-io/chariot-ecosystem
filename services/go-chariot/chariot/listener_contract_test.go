package chariot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
)

func TestListenRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		options string
		want    string
	}{
		{"unknown", `map("methdos", array("GET"))`, "unknown listen option"},
		{"wrong methods type", `map("methods", "GET")`, "array of strings"},
		{"invalid method", `map("methods", array("FETCH"))`, "invalid HTTP method"},
		{"negative timeout", `map("read_timeout_ms", -1)`, "non-negative"},
		{"fractional timeout", `map("read_timeout_ms", 1.5)`, "whole number"},
		{"invalid body limit", `map("max_body_bytes", 0)`, "greater than zero"},
		{"invalid strict json", `map("strict_json", "yes")`, "must be a boolean"},
		{"bind includes port", `map("bind_address", "127.0.0.1:8080")`, "without a port"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rt := newListenerTestRuntime(t)
			port := availableListenerTestPort(t)
			_, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { return(req) }, %s)`, port, test.options))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestListenBodyLimitStrictJSONAndExactBasePath(t *testing.T) {
	rt := newListenerTestRuntime(t)
	port := availableListenerTestPort(t)
	_, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { return(req) }, map(
		"bind_address", "127.0.0.1", "methods", array("POST"), "base_path", "/api",
		"max_body_bytes", 12, "strict_json", true))`, port))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	response, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/apix", port), "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request prefix collision: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("prefix-collision status = %d, want 404", response.StatusCode)
	}

	response, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/api", port), "application/json", strings.NewReader(`{"bad"`))
	if err != nil {
		t.Fatalf("request malformed JSON: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed JSON status = %d, want 400", response.StatusCode)
	}

	response, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/items", port), "application/json", strings.NewReader(`{"value":1234}`))
	if err != nil {
		t.Fatalf("request oversized body: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 413", response.StatusCode)
	}
}

func TestHTTPResponseAndOrdinaryStatusMap(t *testing.T) {
	rt := newListenerTestRuntime(t)
	structuredPort := availableListenerTestPort(t)
	_, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) {
		return(httpResponse(201, map("Location", "/orders/42", "Content-Type", "text/plain"), "created"))
	}, map("bind_address", "127.0.0.1"))`, structuredPort))
	if err != nil {
		t.Fatalf("listen structured response: %v", err)
	}
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", structuredPort))
	if err != nil {
		t.Fatalf("structured response request: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated || response.Header.Get("Location") != "/orders/42" || string(body) != `"created"` {
		t.Fatalf("structured response status=%d location=%q body=%s", response.StatusCode, response.Header.Get("Location"), body)
	}

	mapPort := availableListenerTestPort(t)
	_, err = rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { return(map("status", 202, "body", "ordinary")) }, map("bind_address", "127.0.0.1"))`, mapPort))
	if err != nil {
		t.Fatalf("listen ordinary map: %v", err)
	}
	response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/", mapPort))
	if err != nil {
		t.Fatalf("ordinary map request: %v", err)
	}
	defer response.Body.Close()
	var payload map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode ordinary map: %v", err)
	}
	if response.StatusCode != http.StatusOK || payload["status"] != float64(202) {
		t.Fatalf("ordinary map became transport metadata: status=%d payload=%#v", response.StatusCode, payload)
	}
}

func TestListenerLanguageLifecycle(t *testing.T) {
	rt := newListenerTestRuntime(t)
	port := availableListenerTestPort(t)
	result, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { return(DBNull) }, map("bind_address", "127.0.0.1"))`, port))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	idValue, _ := result.(*MapValue).Get("id")
	id := string(idValue.(Str))

	listed, err := rt.ExecProgram(`listenerList()`)
	if err != nil || listed.(*ArrayValue).Length() != 1 {
		t.Fatalf("listenerList result=%v err=%v", listed, err)
	}
	status, err := rt.ExecProgram(fmt.Sprintf(`listenerStatus("%s")`, id))
	if err != nil || status.(*MapValue).Values["state"] != Str("running") {
		t.Fatalf("listenerStatus result=%v err=%v", status, err)
	}
	stopped, err := rt.ExecProgram(fmt.Sprintf(`listenerStop("%s")`, id))
	if err != nil || stopped != Bool(true) {
		t.Fatalf("listenerStop result=%v err=%v", stopped, err)
	}
	// Language stop acknowledges acceptance only; the Go API remains the
	// blocking completion barrier (even if the listener has already left the registry).
	if err := rt.ShutdownListeners(context.Background()); err != nil {
		t.Fatalf("wait for accepted stop: %v", err)
	}
	missing, err := rt.ExecProgram(fmt.Sprintf(`listenerStatus("%s")`, id))
	if err != nil || missing != DBNull {
		t.Fatalf("stopped listener status=%v err=%v", missing, err)
	}
}

func TestNSQOptionsAreStrict(t *testing.T) {
	handlers := NewMapWithValues(map[string]Value{"decision": Str("handler")})
	_, err := buildNSQListenerOptions(Str("topic"), Str("channel"), handlers,
		NewMapWithValues(map[string]Value{"nsqd": Str("127.0.0.1:4150")}))
	if err == nil || !strings.Contains(err.Error(), "unknown listenNSQ option") {
		t.Fatalf("legacy option error = %v", err)
	}

	options := NewMapWithValues(map[string]Value{
		"nsqd_address": Str("127.0.0.1:4150"), "max_attempts": Number(3),
		"backoff_strategy": Str("full_jitter"), "response_required": Bool(true),
		"decode_error_policy": Str("ack"),
	})
	parsed, err := buildNSQListenerOptions(Str("topic"), Str("channel"), handlers, options)
	if err != nil {
		t.Fatalf("canonical NSQ options: %v", err)
	}
	if parsed.MaxAttempts != 3 || parsed.BackoffStrategy != "full_jitter" || !parsed.ResponseRequired || parsed.DecodeErrorPolicy != nsqFailureAck {
		t.Fatalf("unexpected parsed NSQ options: %+v", parsed)
	}
}

func TestNSQFailurePolicies(t *testing.T) {
	message := &nsq.Message{Attempts: 1, Body: []byte(`not-json`)}
	listener := &nsqListener{opts: nsqListenerOptions{
		MaxAttempts: 2, DecodeErrorPolicy: nsqFailureRequeue,
	}}
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("decode requeue policy acknowledged before max attempts")
	}

	var publishedTopic string
	listener.opts.DeadLetterTopic = "dead_letters"
	listener.publish = func(address, topic string, payload []byte) error {
		publishedTopic = topic
		return nil
	}
	message.Attempts = 2
	if err := listener.handleMessage(message); err != nil {
		t.Fatalf("exhausted decode failure: %v", err)
	}
	if publishedTopic != "dead_letters" {
		t.Fatalf("dead-letter topic = %q", publishedTopic)
	}

	listener.publish = func(address, topic string, payload []byte) error { return errors.New("publish failed") }
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("dead-letter publish failure must requeue")
	}
}

func TestNSQOptionalAndRequiredResponses(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)
	if err := rt.SaveFunction("handler", `function handler(payload) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	binding := &nsqHandlerBinding{messageType: "decision", HandlerName: "handler"}
	message := &nsq.Message{Attempts: 1, Timestamp: time.Now().UnixNano(), Body: []byte(`{"messageType":"decision"}`)}

	listener := &nsqListener{runtime: rt, opts: nsqListenerOptions{
		Handlers: map[string]*nsqHandlerBinding{"decision": binding}, MaxAttempts: 3,
		MissingResponsePolicy: nsqFailureRequeue,
	}}
	if err := listener.handleMessage(message); err != nil {
		t.Fatalf("optional response should acknowledge: %v", err)
	}
	listener.opts.ResponseRequired = true
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("required missing response should requeue")
	}
}

func TestNSQStructuralFailurePolicies(t *testing.T) {
	message := &nsq.Message{Attempts: 1, Timestamp: time.Now().UnixNano()}
	listener := &nsqListener{opts: nsqListenerOptions{
		MaxAttempts: 3, MissingMessageTypePolicy: nsqFailureRequeue,
		MissingHandlerPolicy: nsqFailureAck,
	}}
	message.Body = []byte(`{"value":1}`)
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("missing messageType requeue policy acknowledged")
	}
	message.Body = []byte(`{"messageType":"unknown"}`)
	if err := listener.handleMessage(message); err != nil {
		t.Fatalf("missing handler ack policy requeued: %v", err)
	}
}

func TestNSQHandlerAndResponsePublishFailuresRequeue(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)
	message := &nsq.Message{Attempts: 1, Timestamp: time.Now().UnixNano(), Body: []byte(`{"messageType":"decision"}`)}
	listener := &nsqListener{runtime: rt, opts: nsqListenerOptions{
		Handlers: map[string]*nsqHandlerBinding{
			"decision": {messageType: "decision", HandlerName: "missingHandler", ResponseTopicKey: "contract_response"},
		},
		MaxAttempts: 3,
	}}
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("handler failure must requeue")
	}

	if err := rt.SaveFunction("workingHandler", `function workingHandler(payload) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	listener.opts.Handlers["decision"].HandlerName = "workingHandler"
	if err := RegisterResponseTopic("contract_response", "contract_responses"); err != nil {
		t.Fatalf("register response topic: %v", err)
	}
	listener.publish = func(address, topic string, payload []byte) error { return errors.New("publish unavailable") }
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("response publish failure must requeue")
	}
}

func TestNSQUnknownResponsePolicy(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)
	if err := rt.SaveFunction("handler", `function handler(payload) { return("ok") }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	message := &nsq.Message{Attempts: 1, Timestamp: time.Now().UnixNano(), Body: []byte(`{"messageType":"decision"}`)}
	listener := &nsqListener{runtime: rt, opts: nsqListenerOptions{
		Handlers: map[string]*nsqHandlerBinding{
			"decision": {messageType: "decision", HandlerName: "handler", ResponseTopicKey: "unregistered_contract_response"},
		},
		MaxAttempts: 3, UnknownResponsePolicy: nsqFailureRequeue,
	}}
	if err := listener.handleMessage(message); err == nil {
		t.Fatal("unknown response topic requeue policy acknowledged")
	}
}

func TestHTTPResponseValidation(t *testing.T) {
	rt := newListenerTestRuntime(t)
	_, err := rt.ExecProgram(`httpResponse(99, "bad")`)
	if err == nil || !strings.Contains(err.Error(), "between 100 and 599") {
		t.Fatalf("invalid status error = %v", err)
	}
	_, err = rt.ExecProgram(`httpResponse(200, map("X-Test", 12), "bad")`)
	if err == nil || !strings.Contains(err.Error(), "string or array") {
		t.Fatalf("invalid header error = %v", err)
	}
}

func TestListenerStopHonorsContextAPI(t *testing.T) {
	rt := newListenerTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := rt.ShutdownListeners(ctx); err != nil {
		t.Fatalf("empty shutdown: %v", err)
	}
}

func TestListenersSharingRuntimeSerializeHandlerExecution(t *testing.T) {
	rt := NewRuntime()
	RegisterAll(rt)
	var active atomic.Int32
	var maximum atomic.Int32
	rt.Register("criticalSection", func(args ...Value) (Value, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		return Bool(true), nil
	})
	if err := rt.SaveFunction("handler", `function handler(payload) { return(criticalSection()) }`, ""); err != nil {
		t.Fatalf("save handler: %v", err)
	}
	first := &runtimeListener{runtime: rt, opts: listenerOptions{HandlerName: "handler"}}
	second := &runtimeListener{runtime: rt, opts: listenerOptions{HandlerName: "handler"}}

	var wait sync.WaitGroup
	wait.Add(2)
	for _, listener := range []*runtimeListener{first, second} {
		go func(listener *runtimeListener) {
			defer wait.Done()
			if _, err := listener.invokeHandler(NewMap()); err != nil {
				t.Errorf("invoke handler: %v", err)
			}
		}(listener)
	}
	wait.Wait()
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent callbacks = %d, want 1", maximum.Load())
	}
}
