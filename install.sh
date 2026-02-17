#!/usr/bin/env bash
set -euo pipefail

# Dotman installer
#
# High-level flow:
#  1) Show a safety warning (red box) + ask for confirmation
#  2) Fetch the last up to 5 GitHub releases and let you choose a version
#     (if fewer than 5 releases exist, the selection range is 1..n)
#  3) Detect OS/arch and download the matching release asset
#  4) Install the binary to ~/.local/bin/dotman
#  5) If ~/.local/bin is not on PATH, append a marked block to your shell rc file so
#     future shells can find dotman (the block is removable by uninstall.sh)
#
# Requirements:
#  - bash
#  - curl
#  - jq
# Optional:
#  - GITHUB_TOKEN (recommended for GitHub API rate limits / private repos)
#  - NO_COLOR (set to disable ANSI colors)

print_usage() {
  cat <<'USAGE'
Usage:
  ./install.sh [owner/repo]

Examples:
  ./install.sh christhomas/dotman
  ./install.sh   # tries to infer owner/repo from `git remote get-url origin`

What this does:
  - Welcomes you
  - Lists the last 5 GitHub releases
  - Lets you pick a version to install (selection only; install steps can be added later)
USAGE
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '%sMissing required command:%s %s\n' "${C_RED}" "${C_RESET}" "$1" >&2
    exit 1
  fi
}

init_colors() {
  # Initialize ANSI color codes.
  # Colors are disabled when stdout is not a TTY or when NO_COLOR is set.
  if [[ -t 1 ]] && [[ "${NO_COLOR:-}" == "" ]]; then
    C_RESET=$'\033[0m'
    C_BOLD=$'\033[1m'
    C_DIM=$'\033[2m'
    C_RED=$'\033[0;31m'
    C_GREEN=$'\033[0;32m'
    C_YELLOW=$'\033[0;33m'
    C_BLUE=$'\033[0;34m'
    C_CYAN=$'\033[0;36m'
    C_BG_RED=$'\033[41m'
    C_FG_WHITE=$'\033[97m'
  else
    C_RESET=''
    C_BOLD=''
    C_DIM=''
    C_RED=''
    C_GREEN=''
    C_YELLOW=''
    C_BLUE=''
    C_CYAN=''
    C_BG_RED=''
    C_FG_WHITE=''
  fi
}

info() { printf '%s[INFO]%s %s\n' "$C_BLUE" "$C_RESET" "$*"; }
warn() { printf '%s[WARN]%s %s\n' "$C_YELLOW" "$C_RESET" "$*"; }
ok() { printf '%s[OK]%s %s\n' "$C_GREEN" "$C_RESET" "$*"; }
success() { printf '%s[SUCCESS]%s %s\n' "$C_GREEN" "$C_RESET" "$*"; }
err() { printf '%s[ERROR]%s %s\n' "$C_RED" "$C_RESET" "$*" >&2; }

warn_box_red() {
  # Render a warning in a red background “box”.
  # Falls back to plain text if colors are disabled.
  local msg="$*"
  if [[ -n "${C_BG_RED}" && -n "${C_FG_WHITE}" ]]; then
    local pad
    pad=$(printf '%*s' "$(( ${#msg} + 4 ))" '')
    printf '%s%s%s%s%s\n' "$C_BG_RED" "$C_FG_WHITE" "$C_BOLD" "$pad" "$C_RESET"
    printf '%s%s%s  %s  %s\n' "$C_BG_RED" "$C_FG_WHITE" "$C_BOLD" "$msg" "$C_RESET"
    printf '%s%s%s%s%s\n' "$C_BG_RED" "$C_FG_WHITE" "$C_BOLD" "$pad" "$C_RESET"
  else
    printf '%s\n' "$msg"
  fi
}

github_curl() {
  # Wrapper around curl that adds GitHub API headers safely (no word-splitting issues)
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    curl -H 'Accept: application/vnd.github+json' \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      "$@"
  else
    curl -H 'Accept: application/vnd.github+json' \
      "$@"
  fi
}

