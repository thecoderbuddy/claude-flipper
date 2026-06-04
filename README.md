# Claude Flipper

> Flip between multiple Claude Code accounts without logging out and back in — every time.

<p align="center">
  <a href="https://github.com/thecoderbuddy/claude-flipper/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License" />
  </a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey" alt="Cross platform" />
  <img src="https://img.shields.io/badge/built_with-Go-00ADD8?logo=go" alt="Built with Go" />
  <a href="https://claude.ai/code">
    <img src="https://img.shields.io/badge/Claude_Code-compatible-blueviolet?logo=anthropic" alt="Claude Code compatible" />
  </a>
</p>

<p align="center">
  <a href="#installation">Install</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#commands">Commands</a> •
  <a href="#how-it-works">How It Works</a>
</p>

<p align="center">
  <a href="https://buymeacoffee.com/sharvari">
    <img src="https://img.shields.io/badge/Buy%20Me%20a%20Coffee-support%20this%20project-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black" alt="Buy Me a Coffee" />
  </a>
</p>

---

## The problem it solves

Claude Code only supports one account at a time. If you have a personal subscription and a work subscription, switching means `/logout` → browser → log back in → every single time.

That's five steps and a browser redirect to do something that should be instant.

**Claude Flipper** saves your accounts once and flips between them with a single command.

---

## Before and after

**Without Claude Flipper:**
```
/logout → wait for browser → log in with other account → back to terminal
```
Five steps every time you switch. Easy to end up in the wrong account without noticing.

**With Claude Flipper:**
```
flipper swap && claude
```
One command. Instant. Always know which account is active.

---

## Installation

### macOS

```bash
brew install thecoderbuddy/tap/claude-flipper
flipper setup
source ~/.zshrc
```

`flipper setup` adds a shell wrapper that injects your active account's token every time you run `claude`. This is required for account switching to work.

**To upgrade:**
```bash
brew upgrade thecoderbuddy/tap/claude-flipper
```

> **Note:** Upgrading does not affect your saved accounts. Your slots, credentials, and config backups in `~/.claude-flipper/` are preserved across upgrades.

### Linux
```bash
curl -fsSL https://raw.githubusercontent.com/thecoderbuddy/claude-flipper/main/install.sh | bash
flipper setup
source ~/.bashrc
```

Supports x86_64 and arm64. Installs to `/usr/local/bin/`.

> **Note:** On Linux, Claude Code stores credentials in `~/.claude/.credentials.json` — no Keychain involved. `flipper setup` still adds the shell wrapper as a best practice for consistent behaviour across platforms.

### Windows

Coming soon.

---

## Quick Start

### Step 1 — Save your first account

Open Claude Code and make sure you're logged in with your first account:

```bash
claude
```

Once you're in, **exit Claude Code** (`/exit`). Then run:

```bash
flipper add
```

This captures the session and saves it as slot 1.

### Step 2 — Save your second account

Open Claude Code, log out, then log in with your second account:

```bash
claude
```

Inside Claude Code:
1. Run `/logout` — this logs out your current account
2. Run `claude` again to reopen
3. Run `/login` — this opens the browser, sign in with your second account and complete the flow
4. Run `/exit` once logged in

Then run:

```bash
flipper add
```

This saves the second session as slot 2.

> **Why `/login` after `/logout`?** After `/logout`, Claude Code auto-picks the last used account on next open. Running `/login` forces a fresh login prompt so you can choose a different account.

### Step 3 — Confirm both accounts are saved

```bash
flipper list
```

```
SLOT    ACT  EMAIL                     ORG
----    ---  -----                     ---
1            personal@gmail.com        Personal
2       *    work@company.com          Acme Corp
```

### Step 4 — Flip between them

```bash
flipper swap && claude        # rotate to next account and open Claude
flipper swap 1 && claude      # jump to slot 1 by number
flipper swap work@company.com # jump by email
```

> **Note:** Always exit Claude Code (`/exit`) before swapping. The desktop app overwrites credentials while running.

---

## Commands

| Command | What it does |
|---|---|
| `flipper add` | Save the currently logged-in Claude Code account as a new slot |
| `flipper swap` | Rotate to the next account in the sequence |
| `flipper swap <slot\|email>` | Jump to a specific account by slot number or email |
| `flipper list` | Show all saved accounts — active slot marked in the ACT column |
| `flipper status` | Show which account is currently active |
| `flipper setup` | Add the claude shell wrapper to your shell config (run once after install) |
| `flipper token` | Print the active account's access token (refreshes if expiring) |
| `flipper doctor` | Diagnose token expiry, keychain state, and config for all slots |
| `flipper remove <slot\|email>` | Remove an account by slot number or email |
| `flipper reset` | Remove all saved accounts and wipe all Claude Flipper data |

---

## How it works

**Saving an account (`flipper add`):**
- Reads your current Claude Code session from `~/.claude.json`
- Reads your credentials from the macOS Keychain (macOS) or credentials file (Linux)
- Backs them up to `~/.claude-flipper/` under a numbered slot

**Swapping accounts (`flipper swap`):**
- Backs up the current account credentials and config
- Loads the target account's credentials and config from the backup
- Refreshes the access token if it is about to expire
- Updates `~/.claude.json` with the target account's session

**Why the shell wrapper (`flipper setup`):**

macOS Security.framework (used internally by Claude Code) cannot read Keychain entries written by third-party processes. Rather than fight the Keychain, flipper injects the token via the `ANTHROPIC_AUTH_TOKEN` environment variable — which the Anthropic SDK reads directly, bypassing the Keychain lookup entirely.

The wrapper added by `flipper setup` does this automatically every time you run `claude`:
```bash
claude() { ANTHROPIC_AUTH_TOKEN="$(flipper token)" command claude "$@"; }
```

If anything fails mid-swap, it rolls back automatically — you're never left in a broken state.

---

## Troubleshooting

**"Not logged in" after swap**
- Make sure you ran `flipper setup` and reloaded your shell (`source ~/.zshrc`)
- Check `flipper doctor` to see token expiry and config state for all slots

**Claude.app blocks the swap**
- flipper blocks swaps while Claude.app (desktop) is running — it overwrites credentials
- Quit Claude.app (⌘Q) first, then run `flipper swap`

**Token expired**
- `flipper swap` automatically refreshes tokens before writing them
- If the refresh token is revoked (e.g. you ran `/logout`), re-add the account: `flipper add`

---

## Data location

```
~/.claude-flipper/                   macOS and Windows
~/.local/share/claude-flipper/       Linux (XDG)

├── sequence.json           Master account list and active slot
├── credentials/            Credential backups per slot
└── configs/                Config backups per slot
```

## Privacy

**Claude Flipper does not collect, transmit, or share any data.**

Everything stays on your machine:
- Credentials are stored locally with `0600` permissions
- The only network request flipper makes is refreshing expired OAuth tokens directly with Anthropic's token endpoint
- No telemetry, no analytics
- Open source — you can verify exactly what the code does

---

## Support

If Claude Flipper saves you time, consider buying me a coffee:

<a href="https://buymeacoffee.com/sharvari">
  <img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" />
</a>

## Acknowledgements

Built for [Claude Code](https://claude.ai/code) by [Anthropic](https://anthropic.com).

---

## Contributing

Bug reports, improvements, and platform-specific fixes are welcome.

- Open an issue to report a bug or request a feature
- Fork, fix, and open a PR — keep it focused on one thing

---

## License

MIT — use it, fork it, adapt it.
