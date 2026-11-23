#!/usr/bin/env bash
# For a normal writable Git checkout; environment-owned .git placeholders are
# deliberately not replaced. No commit or remote publication is performed.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
pin=bb03d2c45ce79213c4c9b206bbd87e9846be5a9a
if ! git rev-parse --show-toplevel >/dev/null 2>&1; then git init; fi
if [ ! -f upstream/go.mod ]; then
 git submodule add https://github.com/fosrl/cli.git upstream
else
 # Register an existing clean clone as a real gitlink in the parent index.
 git add .gitmodules
 git update-index --add --cacheinfo "160000,$pin,upstream"
 git submodule absorbgitdirs upstream
fi
git -C upstream checkout --detach "$pin"
python3 scripts/ztarc.py stage
