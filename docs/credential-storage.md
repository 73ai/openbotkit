# Credential Storage

## How it works

API keys are stored in the OS-native keyring via [`zalando/go-keyring`](https://github.com/zalando/go-keyring):

| Platform | Backend |
|----------|---------|
| macOS | Keychain |
| Linux | Secret Service (GNOME Keyring / KDE Wallet) |
| Windows | Credential Manager (WinCred) |

For headless/CI/Docker environments where no keyring daemon is available, use environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`). `ResolveAPIKey` checks the keyring first, then falls back to env vars.

## Credential ref format

Stored in config as `api_key_ref: "keychain:obk/<provider>"`. The `keychain:` prefix is a historical name — it works on all platforms, not just macOS Keychain.

## Design decisions and learnings

### Why keyring-only, no file fallback

Early versions silently fell back to plain-text files (`~/.obk/secrets/`) when the keyring was unavailable. We removed this after researching how production CLI tools handle it:

- **GitHub CLI (`gh`)** does the same silent file fallback and [their team now considers it a mistake](https://github.com/cli/cli/issues/10108). Users unknowingly store tokens in plain text.
- **aws-vault** never auto-falls back. Users must explicitly choose a backend (`--backend=file`).
- **docker-credential-helpers** errors out if the configured helper fails.

**The pattern:** Keyring succeeds or errors. Env vars cover headless. No silent degradation to insecure storage.

### Why `zalando/go-keyring` over `99designs/keyring`

- Pure Go on all platforms — works with `CGO_ENABLED=0` for cross-compilation
- Simple API (`Get`/`Set`/`Delete`) — we don't need backend selection, encrypted file fallback, or kwallet/pass/keyctl support
- Same library used by `gh` (GitHub CLI) and `chezmoi`
- `99designs/keyring` is more powerful but pulls in more dependencies and complexity (backend selection, encrypted file backend, passphrase prompts) that we don't need

### Why errors propagate directly

Keyring errors (locked keychain, unsupported platform, data too big) are returned to the caller. This means:

- Users see the real problem instead of a misleading "file not found" from a hidden fallback
- `obk setup` fails visibly if the keyring isn't working, rather than silently writing secrets to disk
- Debugging is straightforward — the error message tells you what happened

### Backward compatibility

- `go-keyring` can read credentials stored by the old hand-rolled `security` CLI code on macOS (it checks for encoding prefixes and returns raw values as-is)
- New credentials stored by `go-keyring` are base64-encoded in the keychain — this is a one-way migration (no downgrade path, which is fine)
- The `keychain:` ref prefix in config files is preserved
