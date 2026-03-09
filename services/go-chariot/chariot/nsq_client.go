package chariot

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/nsqio/go-nsq"
)

var (
	nsqProducerMu sync.Mutex
	nsqProducers  = make(map[string]*nsq.Producer)
)

// sendNSQMessage publishes the provided payload to the configured nsqd instance.
func sendNSQMessage(topic string, data interface{}) error {
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return errors.New("nsq topic cannot be empty")
	}
	if !cfg.ChariotConfig.NSQEnabled {
		return errors.New("NSQ messaging is disabled (set CHARIOT_NSQ_ENABLED=true)")
	}
	addr, err := ensureNSQDAddress(cfg.ChariotConfig.NSQDAddress)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to encode NSQ payload: %w", err)
	}
	return publishNSQPayload(addr, trimmedTopic, payload)
}

// publishNSQPayload publishes bytes to the nsqd instance identified by addr.
func publishNSQPayload(addr, topic string, payload []byte) error {
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return errors.New("nsq topic cannot be empty")
	}
	resolvedAddr, err := ensureNSQDAddress(addr)
	if err != nil {
		return err
	}
	producer, err := getOrCreateNSQProducer(resolvedAddr)
	if err != nil {
		return err
	}
	return producer.Publish(trimmedTopic, payload)
}

func ensureNSQDAddress(addr string) (string, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return "", errors.New("nsqd address not configured")
	}
	return trimmed, nil
}

func getOrCreateNSQProducer(addr string) (*nsq.Producer, error) {
	nsqProducerMu.Lock()
	defer nsqProducerMu.Unlock()
	if producer, ok := nsqProducers[addr]; ok {
		return producer, nil
	}
	cfg := nsq.NewConfig()
	producer, err := nsq.NewProducer(addr, cfg)
	if err != nil {
		return nil, err
	}
	nsqProducers[addr] = producer
	return producer, nil
}

// valueToJSONBytes serializes a runtime Value into JSON bytes suitable for NSQ payloads.
func valueToJSONBytes(result Value) ([]byte, error) {
	if result == nil || result == DBNull {
		return []byte("null"), nil
	}
	if node, ok := result.(*JSONNode); ok {
		jsonStr, err := node.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to encode JSONNode: %w", err)
		}
		return []byte(jsonStr), nil
	}
	native := ValueToJSON(result)
	payload, err := json.Marshal(native)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value: %w", err)
	}
	return payload, nil
}
