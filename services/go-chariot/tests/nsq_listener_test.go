package tests

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestResponseTopicRegistrationScript(t *testing.T) {
	key := fmt.Sprintf("response_key_%d", time.Now().UnixNano())
	topic := fmt.Sprintf("test.response.%d", time.Now().UnixNano())

	cases := []TestCase{
		{
			Name: "RegisterAndListResponseTopic",
			Script: []string{
				fmt.Sprintf(`registerResponseTopic("%s", "%s")`, key, topic),
				"setq(allTopics, listResponseTopics())",
				fmt.Sprintf(`getProp(allTopics, "%s")`, key),
			},
			ExpectedValue: chariot.Str(topic),
		},
	}

	RunTestCases(t, cases)
}

func TestListenNSQEndToEnd(t *testing.T) {
	addr := requireNSQD(t)

	cfg.ChariotConfig.NSQEnabled = true
	cfg.ChariotConfig.NSQDAddress = addr

	responseTopic := fmt.Sprintf("test_response_%d", time.Now().UnixNano())
	requestTopic := fmt.Sprintf("test_request_%d", time.Now().UnixNano())
	channel := fmt.Sprintf("test_channel_%d#ephemeral", time.Now().UnixNano())
	responseKey := fmt.Sprintf("response-key-%d", time.Now().UnixNano())
	handlerVar := fmt.Sprintf("nsqHandler_%d", time.Now().UnixNano())

	runtimeName := fmt.Sprintf("nsq_runtime_%d", time.Now().UnixNano())
	rt := createNamedRuntime(runtimeName)
	defer chariot.UnregisterRuntime(runtimeName)

	scriptLines := []string{
		fmt.Sprintf(`registerResponseTopic("%s", "%s")`, responseKey, responseTopic),
		fmt.Sprintf(`setq(%s, func(msg) { map("status", "ok", "payload", getProp(msg, "payload")) })`, handlerVar),
		fmt.Sprintf(`setq(listenerInfo, listenNSQ("%s", "%s", map("decision", %s), map("nsqd", "%s")))`, requestTopic, channel, handlerVar, addr),
		"getProp(listenerInfo, \"status\")",
	}

	program := strings.Join(scriptLines, "\n")
	result, err := rt.ExecProgram(program)
	if err != nil {
		t.Fatalf("listenNSQ script failed: %v", err)
	}
	status, ok := result.(chariot.Str)
	if !ok || string(status) != "listening" {
		t.Fatalf("unexpected listenNSQ status: %v", result)
	}

	respCh := make(chan []byte, 1)
	respConsumer, err := nsq.NewConsumer(responseTopic, fmt.Sprintf("resp_channel_%d#ephemeral", time.Now().UnixNano()), nsq.NewConfig())
	if err != nil {
		t.Fatalf("failed to create response consumer: %v", err)
	}
	respConsumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		select {
		case respCh <- append([]byte(nil), msg.Body...):
		default:
		}
		return nil
	}))
	if err := respConsumer.ConnectToNSQD(addr); err != nil {
		respConsumer.Stop()
		t.Fatalf("failed to connect response consumer: %v", err)
	}
	defer respConsumer.Stop()

	time.Sleep(250 * time.Millisecond)

	payload := fmt.Sprintf("hello-%d", time.Now().UnixNano())
	body, err := json.Marshal(map[string]interface{}{
		"messageType":      "decision",
		"payload":          payload,
		"responseTopicKey": responseKey,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	producer, err := nsq.NewProducer(addr, nsq.NewConfig())
	if err != nil {
		t.Fatalf("failed to create nsq producer: %v", err)
	}
	if err := producer.Publish(requestTopic, body); err != nil {
		producer.Stop()
		t.Fatalf("failed to publish test message: %v", err)
	}
	producer.Stop()

	select {
	case resp := <-respCh:
		var decoded map[string]interface{}
		if err := json.Unmarshal(resp, &decoded); err != nil {
			t.Fatalf("invalid response payload: %v", err)
		}
		if decoded["status"] != "ok" {
			t.Fatalf("unexpected response status: %v", decoded["status"])
		}
		if decoded["payload"] != payload {
			t.Fatalf("unexpected response payload: %v", decoded["payload"])
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for NSQ response")
	}
}

func requireNSQD(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("CHARIOT_NSQ_ADDR")
	if addr == "" {
		addr = "127.0.0.1:4150"
	}

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("NSQD not reachable at %s: %v", addr, err)
	}
	_ = conn.Close()

	return addr
}
