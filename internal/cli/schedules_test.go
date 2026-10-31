package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mvanhorn/agent-tincan/internal/client"
	"github.com/mvanhorn/agent-tincan/internal/envelope"
	"github.com/mvanhorn/agent-tincan/internal/relay"
	"github.com/mvanhorn/agent-tincan/internal/testrelay"
)

func TestAskScheduleCommands(t *testing.T) {
	m := testrelay.New(t, relay.Config{})
	useConfig(t, client.Config{Relay: m.URL("grokbot"), Agent: "grokbot"})
	out, err := run(t, askCmd(), "muse", "weekly", "report", "--at", "2h", "--every", "1h")
	if err != nil || !strings.Contains(out, "Scheduled ask #") {
		t.Fatalf("ask = %q, %v", out, err)
	}
	if inbox, err := m.Client(t, "muse").Poll(t.Context(), 0); err != nil || len(inbox.Requests) != 0 {
		t.Fatalf("sent at once: %+v, %v", inbox, err)
	}
	out, err = run(t, schedulesCmd())
	if err != nil || !strings.Contains(out, "grokbot to muse") || !strings.Contains(out, "weekly report") {
		t.Fatalf("schedules = %q, %v", out, err)
	}
	all, err := client.Schedules(t.Context(), m.Client(t, "grokbot"), false)
	if err != nil || len(all) != 1 {
		t.Fatalf("list = %+v, %v", all, err)
	}
	out, err = run(t, schedulesCmd(), "--json")
	var got []envelope.Schedule
	if err != nil || json.Unmarshal([]byte(out), &got) != nil || len(got) != 1 || got[0].To != "muse" {
		t.Fatalf("--json = %q, %v", out, err)
	}
	if out, err := run(t, unscheduleCmd(), "not-a-number"); err == nil {
		t.Fatalf("bad id = %q", out)
	}
	out, err = run(t, unscheduleCmd(), fmt.Sprint(all[0].ID))
	if err != nil || !strings.Contains(out, "Cancelled") {
		t.Fatalf("unschedule = %q, %v", out, err)
	}
}

func TestAskScheduleRefusals(t *testing.T) {
	m := testrelay.New(t, relay.Config{})
	useConfig(t, client.Config{Relay: m.URL("grokbot"), Agent: "grokbot"})
	for name, args := range map[string][]string{
		"several targets": {"muse,instinct", "x", "--every", "1h"},
		"notify":          {"muse", "x", "--every", "1h", "--notify"},
		"parent":          {"muse", "x", "--every", "1h", "--parent", "abc"},
		"attach":          {"muse", "x", "--every", "1h", "--attach", "/tmp/x"},
		"too often":       {"muse", "x", "--every", "5m"},
		"past":            {"muse", "x", "--at", "2001-01-01T00:00:00Z"},
		"bad at":          {"muse", "x", "--at", "tomorrow"},
	} {
		t.Run(name, func(t *testing.T) {
			if out, err := run(t, askCmd(), args...); err == nil {
				t.Fatalf("ask %v = %q", args, out)
			}
		})
	}
	if inbox, err := m.Client(t, "muse").Poll(t.Context(), 0); err != nil || len(inbox.Requests) != 0 {
		t.Fatalf("sent = %+v, %v", inbox, err)
	}
}
