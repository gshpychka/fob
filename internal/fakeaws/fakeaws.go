// Package fakeaws stands in for the IAM Identity Center OIDC and portal services in tests.
package fakeaws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	// StartURL is the instance the server signs users in to.
	StartURL = "https://example.awsapps.com/start"
	// UserCode is the code the approval page shows.
	UserCode = "ABCD-EFGH"
	// VerificationURL is the approval page of a sign-in.
	VerificationURL = StartURL + "/#/device?user_code=" + UserCode

	accessToken = "access-token"
)

// Account is an account assigned to the user, with the roles they can take in it.
type Account struct {
	ID    string
	Name  string
	Roles []string
}

// Server serves both services from one address.
type Server struct {
	URL      string
	accounts []Account

	mu sync.Mutex
	// refusals are the OAuth errors the next redemptions meet, in order.
	refusals []string
	// minted counts the credentials issued, which tells one set from the next.
	minted int
}

// New starts a server that assigns accounts to the user.
func New(t *testing.T, accounts ...Account) *Server {
	server := &Server{accounts: accounts}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /client/register", func(w http.ResponseWriter, _ *http.Request) {
		respond(w, map[string]any{"clientId": "client-id", "clientSecret": "client-secret"})
	})
	mux.HandleFunc("POST /device_authorization", func(w http.ResponseWriter, _ *http.Request) {
		respond(w, map[string]any{"deviceCode": "device-code", "userCode": UserCode, "verificationUriComplete": VerificationURL, "interval": 1})
	})
	mux.HandleFunc("POST /token", server.redeem)
	mux.HandleFunc("GET /assignment/accounts", signedIn(server.listAccounts))
	mux.HandleFunc("GET /assignment/roles", signedIn(server.listAccountRoles))
	mux.HandleFunc("GET /federation/credentials", signedIn(server.mint))
	listener := httptest.NewServer(mux)
	t.Cleanup(listener.Close)
	server.URL = listener.URL
	return server
}

// RefuseRedemptions makes the next device code redemptions fail with these OAuth errors, in order.
func (s *Server) RefuseRedemptions(codes ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refusals = codes
}

func (s *Server) redeem(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.refusals) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		respond(w, map[string]any{"error": s.refusals[0], "error_description": "refused"})
		s.refusals = s.refusals[1:]
		return
	}
	respond(w, map[string]any{"accessToken": accessToken, "expiresIn": 28800})
}

// signedIn guards a portal operation with the access token check.
func signedIn(operation http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-amz-sso_bearer_token") != accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		operation(w, r)
	}
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	entries := make([]any, len(s.accounts))
	for index, account := range s.accounts {
		entries[index] = map[string]string{"accountId": account.ID, "accountName": account.Name}
	}
	page(w, r, "accountList", entries)
}

func (s *Server) listAccountRoles(w http.ResponseWriter, r *http.Request) {
	var entries []any
	for _, account := range s.accounts {
		if account.ID == r.URL.Query().Get("account_id") {
			for _, role := range account.Roles {
				entries = append(entries, map[string]string{"roleName": role})
			}
		}
	}
	page(w, r, "roleList", entries)
}

// page serves a list one entry at a time, so every listing spans pages.
func page(w http.ResponseWriter, r *http.Request, member string, entries []any) {
	start, _ := strconv.Atoi(r.URL.Query().Get("next_token"))
	end := min(start+1, len(entries))
	response := map[string]any{member: entries[start:end]}
	if end < len(entries) {
		response["nextToken"] = strconv.Itoa(end)
	}
	respond(w, response)
}

func (s *Server) mint(w http.ResponseWriter, r *http.Request) {
	accountID, roleName := r.URL.Query().Get("account_id"), r.URL.Query().Get("role_name")
	granted := slices.ContainsFunc(s.accounts, func(account Account) bool {
		return account.ID == accountID && slices.Contains(account.Roles, roleName)
	})
	if !granted {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.minted++
	respond(w, map[string]any{"roleCredentials": map[string]any{
		"accessKeyId":     "ASIA" + strconv.Itoa(s.minted),
		"secretAccessKey": "secret-access-key",
		"sessionToken":    "session-token",
		"expiration":      time.Now().Add(time.Hour).UnixMilli(),
	}})
}

func respond(w http.ResponseWriter, response any) {
	json.NewEncoder(w).Encode(response)
}
