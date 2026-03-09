package chariot

import (
	"fmt"
	"strings"
	"sync"
)

var (
	responseTopicsMu sync.RWMutex
	responseTopics   = make(map[string]string)
)

// RegisterResponseTopic stores a mapping between a logical key and a concrete NSQ topic name.
// Keys are normalized to lowercase to avoid accidental duplication.
func RegisterResponseTopic(key, topic string) error {
	normalized := normalizeResponseTopicKey(key)
	if normalized == "" {
		return fmt.Errorf("response topic key cannot be empty")
	}
	trimmedTopic := strings.TrimSpace(topic)
	if trimmedTopic == "" {
		return fmt.Errorf("response topic value cannot be empty")
	}
	responseTopicsMu.Lock()
	responseTopics[normalized] = trimmedTopic
	responseTopicsMu.Unlock()
	return nil
}

// RegisterResponseTopicsFromSpec registers keys from a comma-separated list of key:topic pairs.
func RegisterResponseTopicsFromSpec(spec string) error {
	entries := strings.Split(spec, ",")
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid response topic entry %q", trimmed)
		}
		if err := RegisterResponseTopic(parts[0], parts[1]); err != nil {
			return err
		}
	}
	return nil
}

// ResolveResponseTopic returns the concrete topic name for a logical key.
func ResolveResponseTopic(key string) (string, bool) {
	normalized := normalizeResponseTopicKey(key)
	if normalized == "" {
		return "", false
	}
	responseTopicsMu.RLock()
	topic, ok := responseTopics[normalized]
	responseTopicsMu.RUnlock()
	return topic, ok
}

// ListResponseTopics returns a copy of the currently registered response topics.
func ListResponseTopics() map[string]string {
	responseTopicsMu.RLock()
	defer responseTopicsMu.RUnlock()
	clone := make(map[string]string, len(responseTopics))
	for k, v := range responseTopics {
		clone[k] = v
	}
	return clone
}

func normalizeResponseTopicKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(trimmed)
}
