# fob

Sign-in to AWS IAM Identity Center, and credentials for the roles it grants.

One sign-in covers every account and role. Each role is an AWS profile named
`<account>/<role>`, and AWS SDKs fetch its credentials through
[`credential_process`](https://docs.aws.amazon.com/sdkref/latest/guide/feature-process-credentials.html).

## Use

```console
$ fob
IAM Identity Center start URL: https://example.awsapps.com/start
IAM Identity Center region [eu-west-1]:
Approve the sign-in: approval page
Code: ABCD-EFGH
Signed in until 18:42.
```

`fob` signs you in when you are signed out, then opens a picker over your roles
and prints the profile you choose.

The first sign-in asks for the instance and remembers it. The approval page is
a link the terminal opens on a click. A sign-in lasts the
[eight hours](https://docs.aws.amazon.com/sdkref/latest/guide/feature-sso-credentials.html)
of an IAM Identity Center access token.

The picker is [fzf](https://github.com/junegunn/fzf). It shows the roles of the
last sync at once and takes in the roles granted right now as soon as AWS has
listed them.

A shell function makes the choice the profile of the shell:

```zsh
fob() {
  if (( $# )); then
    command fob "$@"
  else
    local profile
    profile="$(command fob)" && export AWS_PROFILE="$profile"
  fi
}
```

| Command | Does |
| --- | --- |
| `fob login` | Signs in and syncs. |
| `fob sync` | Gives each role granted right now a profile in the AWS config file and prints the profile names. |
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
| `secure-enclave`, the macOS default | Sealed with [age](https://age-encryption.org) to a key in the Mac's Secure Enclave ([age-plugin-se](https://github.com/remko/age-plugin-se)). The key works on that Mac alone and asks for no password or touch. |
| `systemd-creds`, the Linux default | Sealed by [`systemd-creds --user`](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html) with the machine's credential key, for the owning user. |
| `file` | Plain files that only their owner can read. |

## As a flake input

```nix
{
  inputs.fob = {
    url = "github:gshpychka/fob";
    inputs.nixpkgs.follows = "nixpkgs";
  };
}
```

`fob.packages.<system>.default` is the program.
