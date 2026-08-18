#!/usr/bin/env bash
# =============================================================================
# PatchMon Docker - Environment Setup Script
# =============================================================================
# Downloads docker-compose.yml and env.example if not already present,
# then:
# 1. Creates .env from env.example (an existing .env is kept, see below)
# 2. Fills in any empty POSTGRES_PASSWORD, REDIS_PASSWORD (32 hex)
# 3. Fills in any empty JWT_SECRET, SESSION_SECRET, AI_ENCRYPTION_KEY (64 hex)
# 4. Interactively configures CORS_ORIGIN, TRUST_PROXY, and TZ
#
# Re-running the script is safe: an existing .env is backed up and kept, and
# only secrets that are still empty get generated. That matters most for
# POSTGRES_PASSWORD, which Postgres bakes into the data volume the first time
# it initialises: handing it a new password later does not change the stored
# one, it just makes the server fail to log in with
# "password authentication failed for user". Use --force only when you also
# intend to start from an empty database volume.
#
# Run from any directory:
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/PatchMon/PatchMon/refs/heads/main/docker/setup-env.sh)"
# Or if already downloaded:
#   ./setup-env.sh            # keep an existing .env, fill in what is missing
#   ./setup-env.sh --force    # discard .env and generate a fresh set of secrets
# =============================================================================

set -e

FORCE=0
for arg in "$@"; do
  case "$arg" in
    -f|--force)
      FORCE=1
      ;;
    -h|--help)
      sed -n '2,26p' "${BASH_SOURCE[0]}" 2>/dev/null || true
      exit 0
      ;;
    *)
      echo "Unknown option: $arg (supported: --force)" >&2
      exit 1
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

UPSTREAM="https://raw.githubusercontent.com/PatchMon/PatchMon/refs/heads/main/docker"

# -----------------------------------------------------------------------------
# Download docker-compose.yml if not present
# -----------------------------------------------------------------------------
if [ ! -f "./docker-compose.yml" ]; then
  echo "docker-compose.yml not found. Downloading from upstream..."
  if ! curl -fsSL -o docker-compose.yml "$UPSTREAM/docker-compose.yml"; then
    echo "Error: Failed to download docker-compose.yml." >&2
    exit 1
  fi
  echo "docker-compose.yml downloaded."
fi

# -----------------------------------------------------------------------------
# Ensure env.example exists locally, or download if missing
# -----------------------------------------------------------------------------
if [ ! -f "./env.example" ]; then
  echo "env.example not found. Downloading from upstream..."
  if ! curl -fsSL -o env.example "$UPSTREAM/env.example"; then
    echo "Error: Failed to download env.example." >&2
    exit 1
  fi
  echo "env.example downloaded."
fi

# -----------------------------------------------------------------------------
# Create or preserve .env
# -----------------------------------------------------------------------------
# Re-running this script used to overwrite .env with a fresh env.example and a
# fresh set of secrets. That silently breaks an existing install: Postgres only
# reads POSTGRES_PASSWORD when it initialises an empty data directory, so a
# regenerated password never reaches the database and every later connection
# fails with "password authentication failed for user". Keep the existing file
# by default and only fill in what is still empty.
TIMESTAMP="$(date +%Y%m%d%H%M%S)"

if [ -f .env ]; then
  cp .env ".env.bak.$TIMESTAMP"
  echo "Existing .env backed up to .env.bak.$TIMESTAMP"
  if [ "$FORCE" -eq 1 ]; then
    echo "--force given: replacing .env with a fresh copy of env.example."
    echo "WARNING: this regenerates POSTGRES_PASSWORD. If the database volume"
    echo "         already exists it still holds the old password, and the"
    echo "         server will fail to connect until you either delete the"
    echo "         volume or change the password inside Postgres."
    cp env.example .env
  else
    echo "Keeping it. Only secrets that are still empty will be generated."
    echo "(Run with --force to start over from env.example.)"
  fi
else
  echo "Copying env.example to .env"
  cp env.example .env
fi

echo ""

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

# Read the value of KEY from .env, empty if the key is absent or commented out.
read_env_var() {
  [ -f .env ] || return 0
  grep -E "^$1=" .env | tail -n 1 | cut -d= -f2- || true
}

# Set KEY=VALUE in .env, appending the line if it is not already there.
# perl rather than sed because generated values and URLs contain / and &.
set_env_var() {
  PM_KEY="$1" PM_VALUE="$2" perl -i -pe 's/^\Q$ENV{PM_KEY}\E=.*/$ENV{PM_KEY}=$ENV{PM_VALUE}/' .env
  if ! grep -qE "^$1=" .env; then
    printf '%s=%s\n' "$1" "$2" >> .env
  fi
}

