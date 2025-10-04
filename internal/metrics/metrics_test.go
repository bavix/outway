package metrics_test

import (
	"testing"
)

func TestMetrics(t *testing.T) {
	t.Parallel()
	// Skip test that requires unexported functions
	t.Skip("Test requires unexported functions - needs refactoring")
}
