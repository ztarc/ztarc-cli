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


## Linux amd64 user-client lab follow-up — 2026-11-26

The original signed candidate at commit914a7975156af88b0937b58aa0aca0d5e8592a87
failed interactive login whenever device polling returned HTTP429. The CLI exited
before collecting the owner's completed device approval. Patch003 and the polling
/API overrides now retry429 with Retry-After and bounded backoff until code expiry;
non429 failures still stop. Full `make test` passed, including six new retry tests
(the optional Docker installer test was skipped).

An explicitly unsigned development binary `0.23.0-ztarc.1-dev.pollretry`, SHA256
`ab9246c30abb977aac08167b84d04539cd9e88464ab844f4f626e6e348a37bdd`, ran on the
owner's Linux amd64 VM192.168.57.250. Official owner-approved login succeeded for
demo@ztarc.io/demo-tenant. Client22/100.90.128.2 connected through the existing
signed-candidate Spoke192.168.57.200. At08:35:40UTC, HTTP by overlay IP/private
hostname and UDP returned exact proof markers. Spoke forwarding logs and target
peer192.168.57.200 proved tunnel traversal despite colocated client and target.
Normal `down` removed the overlay route/interface, restored LAN DNS, and left
HTTPS working; overlay access failed again as expected. Local accounts remained
user-owned0700/0600. Runtime was stopped and the local user session retained.

These results apply only to the development build, not to the signed candidate's
user login. Review/sign a new candidate and retest its exact bytes before stable.
No signed distribution bytes, stable pin, or default were modified. Owner skipped
24-hour observation and connector migration/rollback tests; neither is marked
passed. Other acceptance gates remain as listed above. Operational evidence lives
in the companion console repository's deploy/evidence/cli-stage-a-client-20261126.json.

## Signed retry-fix candidate

Version `0.23.0-ztarc.2` contains the bounded device-approval HTTP 429 retry fix.
The candidate must be built and signed from its clean commit by `candidate.yml`;
the earlier development-binary smoke does not substitute for testing these bytes.
