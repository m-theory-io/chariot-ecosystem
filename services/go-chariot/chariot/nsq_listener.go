package chariot

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

type nsqHandlerBinding struct {
	messageType      string
	HandlerName      string
	HandlerFn        *FunctionValue
	ResponseTopicKey string
}

type nsqListenerOptions struct {
	Topic                   string
	Channel                 string
	NSQDAddress             string
	LookupdAddresses        []string
	Handlers                map[string]*nsqHandlerBinding
	MaxInFlight             int
	Concurrency             int
	OnStartProgram          string
	OnExitProgram           string
	ResponseTopicKeyField   string
	ResponseProducerAddress string
}

type nsqListener struct {
	runtime  *Runtime
	opts     nsqListenerOptions
	consumer *nsq.Consumer
	execMu   sync.Mutex
}

var (
	nsqListenersMu      sync.Mutex
	runtimeNSQListeners = make(map[*Runtime]map[string]*nsqListener)
)

func startRuntimeNSQListener(rt *Runtime, opts nsqListenerOptions) (*nsqListener, error) {
	opts.Topic = strings.TrimSpace(opts.Topic)
	opts.Channel = strings.TrimSpace(opts.Channel)
	if opts.Topic == "" {
		opts.Topic = strings.TrimSpace(cfg.ChariotConfig.NSQDefaultTopic)
	}
	if opts.Channel == "" {
		opts.Channel = strings.TrimSpace(cfg.ChariotConfig.NSQDefaultChannel)
	}
	if opts.Topic == "" {
		return nil, errors.New("nsq listen requires a topic")
	}
	if opts.Channel == "" {
		return nil, errors.New("nsq listen requires a channel")
	}
	if len(opts.Handlers) == 0 {
		return nil, errors.New("nsq listen requires at least one handler binding")
	}
	if opts.ResponseTopicKeyField == "" {
		opts.ResponseTopicKeyField = "responseTopicKey"
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.MaxInFlight <= 0 {
		opts.MaxInFlight = 200
	}
	opts.NSQDAddress = strings.TrimSpace(opts.NSQDAddress)
	if opts.NSQDAddress == "" {
		opts.NSQDAddress = strings.TrimSpace(cfg.ChariotConfig.NSQDAddress)
	}
	if opts.ResponseProducerAddress == "" {
		opts.ResponseProducerAddress = opts.NSQDAddress
	}
	if opts.NSQDAddress == "" && len(opts.LookupdAddresses) == 0 {
		return nil, errors.New("nsq listen requires either nsqd address or lookupd addresses")
	}
	config := nsq.NewConfig()
	config.MaxInFlight = opts.MaxInFlight
	consumer, err := nsq.NewConsumer(opts.Topic, opts.Channel, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create nsq consumer: %w", err)
	}
	listener := &nsqListener{runtime: rt, opts: opts, consumer: consumer}
	consumer.AddConcurrentHandlers(nsq.HandlerFunc(listener.handleMessage), opts.Concurrency)
	if opts.OnStartProgram != "" {
		if err := listener.runtime.RunProgram(opts.OnStartProgram, 0); err != nil {
			cfg.ChariotLogger.Warn("listenNSQ onStart script failed", zapError(err), zap.String("topic", opts.Topic), zap.String("channel", opts.Channel))
		}
	}
	if len(opts.LookupdAddresses) > 0 {
		if err := consumer.ConnectToNSQLookupds(opts.LookupdAddresses); err != nil {
			consumer.Stop()
			return nil, fmt.Errorf("failed to connect to nsq lookupd: %w", err)
		}
	} else {
		if err := consumer.ConnectToNSQD(opts.NSQDAddress); err != nil {
			consumer.Stop()
			return nil, fmt.Errorf("failed to connect to nsqd: %w", err)
		}
	}
	if err := registerRuntimeNSQListener(rt, listener); err != nil {
		consumer.Stop()
		return nil, err
	}
	cfg.ChariotLogger.Info("nsq listener started", zap.String("topic", opts.Topic), zap.String("channel", opts.Channel))
	go listener.waitForStop()
	return listener, nil
}

func (l *nsqListener) waitForStop() {
	<-l.consumer.StopChan
	unregisterRuntimeNSQListener(l.runtime, listenerKey(l.opts.Topic, l.opts.Channel))
	if l.opts.OnExitProgram != "" {
		if err := l.runtime.RunProgram(l.opts.OnExitProgram, 0); err != nil {
			cfg.ChariotLogger.Warn("listenNSQ onExit script failed", zapError(err), zap.String("topic", l.opts.Topic), zap.String("channel", l.opts.Channel))
		}
	}
	cfg.ChariotLogger.Info("nsq listener stopped", zap.String("topic", l.opts.Topic), zap.String("channel", l.opts.Channel))
}

func registerRuntimeNSQListener(rt *Runtime, l *nsqListener) error {
	nsqListenersMu.Lock()
	defer nsqListenersMu.Unlock()
	key := listenerKey(l.opts.Topic, l.opts.Channel)
	if _, ok := runtimeNSQListeners[rt]; !ok {
		runtimeNSQListeners[rt] = make(map[string]*nsqListener)
	}
	if _, exists := runtimeNSQListeners[rt][key]; exists {
		return fmt.Errorf("nsq listener for topic %s channel %s already registered", l.opts.Topic, l.opts.Channel)
	}
	runtimeNSQListeners[rt][key] = l
	return nil
}

func unregisterRuntimeNSQListener(rt *Runtime, key string) {
	nsqListenersMu.Lock()
	defer nsqListenersMu.Unlock()
	if perRuntime, ok := runtimeNSQListeners[rt]; ok {
		delete(perRuntime, key)
		if len(perRuntime) == 0 {
			delete(runtimeNSQListeners, rt)
		}
	}
}

func listenerKey(topic, channel string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(topic)), strings.ToLower(strings.TrimSpace(channel)))
}

