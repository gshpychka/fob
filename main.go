// Command idc signs in to AWS IAM Identity Center and serves the credentials of the roles it grants.
package main

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gshpychka/idc/internal/credentials"
	"github.com/gshpychka/idc/internal/portal"
	"github.com/gshpychka/idc/internal/session"
	"github.com/gshpychka/idc/internal/ssooidc"
	"github.com/gshpychka/idc/internal/store"
)

const program = "idc"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	app, err := fromEnvironment()
	if err == nil {
		err = app.run(ctx, os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// variable names one of the program's own environment variables.
func variable(suffix string) string {
	return strings.ToUpper(program) + "_" + suffix
}

// fromEnvironment wires the program to the instance, the store and the files its environment names.
func fromEnvironment() (app, error) {
	instance := session.Instance{
		StartURL: os.Getenv(variable("SSO_START_URL")),
		Region:   os.Getenv(variable("SSO_REGION")),
	}
	if instance.StartURL == "" || instance.Region == "" {
		return app{}, fmt.Errorf("%s and %s name the IAM Identity Center instance and are both required",
			variable("SSO_START_URL"), variable("SSO_REGION"))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return app{}, err
	}
	secrets, err := store.Open(
		cmp.Or(store.Kind(os.Getenv(variable("STORE"))), store.DefaultKind()),
		filepath.Join(cmp.Or(os.Getenv("XDG_STATE_HOME"), filepath.Join(home, ".local", "state")), program),
	)
	if err != nil {
		return app{}, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	signIn := session.Session{
		Instance:   instance,
		OIDC:       ssooidc.Client{Endpoint: ssooidc.Endpoint(instance.Region), HTTP: client},
		Secrets:    secrets,
		ClientName: program,
		Sleep:      session.Sleep,
	}
	portalClient := portal.Client{Endpoint: portal.Endpoint(instance.Region), HTTP: client}
	return app{
		stdout:        os.Stdout,
		stderr:        os.Stderr,
		hyperlinks:    isTerminal(os.Stderr),
		session:       signIn,
		portal:        portalClient,
		credentials:   credentials.Provider{Session: signIn, Portal: portalClient, Secrets: secrets},
		awsConfigPath: cmp.Or(os.Getenv("AWS_CONFIG_FILE"), filepath.Join(home, ".aws", "config")),
	}, nil
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
