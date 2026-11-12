package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gshpychka/idc/internal/credentials"
	"github.com/gshpychka/idc/internal/fakeaws"
	"github.com/gshpychka/idc/internal/portal"
	"github.com/gshpychka/idc/internal/session"
	"github.com/gshpychka/idc/internal/ssooidc"
	"github.com/gshpychka/idc/internal/store"
)

// newApp wires the commands to a stand-in for AWS and to a user who types answers.
func newApp(t *testing.T, aws *fakeaws.Server, answers string) (a app, stdout, stderr *bytes.Buffer) {
	secrets, err := store.Open(store.File, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signIn := session.Session{
		Instance:   session.Instance{StartURL: fakeaws.StartURL, Region: "eu-west-1"},
		OIDC:       ssooidc.Client{Endpoint: aws.URL, HTTP: http.DefaultClient},
		Secrets:    secrets,
		ClientName: program,
		Sleep:      func(context.Context, time.Duration) error { return nil },
	}
	portalClient := portal.Client{Endpoint: aws.URL, HTTP: http.DefaultClient}
	stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	return app{
		stdout:        stdout,
		stderr:        stderr,
		user:          prompter{in: bufio.NewReader(strings.NewReader(answers)), out: io.Discard},
		session:       signIn,
		portal:        portalClient,
		credentials:   credentials.Provider{Session: signIn, Portal: portalClient, Secrets: secrets},
		awsConfigPath: filepath.Join(t.TempDir(), "config"),
	}, stdout, stderr
}

func TestSignInSyncAndCredentials(t *testing.T) {
	aws := fakeaws.New(t,
		fakeaws.Account{ID: "111111111111", Name: "Data Platform", Roles: []string{"ReadOnly", "Admin"}},
		fakeaws.Account{ID: "222222222222", Name: "billing", Roles: []string{"Admin"}},
	)
	a, stdout, stderr := newApp(t, aws, "")
	// run returns what a command prints to standard output.
	run := func(arguments ...string) string {
		t.Helper()
		stdout.Reset()
		if err := a.run(context.Background(), arguments); err != nil {
			t.Fatalf("%v: %v", arguments, err)
		}
		return stdout.String()
	}

	if err := a.run(context.Background(), []string{"sync"}); !errors.Is(err, session.ErrSignedOut) {
		t.Fatalf("sync before login: %v", err)
	}

	run("login")
	if shown := stderr.String(); !strings.Contains(shown, fakeaws.VerificationURL) || !strings.Contains(shown, fakeaws.UserCode) {
		t.Errorf("login showed %q", shown)
	}

	if profiles := run("sync"); profiles != "Data-Platform/Admin\nData-Platform/ReadOnly\nbilling/Admin\n" {
		t.Errorf("sync printed %q", profiles)
	}
	config, err := os.ReadFile(a.awsConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if profile := "[profile Data-Platform/Admin]\ncredential_process = idc creds 111111111111 Admin\n"; !strings.Contains(string(config), profile) {
		t.Errorf("config lacks %q:\n%s", profile, config)
	}

	first := run("creds", "111111111111", "Admin")
	if !strings.Contains(first, `"AccessKeyId":"ASIA1"`) {
		t.Errorf("creds printed %q", first)
	}
	if second := run("creds", "111111111111", "Admin"); second != first {
		t.Errorf("stored credentials were replaced: %q, then %q", first, second)
	}
}

func TestSyncReplacesAForeignConfigOnlyWithConsent(t *testing.T) {
	const foreign = "[default]\nregion = eu-west-1\n"
	for answer, replaced := range map[string]bool{"n\n": false, "y\n": true} {
		a, _, _ := newApp(t, fakeaws.New(t), answer)
		if err := a.run(context.Background(), []string{"login"}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(a.awsConfigPath, []byte(foreign), 0o600); err != nil {
			t.Fatal(err)
		}
		err := a.run(context.Background(), []string{"sync"})
		config, _ := os.ReadFile(a.awsConfigPath)
		if (err == nil) != replaced || (string(config) != foreign) != replaced {
			t.Errorf("answer %q: err = %v, config = %q", answer, err, config)
		}
	}
}
