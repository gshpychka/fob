// Package credentials serves the temporary AWS credentials of a role.
package credentials

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gshpychka/fob/internal/portal"
	"github.com/gshpychka/fob/internal/session"
	"github.com/gshpychka/fob/internal/store"
)

// Provider hands out role credentials, keeping each set in the store for as long as it serves.
type Provider struct {
	Session session.Session
	Portal  portal.Client
	Secrets store.Store
}

// Credentials served have at least this long to live: botocore asks again on every call for any that expire sooner.
const minimumLifetime = 15 * time.Minute

// Get returns credentials for a role.
func (p Provider) Get(ctx context.Context, accountID, roleName string) (portal.RoleCredentials, error) {
	name := "role." + accountID + "." + roleName
	var issued portal.RoleCredentials
	// A stored set that fails to read is replaced like one that has expired.
	if secret, err := p.Secrets.Get(ctx, name); err == nil && json.Unmarshal(secret, &issued) == nil {
		if time.Until(issued.Expiration) > minimumLifetime {
			return issued, nil
		}
	}
	accessToken, err := p.Session.AccessToken(ctx)
	if err != nil {
		return portal.RoleCredentials{}, err
	}
	if issued, err = p.Portal.GetRoleCredentials(ctx, accessToken, accountID, roleName); err != nil {
		return portal.RoleCredentials{}, err
	}
	secret, err := json.Marshal(issued)
	if err != nil {
		return portal.RoleCredentials{}, err
	}
	return issued, p.Secrets.Set(ctx, name, secret)
}