func (l *nsqListener) handleMessage(msg *nsq.Message) error {
	payload, err := l.decodeMessage(msg)
	if err != nil {
		cfg.ChariotLogger.Warn("nsq message decode failed", zapError(err))
		return nil
	}
	rawType, _ := payload["messageType"].(string)
	normalizedType := normalizeMessageTypeKey(rawType)
	if normalizedType == "" {
		cfg.ChariotLogger.Warn("nsq message missing messageType")
		return nil
	}
	binding, ok := l.opts.Handlers[normalizedType]
	if !ok {
		cfg.ChariotLogger.Warn("no handler registered for nsq message type", zap.String("messageType", rawType))
		return nil
	}
	valuePayload, err := JSONToValue(payload)
	if err != nil {
		cfg.ChariotLogger.Error("failed to convert nsq payload", zapError(err))
		return err
	}
	result, err := l.invokeHandler(binding, valuePayload)
	if err != nil {
		cfg.ChariotLogger.Error("nsq handler failed", zap.String("messageType", rawType), zapError(err))
		return err
	}
	responseKey := l.extractResponseTopicKey(payload, binding)
	if responseKey == "" {
		cfg.ChariotLogger.Warn("no response topic key resolved", zap.String("messageType", rawType))
		return nil
	}
	topicName, ok := ResolveResponseTopic(responseKey)
	if !ok {
		cfg.ChariotLogger.Error("response topic key not registered", zap.String("key", responseKey))
		return nil
	}
	bytes, err := valueToJSONBytes(result)
	if err != nil {
		cfg.ChariotLogger.Error("failed to encode nsq handler result", zapError(err))
		return err
	}
	addr := l.opts.ResponseProducerAddress
	if addr == "" {
		addr = l.opts.NSQDAddress
	}
	if err := publishNSQPayload(addr, topicName, bytes); err != nil {
		cfg.ChariotLogger.Error("failed to publish nsq response", zap.String("topic", topicName), zapError(err))
		return err
	}
	return nil
}

func (l *nsqListener) invokeHandler(binding *nsqHandlerBinding, payload Value) (Value, error) {
	fn := binding.HandlerFn
	if fn == nil {
		var ok bool
		fn, ok = l.runtime.GetFunction(binding.HandlerName)
		if !ok {
			return nil, fmt.Errorf("handler '%s' not found", binding.HandlerName)
		}
	}
	l.execMu.Lock()
	defer l.execMu.Unlock()
	return executeFunctionValue(l.runtime, fn, []Value{payload})
}

func (l *nsqListener) extractResponseTopicKey(payload map[string]interface{}, binding *nsqHandlerBinding) string {
	field := strings.TrimSpace(l.opts.ResponseTopicKeyField)
	var key string
	if field != "" {
		if raw, ok := payload[field]; ok {
			key, _ = raw.(string)
		}
	}
	if key == "" {
		key = binding.ResponseTopicKey
	}
	return normalizeResponseTopicKey(key)
}

func (l *nsqListener) decodeMessage(msg *nsq.Message) (map[string]interface{}, error) {
	var payload interface{}
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		return nil, fmt.Errorf("invalid nsq message payload: %w", err)
	}
	data, ok := payload.(map[string]interface{})
	if !ok {
		return nil, errors.New("nsq message must be a JSON object")
	}
	meta := map[string]interface{}{
		"attempts":  int(msg.Attempts),
		"timestamp": time.Unix(0, msg.Timestamp).UTC().Format(time.RFC3339Nano),
		"id":        hex.EncodeToString(msg.ID[:]),
	}
	data["_nsq"] = meta
	data["_raw"] = string(msg.Body)
	return data, nil
}

