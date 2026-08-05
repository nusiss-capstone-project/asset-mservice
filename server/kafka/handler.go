package kafka

import "sync"

var (
	handlerMu sync.RWMutex
	handlers  = make(map[string][]Handler)
)

// RegisterHandler registers a handler for a topic.
// The same topic can register multiple handlers; they run in parallel per message.
func RegisterHandler(topic string, handler Handler) {
	if topic == "" || handler == nil {
		return
	}
	handlerMu.Lock()
	defer handlerMu.Unlock()
	handlers[topic] = append(handlers[topic], handler)
}

// HandlersForTopic returns registered handlers for a topic.
func HandlersForTopic(topic string) []Handler {
	handlerMu.RLock()
	defer handlerMu.RUnlock()
	hs := handlers[topic]
	if len(hs) == 0 {
		return nil
	}
	result := make([]Handler, len(hs))
	copy(result, hs)
	return result
}

// RegisteredTopics returns topics that have at least one handler.
func RegisteredTopics() []string {
	handlerMu.RLock()
	defer handlerMu.RUnlock()
	topics := make([]string, 0, len(handlers))
	for topic, hs := range handlers {
		if len(hs) > 0 {
			topics = append(topics, topic)
		}
	}
	return topics
}
