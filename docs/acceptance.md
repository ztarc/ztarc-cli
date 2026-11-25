# Acceptance gates

Source/build, unit and local smoke evidence is recorded in `release/local-evidence.json`
after checks. It is development evidence, not a stable promotion decision.

The following remain mandatory external acceptance gates:

- Native Console adapter authorization: cross-user/org isolation, tenant expiry,
  password/SSO/MFA, pending approvals, cancel/expiry/retry/429, rotated sessions,
  organization changes, logout and approval cleanup.
- Linux amd64 and arm64 physical/VM dataplane: TUN/routes/DNS, suspend/reconnect,
  abrupt termination with watchdog restoration, reboot and service lifecycle.
- Container site and machine traffic, signal shutdown, read-only credential
  mounts, permissions and networking/capabilities for each advertised mode.
- Machine without grants denied; selected grants limited; revoke/Block cuts
  active access and reconnect; user logout independent; credential replacement
  leaves old identity blocked. Console A1 lifecycle is a separate workstream.
- Real standard systemd and Docker migration, interrupted cutover/reboot and
  tested rollback without duplicate site processes.
- Real GitHub OIDC signing, wrong-identity/signature rejection, registry digest
  verification and distribution failure handling. No local signature is claimed.
- Minimum 24-hour canary measuring actual resource access, with evidence URLs.

No platform or networking command is marked supported solely from cross-build.
Deferred stage B/C/D commands remain blocked. Production stable promotion,
console pin changes and host migrations require their own explicit operations.

### Security boundary follow-up (2026-11-25)

Stage A site tunnels force SSH off and ignore remote PAM/browser-gateway messages.
SSH/auth-daemon, updown scripts, pprof, blueprints, and provisioning inputs are
rejected before runtime initialization. Root site configuration must be a root-owned
private regular file. Site credential persistence belongs to CLI/service enrollment;
the embedded engine cannot create a fallback credential file.

Remote console endpoints require HTTPS with certificate verification; HTTP is
permitted only for `localhost`, `127.0.0.1`, and `::1` development endpoints.
`SKIP_TLS_VERIFY` no longer disables verification. Use system trust or configured
custom CAs. Credential-bearing HTTP redirects must stay on the original origin.

System commands and machine credentials do not load a user's profile. Detached
user runtime logs must stay below the initiating account's
`~/.config/ztarc-cli/logs/`; root machine logs use
`/var/log/ztarc-cli/client.log`. Elevated runtimes append through checked file
descriptors and do not rotate user-directory logs. Launch interactive user clients
without sudo so their log directories are created before elevation; a direct sudo
launch requires those user directories to exist already.

These boundaries apply to all packaged Stage A runtimes. They do not establish
server authorization or satisfy the external console/server acceptance gates.