# Generate KEY only when it has no value yet, so re-runs keep working secrets.
# Each secret gets its own random value.
ensure_secret() {
  local key="$1" bytes="$2"
  if [ -n "$(read_env_var "$key")" ]; then
    echo "  $key: kept"
    return 0
  fi
  set_env_var "$key" "$(openssl rand -hex "$bytes")"
  echo "  $key: generated"
  GENERATED_SECRETS="$GENERATED_SECRETS $key"
}

# -----------------------------------------------------------------------------
# Secrets
# -----------------------------------------------------------------------------
GENERATED_SECRETS=""

echo "Secrets:"
ensure_secret POSTGRES_PASSWORD 32
ensure_secret REDIS_PASSWORD 32
ensure_secret JWT_SECRET 64
ensure_secret SESSION_SECRET 64
ensure_secret AI_ENCRYPTION_KEY 64

# A generated POSTGRES_PASSWORD only reaches Postgres on a first start with an
# empty data directory. If the volume is already there, say so now rather than
# letting the server crash-loop on an authentication failure later.
case " $GENERATED_SECRETS " in
  *" POSTGRES_PASSWORD "*)
    if command -v docker >/dev/null 2>&1 &&
       docker volume ls -q 2>/dev/null | grep -qx "patchmon_postgres_data"; then
      echo ""
      echo "WARNING: the patchmon_postgres_data volume already exists, so the"
      echo "         database still uses its original password and will reject"
      echo "         the one just written to .env."
      echo ""
      echo "         Point .env at the existing database by restoring the old"
      echo "         POSTGRES_PASSWORD from a .env.bak.* file, or set the new"
      echo "         password on the database itself:"
      echo ""
      echo "           docker compose up -d database"
      echo "           docker compose exec database psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" \\"
      echo "             -c \"ALTER USER \\\"\$POSTGRES_USER\\\" WITH PASSWORD '<new password>';\""
      echo ""
      echo "         Deleting the volume also works, and destroys all PatchMon data."
    fi
    ;;
esac

echo ""
echo "Done. .env is ready."
echo ""

# -----------------------------------------------------------------------------
# Interactive CORS_ORIGIN builder (skip if not a TTY, e.g. CI)
# -----------------------------------------------------------------------------
if [ -t 0 ]; then
echo "=== CORS Origin Configuration ==="
echo "PatchMon runs on port 3000 by default. If using a reverse proxy or different host, enter the full URL you will use to access it."
echo ""

# Reverse proxy / TRUST_PROXY
# Default the prompt to whatever .env already says, so pressing enter on a
# re-run keeps the current setting instead of resetting it.
if [ "$(read_env_var TRUST_PROXY)" = "true" ]; then
  proxy_default="y"
else
  proxy_default="n"
fi
read -r -p "Will you be accessing PatchMon via a reverse proxy (nginx, Caddy, etc.)? (y/n) [$proxy_default]: " use_proxy
use_proxy=${use_proxy:-$proxy_default}
if [ "$use_proxy" = "y" ] || [ "$use_proxy" = "Y" ]; then
  TRUST_PROXY_VALUE="true"
  echo "TRUST_PROXY will be set to true (server will trust X-Forwarded-* headers)."
else
  TRUST_PROXY_VALUE="false"
fi
export TRUST_PROXY_VALUE
# Uncomment or replace existing TRUST_PROXY line (matches both "# TRUST_PROXY=..." and "TRUST_PROXY=...")
perl -i -pe 's/^#?\s*TRUST_PROXY=.*/TRUST_PROXY=$ENV{TRUST_PROXY_VALUE}/' .env
if ! grep -q "^TRUST_PROXY=" .env; then
  echo "TRUST_PROXY=$TRUST_PROXY_VALUE" >> .env
fi

echo ""

# Timezone (TZ)
read -r -p "Do you want to change the timezone from UTC? (y/n) [n]: " change_tz
change_tz=${change_tz:-n}
if [ "$change_tz" = "y" ] || [ "$change_tz" = "Y" ]; then
  echo "Examples: Europe/London, America/New_York, America/Los_Angeles, Asia/Tokyo, Australia/Sydney"
  while true; do
    read -r -p "Enter timezone (IANA format, e.g. Europe/London) [UTC]: " tz_input
    tz_input=$(echo "${tz_input:-UTC}" | tr -d ' ')
    if [ -z "$tz_input" ]; then
      tz_input="UTC"
    fi
    # Validate: check zoneinfo path (Linux/macOS) or use zdump
    if [ -f "/usr/share/zoneinfo/$tz_input" ]; then
      TZ_VALUE="$tz_input"
      break
    elif command -v zdump >/dev/null 2>&1 && zdump "$tz_input" >/dev/null 2>&1; then
      TZ_VALUE="$tz_input"
      break
    elif [ "$tz_input" = "UTC" ] || [ "$tz_input" = "Etc/UTC" ] || [ "$tz_input" = "GMT" ] || [ "$tz_input" = "Etc/GMT" ]; then
      TZ_VALUE="$tz_input"
      break
    else
      echo "Invalid timezone: $tz_input. Use IANA format (e.g. America/New_York). Try 'timedatectl list-timezones' for a full list."
    fi
  done
  export TZ_VALUE
  perl -i -pe 's/^#?\s*TZ=.*/TZ=$ENV{TZ_VALUE}/' .env
  if ! grep -q "^TZ=" .env; then
    echo "TZ=$TZ_VALUE" >> .env
  fi
  echo "TZ will be set to: $TZ_VALUE"
