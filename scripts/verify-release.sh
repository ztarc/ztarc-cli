#!/usr/bin/env bash
set -euo pipefail
: "${ZTARC_CERTIFICATE_IDENTITY:?Set the trusted release workflow identity independently of downloaded files}"
: "${ZTARC_OIDC_ISSUER:=https://token.actions.githubusercontent.com}"
dist="${1:-dist}"
for file in SHA256SUMS manifest.json install.sh; do
 cosign verify-blob "$dist/$file" --bundle "$dist/$file.sigstore.json" \
  --certificate-identity "$ZTARC_CERTIFICATE_IDENTITY" --certificate-oidc-issuer "$ZTARC_OIDC_ISSUER"
done
python3 "$(dirname "${BASH_SOURCE[0]}")/ztarc.py" verify --dist "$dist"
