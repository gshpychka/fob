package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gshpychka/idc/internal/awsconfig"
	"github.com/gshpychka/idc/internal/credentials"
	"github.com/gshpychka/idc/internal/portal"
	"github.com/gshpychka/idc/internal/roles"
	"github.com/gshpychka/idc/internal/session"
)

// app runs the program's commands.
type app struct {
	stdout io.Writer
	stderr io.Writer
	// hyperlinks says that stderr is a terminal, which renders links.
	hyperlinks bool

	session     session.Session
	portal      portal.Client
	credentials credentials.Provider
	// awsConfigPath is the AWS shared config file that sync writes.
	awsConfigPath string
}

var errUsage = errors.New("usage: " + program + " login | sync | creds <account-id> <role-name>")

// run executes the command line that follows the program's name.
func (a app) run(ctx context.Context, arguments []string) error {
	err := errUsage
	switch {
	case len(arguments) == 1 && arguments[0] == "login":
		err = a.login(ctx)
	case len(arguments) == 1 && arguments[0] == "sync":
		err = a.sync(ctx)
	case len(arguments) == 3 && arguments[0] == "creds":
		err = a.creds(ctx, arguments[1], arguments[2])
	}
	if errors.Is(err, session.ErrSignedOut) || errors.Is(err, portal.ErrUnauthorized) {
		return fmt.Errorf("%w; sign in with `%s login`", err, program)
	}
	return err
}

// login signs the user in.
func (a app) login(ctx context.Context) error {
	expiry, err := a.session.Login(ctx, func(approval session.Approval) {
		page := approval.URL
		if a.hyperlinks {
			page = hyperlink("approval page", approval.URL)
		}
		fmt.Fprintf(a.stderr, "Approve the sign-in: %s\nCode: %s\n", page, approval.Code)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stderr, "Signed in until %s.\n", expiry.Format("15:04"))
	return nil
}

// hyperlink is underlined text that a terminal opens as a link on a click.
//
// https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
func hyperlink(text, url string) string {
	return "\x1b]8;;" + url + "\x1b\\\x1b[4m" + text + "\x1b[24m\x1b]8;;\x1b\\"
}

// sync gives every role granted right now a profile in the AWS config, and prints the profiles.
func (a app) sync(ctx context.Context) error {
	accessToken, err := a.session.AccessToken(ctx)
	if err != nil {
		return err
	}
	granted, err := roles.Fetch(ctx, a.portal, accessToken)
	if err != nil {
		return err
	}
	profiles := roles.Profiles(granted)
	if err := awsconfig.Write(a.awsConfigPath, program, profiles); err != nil {
		return err
	}
	for _, profile := range profiles {
		fmt.Fprintln(a.stdout, profile.Name)
	}
	return nil
}

// creds prints a role's credentials in the format of credential_process.
//
// https://docs.aws.amazon.com/sdkref/latest/guide/feature-process-credentials.html
func (a app) creds(ctx context.Context, accountID, roleName string) error {
	issued, err := a.credentials.Get(ctx, accountID, roleName)
	if err != nil {
		return err
	}
	return json.NewEncoder(a.stdout).Encode(map[string]any{
		"Version":         1,
		"AccessKeyId":     issued.AccessKeyID,
		"SecretAccessKey": issued.SecretAccessKey,
		"SessionToken":    issued.SessionToken,
		"Expiration":      issued.Expiration.UTC().Format(time.RFC3339),
	})
}
