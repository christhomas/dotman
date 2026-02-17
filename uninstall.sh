#!/usr/bin/env bash
set -euo pipefail

# Dotman uninstaller
#
# High-level flow:
#  1) Remove the installed dotman binary at ~/.local/bin/dotman
#  2) If ~/.local/bin is empty after removal, remove the marked PATH block that install.sh added
#     to the appropriate shell rc file.
#
# Safety notes:
#  - We only remove the PATH block when ~/.local/bin is empty, so we don't break other
#    tools that may also rely on ~/.local/bin.
#  - We do NOT remove the ~/.local/bin directory.
#  - The PATH block is identified by unique marker lines inserted by install.sh.
#
# Optional:
#  - NO_COLOR (set to disable ANSI colors)

print_usage() {
  cat <<'USAGE'
Usage:
  ./uninstall.sh

What this does:
  - Removes ~/.local/bin/dotman
  - If ~/.local/bin is empty after removal, removes the PATH block added by install.sh
USAGE
}

init_colors() {
  # Same color policy as install.sh: enable only for interactive TTY and when NO_COLOR is unset.
  if [[ -t 1 ]] && [[ "${NO_COLOR:-}" == "" ]]; then
    C_RESET=$'\033[0m'
    C_BOLD=$'\033[1m'
    C_RED=$'\033[0;31m'
    C_GREEN=$'\033[0;32m'
    C_YELLOW=$'\033[0;33m'
    C_BLUE=$'\033[0;34m'
    C_CYAN=$'\033[0;36m'
  else
    C_RESET=''
    C_BOLD=''
    C_RED=''
    C_GREEN=''
    C_YELLOW=''
    C_BLUE=''
    C_CYAN=''
  fi
}

info() { printf '%s[INFO]%s %s\n' "$C_BLUE" "$C_RESET" "$*"; }
warn() { printf '%s[WARN]%s %s\n' "$C_YELLOW" "$C_RESET" "$*"; }
ok() { printf '%s[OK]%s %s\n' "$C_GREEN" "$C_RESET" "$*"; }
err() { printf '%s[ERROR]%s %s\n' "$C_RED" "$C_RESET" "$*" >&2; }

remove_marker_block_from_file() {
  # Remove the exact block between the marker lines (inclusive markers are removed too).
  # This is how we undo the PATH modification performed by install.sh.
  local file="$1"
  local block_start='# dotman_installer_please_do_not_edit_start'
  local block_end='# dotman_installer_please_do_not_edit_end'

  [[ -f "$file" ]] || return 0

  if ! grep -Fqs "$block_start" "$file"; then
    return 0
  fi

  local tmp
  tmp=$(mktemp)

  awk -v start="$block_start" -v end="$block_end" '
    $0 == start {inblock=1; next}
    $0 == end {inblock=0; next}
    inblock != 1 {print}
  ' "$file" >"$tmp"

  mv "$tmp" "$file"
  ok "Removed dotman PATH block from $file"
}

get_shell_rc_file() {
  # Choose which shell rc file to modify based on $SHELL.
  # Note: this intentionally mirrors install.sh.
  local shell_name
  shell_name=$(basename "${SHELL:-}")

  case "$shell_name" in
    zsh)
      printf '%s\n' "$HOME/.zshrc"
      ;;
    bash)
      if [[ -f "$HOME/.bashrc" ]]; then
        printf '%s\n' "$HOME/.bashrc"
      else
        printf '%s\n' "$HOME/.bash_profile"
      fi
      ;;
    fish)
      printf '%s\n' "$HOME/.config/fish/config.fish"
      ;;
    *)
      printf '%s\n' ''
      ;;
  esac
}

is_dir_empty() {
  # Return 0 if directory is empty or does not exist; 1 otherwise.
  # This uses `find ... -print -quit` for portability.
  local dir="$1"
  [[ -d "$dir" ]] || return 0

  # Portable emptiness check (works on macOS bash 3.2)
  if find "$dir" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null | grep -q .; then
    return 1
  fi
  return 0
}

main() {
  init_colors

  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    print_usage
    exit 0
  fi

  printf '%s%sDotman uninstaller%s\n\n' "$C_BOLD" "$C_CYAN" "$C_RESET"

  local bin_dir="$HOME/.local/bin"
  local bin_path="$bin_dir/dotman"

  if [[ -f "$bin_path" ]]; then
    rm -f "$bin_path"
    ok "Removed $bin_path"
  else
    warn "Binary not found: $bin_path"
  fi

  if is_dir_empty "$bin_dir"; then
    info "$bin_dir is empty"

    local rc_file
    rc_file=$(get_shell_rc_file)
    if [[ -n "$rc_file" ]]; then
      remove_marker_block_from_file "$rc_file"
    else
      warn 'Could not determine your shell rc file from $SHELL; not removing PATH block.'
    fi
  else
    info "$bin_dir is not empty; leaving PATH block in place"
  fi

  ok 'Uninstall complete.'
}

main "$@"
