package chariot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	Topic                    string
	Channel                  string
	NSQDAddress              string
	LookupdAddresses         []string
	Handlers                 map[string]*nsqHandlerBinding
	MaxInFlight              int
	Concurrency              int
	OnStartProgram           string
	OnExitProgram            string
	ResponseTopicKeyField    string
	ResponseProducerAddress  string
	ResponseRequired         bool
	MaxAttempts              int
	BackoffStrategy          string
	BackoffMultiplier        time.Duration
	MaxBackoffDuration       time.Duration
	DeadLetterTopic          string
	DecodeErrorPolicy        string
	MissingMessageTypePolicy string
	MissingHandlerPolicy     string
	MissingResponsePolicy    string
	UnknownResponsePolicy    string
}

const (
	nsqFailureAck        = "ack"
	nsqFailureRequeue    = "requeue"
	nsqFailureDeadLetter = "dead_letter"
)

type nsqListener struct {
	runtime      *Runtime
	opts         nsqListenerOptions
	consumer     *nsq.Consumer
	publish      func(string, string, []byte) error
	metrics      listenerMetrics
	done         chan struct{}
	shutdownOnce sync.Once
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
	if opts.MaxAttempts == 0 {
		opts.MaxAttempts = 5
	}
	if opts.BackoffStrategy == "" {
		opts.BackoffStrategy = "exponential"
	}
	if opts.BackoffMultiplier == 0 {
		opts.BackoffMultiplier = time.Second
	}
	if opts.MaxBackoffDuration == 0 {
		opts.MaxBackoffDuration = 2 * time.Minute
	}
	if opts.DecodeErrorPolicy == "" {
		opts.DecodeErrorPolicy = nsqFailureDeadLetter
	}
	if opts.MissingMessageTypePolicy == "" {
		opts.MissingMessageTypePolicy = nsqFailureDeadLetter
	}
	if opts.MissingHandlerPolicy == "" {
		opts.MissingHandlerPolicy = nsqFailureDeadLetter
	}
	if opts.MissingResponsePolicy == "" {
		opts.MissingResponsePolicy = nsqFailureDeadLetter
	}
	if opts.UnknownResponsePolicy == "" {
		opts.UnknownResponsePolicy = nsqFailureDeadLetter
	}
	opts.NSQDAddress = strings.TrimSpace(opts.NSQDAddress)
	if opts.NSQDAddress == "" {
		opts.NSQDAddress = strings.TrimSpace(cfg.ChariotConfig.NSQDAddress)
	}
	if opts.ResponseProducerAddress == "" {
		opts.ResponseProducerAddress = opts.NSQDAddress
	}
	if opts.DeadLetterTopic != "" && opts.ResponseProducerAddress == "" {
		return nil, errors.New("dead_letter_topic requires nsqd_address or response_nsqd_address")
	}
	if opts.NSQDAddress == "" && len(opts.LookupdAddresses) == 0 {
		return nil, errors.New("nsq listen requires either nsqd address or lookupd addresses")
	}
	config := nsq.NewConfig()
	config.MaxInFlight = opts.MaxInFlight
	// Chariot enforces MaxAttempts in handleMessage. Leave the client unlimited
	// so a failed dead-letter publish can requeue and retry instead of being
	// discarded by go-nsq before the handler is invoked again.
	config.MaxAttempts = 0
	config.BackoffMultiplier = opts.BackoffMultiplier
	config.MaxBackoffDuration = opts.MaxBackoffDuration
	switch opts.BackoffStrategy {
	case "exponential":
		config.BackoffStrategy = &nsq.ExponentialStrategy{}
	case "full_jitter":
		config.BackoffStrategy = &nsq.FullJitterStrategy{}
	default:
		return nil, fmt.Errorf("unsupported NSQ backoff strategy %q", opts.BackoffStrategy)
	}
	consumer, err := nsq.NewConsumer(opts.Topic, opts.Channel, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create nsq consumer: %w", err)
	}
	listener := &nsqListener{
		runtime:  rt,
		opts:     opts,
		consumer: consumer,
		publish:  publishNSQPayload,
		metrics:  newListenerMetrics(),
		done:     make(chan struct{}),
	}
	consumer.AddConcurrentHandlers(nsq.HandlerFunc(listener.handleMessage), opts.Concurrency)
	if opts.OnStartProgram != "" {
		if err := listener.runtime.RunProgram(opts.OnStartProgram, 0); err != nil {
			consumer.Stop()
			listener.metrics.recordError(err)
			listener.metrics.setState("failed")
			return nil, fmt.Errorf("listenNSQ onStart script failed: %w", err)
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
	listener.metrics.setState("running")
	cfg.ChariotLogger.Info("nsq listener started", zap.String("topic", opts.Topic), zap.String("channel", opts.Channel))
	go listener.waitForStop()
	return listener, nil
}

func (l *nsqListener) waitForStop() {
	defer close(l.done)
	<-l.consumer.StopChan
	defer unregisterRuntimeNSQListener(l.runtime, listenerKey(l.opts.Topic, l.opts.Channel))
	if l.opts.OnExitProgram != "" {
		if err := l.runtime.WithExecution(func() error {
			return l.runtime.RunProgram(l.opts.OnExitProgram, 0)
		}); err != nil {
			l.metrics.recordTerminalError(fmt.Errorf("listenNSQ onExit script failed: %w", err))
			cfg.ChariotLogger.Warn("listenNSQ onExit script failed", zapError(err), zap.String("topic", l.opts.Topic), zap.String("channel", l.opts.Channel))
		}
	}
	if l.metrics.terminalErr() != nil {
		l.metrics.setState("failed")
	} else {
		l.metrics.setState("stopped")
	}
	cfg.ChariotLogger.Info("nsq listener stopped", zap.String("topic", l.opts.Topic), zap.String("channel", l.opts.Channel))
}

func (l *nsqListener) requestStop(ctx context.Context) (started bool) {
	l.shutdownOnce.Do(func() {
		select {
		case <-l.done:
			return
		default:
		}
		started = true
		l.metrics.setState("stopping")
		// Consumer.Stop can initiate connection teardown; never run it on the
		// interpreter's guarded call stack or wait for consumer handlers here.
		go l.consumer.Stop()
	})
	return started
}

func (l *nsqListener) Shutdown(ctx context.Context) error {
	select {
	case <-l.done:
		return l.metrics.terminalErr()
	default:
	}

	l.requestStop(ctx)
	select {
	case <-l.done:
		return l.metrics.terminalErr()
	case <-ctx.Done():
		l.metrics.recordError(ctx.Err())
		l.metrics.setState("failed")
		return ctx.Err()
	}
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

func (l *nsqListener) handleMessage(msg *nsq.Message) (resultErr error) {
	l.metrics.recordStart()
	var processingErr error
	defer func() {
		if processingErr != nil {
			l.metrics.recordFailure(processingErr)
		} else {
			l.metrics.recordSuccess()
		}
	}()

	payload, err := l.decodeMessage(msg)
	if err != nil {
		processingErr = err
		cfg.ChariotLogger.Warn("nsq message decode failed", zapError(err))
		return l.applyFailurePolicy(msg, optsPolicy(l.opts.DecodeErrorPolicy), err)
	}
	rawType, _ := payload["messageType"].(string)
	normalizedType := normalizeMessageTypeKey(rawType)
	if normalizedType == "" {
		processingErr = errors.New("nsq message missing messageType")
		cfg.ChariotLogger.Warn("nsq message missing messageType")
		return l.applyFailurePolicy(msg, optsPolicy(l.opts.MissingMessageTypePolicy), processingErr)
	}
	binding, ok := l.opts.Handlers[normalizedType]
	if !ok {
		processingErr = fmt.Errorf("no handler registered for nsq message type %q", rawType)
		cfg.ChariotLogger.Warn("no handler registered for nsq message type", zap.String("messageType", rawType))
		return l.applyFailurePolicy(msg, optsPolicy(l.opts.MissingHandlerPolicy), processingErr)
	}
	valuePayload, err := JSONToValue(payload)
	if err != nil {
		processingErr = err
		cfg.ChariotLogger.Error("failed to convert nsq payload", zapError(err))
		return l.applyTransientFailure(msg, err)
	}
	result, err := l.invokeHandler(binding, valuePayload)
	if err != nil {
		processingErr = err
		cfg.ChariotLogger.Error("nsq handler failed", zap.String("messageType", rawType), zapError(err))
		return l.applyTransientFailure(msg, err)
	}
	responseKey := l.extractResponseTopicKey(payload, binding)
	if responseKey == "" {
		if !l.opts.ResponseRequired {
			return nil
		}
		processingErr = errors.New("no response topic key resolved")
		cfg.ChariotLogger.Warn("no response topic key resolved", zap.String("messageType", rawType))
		return l.applyFailurePolicy(msg, optsPolicy(l.opts.MissingResponsePolicy), processingErr)
	}
	topicName, ok := ResolveResponseTopic(responseKey)
	if !ok {
		processingErr = fmt.Errorf("response topic key %q is not registered", responseKey)
		cfg.ChariotLogger.Error("response topic key not registered", zap.String("key", responseKey))
		return l.applyFailurePolicy(msg, optsPolicy(l.opts.UnknownResponsePolicy), processingErr)
	}
	bytes, err := valueToJSONBytes(result)
	if err != nil {
		processingErr = err
		cfg.ChariotLogger.Error("failed to encode nsq handler result", zapError(err))
		return l.applyTransientFailure(msg, err)
	}
	addr := l.opts.ResponseProducerAddress
	if addr == "" {
		addr = l.opts.NSQDAddress
	}
	if err := l.publishPayload(addr, topicName, bytes); err != nil {
		processingErr = err
		cfg.ChariotLogger.Error("failed to publish nsq response", zap.String("topic", topicName), zapError(err))
		return l.applyTransientFailure(msg, err)
	}
	return nil
}

func optsPolicy(policy string) string {
	if policy == "" {
		return nsqFailureDeadLetter
	}
	return policy
}

func (l *nsqListener) applyTransientFailure(msg *nsq.Message, failure error) error {
	if int(msg.Attempts) < l.opts.MaxAttempts {
		return failure
	}
	return l.deadLetterOrAck(msg, failure)
}

func (l *nsqListener) applyFailurePolicy(msg *nsq.Message, policy string, failure error) error {
	switch policy {
	case nsqFailureAck:
		return nil
	case nsqFailureRequeue:
		return l.applyTransientFailure(msg, failure)
	case nsqFailureDeadLetter:
		return l.deadLetterOrAck(msg, failure)
	default:
		return fmt.Errorf("invalid NSQ failure policy %q", policy)
	}
}

func (l *nsqListener) deadLetterOrAck(msg *nsq.Message, failure error) error {
	if l.opts.DeadLetterTopic == "" {
		return nil
	}
	address := l.opts.ResponseProducerAddress
	if address == "" {
		address = l.opts.NSQDAddress
	}
	if err := l.publishPayload(address, l.opts.DeadLetterTopic, msg.Body); err != nil {
		return fmt.Errorf("publish NSQ dead letter after %v: %w", failure, err)
	}
	return nil
}

func (l *nsqListener) publishPayload(address, topic string, payload []byte) error {
	if l.publish != nil {
		return l.publish(address, topic, payload)
	}
	return publishNSQPayload(address, topic, payload)
}

func (l *nsqListener) id() string {
	return fmt.Sprintf("nsq:%s:%s", l.opts.Topic, l.opts.Channel)
}

func (l *nsqListener) status() ListenerStatus {
	address := l.opts.NSQDAddress
	if address == "" {
		address = strings.Join(l.opts.LookupdAddresses, ",")
	}
	return l.metrics.snapshot(ListenerStatus{
		ID:      l.id(),
		Kind:    ListenerNSQ,
		Address: address,
		Topic:   l.opts.Topic,
		Channel: l.opts.Channel,
	})
}

func runtimeNSQListenerSnapshot(rt *Runtime) []*nsqListener {
	nsqListenersMu.Lock()
	defer nsqListenersMu.Unlock()
	perRuntime := runtimeNSQListeners[rt]
	listeners := make([]*nsqListener, 0, len(perRuntime))
	for _, listener := range perRuntime {
		listeners = append(listeners, listener)
	}
	return listeners
}

func listRuntimeNSQListenerStatuses(rt *Runtime) []ListenerStatus {
	listeners := runtimeNSQListenerSnapshot(rt)
	statuses := make([]ListenerStatus, 0, len(listeners))
	for _, listener := range listeners {
		statuses = append(statuses, listener.status())
	}
	return statuses
}

func stopRuntimeNSQListener(ctx context.Context, rt *Runtime, id string) (bool, error) {
	for _, listener := range runtimeNSQListenerSnapshot(rt) {
		if listener.id() == id {
			return true, listener.Shutdown(ctx)
		}
	}
	return false, nil
}

func (l *nsqListener) invokeHandler(binding *nsqHandlerBinding, payload Value) (result Value, err error) {
	err = l.runtime.WithExecution(func() error {
		fn := binding.HandlerFn
		if fn == nil {
			var ok bool
			fn, ok = l.runtime.GetFunction(binding.HandlerName)
			if !ok {
				return fmt.Errorf("handler '%s' not found", binding.HandlerName)
			}
		}
		var execErr error
		result, execErr = executeFunctionValue(l.runtime, fn, []Value{payload})
		return execErr
	})
	return result, err
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
	opts := nsqListenerOptions{
		Topic: topic, Channel: channel, Handlers: handlers,
		ResponseTopicKeyField: "responseTopicKey", MaxAttempts: 5,
		BackoffStrategy: "exponential", BackoffMultiplier: time.Second,
		MaxBackoffDuration: 2 * time.Minute, DecodeErrorPolicy: nsqFailureDeadLetter,
		MissingMessageTypePolicy: nsqFailureDeadLetter, MissingHandlerPolicy: nsqFailureDeadLetter,
		MissingResponsePolicy: nsqFailureDeadLetter, UnknownResponsePolicy: nsqFailureDeadLetter,
	}
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
		switch key {
		case "nsqd_address":
			address, ok := value.(string)
			if !ok || strings.TrimSpace(address) == "" {
				return fmt.Errorf("nsqd_address must be a non-empty string, got %T", value)
			}
			opts.NSQDAddress = strings.TrimSpace(address)
		case "lookupd_addresses":
			slice, err := parseStrictStringSlice(value)
			if err != nil {
				return fmt.Errorf("invalid lookupd_addresses: %w", err)
			}
			opts.LookupdAddresses = slice
		case "concurrency":
			number, err := parsePositiveInt(value, "concurrency")
			if err != nil {
				return err
			}
			opts.Concurrency = number
		case "max_in_flight":
			number, err := parsePositiveInt(value, "max_in_flight")
			if err != nil {
				return err
			}
			opts.MaxInFlight = number
		case "max_attempts":
			number, err := parsePositiveInt(value, "max_attempts")
			if err != nil {
				return err
			}
			if number > 65535 {
				return errors.New("max_attempts must be at most 65535")
			}
			opts.MaxAttempts = number
		case "backoff_strategy":
			strategy, ok := value.(string)
			if !ok || (strategy != "exponential" && strategy != "full_jitter") {
				return errors.New("backoff_strategy must be \"exponential\" or \"full_jitter\"")
			}
			opts.BackoffStrategy = strategy
		case "backoff_multiplier_ms":
			duration, err := parseDurationMillis(value, "backoff_multiplier_ms")
			if err != nil {
				return err
			}
			opts.BackoffMultiplier = duration
		case "max_backoff_ms":
			duration, err := parseDurationMillis(value, "max_backoff_ms")
			if err != nil {
				return err
			}
			opts.MaxBackoffDuration = duration
		case "dead_letter_topic":
			topic, ok := value.(string)
			if !ok {
				return fmt.Errorf("dead_letter_topic must be a string, got %T", value)
			}
			opts.DeadLetterTopic = strings.TrimSpace(topic)
		case "decode_error_policy":
			policy, err := parseNSQFailurePolicy(value, key)
			if err != nil {
				return err
			}
			opts.DecodeErrorPolicy = policy
		case "missing_message_type_policy":
			policy, err := parseNSQFailurePolicy(value, key)
			if err != nil {
				return err
			}
			opts.MissingMessageTypePolicy = policy
		case "missing_handler_policy":
			policy, err := parseNSQFailurePolicy(value, key)
			if err != nil {
				return err
			}
			opts.MissingHandlerPolicy = policy
		case "missing_response_policy":
			policy, err := parseNSQFailurePolicy(value, key)
			if err != nil {
				return err
			}
			opts.MissingResponsePolicy = policy
		case "unknown_response_policy":
			policy, err := parseNSQFailurePolicy(value, key)
			if err != nil {
				return err
			}
			opts.UnknownResponsePolicy = policy
		case "on_start":
			program, ok := value.(string)
			if !ok {
				return fmt.Errorf("on_start must be a string, got %T", value)
			}
			opts.OnStartProgram = strings.TrimSpace(program)
		case "on_exit":
			program, ok := value.(string)
			if !ok {
				return fmt.Errorf("on_exit must be a string, got %T", value)
			}
			opts.OnExitProgram = strings.TrimSpace(program)
		case "response_topic_field":
			field, ok := value.(string)
			if !ok || strings.TrimSpace(field) == "" {
				return fmt.Errorf("response_topic_field must be a non-empty string")
			}
			opts.ResponseTopicKeyField = strings.TrimSpace(field)
		case "response_nsqd_address":
			address, ok := value.(string)
			if !ok || strings.TrimSpace(address) == "" {
				return fmt.Errorf("response_nsqd_address must be a non-empty string")
			}
			opts.ResponseProducerAddress = strings.TrimSpace(address)
		case "response_required":
			required, ok := value.(bool)
			if !ok {
				return fmt.Errorf("response_required must be a boolean, got %T", value)
			}
			opts.ResponseRequired = required
		default:
			return fmt.Errorf("unknown listenNSQ option %q", key)
		}
	}
	return nil
}

func parseNSQFailurePolicy(value interface{}, name string) (string, error) {
	policy, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string, got %T", name, value)
	}
	switch policy {
	case nsqFailureAck, nsqFailureRequeue, nsqFailureDeadLetter:
		return policy, nil
	default:
		return "", fmt.Errorf("%s must be ack, requeue, or dead_letter", name)
	}
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