func buildNSQListenerOptions(topicArg, channelArg, handlersArg Value, optArg Value) (nsqListenerOptions, error) {
	topic, err := valueToNonEmptyString(topicArg, "topic")
	if err != nil {
		return nsqListenerOptions{}, err
	}
	channel, err := valueToNonEmptyString(channelArg, "channel")
	if err != nil {
		return nsqListenerOptions{}, err
	}
	handlers, err := parseNSQHandlersValue(handlersArg)
	if err != nil {
		return nsqListenerOptions{}, err
	}
	opts := nsqListenerOptions{Topic: topic, Channel: channel, Handlers: handlers, ResponseTopicKeyField: "responseTopicKey"}
	if optArg != nil && optArg != DBNull {
		if err := applyNSQListenerOptionValue(&opts, optArg); err != nil {
			return nsqListenerOptions{}, err
		}
	}
	return opts, nil
}

func parseNSQHandlersValue(value Value) (map[string]*nsqHandlerBinding, error) {
	bindings := make(map[string]*nsqHandlerBinding)
	switch v := value.(type) {
	case *MapValue:
		for key, raw := range v.Values {
			binding, err := parseNSQHandlerBinding(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid handler for messageType %s: %w", key, err)
			}
			normalized := normalizeMessageTypeKey(key)
			if normalized == "" {
				return nil, fmt.Errorf("messageType key cannot be empty")
			}
			binding.messageType = normalized
			bindings[normalized] = binding
		}
	case *JSONNode:
		native := convertValueToNative(v)
		data, ok := native.(map[string]interface{})
		if !ok {
			return nil, errors.New("handlers JSON must be an object")
		}
		for key, raw := range data {
			var binding *nsqHandlerBinding
			var err error
			switch entry := raw.(type) {
			case string:
				binding, err = parseNSQHandlerBinding(Str(entry))
			case map[string]interface{}:
				binding, err = parseNSQHandlerBindingNative(entry)
			default:
				err = fmt.Errorf("unsupported handler entry type %T", raw)
			}
			if err != nil {
				return nil, fmt.Errorf("invalid handler for messageType %s: %w", key, err)
			}
			normalized := normalizeMessageTypeKey(key)
			if normalized == "" {
				return nil, fmt.Errorf("messageType key cannot be empty")
			}
			binding.messageType = normalized
			bindings[normalized] = binding
		}
	default:
		return nil, fmt.Errorf("handlers argument must be a map, got %T", value)
	}
	if len(bindings) == 0 {
		return nil, errors.New("at least one handler must be registered")
	}
	return bindings, nil
}

func parseNSQHandlerBinding(raw Value) (*nsqHandlerBinding, error) {
	switch v := raw.(type) {
	case Str:
		name := strings.TrimSpace(string(v))
		if name == "" {
			return nil, errors.New("handler name cannot be empty")
		}
		return &nsqHandlerBinding{HandlerName: name}, nil
	case *FunctionValue:
		return &nsqHandlerBinding{HandlerFn: v}, nil
	case *MapValue:
		return parseNSQHandlerBindingMap(v.Values)
	case *JSONNode:
		native := convertValueToNative(v)
		if data, ok := native.(map[string]interface{}); ok {
			return parseNSQHandlerBindingNative(data)
		}
		return nil, errors.New("handler object must be map-like")
	default:
		return nil, fmt.Errorf("unsupported handler binding type %T", raw)
	}
}

func parseNSQHandlerBindingMap(data map[string]Value) (*nsqHandlerBinding, error) {
	binding := &nsqHandlerBinding{}
	if handlerVal, ok := data["handler"]; ok {
		if err := assignHandlerReference(binding, handlerVal); err != nil {
			return nil, err
		}
	} else if handlerVal, ok := data["handlerName"]; ok {
		if err := assignHandlerReference(binding, handlerVal); err != nil {
			return nil, err
		}
	}
	if respVal, ok := data["responseTopicKey"]; ok {
		if str, ok := respVal.(Str); ok {
			binding.ResponseTopicKey = normalizeResponseTopicKey(string(str))
		}
	}
	if binding.HandlerFn == nil && binding.HandlerName == "" {
		return nil, errors.New("handler binding requires a handler reference")
	}
	return binding, nil
}