else
  echo "Keeping default timezone (UTC)."
fi

echo ""

# Read existing CORS_ORIGIN from .env if present
current_cors=$(grep -E "^CORS_ORIGIN=" .env 2>/dev/null | cut -d= -f2- | tr -d '"' || true)
cors_origins=()
if [ -n "$current_cors" ]; then
  IFS=',' read -ra cors_origins <<< "$current_cors"
fi

# Default if empty
if [ ${#cors_origins[@]} -eq 0 ]; then
  cors_origins=("http://localhost:3000")
fi

# Main URL prompt
echo "PatchMon runs on port 3000 by default. Include the port in your URL unless you are"
echo "terminating it at a reverse proxy on a standard port (e.g. https://patchmon.example.com)."
echo "Examples:  http://192.168.1.10:3000   https://patchmon.local:3000   https://patchmon.example.com"
echo ""
read -r -p "What URL will you use to access PatchMon? [http://localhost:3000]: " input_url
if [ -n "$input_url" ]; then
  cors_origins=("$input_url")
fi

# Add/remove loop
while true; do
  echo ""
  echo "Current CORS origins:"
  for i in "${!cors_origins[@]}"; do
    echo "  $((i + 1)). ${cors_origins[$i]}"
  done
  echo ""
  read -r -p "Add (a), Remove (r), or Done (d) [d]: " action
  action=${action:-d}
  case "$action" in
    a|A)
      read -r -p "Enter URL to add: " new_url
      if [ -n "$new_url" ]; then
        cors_origins+=("$new_url")
      fi
      ;;
    r|R)
      if [ ${#cors_origins[@]} -eq 0 ]; then
        echo "No origins to remove."
      else
        read -r -p "Enter number to remove (1-${#cors_origins[@]}): " idx
        if [[ "$idx" =~ ^[0-9]+$ ]] && [ "$idx" -ge 1 ] && [ "$idx" -le ${#cors_origins[@]} ]; then
          unset 'cors_origins[idx-1]'
          cors_origins=("${cors_origins[@]}")
        else
          echo "Invalid number."
        fi
      fi
      ;;
    d|D)
      break
      ;;
    *)
      echo "Invalid choice. Use a, r, or d."
      ;;
  esac
done

# Build final value
if [ ${#cors_origins[@]} -gt 0 ]; then
  CORS_VALUE=$(IFS=','; echo "${cors_origins[*]}")
  echo ""
  echo "CORS_ORIGIN will be set to: $CORS_VALUE"
  # Update .env (use perl to avoid sed escaping issues with URLs)
  export CORS_VALUE
  perl -i -pe 's/^CORS_ORIGIN=.*/CORS_ORIGIN=$ENV{CORS_VALUE}/' .env
  # Ensure the line exists if it was missing
  if ! grep -q "^CORS_ORIGIN=" .env; then
    echo "CORS_ORIGIN=$CORS_VALUE" >> .env
  fi
fi

echo ""
echo "============================================================"
echo " Setup complete!"
echo "============================================================"
echo ""
echo "NOTE: Before starting for the first time, ensure that:"
echo "  - DNS records are configured to point your domain(s) to this host"
echo "  - If using a reverse proxy (nginx, Caddy, Traefik, etc.), configure"
echo "    it to forward traffic to this host on port 3000"
echo ""
echo "Access your PatchMon server using the following URL(s):"
if [ -n "${CORS_VALUE:-}" ]; then
  IFS=',' read -ra _display_origins <<< "$CORS_VALUE"
  for _url in "${_display_origins[@]}"; do
    echo "  -> $_url"
  done
else
  echo "  (no CORS_ORIGIN configured — edit .env before starting)"
fi
echo ""
echo "Start PatchMon with:"
echo ""
echo "  docker compose up -d"
echo ""
echo "Edit .env to configure PORT if needed (default: 3000)."
else
  echo "Non-interactive mode: skipping CORS_ORIGIN configuration. Edit .env to set CORS_ORIGIN and PORT if needed."
  echo ""
  echo "NOTE: Before starting for the first time, ensure DNS and any reverse proxy"
  echo "are configured to point to this host before running:"
  echo ""
  echo "  docker compose up -d"
fi
