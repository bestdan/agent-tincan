package client_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mvanhorn/agent-tincan/internal/client"
)

// oldRelay is a relay that answers an invite as a relay from before guests
// does: it mints a code and ignores the guest field. guests sets its
// capability, so the second check is reachable too.
func oldRelay(t *testing.T, guests bool) (*client.Relay, *int) {
	t.Helper()
	invites := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/capabilities":
			_ = json.NewEncoder(w).Encode(client.Capabilities{Guests: guests})
		case "/v1/admin/invite":
			invites++
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "friend", "code": "AAAA-BBBB"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	r, err := client.NewRelayFor(client.Config{Relay: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return r, &invites
}

// A relay without the capability never sees the invite: an old relay would
// ignore the guest field and mint a full teammate.
func TestInviteGuestRefusesARelayWithoutTheCapability(t *testing.T) {
	r, invites := oldRelay(t, false)
	_, err := r.InviteGuest(t.Context(), "friend", client.GuestInvite{Ask: []string{"muse"}, ForSeconds: 3600})
	if !errors.Is(err, client.ErrGuestsUnsupported) {
		t.Fatalf("err = %v", err)
	}
	if *invites != 0 {
		t.Fatalf("%d invites sent", *invites)
	}
}

// A relay that took the invite but echoed no limits gets no code printed.
func TestInviteGuestRefusesACodeWithNoLimits(t *testing.T) {
	r, _ := oldRelay(t, true)
	inv, err := r.InviteGuest(t.Context(), "friend", client.GuestInvite{Ask: []string{"muse"}, ForSeconds: 3600})
	if !errors.Is(err, client.ErrGuestsUnsupported) || inv.Code != "" {
		t.Fatalf("invite = %+v, %v", inv, err)
	}
	if strings.Contains(err.Error(), "AAAA") {
		t.Fatalf("error carries the code: %v", err)
	}
}
