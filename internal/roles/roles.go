// Package roles is the inventory of roles the signed-in user can take.
package roles

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/gshpychka/fob/internal/portal"
)

// Role is a role the user can take in an AWS account.
type Role struct {
	AccountID   string
	AccountName string
	Name        string
}

// Requests in flight to the portal at once.
const concurrency = 8

// Fetch lists the roles of every account assigned to the user.
func Fetch(ctx context.Context, client portal.Client, accessToken string) ([]Role, error) {
	accounts, err := client.ListAccounts(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	// The first failed listing cancels the others.
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	names := make([][]string, len(accounts))
	slots := make(chan struct{}, concurrency)
	var listings sync.WaitGroup
	for index, account := range accounts {
		slots <- struct{}{}
		listings.Go(func() {
			defer func() { <-slots }()
			roleNames, err := client.ListAccountRoles(ctx, accessToken, account.ID)
			if err != nil {
				cancel(err)
			}
			names[index] = roleNames
		})
	}
	listings.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	var roles []Role
	for index, account := range accounts {
		for _, name := range names[index] {
			roles = append(roles, Role{AccountID: account.ID, AccountName: account.Name, Name: name})
		}
	}
	return roles, nil
}

// Profile is a role under the name that selects it.
type Profile struct {
	Name string
	Role Role
}

// Profiles names each role "<account>/<role>" and orders the result by name.
func Profiles(roles []Role) []Profile {
	// Account names are free text, so several accounts can share a label; their IDs tell those apart.
	accountsByLabel := map[string][]string{}
	for _, role := range roles {
		label := accountLabel(role.AccountName)
		if !slices.Contains(accountsByLabel[label], role.AccountID) {
			accountsByLabel[label] = append(accountsByLabel[label], role.AccountID)
		}
	}
	profiles := make([]Profile, len(roles))
	for index, role := range roles {
		label := accountLabel(role.AccountName)
		if len(accountsByLabel[label]) > 1 {
			label += "-" + role.AccountID
		}
		profiles[index] = Profile{Name: label + "/" + role.Name, Role: role}
	}
	slices.SortFunc(profiles, func(a, b Profile) int { return strings.Compare(a.Name, b.Name) })
	return profiles
}

// accountLabel is an account's name as one word.
func accountLabel(name string) string {
	return strings.Join(strings.Fields(name), "-")
}
