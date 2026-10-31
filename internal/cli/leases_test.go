package cli

import (
	"strings"
	"testing"

	"github.com/mvanhorn/agent-tincan/internal/client"
	"github.com/mvanhorn/agent-tincan/internal/relay"
	"github.com/mvanhorn/agent-tincan/internal/testrelay"
)

// tincan lease takes, lists and drops a lease as the configured agent. A
// conflict is an error naming the holder, and an admin device drops any
// lease with --relay.
func TestLeaseCommands(t *testing.T) {
	m := testrelay.New(t, relay.Config{})
	useConfig(t, client.Config{Relay: m.URL("muse"), Agent: "muse"})
	if out, err := run(t, leaseCmd(), "acquire", "repo/internal/relay/*", "--ttl", "30m", "--note", "tests"); err != nil || !strings.Contains(out, "Leased") || !strings.Contains(out, "repo/internal/relay/*") {
		t.Fatalf("acquire = %q, %v", out, err)
	}
	if out, err := run(t, leaseCmd(), "acquire", "repo/internal/relay/*"); err != nil || !strings.Contains(out, "Renewed") {
		t.Fatalf("renew = %q, %v", out, err)
	}
	if out, err := run(t, leaseCmd(), "list"); err != nil || !strings.Contains(out, "muse") || !strings.Contains(out, `note="tests"`) {
		t.Fatalf("list = %q, %v", out, err)
	}
	if _, err := run(t, leaseCmd(), "acquire", "x", "--ttl", "25h"); err == nil {
		t.Fatal("took a 25-hour lease")
	}
	if out, err := run(t, leaseCmd(), "release", "repo/internal/relay/*"); err != nil || !strings.Contains(out, "Released") {
		t.Fatalf("release = %q, %v", out, err)
	}
	if out, err := run(t, leaseCmd(), "list"); err != nil || !strings.Contains(out, "No leases") {
		t.Fatalf("empty list = %q, %v", out, err)
	}

	if _, err := client.AcquireLease(t.Context(), m.Client(t, "grokbot"), "shared", "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, leaseCmd(), "acquire", "shared"); err == nil || !strings.Contains(err.Error(), "grokbot") {
		t.Fatalf("conflict = %v", err)
	}
	if _, err := run(t, leaseCmd(), "release", "shared"); err == nil {
		t.Fatal("released a teammate's lease")
	}
	if out, err := run(t, leaseCmd(), "release", "shared", "--relay", m.URL("admin")); err != nil || !strings.Contains(out, "grokbot") {
		t.Fatalf("admin release = %q, %v", out, err)
	}
}
