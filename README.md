# ZTARC CLI

ZTARC distribution of Pangolin CLI 0.23.0, pinned to
`bb03d2c45ce79213c4c9b206bbd87e9846be5a9a`. Product version:
`0.23.0-ztarc.2`; Newt 1.20.1; Olm 1.10.2; Go 1.26.10.

This repository implements development stages 0–3. Artifacts are candidates:
Linux dataplane, reboot, container and 24-hour canary acceptance remain required.
No release is published, console integration enabled, or existing host migrated
by building this repository. See [design](docs/design.md) and
[acceptance status](docs/acceptance.md).

## Build

Requires Git, Python 3.12+, and Go capable of selecting toolchain `go1.26.10`.
The first build downloads the toolchain/modules using Go checksum verification.
Docker Buildx is needed only for OCI packaging; Cosign 3.1.4 for release signing.

```sh
git submodule update --init --recursive
make test
make package
python3 tests/integration/smoke.py
python3 scripts/container.py
```

For a new writable Git checkout containing an existing `upstream` clone, run
`scripts/bootstrap.sh` once to register the gitlink. It does not create a commit
or publish anything. Environment-owned read-only `.git` placeholders must be
replaced by a normal checkout outside that environment. Builds without a committed HEAD
record `ztarcCommit: uncommitted`; release signing requires a clean, committed checkout.

Build outputs are under `dist/`. `build/src/` is disposable, contains patched
source and vendored dependencies, and must never be edited as the permanent
source of a change. Behavioral patches live under `brand/patches`, owned files
under `brand/overrides`, and count-checked branding rules under
`brand/replacements`. Dependency namespace transforms are explicit in release
metadata. `upstream/` must remain clean.

The source archive includes pristine upstream, transformations, metadata, and
`staged-source/` with patched source and all vendored module source/licenses.
For an offline source build with the pinned Go toolchain already installed:

```sh
cd staged-source
GOTOOLCHAIN=go1.26.10 CGO_ENABLED=0 go build -mod=vendor -buildvcs=false -trimpath -o ztarc-cli .
```

That plain source build displays a development upstream version. Use the build
pipeline to stamp release metadata and provenance.

## Usage

After verifying and installing the selected immutable artifact:

```sh
ztarc-cli version --json
ztarc-cli login                          # ordinary user; default ZTARC Console
ztarc-cli login https://staging.example  # explicit server remains supported
ztarc-cli up client                      # elevation for networking only
ztarc-cli status
ztarc-cli down
```

Set `ZTARC_CONSOLE_URL` to change the default console without rebuilding:

```sh
export ZTARC_CONSOLE_URL=https://console.example.com
ztarc-cli login
```

An explicit login URL or the saved host when logging back into the same account
takes precedence. If the variable is unset or blank, the default remains
`https://console.ztarc.io`. Hosts without a scheme use HTTPS.

The native API adapter in ZTARC Console must be implemented before production
login acceptance. CLI accounts live in `~/.config/ztarc-cli`; no desktop account,
token, or socket is imported. Deferred commands are labelled in help and reject
execution before auth/network mutation.

For machine mode, securely provision `/etc/ztarc/client.json` with this shape:

```json
{"id":"<olm-id>","secret":"<olm-secret>","endpoint":"https://console.ztarc.io","org":"<org-id>"}
```

Use a root-owned `0700` directory and root-owned `0600` regular file. Never put
real credentials into shell history, argv, source control, or an image layer.

```sh
sudo ztarc-cli service install client --credentials-file /etc/ztarc/client.json
sudo ztarc-cli service status client
sudo ztarc-cli service logs client -f
```

For Spoke mode, provision a root-only Newt JSON config at `/etc/ztarc/site.json`
with `id`, `secret`, and `endpoint`, plus reviewed Newt operational settings.

```sh
sudo ztarc-cli config validate-site /etc/ztarc/site.json
sudo ztarc-cli service install site --config-file /etc/ztarc/site.json
```

Installation starts the explicitly requested service. Installing the binary
alone never enrolls, creates a device, or changes any service. Service install
refuses conflicting units; uninstall does not erase credential files.

See [distribution and signature verification](docs/distribution.md),
[console contract](docs/console-contract.md), and [host migration](docs/migration.md).
