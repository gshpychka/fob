// Package ssooidc is a client for the IAM Identity Center OIDC service.
//
// https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/Welcome.html
package ssooidc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Endpoint is the service's address in a region.
func Endpoint(region string) string {
	return "https://oidc." + region + ".amazonaws.com"
}

// Client calls the service at one endpoint.
type Client struct {
	Endpoint string
	HTTP     *http.Client
}

// Registration is an OAuth client registered with the service.
type Registration struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (c Client) RegisterClient(ctx context.Context, name string) (Registration, error) {
	var registration Registration
	request := map[string]string{"clientName": name, "clientType": "public"}
	return registration, c.post(ctx, "/client/register", request, &registration)
}

// DeviceAuthorization is a pending sign-in that the user approves in a browser.
type DeviceAuthorization struct {
	DeviceCode string `json:"deviceCode"`
	UserCode   string `json:"userCode"`
	// VerificationURL opens the approval page with the user code filled in.
	VerificationURL string `json:"verificationUriComplete"`
	// Interval is the minimum number of seconds between redemption attempts.
	Interval int `json:"interval"`
}

func (c Client) StartDeviceAuthorization(ctx context.Context, registration Registration, startURL string) (DeviceAuthorization, error) {
	var authorization DeviceAuthorization
	request := map[string]string{
		"clientId":     registration.ClientID,
		"clientSecret": registration.ClientSecret,
		"startUrl":     startURL,
	}
	return authorization, c.post(ctx, "/device_authorization", request, &authorization)
}

// Token is an access token for the portal.
type Token struct {
	AccessToken string `json:"accessToken"`
	// ExpiresIn is the token's lifetime in seconds.
	ExpiresIn int `json:"expiresIn"`
}

// RedeemDeviceCode exchanges an approved device authorization for a token.
func (c Client) RedeemDeviceCode(ctx context.Context, registration Registration, deviceCode string) (Token, error) {
	var token Token
	request := map[string]string{
		"clientId":     registration.ClientID,
		"clientSecret": registration.ClientSecret,
		"grantType":    "urn:ietf:params:oauth:grant-type:device_code",
		"deviceCode":   deviceCode,
	}
	return token, c.post(ctx, "/token", request, &token)
}

// ErrorCode is an OAuth 2.0 error code.
type ErrorCode string

const (
	AuthorizationPending ErrorCode = "authorization_pending"
	SlowDown             ErrorCode = "slow_down"
)

// Error is a request the service refused.
type Error struct {
	Code        ErrorCode `json:"error"`
	Description string    `json:"error_description"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Description)
}

func (c Client) post(ctx context.Context, path string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpResponse, err := c.HTTP.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	payload, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}
	if httpResponse.StatusCode == http.StatusOK {
		return json.Unmarshal(payload, response)
	}
	// A refusal carries an OAuth error body.
	var refusal Error
	if json.Unmarshal(payload, &refusal) == nil && refusal.Code != "" {
		return &refusal
	}
	return fmt.Errorf("POST %s: %s", path, httpResponse.Status)
}
