# Candidate distribution and trust

`make package` generates Linux amd64/arm64 binaries, immutable installer bytes,
`manifest.json`, `release-id`, provenance, checksums and complete source archive.
`release/manifest.schema.json` is schema v1; incompatible changes require a new
schema version. `release-id` hashes the exact manifest bytes and is not included
inside the manifest. Channel/pin selection belongs to console/deployment and is
not inferred from the product version suffix. Artifact URLs never use `latest`.

`python3 scripts/container.py` builds a multi-platform OCI archive without a
registry push, attaches its digest/archive checksum to the manifest, and removes
stale signatures. It uses the reviewed base digest in `release/container.json`.
The local builder must support both architectures natively or through QEMU;
the candidate workflow configures QEMU on its disposable runner. To build only
amd64 locally use `python3 scripts/container.py --platform linux/amd64`; the
manifest records only the platforms actually packaged. This is not arm64 runtime
evidence. The base and Dockerfile frontend are pinned; Alpine package repository contents may still change. Exact
candidate bytes, rather than rebuilds, must be promoted after acceptance.

Signing is Sigstore/Cosign keyless with GitHub OIDC. The manually triggered
`candidate.yml` workflow runs only for main/release-tag refs, uses the
`cli-release` environment, grants `id-token: write` only to the candidate job,
and uploads signed artifacts without creating a GitHub Release or pushing an
image. Protect release refs, workflow edits, and that environment in GitHub.

The signed objects are `SHA256SUMS`, `manifest.json`, and `install.sh`.
SHA256SUMS binds the final binaries, OCI archive, source archive and provenance.
Bundles include certificate/signature/transparency evidence. Verifiers must
configure the expected workflow identity/ref independently of downloaded files:

```sh
export ZTARC_CERTIFICATE_IDENTITY='https://github.com/ORG/REPO/.github/workflows/candidate.yml@refs/tags/v0.23.0-ztarc.1'
export ZTARC_OIDC_ISSUER='https://token.actions.githubusercontent.com'
scripts/verify-release.sh dist
```

Use the actual release repository/ref, never an unrestricted identity wildcard.
No local OIDC identity or fabricated production signature is provided. A missing
signature/identity is a release failure. `make verify` checks byte integrity only;
it does not claim signer verification.

Before publishing an image, copy the tested OCI archive with digest preservation,
verify the registry digest equals the manifest, and Cosign-sign that digest using
the protected release workflow. Record publication evidence separately; do not
change immutable manifest bytes to flip a channel. No image is advertised as
available merely because an OCI archive exists locally.

The installer embeds the release version and binary hashes, uses HTTPS, verifies
bytes before replacement, checks a host installation lock, refuses symlink
paths/running ZTARC binaries, retains `ztarc-cli.previous`, and does not touch
credentials or services. Verify its signature before execution. Downloading an
installer and checksum through the same HTTPS origin alone does not provide an
independent trust anchor. This is a manual installation mechanism, not the
future signed automatic updater with anti-replay and key rotation.

For container use, select the verified image digest, mount a root-owned `0600`
config read-only at `/etc/ztarc/client.json` or `/etc/ztarc/site.json`, and pass
`client` or `site` to the entrypoint. Machine tunnels need `/dev/net/tun` and
NET_ADMIN; choose host/bridge networking only after dataplane validation. Do not
use `--privileged` as a blanket default. Keep both packaging modes' Spoke
capabilities equal. No self-modification or credential-to-argv translation occurs.
