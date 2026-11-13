// Package portal is a client for the IAM Identity Center portal service.
//
// https://docs.aws.amazon.com/singlesignon/latest/PortalAPIReference/Welcome.html
package portal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"
)

// Endpoint is the service's address in a region.
func Endpoint(region string) string {
	return "https://portal.sso." + region + ".amazonaws.com"
}

// ErrUnauthorized reports an access token the service rejects.
var ErrUnauthorized = errors.New("access token rejected")

// Client calls the service at one endpoint.
type Client struct {
	Endpoint string
	HTTP     *http.Client
}

// Account is an AWS account assigned to the user.
type Account struct {
	ID   string `json:"accountId"`
	Name string `json:"accountName"`
}

func (c Client) ListAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	all, err := c.list(ctx, accessToken, "/assignment/accounts", url.Values{})
	return all.Accounts, err
}

// ListAccountRoles names the roles the user can take in an account.
func (c Client) ListAccountRoles(ctx context.Context, accessToken, accountID string) ([]string, error) {
	all, err := c.list(ctx, accessToken, "/assignment/roles", url.Values{"account_id": {accountID}})
	names := make([]string, len(all.Roles))
	for index, role := range all.Roles {
		names[index] = role.Name
	}
	return names, err
}

// listing is a page of either list operation, which fills the member it is about.
type listing struct {
	Accounts []Account `json:"accountList"`
	Roles    []struct {
		Name string `json:"roleName"`
	} `json:"roleList"`
	NextToken string `json:"nextToken"`
}

// list gathers every page of a list operation.
func (c Client) list(ctx context.Context, accessToken, path string, query url.Values) (listing, error) {
	var all listing
	// The largest page the service serves.
	query.Set("max_result", "100")
	for {
		var page listing
		if err := c.get(ctx, accessToken, path, query, &page); err != nil {
			return listing{}, err
		}
		all.Accounts = append(all.Accounts, page.Accounts...)
		all.Roles = append(all.Roles, page.Roles...)
		if page.NextToken == "" {
			return all, nil
		}
		query.Set("next_token", page.NextToken)
	}
}

// RoleCredentials are temporary AWS credentials for a role.
type RoleCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

func (c Client) GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName string) (RoleCredentials, error) {
	var response struct {
		RoleCredentials struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			SessionToken    string `json:"sessionToken"`
			// Milliseconds since the Unix epoch.
			Expiration int64 `json:"expiration"`
		} `json:"roleCredentials"`
	}
	query := url.Values{"account_id": {accountID}, "role_name": {roleName}}
	if err := c.get(ctx, accessToken, "/federation/credentials", query, &response); err != nil {
		return RoleCredentials{}, err
	}
	issued := response.RoleCredentials
	return RoleCredentials{
		AccessKeyID:     issued.AccessKeyID,
		SecretAccessKey: issued.SecretAccessKey,
		SessionToken:    issued.SessionToken,
		Expiration:      time.UnixMilli(issued.Expiration),
	}, nil
}

// errThrottled reports a request the service turns away for the rate of requests.
var errThrottled = errors.New("IAM Identity Center is throttling requests")

// A throttled response comes without a time to retry at, so the pauses before the retries double from the first to the last.
const (
	firstPause = 200 * time.Millisecond
	lastPause  = 3200 * time.Millisecond
)

// get performs a request, and repeats one that the service throttles.
func (c Client) get(ctx context.Context, accessToken, path string, query url.Values, response any) error {
	for pause := firstPause; ; pause *= 2 {
		err := c.attempt(ctx, accessToken, path, query, response)
		if !errors.Is(err, errThrottled) || pause > lastPause {
			return err
		}
		// A random share of the pause spreads out requests that were throttled together.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pause/2 + rand.N(pause/2)):
		}
	}
}

func (c Client) attempt(ctx context.Context, accessToken, path string, query url.Values, response any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("x-amz-sso_bearer_token", accessToken)
	httpResponse, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	switch httpResponse.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(httpResponse.Body).Decode(response)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return errThrottled
	default:
		return fmt.Errorf("GET %s: %s", path, httpResponse.Status)
	}
}
