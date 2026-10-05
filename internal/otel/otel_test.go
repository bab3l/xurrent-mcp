package otel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitTracing_Noop(t *testing.T) {
	t.Parallel()
	shutdown, err := InitTracing()
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	shutdown() // should not panic
}

func TestInitMetrics_Noop(t *testing.T) {
	t.Parallel()
	shutdown, err := InitMetrics()
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	shutdown() // should not panic
}
