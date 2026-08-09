package kafka

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixedTopic(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_PREFIX", "")
	require.Equal(t, "payment.transaction.updated", PrefixedTopic("payment.transaction.updated"))

	t.Setenv("KAFKA_TOPIC_PREFIX", "dev.")
	require.Equal(t, "dev.payment.transaction.updated", PrefixedTopic("payment.transaction.updated"))
	require.Equal(t, "", PrefixedTopic(""))
	require.Equal(t, []string{"dev.a", "dev.b"}, PrefixedTopics([]string{"a", "b"}))
}

func TestLogicalTopic(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_PREFIX", "dev.")
	require.Equal(t, "payment.transaction.updated", LogicalTopic("dev.payment.transaction.updated"))
	require.Equal(t, "other.topic", LogicalTopic("other.topic"))

	require.NoError(t, os.Unsetenv("KAFKA_TOPIC_PREFIX"))
	require.Equal(t, "dev.payment.transaction.updated", LogicalTopic("dev.payment.transaction.updated"))
}
