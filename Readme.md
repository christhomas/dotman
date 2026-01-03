# Dotman

## 🔧 Lets build a simpler dotfile manager

We want a tool that:

1. **Tracks dotfiles in a Git repo** — using real filenames and real directory structure (no `dot_` renaming)
2. **Copies files to and from the home directory** — no symlinks, just explicit syncs
3. **Detects changes** in either direction:
   - If the file changed in the repo: _“Do you want to overwrite your home version with the repo version?”_
   - If the file changed in your home dir: _“Do you want to stage this change into the repo?”_
4. **Gives you a clear, minimal UX** for syncing both ways
5. **Provides a simpler Git UX** — wraps Git with limited, task-specific operations relevant to dotfile workflows
6. **Supports cross-platform** — works on Linux, macOS, and Windows (In theory)

No symlinks, no shadow config repo, no fake naming like `dot_zshrc`, and no indirection. Just a plain, transparent, Git-backed dotfile workflow.

## 📜 Dotman Rules for Managing Dotfiles

- The **Git clone is the source of truth**. Dotman assumes the Git repo reflects the canonical state.
- When you **apply dotfiles to your system**, they are copied over. You’ll be asked whether to overwrite each conflicting file.
- When the **Git repo receives new changes** (e.g., via `git pull`), dotman can prompt you to re-apply them to your system.
- When the **system version of a file changes**, dotman will offer to stage and commit those changes back to the repo with an optional commit message.
- If **both the system and the repo have changes**, dotman enters a conflict resolution state. You’ll be prompted to choose between:
  - Keeping the local system version
  - Overwriting it with the repo version
  - Manually resolving the difference ("you are in a dark place")
- Before **any new submission** (submit), dotman will automatically run `git pull` to ensure you’re working from the latest version.
- You may use `--no-pull` to skip that step, but then you must accept the consequences — you may need to manually resolve merge conflicts.
- In case of **conflicts**, dotman will try to help, but assumes you may have to "get your hands dirty" and resolve it like you would with normal Git.

## 🧠 Motivation

- Avoid filesystem complexities (e.g. symlinks not working in WSL, Docker, Dropbox, etc.)
- Avoid the “hidden behavior” of tools like `chezmoi`, `yadm`, or `stow`
- Provide full Git lifecycle handling with simple UX wrappers — but **not** a replacement for Git itself
- TODO: Allow for **read-only setups** in cloud environments, shared workstations, or deployment scenarios where config should be loaded but not modified
- I tried other dotfile managers, but I was always a bit surprised to find that when it comes to uploading your changes to the repo where the files are stored, suddenly the tool is not managing everything anymore, and you have to remember to `git pull`, `git commit`, `git push`, resolve conflicts, etc. So the tool only seems to do half the job.

---

## 🛠️ Problems

- If on machine 1 modifications are made and published. Then on machine 2 modifications are made and submitted. Then you might run into conflict issues if you've attempted to edit the same thing

## 🎯 Let us build the tool!

It would:
- Maintain a Git repo that the user initializes from
- Copy files from the repo into `$HOME` as needed (**apply**)
- Copy files from `$HOME` into the repo (**submit**)
- Show diffs between the two (**status**)
- Be cross-platform and not rely on symlinks
- Provide Git lifecycle automation with clear choices, not full Git replacement

---

## ✅ Comparison Table

| Feature                   | ✅ Your Tool       | ❌ ChezMoi              | ❌ YADM              | ❌ Stow |
|---------------------------|--------------------|------------------------|----------------------|---------|
| Real filenames            | ✅                 | ❌ (`dot_` indirection) | ✅                   | ✅      |
| Git integration           | ✅ (task-specific) | ❌ (manual Git)         | ❌ (manual Git)      | ❌      |
| Apply/Submit sync model   | ✅                 | ❌ (one-way apply)      | ❌ (manual Git only) | ❌      |
| Secrets support           | Optional           | ✅                      | ✅                   | ❌      |
| Works without symlinks    | ✅                 | ✅                      | ✅ (by default)      | ❌      |
| UX clarity                | ✅                 | ❌                      | ❌                   | ❌      |
| Read-only mode            | ✅                 | ❌                      | ❌                   | ❌      |

---

## 🛠 Core Commands

```bash
# Initialize dotman
$ dotman init <repo-url> <target-dir>

# Add a file to the repo by copying it
$ dotman add ~/.zshrc

# Apply changes: shows diff, asks what to do per file
$ dotman apply

# Submit file(s) from home → repo
$ dotman submit

# Publish file(s) from repo → home
$ dotman publish
```

---

## 📁 Repo Layout

The Git repo should mirror the real structure:

```
~/.dotman/
├── hooks/
│   └── bootstrap.sh
└── home/
    ├── .zshrc
    ├── .Brewfile
    └── .ghostty/
        └── config
```

No renaming. The file `.dotman/home/.zshrc` corresponds exactly to `~/.zshrc`.

---

## 🧪 Internal Data

Optionally, `dotman` could maintain a metadata file like `.dotman.json` to keep track of the last synced checksums and locations.

---

## 🔐 Optional Features

- Auto-commit after `dotman publish`
- Simple conflict resolution with colorized diff
- Can run as a dry-run
- Git integration (`dotman show`, `dotman commit`, `dotman publish`)
- Config file to specify tracked paths manually
- Read-only clone support for ephemeral or CI setups

---

## 📋 Project Task Checklist

### 🎯 Core Goals
- [x] Track real dotfiles using exact paths and filenames
- [x] Avoid symlinks or file renaming (`dot_*`)
- [x] Enable pull/push file sync model between `$HOME` and repo
- [x] Cross-platform, minimal setup, Git-friendly
- [ ] Read-only mode for Git-based consumption without modification

### 🛠 Core Commands
- [x] `dotman init <repourl> <targetdir>` — initialize dotman in existing folder
- [x] `dotman apply` — copy from home → repo
- [x] `dotman add <file>` — add file from `$HOME` into repo
- [x] `dotman submit` — stage and commit changes from `$HOME` back to the repo
- [x] `dotman publish` — copy from repo → home

### 🔧 Internal Functionality
- [x] Set up Cobra CLI framework
- [x] Project skeleton with Go modules
- [x] `$HOME` and `$XDG_DATA_HOME` detection
- [ ] Track known files in `.dotman/config.json`
- [x] Fully integrated Git lifecycle: commit, push, pull, etc
- [ ] Implement read-only repo mode logic (disable write paths)

### 🧪 UX Enhancements
- [x] Interactive diffs like `git add -p`
- [x] Dry-run support
- [x] Pretty terminal output and prompts
- [x] Logging / verbosity flags

### 🧠 Future Features
- [ ] Host-specific or profile-based overrides
- [ ] Secrets encryption support
