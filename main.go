// Command fob signs in to AWS IAM Identity Center and serves the credentials of the roles it grants.
package main

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/gshpychka/fob/internal/config"
	"github.com/gshpychka/fob/internal/credentials"
	"github.com/gshpychka/fob/internal/portal"
	"github.com/gshpychka/fob/internal/session"
	"github.com/gshpychka/fob/internal/ssooidc"
	"github.com/gshpychka/fob/internal/store"
)

const program = "fob"

// settingsVariable is the environment variable that names the settings file.
var settingsVariable = strings.ToUpper(program) + "_CONFIG_FILE"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// directory is the program's own directory under an XDG base directory.
	directory := func(variable, fallback string) string {
		return filepath.Join(cmp.Or(os.Getenv(variable), filepath.Join(home, fallback)), program)
	}
	settingsPath := cmp.Or(os.Getenv(settingsVariable), filepath.Join(directory("XDG_CONFIG_HOME", ".config"), "config.json"))
	settings, err := config.Load(settingsPath)
	if err != nil {
		return err
	}
	user := prompter{in: bufio.NewReader(os.Stdin), out: os.Stderr}
	// The commands that sign the user in ask for the instance.
	if len(arguments) == 0 || slices.Equal(arguments, []string{"login"}) {
		if settings.Instance, err = identify(settings.Instance, user); err != nil {
			return err
		}
	}
	secrets, err := store.Open(cmp.Or(settings.Store, store.DefaultKind()), directory("XDG_STATE_HOME", ".local/state"))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	signIn := session.Session{
		Instance:   settings.Instance,
		OIDC:       ssooidc.Client{Endpoint: ssooidc.Endpoint(settings.Region), HTTP: client},
		Secrets:    secrets,
		ClientName: program,
		Sleep:      session.Sleep,
	}
	portalClient := portal.Client{Endpoint: portal.Endpoint(settings.Region), HTTP: client}
	commands := app{
		stdout:           os.Stdout,
		stderr:           os.Stderr,
		unhooked:         isTerminal(os.Stdout),
		terminal:         isTerminal(os.Stderr),
		user:             user,
		choose:           choose,
		syncInBackground: syncInBackground,
		session:          signIn,
		portal:           portalClient,
		credentials:      credentials.Provider{Session: signIn, Portal: portalClient, Secrets: secrets},
		awsConfigPath:    cmp.Or(os.Getenv("AWS_CONFIG_FILE"), filepath.Join(home, ".aws", "config")),
		remember:         func() error { return config.Save(settingsPath, settings) },
	}
	return commands.run(ctx, arguments)
}

// identify asks the user for the parts of the instance that the settings lack.
func identify(instance session.Instance, user prompter) (session.Instance, error) {
	var err error
	if instance.StartURL == "" {
		if instance.StartURL, err = user.ask("IAM Identity Center start URL", ""); err != nil {
			return instance, err
		}
	}
	if instance.Region == "" {
		suggestion := cmp.Or(os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"))
		instance.Region, err = user.ask("IAM Identity Center region", suggestion)
	}
	return instance, err
}

// syncInBackground starts a sync that outlives the program, and returns the profile names it prints.
func syncInBackground() (io.Reader, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	sync := exec.Command(executable, "sync")
	// A session of its own keeps the terminal's interrupt and hangup away from the sync.
	sync.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	names, err := sync.StdoutPipe()
	if err != nil {
		return nil, err
	}
	return names, sync.Start()
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
