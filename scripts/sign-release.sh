#!/usr/bin/env bash
set -euo pipefail
python3 scripts/ztarc.py release-check
python3 scripts/ztarc.py verify
for file in dist/SHA256SUMS dist/manifest.json dist/install.sh; do
 cosign sign-blob --yes --bundle "$file.sigstore.json" "$file"
done
scripts/verify-release.sh dist