func parseNSQHandlerBindingNative(data map[string]interface{}) (*nsqHandlerBinding, error) {
	binding := &nsqHandlerBinding{}
	if handlerVal, ok := data["handler"].(string); ok {
		binding.HandlerName = strings.TrimSpace(handlerVal)
	}
	if binding.HandlerName == "" {
		return nil, errors.New("handler name is required in JSON handler config")
	}
	if respVal, ok := data["responseTopicKey"].(string); ok {
		binding.ResponseTopicKey = normalizeResponseTopicKey(respVal)
	}
	return binding, nil
}

func assignHandlerReference(binding *nsqHandlerBinding, raw Value) error {
	switch hv := raw.(type) {
	case Str:
		binding.HandlerName = strings.TrimSpace(string(hv))
	case *FunctionValue:
		binding.HandlerFn = hv
	default:
		return fmt.Errorf("handler reference must be a string or function, got %T", raw)
	}
	return nil
}

func applyNSQListenerOptionValue(opts *nsqListenerOptions, raw Value) error {
	switch v := raw.(type) {
	case *MapValue:
		return applyNSQListenerOptionsMap(opts, convertValueMapToNative(v))
	case *JSONNode:
		native := convertValueToNative(v)
		data, ok := native.(map[string]interface{})
		if !ok {
			return fmt.Errorf("nsq options must be an object, got %T", native)
		}
		return applyNSQListenerOptionsMap(opts, data)
	default:
		return fmt.Errorf("unsupported nsq options type %T", raw)
	}
}

func applyNSQListenerOptionsMap(opts *nsqListenerOptions, data map[string]interface{}) error {
	for key, value := range data {
		switch strings.ToLower(key) {
		case "nsqd", "address", "nsq_addr":
			if s, ok := value.(string); ok {
				opts.NSQDAddress = strings.TrimSpace(s)
			}
		case "lookupd", "lookupds":
			slice, err := parseStringSlice(value)
			if err != nil {
				return err
			}
			opts.LookupdAddresses = slice
		case "concurrency":
			if n, ok := parseInt(value); ok {
				opts.Concurrency = n
			}
		case "maxinflight", "max_inflight":
			if n, ok := parseInt(value); ok {
				opts.MaxInFlight = n
			}
		case "onstart", "on_start":
			if s, ok := value.(string); ok {
				opts.OnStartProgram = strings.TrimSpace(s)
			}
		case "onexit", "on_exit":
			if s, ok := value.(string); ok {
				opts.OnExitProgram = strings.TrimSpace(s)
			}
		case "responsefield", "response_topic_field":
			if s, ok := value.(string); ok {
				opts.ResponseTopicKeyField = strings.TrimSpace(s)
			}
		case "responseaddr", "response_nsqd":
			if s, ok := value.(string); ok {
				opts.ResponseProducerAddress = strings.TrimSpace(s)
			}
		case "topic":
			if s, ok := value.(string); ok {
				opts.Topic = strings.TrimSpace(s)
			}
		case "channel":
			if s, ok := value.(string); ok {
				opts.Channel = strings.TrimSpace(s)
			}
		}
	}
	return nil
}

func convertValueMapToNative(m *MapValue) map[string]interface{} {
	native := make(map[string]interface{}, len(m.Values))
	for k, v := range m.Values {
		native[k] = convertValueToNative(v)
	}
	return native
}

func valueToNonEmptyString(val Value, name string) (string, error) {
	switch v := val.(type) {
	case Str:
		trimmed := strings.TrimSpace(string(v))
		if trimmed == "" {
			return "", fmt.Errorf("%s cannot be empty", name)
		}
		return trimmed, nil
	default:
		return "", fmt.Errorf("%s must be a string, got %T", name, val)
	}
}

func parseInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	case int:
		return v, true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func normalizeMessageTypeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildNSQListenerResult(opts nsqListenerOptions) *MapValue {
	values := map[string]Value{
		"status":  Str("listening"),
		"topic":   Str(opts.Topic),
		"channel": Str(opts.Channel),
	}
	arr := NewArray()
	for _, binding := range opts.Handlers {
		handlerDesc := NewMapWithValues(map[string]Value{
			"messageType": Str(binding.messageType),
		})
		name := binding.HandlerName
		if name == "" && binding.HandlerFn != nil {
			name = "<inline>"
		}
		if name != "" {
			handlerDesc.Set("handler", Str(name))
		}
		if binding.ResponseTopicKey != "" {
			handlerDesc.Set("responseTopicKey", Str(binding.ResponseTopicKey))
		}
		arr.Append(handlerDesc)
	}
	values["handlers"] = arr
	return NewMapWithValues(values)
}