detect_asset_name() {
  # Map current OS/arch to the asset naming scheme produced by .github/workflows/release.yml
  local os
  local arch

  os=$(uname -s)
  arch=$(uname -m)

  case "$os" in
    Linux) os='linux' ;;
    Darwin) os='macos' ;;
    *)
      err "Unsupported OS: $os"
      exit 1
      ;;
  esac

  case "$arch" in
    x86_64|amd64) arch='amd64' ;;
    arm64|aarch64) arch='arm64' ;;
    *)
      err "Unsupported architecture: $arch"
      exit 1
      ;;
  esac

  printf 'dotman-%s-%s\n' "$os" "$arch"
}

ensure_local_bin_on_path() {
  # Ensure ~/.local/bin is on PATH for future shells.
  #
  # We append a marked block so uninstall.sh can remove it safely if ~/.local/bin becomes empty.
  # The marker lines are intentionally unique:
  #   # dotman_installer_please_do_not_edit_start
  #   ...
  #   # dotman_installer_please_do_not_edit_end
  local bin_dir="$HOME/.local/bin"
  local block_start='# dotman_installer_please_do_not_edit_start'
  local block_end='# dotman_installer_please_do_not_edit_end'

  case ":${PATH:-}:" in
    *":${bin_dir}:"*)
      return 0
      ;;
  esac

  local shell_name
  shell_name=$(basename "${SHELL:-}")

  local rc_file=''
  local snippet=''

  case "$shell_name" in
    zsh)
      rc_file="$HOME/.zshrc"
      snippet='export PATH="$PATH:$HOME/.local/bin"'
      ;;
    bash)
      # Prefer ~/.bashrc if present; otherwise use ~/.bash_profile
      if [[ -f "$HOME/.bashrc" ]]; then
        rc_file="$HOME/.bashrc"
      else
        rc_file="$HOME/.bash_profile"
      fi
      snippet='export PATH="$PATH:$HOME/.local/bin"'
      ;;
    fish)
      rc_file="$HOME/.config/fish/config.fish"
      snippet='set -gx PATH $PATH $HOME/.local/bin'
      ;;
    *)
      warn "~/.local/bin is not currently on PATH. Please add it for your shell (${shell_name})."
      info 'Suggested: export PATH="$PATH:$HOME/.local/bin"'
      return 0
      ;;
  esac

  mkdir -p "$(dirname "$rc_file")"
  touch "$rc_file"

  if grep -Fqs "$block_start" "$rc_file"; then
    return 0
  fi

  if grep -Fqs "$snippet" "$rc_file"; then
    return 0
  fi

  {
    printf '\n%s\n' "$block_start"
    printf '%s\n' "$snippet"
    printf '%s\n' "$block_end"
    printf '\n'
  } >>"$rc_file"
  ok "Added ~/.local/bin to PATH in $rc_file"
}

