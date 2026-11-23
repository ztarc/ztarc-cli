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
