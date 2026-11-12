package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gshpychka/fob/internal/awsconfig"
	"github.com/gshpychka/fob/internal/credentials"
	"github.com/gshpychka/fob/internal/portal"
	"github.com/gshpychka/fob/internal/roles"
	"github.com/gshpychka/fob/internal/session"
)

// app runs the program's commands.
type app struct {
	stdout io.Writer
	stderr io.Writer
	// terminal says that stderr is a terminal, which renders links and a status line.
	terminal bool
	user     prompter
	// choose lets the user pick one of lines, which the output of a refresh command replaces.
	choose func(ctx context.Context, lines []string, refresh string) (string, error)

	session     session.Session
	portal      portal.Client
	credentials credentials.Provider
	// awsConfigPath is the AWS shared config file that holds the profiles.
	awsConfigPath string
	// remember records the instance in the settings file.
	remember func() error
}

var errUsage = errors.New("usage: " + program + " [login | sync | creds <account-id> <role-name>]")

// run executes the command line that follows the program's name.
func (a app) run(ctx context.Context, arguments []string) error {
	err := errUsage
	switch {
	case len(arguments) == 0:
		err = a.pick(ctx)
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

// pick prints the profile the user chooses, after a sign-in when they are signed out.
func (a app) pick(ctx context.Context) error {
	_, err := a.session.AccessToken(ctx)
	if errors.Is(err, session.ErrSignedOut) {
		err = a.signIn(ctx)
	}
	if err != nil {
		return err
	}
	// The picker opens on the profiles of the last sync and runs the next one meanwhile.
	names, err := awsconfig.Read(a.awsConfigPath, program)
	refresh := program + " sync"
	if len(names) == 0 {
		// A config without profiles gets its sync before the picker opens.
		names, err = a.refresh(ctx)
		refresh = ""
	}
	if err != nil {
		return err
	}
	profile, err := a.choose(ctx, names, refresh)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, profile)
	return nil
}

// login signs the user in and refreshes the profiles.
func (a app) login(ctx context.Context) error {
	if err := a.signIn(ctx); err != nil {
		return err
	}
	_, err := a.refresh(ctx)
	return err
}

// signIn takes the user through a sign-in and remembers its instance.
func (a app) signIn(ctx context.Context) error {
	expiry, err := a.session.Login(ctx, func(approval session.Approval) {
		page := approval.URL
		if a.terminal {
			page = hyperlink("approval page", approval.URL)
		}
		fmt.Fprintf(a.stderr, "Approve the sign-in: %s\nCode: %s\n", page, approval.Code)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stderr, "Signed in until %s.\n", expiry.Format("15:04"))
	return a.remember()
}

// hyperlink is underlined text that a terminal opens as a link on a click.
//
// https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
func hyperlink(text, url string) string {
	return "\x1b]8;;" + url + "\x1b\\\x1b[4m" + text + "\x1b[24m\x1b]8;;\x1b\\"
}

// sync refreshes the profiles and prints their names.
func (a app) sync(ctx context.Context) error {
	names, err := a.refresh(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Fprintln(a.stdout, name)
	}
	return nil
}

// refresh gives every role granted right now a profile in the AWS config and returns the profiles' names.
func (a app) refresh(ctx context.Context) ([]string, error) {
	_, err := awsconfig.Read(a.awsConfigPath, program)
	if errors.Is(err, awsconfig.ErrForeign) {
		if !a.user.confirm(fmt.Sprintf("%s holds a config %s did not write. Replace it?", a.awsConfigPath, program)) {
			return nil, fmt.Errorf("%s stays as it is; AWS_CONFIG_FILE names the file %s writes", a.awsConfigPath, program)
		}
	} else if err != nil {
		return nil, err
	}
	accessToken, err := a.session.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	a.status("Syncing roles")
	granted, err := roles.Fetch(ctx, a.portal, accessToken, func(listed, total int) {
		a.status(fmt.Sprintf("Syncing roles: %d of %d accounts", listed, total))
	})
	a.status("")
	if err != nil {
		return nil, err
	}
	profiles := roles.Profiles(granted)
	if err := awsconfig.Write(a.awsConfigPath, program, profiles); err != nil {
		return nil, err
	}
	names := make([]string, len(profiles))
	for i, profile := range profiles {
		names[i] = profile.Name
	}
	return names, nil
}

// status puts text on the terminal's status line, in place of the text that was there.
func (a app) status(text string) {
	if a.terminal {
		fmt.Fprint(a.stderr, "\r\x1b[K"+text)
	}
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
