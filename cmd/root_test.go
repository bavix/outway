package cmd_test

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	t.Parallel()
	// Skip test that requires unexported functions
	t.Skip("Test requires unexported functions - needs refactoring")
}
