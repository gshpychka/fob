# idc

Sign-in to AWS IAM Identity Center, and credentials for the roles it grants.

One sign-in covers every account and role. Each role is an AWS profile named
`<account>/<role>`, and AWS SDKs fetch its credentials through
[`credential_process`](https://docs.aws.amazon.com/sdkref/latest/guide/feature-process-credentials.html).

## Commands

| Command | Effect |
| --- | --- |
| `idc login` | Signs in. The approval page prints as a short link the terminal opens on a click. |
| `idc sync` | Lists the roles granted right now, gives each a profile in the AWS config file and prints the profile names. |
| `idc creds <account-id> <role-name>` | Prints a role's credentials for `credential_process`. |

A sign-in lasts the
[eight hours](https://docs.aws.amazon.com/sdkref/latest/guide/feature-sso-credentials.html)
of an IAM Identity Center access token. Role credentials stay in the store until
15 minutes before they expire.

## Configuration

| Variable | Meaning |
| --- | --- |
| `IDC_SSO_START_URL` | Start URL of the IAM Identity Center instance. |
| `IDC_SSO_REGION` | Region of the instance. |
| `IDC_STORE` | Where secrets rest: `secure-enclave` (macOS default), `systemd-creds` (Linux default) or `file`. |
| `AWS_CONFIG_FILE` | The AWS config file `sync` owns. Defaults to `~/.aws/config`. |
| `XDG_STATE_HOME` | Secrets live in its `idc` directory. Defaults to `~/.local/state`. |

## Stores

| Store | Secrets at rest |
| --- | --- |
| `secure-enclave` | Sealed with [age](https://age-encryption.org) to a key in the Mac's Secure Enclave ([age-plugin-se](https://github.com/remko/age-plugin-se)). The key works on that Mac alone and asks for no password or touch. |
| `systemd-creds` | Sealed by [`systemd-creds --user`](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html) with the machine's credential key, for the owning user. |
| `file` | Plain files that only their owner can read. |

## In a dev shell

```nix
{
  inputs.idc = {
    url = "github:gshpychka/idc";
    inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs =
    { nixpkgs, idc, ... }:
    {
      devShells.x86_64-linux.default = nixpkgs.legacyPackages.x86_64-linux.mkShell {
        packages = [ idc.packages.x86_64-linux.default ];
        IDC_SSO_START_URL = "https://example.awsapps.com/start";
        IDC_SSO_REGION = "eu-west-1";
      };
    };
}
```

## Switching roles

A shell function picks a profile from the roles granted at that moment:

```zsh
awsp() {
  local profiles profile
  profiles="$(idc sync)" || { idc login && profiles="$(idc sync)" } || return
  profile="$(fzf <<< "$profiles")" && export AWS_PROFILE="$profile"
}
```
