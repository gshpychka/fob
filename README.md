# fob

Sign-in to AWS IAM Identity Center, and credentials for the roles it grants.

One sign-in covers every account and role. Each role is an AWS profile named
`<account>/<role>`, and AWS SDKs fetch its credentials through
[`credential_process`](https://docs.aws.amazon.com/sdkref/latest/guide/feature-process-credentials.html).

## Install

With Nix, `packages.<system>.default` of the flake is the program, with the
tools it runs:

```nix
{
  inputs.fob = {
    url = "github:gshpychka/fob";
    inputs.nixpkgs.follows = "nixpkgs";
  };
}
```

With Go:

```console
$ go install github.com/gshpychka/fob@latest
```

A build by Go runs [fzf](https://github.com/junegunn/fzf) 0.61 or later from
the `PATH`, and on macOS [age](https://age-encryption.org) and
[age-plugin-se](https://github.com/remko/age-plugin-se) as well.

A line in `~/.zshrc` or `~/.bashrc` lets `fob` set the shell's AWS profile:

```sh
eval "$(fob init)"
```

Where `fob` joins the `PATH` after the shell has started, as in a direnv dev
shell, the startup file holds the output of `fob init` itself.

## Use

```console
$ fob
▸ IAM Identity Center start URL: https://example.awsapps.com/start
▸ IAM Identity Center region [eu-west-1]:
▸ Approve the sign-in: approval page
  Code: ABCD-EFGH
✓ Signed in until 18:42
✓ Synced 42 roles
✓ data-platform/Admin
```

`fob` signs you in when you are signed out, then opens a picker over your roles
and makes the profile you choose the shell's `AWS_PROFILE`.

The first sign-in asks for the instance and remembers it. The approval page is
a link the terminal opens on a click. A sign-in lasts the
[eight hours](https://docs.aws.amazon.com/sdkref/latest/guide/feature-sso-credentials.html)
of an IAM Identity Center access token.

The picker is fzf. It opens at once on the roles of the last sync. A sync in the
background adds the roles granted since, and runs to its end after you pick.

[`NO_COLOR`](https://no-color.org/) turns the colours off.

| Command | Does |
| --- | --- |
| `fob login` | Signs in and syncs. |
| `fob sync` | Gives each role granted right now a profile in the AWS config file and prints the profile names. |
| `fob init` | Prints the shell function that exports the profile picked in `fob`. |
| `fob creds <account-id> <role-name>` | Prints a role's credentials for `credential_process`. The profiles call it, and it keeps the credentials in the store until 15 minutes before they expire. |

## Files

| File | Holds |
| --- | --- |
| `$FOB_CONFIG_FILE`, or `$XDG_CONFIG_HOME/fob/config.json` | The settings: `startUrl`, `region` and, to override the platform's default, `store`. |
| `$XDG_STATE_HOME/fob/` | The secrets, in the store. |
| `$AWS_CONFIG_FILE`, or `~/.aws/config` | The profiles. `fob` owns this file and asks before it replaces a config it did not write. |

## Stores

| `store` | Secrets at rest |
| --- | --- |
| `secure-enclave`, the macOS default | Sealed with [age](https://age-encryption.org), each secret to a key of its own in the Mac's Secure Enclave ([age-plugin-se](https://github.com/remko/age-plugin-se)). A key works on that Mac alone and takes Touch ID for each use. A change of the enrolled fingerprints signs you out. |
| `systemd-creds`, the Linux default | Sealed by [`systemd-creds --user`](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html) with the machine's credential key, for the owning user. |
| `file` | Plain files that only their owner can read. |

