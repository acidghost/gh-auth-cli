# gh-auth-cli

A small host-side credential provider that authenticates a user through their own GitHub App and prints a user access token for [`nono`](https://github.com/nolabs-ai/nono) CLI Command Capture (`cmd://`).

This is an MVP for GitHub.com on macOS and Linux. It stores the GitHub App client secret and OAuth tokens in the operating-system keyring; the plaintext config contains identifiers and account metadata only.

## GitHub App setup

Create a GitHub App under **Settings → Developer settings → GitHub Apps**:

1. Set the callback URL to `http://127.0.0.1:17836/callback`.
2. Grant only the repository and organization permissions your sandboxed tools need.
3. Enable expiring user authorization tokens (recommended).
4. Generate a client secret.
5. Install the app on only the repositories it should be able to access.

The CLI needs the app's **Client ID** and **client secret**. It does not need the app private key.

## Build

```bash
go build -o gh-auth-cli .
```

Or with `just`:

```bash
just build
```

## Usage

Configure the app. The client secret is prompted without terminal echo and saved to the OS keyring:

```bash
gh-auth-cli configure --client-id Iv1.example
```

For automation, pass the secret over stdin rather than as a command-line argument:

```bash
printf '%s' "$GITHUB_APP_CLIENT_SECRET" |
  gh-auth-cli configure --client-id Iv1.example --client-secret-stdin
```

Authenticate in the browser using authorization-code flow with PKCE:

```bash
gh-auth-cli login
```

Inspect non-secret status:

```bash
gh-auth-cli status
```

Print the token. This command writes only the token and a newline to stdout; errors go to stderr:

```bash
gh-auth-cli token --non-interactive --min-validity 20m
```

Remove the current user token, or all local app configuration:

```bash
gh-auth-cli logout
gh-auth-cli logout --all
```

Configuration is stored at the platform user config location under `gh-auth-cli/config.json`. Secrets use keyring service `gh-auth-cli`.

## Logging

The CLI writes structured JSON logs to a platform-specific file:

- macOS: `~/Library/Logs/gh-auth-cli.log`
- Linux and other supported Unix systems: `~/.local/share/gh-auth-cli.log`

The default level is `info`. Set `GH_AUTH_CLI_LOG_LEVEL=debug` for detailed OAuth, keyring, token-validity, and refresh diagnostics:

```bash
GH_AUTH_CLI_LOG_LEVEL=debug gh-auth-cli login
```

Other zerolog levels such as `warn` and `error` are also accepted. Tokens, client secrets, authorization codes, OAuth state, and PKCE verifiers are never logged. File logging does not change the `token` command's stdout contract.

## `nono` integration

A minimal custom credential route is available at [`examples/nono-profile.json`](examples/nono-profile.json):

Run `gh-auth-cli login` before starting `nono`. The current MVP intentionally does not initiate an interactive login from `token`.

The 20-minute minimum validity is longer than the example's 15-minute capture cache, preventing `nono` from caching a token past its expiration.

## Security model and MVP limitations

- The client secret and complete token envelope are stored in macOS Keychain or Linux Secret Service through `go-keyring`.
- OAuth state and the PKCE verifier exist only for the duration of login.
- The callback listener binds only to the configured loopback address and waits for a state-matching callback.
- Refresh is serialized with a per-account file lock, then the token is reloaded before deciding whether to refresh.
- The config file is created with mode `0600`; its directory uses `0700`.
- GitHub App permissions, app installation scope, and `nono` endpoint rules are the actual authorization boundary. GitHub App user tokens do not gain narrower permissions from OAuth scopes.
- This MVP supports one GitHub.com app and one active GitHub account. It does not support GitHub Enterprise, multiple profiles, revocation through GitHub, device flow, or policy generation.
- If GitHub rotates a refresh token but keyring persistence fails, run `gh-auth-cli login` again before the returned access token expires.