infer_repo_from_git_remote() {
  # Infer owner/repo from the current git remote "origin".
  if ! command -v git >/dev/null 2>&1; then
    return 1
  fi

  local url
  if ! url=$(git remote get-url origin 2>/dev/null); then
    return 1
  fi

  # Supported:
  # - https://github.com/owner/repo.git
  # - git@github.com:owner/repo.git
  # - https://github.com/owner/repo
  url=${url%.git}

  if [[ "$url" =~ ^https?://github.com/([^/]+/[^/]+)$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return 0
  fi

  if [[ "$url" =~ ^git@github.com:([^/]+/[^/]+)$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return 0
  fi

  return 1
}

parse_release_tags() {
  # Parse tag_name values from GitHub Releases JSON.
  # Reads GitHub releases JSON from stdin and prints tag_name values (one per line)
  if command -v jq >/dev/null 2>&1; then
    jq -r '.[].tag_name'
    return 0
  fi

  # Minimal fallback without jq.
  # This is intentionally simple: it extracts lines containing "tag_name".
  sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]\+\)".*/\1/p'
}

main() {
  init_colors
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    print_usage
    exit 0
  fi

  printf '%s%sDotman installer%s\n\n' "$C_BOLD" "$C_CYAN" "$C_RESET"
  warn_box_red 'You are very brave to run a script directly from the internet, please read the code first?'
  printf '\n'

  local confirm
  read -r -p 'Are you sure you want to install this program? (y/N): ' confirm
  confirm=$(printf '%s' "$confirm" | tr '[:upper:]' '[:lower:]')
  if [[ "$confirm" != "y" && "$confirm" != "yes" ]]; then
    warn 'Cancelled.'
    exit 0
  fi

  require_cmd curl
  require_cmd jq

  local repo="${1:-}"
  if [[ -z "$repo" ]]; then
    if repo=$(infer_repo_from_git_remote); then
      :
    else
      err 'Could not determine GitHub repo.'
      err 'Provide it as an argument like: ./install.sh owner/repo'
      exit 1
    fi
  fi

  info "Fetching last 5 GitHub releases for ${repo}..."
  printf '\n'

  local releases_json
  if ! releases_json=$(github_curl -fsSL \
    "https://api.github.com/repos/${repo}/releases?per_page=5"); then
    err 'Failed to fetch releases from GitHub API.'
    err 'If this is a private repo, you may need to set GITHUB_TOKEN in your environment.'
    exit 1
  fi

  tags=()
  while IFS= read -r line; do
    tags+=("$line")
  done < <(printf '%s' "$releases_json" | parse_release_tags)

  if [[ ${#tags[@]} -eq 0 ]]; then
    err 'No releases found (or failed to parse).'
    exit 1
  fi

  printf '%sVersions available to install:%s\n' "$C_BOLD" "$C_RESET"
  local i
  for i in "${!tags[@]}"; do
    printf '  %s%d)%s %s\n' "$C_CYAN" "$((i + 1))" "$C_RESET" "${tags[$i]}"
  done
  printf '\n'

  local max_choice
  max_choice=${#tags[@]}

  local selection=""
  while :; do
    read -r -p "Pick a version (1-${max_choice}): " selection
    if [[ "$selection" =~ ^[0-9]+$ ]] && (( selection >= 1 && selection <= max_choice )); then
      break
    fi
    err 'Invalid selection.'
  done

  local chosen="${tags[$((selection - 1))]}"

  ok "Selected version to install: ${chosen}"

  local asset_name
  asset_name=$(detect_asset_name)
  info "Detected platform asset: ${asset_name}"

  info "Fetching release metadata for ${chosen}..."
  local release_json
  if ! release_json=$(github_curl -fsSL \
    "https://api.github.com/repos/${repo}/releases/tags/${chosen}"); then
    err "Failed to fetch release metadata for tag ${chosen}"
    exit 1
  fi

  local download_url
  download_url=$(printf '%s' "$release_json" | jq -r --arg name "$asset_name" '.assets[]? | select(.name == $name) | .browser_download_url' | head -n 1)
  if [[ -z "$download_url" || "$download_url" == "null" ]]; then
    err "Could not find a release asset named ${asset_name} for tag ${chosen}"
    err 'Available assets:'
    printf '%s' "$release_json" | jq -r '.assets[]?.name' >&2
    exit 1
  fi

  local bin_dir="$HOME/.local/bin"
  mkdir -p "$bin_dir"

  local tmp
  tmp=$(mktemp)
  info "Downloading ${download_url}..."
  if ! github_curl -fL --retry 3 --retry-delay 1 \
    -o "$tmp" \
    "$download_url"; then
    rm -f "$tmp"
    err 'Download failed.'
    exit 1
  fi

  local dest="$bin_dir/dotman"
  mv "$tmp" "$dest"
  chmod +x "$dest"

  ok "Installed dotman to: ${dest}"

  ensure_local_bin_on_path

  printf '\n'
  success 'Installation complete.'
  success "Binary installed at: ${dest}"
  success 'Close this terminal and open a new one to ensure your updated PATH is loaded.'
  success 'Then run: dotman --help'
}

main "$@"
