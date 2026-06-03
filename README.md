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

---

## The problem it solves

Claude Code only supports one account at a time. If you have a personal subscription and a work subscription, switching means `/logout` → browser → log back in → every single time.

That's five steps and a browser redirect to do something that should be instant.

**Claude Flipper** saves your accounts once and flips between them with a single command.

---

## Before and after

| Before | After |
|---|---|
| `/logout` → browser → log back in every time | `flipper swap` — done |
| Risk logging into the wrong account mid-session | Named slots — always know which account is active |
| No way to see which account is active | `flipper status` shows you instantly |
| Credentials scattered across browser sessions | One tool manages all accounts securely |

---

## Installation

### macOS
```bash
brew install thecoderbuddy/tap/claude-flipper
```

That's it — no Go installation required.

### Linux / Windows

**Prerequisites:** [Go 1.21+](https://go.dev/dl/)

```bash
git clone https://github.com/thecoderbuddy/claude-flipper.git
cd claude-flipper
go build -o flipper .
```

**Linux** — move the binary to your PATH:
```bash
mv flipper /usr/local/bin/
```

**Windows** — move `flipper.exe` to a folder in your `PATH`, or run it directly.

Verify the install:
```bash
flipper --help
```

---

## Quick Start

### 1. Log in and save your first account

Make sure you're logged into Claude Code with your first account, then:

```bash
flipper add
```

This captures your current session and saves it to slot 1.

### 2. Log in and save your second account

Open Claude Code, log out, log in with your second account, exit. Then:

```bash
flipper add
```

This saves your second session to slot 2.

### 3. Check both accounts are saved

```bash
flipper list
```

```
SLOT    ACT  EMAIL                     ORG
----    ---  -----                     ---
1            personal@gmail.com        Personal
2       *    work@company.com          Acme Corp
```

### 4. Flip between them

```bash
flipper swap                      # rotate to the next account
flipper swap 1                    # jump to slot 1 by number
flipper swap personal@gmail.com   # jump by email
```

That's it. Open Claude Code and you're in the right account.

---

## Commands

| Command | What it does |
|---|---|
| `flipper add` | Save the currently logged-in Claude Code account as a new slot |
| `flipper swap` | Rotate to the next account in the sequence |
| `flipper swap <slot\|email>` | Jump to a specific account by slot number or email |
| `flipper list` | Show all saved accounts — active slot marked in the ACT column |
| `flipper status` | Show which account is currently active |
| `flipper remove <slot\|email>` | Remove an account by slot number or email |
| `flipper reset` | Remove all saved accounts and wipe all Claude Flipper data |

---

## How it works

**Saving an account (`flipper add`):**
- Reads your current Claude Code session from `~/.claude.json`
- Reads your credentials from the macOS Keychain (macOS), credentials file (Linux), or Credential Manager (Windows)
- Backs them up to `~/.claude-flipper/` under a numbered slot

**Swapping accounts (`flipper swap`):**
- Backs up the current account credentials and config
- Loads the target account's credentials and config from the backup
- Writes the target credentials back to the Keychain / credentials file
- Updates `~/.claude.json` with the target account's session

If anything fails mid-swap, it rolls back automatically — you're never left in a broken state.

**Credentials storage:**
- macOS → macOS Keychain (native, encrypted)
- Linux → file-based, `0600` permissions
- Windows → Windows Credential Manager

---

## Data location

```
~/.claude-flipper/                   macOS and Windows
~/.local/share/claude-flipper/       Linux (XDG)

├── sequence.json           Master account list and active slot
├── credentials/            Credential backups per slot
└── configs/                Config backups per slot
```

---

## Contributing

Bug reports, improvements, and platform-specific fixes are welcome.

- Open an issue to report a bug or request a feature
- Fork, fix, and open a PR — keep it focused on one thing

---

## License

MIT — use it, fork it, adapt it.
