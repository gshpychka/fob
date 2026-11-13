package session_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/gshpychka/fob/internal/fakeaws"
	"github.com/gshpychka/fob/internal/session"
	"github.com/gshpychka/fob/internal/ssooidc"
	"github.com/gshpychka/fob/internal/store"
)

func TestLoginPacesRedemptionAsTheServiceDirects(t *testing.T) {
	aws := fakeaws.New(t)
	aws.RefuseRedemptions("authorization_pending", "slow_down")
	secrets, err := store.Open(t.Context(), store.File, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var waits []time.Duration
	signIn := session.Session{
		Instance: session.Instance{StartURL: fakeaws.StartURL, Region: "eu-west-1"},
		OIDC:     ssooidc.Client{Endpoint: aws.URL, HTTP: http.DefaultClient},
		Secrets:  secrets,
		Sleep: func(_ context.Context, wait time.Duration) error {
			waits = append(waits, wait)
			return nil
		},
	}
	if _, err := signIn.Login(context.Background(), func(session.Approval) {}); err != nil {
		t.Fatal(err)
	}
	// The service asks for one second between attempts, and slow_down adds five.
	if want := []time.Duration{time.Second, time.Second, 6 * time.Second}; !slices.Equal(waits, want) {
		t.Errorf("waited %v, want %v", waits, want)
	}
}
