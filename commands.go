package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
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
	// unhooked says that stdout is a terminal, where the shell function is absent.
	unhooked bool
	// terminal says that stderr is a terminal, which renders links and a status line.
	terminal bool
	// pen styles the text on stderr.
	pen  pen
	user prompter
	// choose lets the user pick one of the lines that arrive while the picker is open.
	choose func(ctx context.Context, lines io.Reader) (string, error)
	// syncInBackground starts a sync that outlives the program, and returns the profile names it prints.
	syncInBackground func() (io.Reader, error)

	session     session.Session
	portal      portal.Client
	credentials credentials.Provider
	// awsConfigPath is the AWS shared config file that holds the profiles.
	awsConfigPath string
	// remember records the instance in the settings file.
	remember func() error
}

var errUsage = errors.New("usage: " + program + " [login | sync | init | creds <account-id> <role-name>]")

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
	case len(arguments) == 1 && arguments[0] == "init":
		_, err = fmt.Fprint(a.stdout, shellFunction)
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
	// The picker opens on the profiles of the last sync, and a sync in the background adds its own to them.
	names, err := awsconfig.Read(a.awsConfigPath, program)
	fresh := io.Reader(strings.NewReader(""))
	if len(names) == 0 {
		// A config without profiles gets its sync before the picker opens.
		names, err = a.refresh(ctx)
	} else {
		fresh, err = a.syncInBackground()
	}
	if err != nil {
		return err
	}
	last := strings.NewReader(strings.Join(names, "\n") + "\n")
	profile, err := a.choose(ctx, io.MultiReader(last, fresh))
	if err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, profile)
	if a.unhooked {
		fmt.Fprintln(a.stderr, a.pen.paint(faint, "To set AWS_PROFILE in the shell, add to its startup file:"), shellSetup)
	} else {
		a.done(a.pen.profile(profile))
	}
	return nil
}

// done tells the user that a step has succeeded.
func (a app) done(text string) {
	fmt.Fprintln(a.stderr, a.pen.paint(green, "✓"), text)
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
			page = hyperlink(a.pen.paint(ember, "approval page"), approval.URL)
		}
		fmt.Fprintf(a.stderr, "%s Approve the sign-in: %s\n  Code: %s\n", a.pen.paint(cyan, "▸"), page, a.pen.paint(bold, approval.Code))
		a.status(a.pen.paint(faint, "Waiting for the approval…"))
	})
	a.status("")
	if err != nil {
		return err
	}
	a.done("Signed in until " + expiry.Format("15:04"))
	return a.remember()
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
		a.status(fmt.Sprintf("Syncing roles %s %d/%d", a.pen.bar(listed, total), listed, total))
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
	a.done(fmt.Sprintf("Synced %d roles", len(names)))
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
