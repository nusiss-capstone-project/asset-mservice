package kafka

import (
	"os"
	"strings"
)

const topicPrefixEnv = "KAFKA_TOPIC_PREFIX"

// TopicPrefix returns KAFKA_TOPIC_PREFIX (empty when unset).
func TopicPrefix() string {
	return os.Getenv(topicPrefixEnv)
}

// PrefixedTopic prepends KAFKA_TOPIC_PREFIX when set.
func PrefixedTopic(topic string) string {
	if topic == "" {
		return topic
	}
	prefix := TopicPrefix()
	if prefix == "" {
		return topic
	}
	return prefix + topic
}

// PrefixedTopics applies PrefixedTopic to each topic.
func PrefixedTopics(topics []string) []string {
	if len(topics) == 0 {
		return topics
	}
	out := make([]string, len(topics))
	for i, topic := range topics {
		out[i] = PrefixedTopic(topic)
	}
	return out
}

// LogicalTopic strips KAFKA_TOPIC_PREFIX when present so handler lookup uses logical names.
func LogicalTopic(topic string) string {
	prefix := TopicPrefix()
	if prefix == "" || topic == "" {
		return topic
	}
	return strings.TrimPrefix(topic, prefix)
}
