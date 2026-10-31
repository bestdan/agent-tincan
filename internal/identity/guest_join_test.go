package identity

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A plain code that outlives its name becoming a guest dies at join: the
// tier does not change quietly, even with a code already in hand.
func TestPlainCodeCannotTakeAGuestName(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()
	who := staticResolver{"100.0.0.5:1": Node{ID: "nNEW", Name: "newer", User: "owner@example.com"}}
	d := NewDirectory(st, who, Config{})
	if err := st.PutAgent(ctx, Agent{Name: "friend", NodeID: "nOLD", Guest: GuestTerms{Until: time.Now().Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutInvite(ctx, Invite{Code: hashInviteCode(d.invitePepper, "AAAA-BBBB"), Name: "friend", Expires: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Join(ctx, "100.0.0.5:1", "AAAA-BBBB"); !errors.Is(err, ErrGuestTier) {
		t.Fatalf("join = %v", err)
	}
	if a, _, _ := st.AgentByName(ctx, "friend"); a.NodeID != "nOLD" || !a.Guest.Set() {
		t.Fatalf("agent = %+v", a)
	}
}

type staticResolver map[string]Node

func (r staticResolver) WhoIs(_ context.Context, addr string) (Node, error) {
	n, ok := r[addr]
	if !ok {
		return Node{}, errors.New("unknown node")
	}
	return n, nil
}
