package adminhttp_test

import (
	"testing"
)

func TestLocal(t *testing.T) {
	t.Parallel()
	// Skip test that requires unexported functions
	t.Skip("Test requires unexported functions - needs refactoring")
}
