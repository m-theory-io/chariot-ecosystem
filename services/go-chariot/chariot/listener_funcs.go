package chariot

import (
	"errors"
	"fmt"
)

// RegisterListenerFunctions registers application-server builtins: HTTP and
// NSQ listeners plus NSQ response-topic management. These are excluded from
// embedded runtimes via Without(CategoryListeners) or EmbeddedProfile().
func RegisterListenerFunctions(rt *Runtime) {
	rt.Register("listen", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("listen requires 2 or 3 arguments: port, handler, and optional options map")
		}

		for i, arg := range args {
			if entry, ok := arg.(ScopeEntry); ok {
				args[i] = entry.Value
			}
		}

		port, err := parseListenerPort(args[0])
		if err != nil {
			return nil, err
		}

		handlerRef, err := resolveListenerHandler(args[1])
		if err != nil {
			return nil, err
		}

		opts := listenerOptions{
			Port:        port,
			HandlerName: handlerRef.name,
			HandlerFn:   handlerRef.fn,
		}

		if len(args) >= 3 {
			if err := applyListenerOptionValue(&opts, args[2]); err != nil {
				return nil, err
			}
		}

		listener, err := startRuntimeListener(rt, opts)
		if err != nil {
			return nil, err
		}

		return buildListenerResult(listener), nil
	})

	rt.Register("httpResponse", func(args ...Value) (Value, error) {
		return buildHTTPResponseValue(args...)
	})

	rt.Register("listenerList", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("listenerList accepts no arguments")
		}
		result := NewArray()
		for _, status := range rt.ListListeners() {
			result.Append(listenerStatusValue(status))
		}
		return result, nil
	})

	rt.Register("listenerStatus", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("listenerStatus requires 1 argument: listener id")
		}
		id, err := listenerIDArgument(args[0])
		if err != nil {
			return nil, err
		}
		for _, status := range rt.ListListeners() {
			if status.ID == id {
				return listenerStatusValue(status), nil
			}
		}
		return DBNull, nil
	})

	rt.Register("listenerStop", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("listenerStop requires 1 argument: listener id")
		}
		id, err := listenerIDArgument(args[0])
		if err != nil {
			return nil, err
		}
		// true means accepted, not drained. Waiting here would deadlock a
		// guarded execution against its own handler or the listener's exit hook.
		if err := rt.RequestStopListener(id); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("listenNSQ", func(args ...Value) (Value, error) {
		if len(args) < 3 {
			return nil, errors.New("listenNSQ requires 3 arguments: topic, channel, handlers map")
		}
		for i, arg := range args {
			if entry, ok := arg.(ScopeEntry); ok {
				args[i] = entry.Value
			}
		}
		var optsArg Value
		if len(args) >= 4 {
			optsArg = args[3]
		}
		opts, err := buildNSQListenerOptions(args[0], args[1], args[2], optsArg)
		if err != nil {
			return nil, err
		}
		listener, err := startRuntimeNSQListener(rt, opts)
		if err != nil {
			return nil, err
		}
		return buildNSQListenerResult(listener.opts), nil
	})

	rt.Register("registerResponseTopic", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("registerResponseTopic requires 2 arguments: key and topic")
		}
		key, ok := args[0].(Str)
		if !ok {
			return nil, fmt.Errorf("response topic key must be a string, got %T", args[0])
		}
		topic, ok := args[1].(Str)
		if !ok {
			return nil, fmt.Errorf("response topic value must be a string, got %T", args[1])
		}
		if err := RegisterResponseTopic(string(key), string(topic)); err != nil {
			return nil, err
		}
		return Bool(true), nil
	})

	rt.Register("listResponseTopics", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("listResponseTopics accepts no arguments")
		}
		registered := ListResponseTopics()
		result := NewMap()
		for k, v := range registered {
			result.Set(k, Str(v))
		}
		return result, nil
	})
}
