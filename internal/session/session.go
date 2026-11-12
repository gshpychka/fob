// Package session owns the sign-in to an IAM Identity Center instance.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gshpychka/idc/internal/ssooidc"
	"github.com/gshpychka/idc/internal/store"
)

// ErrSignedOut reports that the user has to sign in.
var ErrSignedOut = errors.New("signed out")

// Instance identifies an IAM Identity Center instance.
type Instance struct {
	StartURL string
	Region   string
}

// Session is the sign-in of one user to one instance, kept in a store.
type Session struct {
	Instance Instance
	OIDC     ssooidc.Client
	Secrets  store.Store
	// ClientName is how the approval page names the requester.
	ClientName string
	// Sleep waits between redemption attempts.
	Sleep func(context.Context, time.Duration) error
}

// record is the stored sign-in.
type record struct {
	Instance    Instance
	AccessToken string
	ExpiresAt   time.Time
}

const secretName = "session"

// Approval is where the user approves a sign-in.
type Approval struct {
	// URL opens the approval page with the code filled in.
	URL  string
	Code string
}

// Login signs the user in through the device authorization grant and returns when the sign-in ends.
// present shows the user where to approve it.
func (s Session) Login(ctx context.Context, present func(Approval)) (time.Time, error) {
	registration, err := s.OIDC.RegisterClient(ctx, s.ClientName)
	if err != nil {
		return time.Time{}, err
	}
	authorization, err := s.OIDC.StartDeviceAuthorization(ctx, registration, s.Instance.StartURL)
	if err != nil {
		return time.Time{}, err
	}
	present(Approval{URL: authorization.VerificationURL, Code: authorization.UserCode})
	token, err := s.awaitApproval(ctx, registration, authorization)
	if err != nil {
		return time.Time{}, err
	}
	signIn := record{
		Instance:    s.Instance,
		AccessToken: token.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
	}
	secret, err := json.Marshal(signIn)
	if err != nil {
		return time.Time{}, err
	}
	return signIn.ExpiresAt, s.Secrets.Set(ctx, secretName, secret)
}

// RFC 8628 sets the redemption interval of a response that names none, and the step a slow_down adds to it.
const (
	defaultInterval = 5 * time.Second
	slowDownStep    = 5 * time.Second
)

// awaitApproval redeems the device code until the user has approved the sign-in.
func (s Session) awaitApproval(ctx context.Context, registration ssooidc.Registration, authorization ssooidc.DeviceAuthorization) (ssooidc.Token, error) {
	interval := time.Duration(authorization.Interval) * time.Second
	if interval == 0 {
		interval = defaultInterval
	}
	for {
		if err := s.Sleep(ctx, interval); err != nil {
			return ssooidc.Token{}, err
		}
		token, err := s.OIDC.RedeemDeviceCode(ctx, registration, authorization.DeviceCode)
		var refusal *ssooidc.Error
		switch {
		case err == nil:
			return token, nil
		case !errors.As(err, &refusal):
			return ssooidc.Token{}, err
		case refusal.Code == ssooidc.SlowDown:
			interval += slowDownStep
		case refusal.Code != ssooidc.AuthorizationPending:
			return ssooidc.Token{}, err
		}
	}
}

// AccessToken returns the token of a sign-in to this instance that is in force, or ErrSignedOut.
func (s Session) AccessToken(ctx context.Context) (string, error) {
	secret, err := s.Secrets.Get(ctx, secretName)
	if errors.Is(err, store.ErrNotFound) {
		return "", ErrSignedOut
	}
	if err != nil {
		return "", err
	}
	var signIn record
	if err := json.Unmarshal(secret, &signIn); err != nil {
		return "", err
	}
	if signIn.Instance != s.Instance || !time.Now().Before(signIn.ExpiresAt) {
		return "", ErrSignedOut
	}
	return signIn.AccessToken, nil
}

// Sleep waits for a duration or the context's end.
func Sleep(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}
