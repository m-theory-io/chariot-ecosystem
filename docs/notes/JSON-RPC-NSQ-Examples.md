Awesome — here are two tiny, runnable Go examples to show the difference in *shape* between JSON-RPC (sync request/response over HTTP) and NSQ (async pub/sub work queue).

---

# JSON-RPC over HTTP (hand-rolled 2.0)

## Server (`jsonrpc_server/main.go`)

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type RPCResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	Result  any          `json:"result,omitempty"`
	Error   *RPCError    `json:"error,omitempty"`
	ID      any          `json:"id,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeErr(w http.ResponseWriter, id any, code int, msg string) {
	_ = json.NewEncoder(w).Encode(RPCResponse{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: code, Message: msg},
		ID:      id,
	})
}

func rpcHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, nil, -32700, "Parse error")
		return
	}
	if req.JSONRPC != "2.0" {
		writeErr(w, req.ID, -32600, "Invalid Request")
		return
	}

	switch req.Method {
	case "add":
		var nums []float64
		if err := json.Unmarshal(req.Params, &nums); err != nil || len(nums) != 2 {
			writeErr(w, req.ID, -32602, "Invalid params")
			return
		}
		_ = json.NewEncoder(w).Encode(RPCResponse{
			JSONRPC: "2.0",
			Result:  nums[0] + nums[1],
			ID:      req.ID,
		})
	case "echo":
		_ = json.NewEncoder(w).Encode(RPCResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(req.Params),
			ID:      req.ID,
		})
	default:
		writeErr(w, req.ID, -32601, "Method not found")
	}
}

func main() {
	http.HandleFunc("/rpc", rpcHandler)
	log.Println("JSON-RPC server listening on :8080 /rpc")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Client (`jsonrpc_client/main.go`)

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      int         `json:"id,omitempty"`
}

func call(method string, params any, id int) {
	body, _ := json.Marshal(RPCRequest{JSONRPC: "2.0", Method: method, Params: params, ID: id})
	resp, err := http.Post("http://localhost:8080/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	fmt.Printf("%s -> %v\n", method, out)
}

func main() {
	call("add", []float64{3, 5}, 1)
	call("echo", map[string]string{"msg": "hello"}, 2)
	call("missingMethod", nil, 3)
}
```

**Run:**

```bash
# in jsonrpc_server
go mod init example.com/jsonrpc_server && go mod tidy
go run .

# in another shell, jsonrpc_client
go mod init example.com/jsonrpc_client && go mod tidy
go run .
```

> This pattern is **synchronous**: client sends a request and immediately receives a response.

---

# NSQ message broker (pub/sub work queue)

**Prereq:** run `nsqd` locally (no lookupd for simplicity).

```bash
docker run --rm -p 4150:4150 -p 4151:4151 nsqio/nsq /nsqd
# 4150 = TCP (NSQD), 4151 = HTTP admin for nsqd
```

## Producer (`nsq_producer/main.go`)

```go
package main

import (
	"log"
	"time"

	"github.com/nsqio/go-nsq"
)

func main() {
	cfg := nsq.NewConfig()
	prod, err := nsq.NewProducer("127.0.0.1:4150", cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer prod.Stop()

	topic := "jobs"
	for i := 1; i <= 5; i++ {
		msg := []byte(time.Now().Format(time.RFC3339Nano))
		if err := prod.Publish(topic, msg); err != nil {
			log.Fatal("publish:", err)
		}
		log.Println("published:", string(msg))
		time.Sleep(300 * time.Millisecond)
	}
}
```

## Consumer (`nsq_consumer/main.go`)

```go
package main

import (
	"log"

	"github.com/nsqio/go-nsq"
)

type handler struct{}

func (h *handler) HandleMessage(m *nsq.Message) error {
	log.Printf("got message: %s (attempts=%d)", string(m.Body), m.Attempts)
	// do work... return error to requeue
	return nil
}

func main() {
	cfg := nsq.NewConfig()
	consumer, err := nsq.NewConsumer("jobs", "workers", cfg) // topic: jobs, channel: workers
	if err != nil {
		log.Fatal(err)
	}
	consumer.AddHandler(&handler{})

	if err := consumer.ConnectToNSQD("127.0.0.1:4150"); err != nil {
		log.Fatal(err)
	}
	<-consumer.StopChan
}
```

**Run:**

```bash
# producer
go mod init example.com/nsq_producer
go get github.com/nsqio/go-nsq@v1.1.0
go run .

# consumer (in another shell)
go mod init example.com/nsq_consumer
go get github.com/nsqio/go-nsq@v1.1.0
go run .
```

> This pattern is **asynchronous**: producer publishes and returns immediately; one or more consumers process messages later. NSQ gives you buffering, retries, fan-out via channels, and horizontal scale almost for free.

---

## When to pick which (rule of thumb)

* **JSON-RPC (HTTP/WebSocket/STDIO):**

  * You need a **direct**, **low-latency** call with a **definite return value**.
  * Perfect for **tool invocation**, control-plane ops, and simple request/response APIs.

* **NSQ (or any broker):**

  * You need **asynchronous**, **durable**, **scalable** task distribution or event streams.
  * Perfect for **pipelines, background jobs, fan-out/fan-in**, and **decoupled microservices**.

If you want, I can also show:

* a **hybrid**: JSON-RPC endpoint that **enqueues** work to NSQ and returns a job id, plus a small **status** API; or
* add **dead-letter** + **backoff** to the NSQ example.
