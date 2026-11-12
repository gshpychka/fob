# idc

Sign-in to AWS IAM Identity Center, and credentials for the roles it grants.

One sign-in covers every account and role. Each role is an AWS profile named
`<account>/<role>`, and AWS SDKs fetch its credentials through
[`credential_process`](https://docs.aws.amazon.com/sdkref/latest/guide/feature-process-credentials.html).

## Use

```console
$ idc login
IAM Identity Center start URL: https://example.awsapps.com/start
IAM Identity Center region [eu-west-1]:
Approve the sign-in: approval page
Code: ABCD-EFGH
Signed in until 18:42.
```

The first sign-in asks for the instance and remembers it. The approval page is
a link the terminal opens on a click. A sign-in lasts the
[eight hours](https://docs.aws.amazon.com/sdkref/latest/guide/feature-sso-credentials.html)
of an IAM Identity Center access token.

```console
$ idc sync
billing/Admin
data-platform/ReadOnly
$ AWS_PROFILE=billing/Admin aws sts get-caller-identity
```

`idc sync` lists the roles granted right now, gives each a profile in the AWS
config file and prints the profile names. The profiles call
`idc creds <account-id> <role-name>`, which keeps a role's credentials in the
store until 15 minutes before they expire.

A shell function switches roles:

```zsh
awsp() {
  local profiles profile
  profiles="$(idc sync)" || { idc login && profiles="$(idc sync)" } || return
  profile="$(fzf <<< "$profiles")" && export AWS_PROFILE="$profile"
}
```

## Files

| File | Holds |
| --- | --- |
| `$IDC_CONFIG_FILE`, or `$XDG_CONFIG_HOME/idc/config.json` | The settings: `startUrl`, `region` and, to override the platform's default, `store`. |
| `$XDG_STATE_HOME/idc/` | The secrets, in the store. |
| `$AWS_CONFIG_FILE`, or `~/.aws/config` | The profiles. `sync` owns this file and asks before it replaces a config it did not write. |

## Stores

| `store` | Secrets at rest |
| --- | --- |
| `secure-enclave`, the macOS default | Sealed with [age](https://age-encryption.org) to a key in the Mac's Secure Enclave ([age-plugin-se](https://github.com/remko/age-plugin-se)). The key works on that Mac alone and asks for no password or touch. |
| `systemd-creds`, the Linux default | Sealed by [`systemd-creds --user`](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html) with the machine's credential key, for the owning user. |
| `file` | Plain files that only their owner can read. |

## As a flake input

```nix
{
  inputs.idc = {
    url = "github:gshpychka/idc";
    inputs.nixpkgs.follows = "nixpkgs";
  };
}
```

`idc.packages.<system>.default` is the program.
