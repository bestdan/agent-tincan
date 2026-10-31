//go:build !unix

package cli

import (
	"strings"
	"testing"
)

func TestListenPerRequestRefusesThisPlatform(t *testing.T) {
	cmd := listenCmd()
	cmd.SetArgs([]string{"--exec", "true", "--per-request"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	err := cmd.ExecuteContext(t.Context())
	if err == nil || !strings.Contains(err.Error(), "does not run on this platform") {
		t.Fatalf("listen --per-request = %v", err)
	}
}
