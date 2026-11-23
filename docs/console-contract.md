# Console handoff contract v1

The implementation owns CLI stages 0–3. Console/middleware must separately review
native authorization and implement distribution routes before production use.
`release/endpoints.json` inventories pinned upstream HTTP consumers; it is not
an ingress allowlist and its empty authorization evidence means server acceptance
is pending. Paths below are relative to `/api/v1`, except the reachability root.

| Consumer | Method/path | Request | Identity and allowed sensitive result |
| --- | --- | --- | --- |
| Login start | POST `/auth/device-web-auth/start` | Upstream `DeviceWebAuthStartRequest` | Native device authorization; return code/expiry/approval URL only |
| Login poll | GET `/auth/device-web-auth/poll/:code` | Code in native path | Bound authorization code; own native session only |
| Identity | GET `/user`, GET `/server-info` | None | Native user; own identity and server metadata |
| Org selection | GET `/user/:userId/orgs` | Native pagination where used | Requested user must equal session user; only permitted organizations |
| Org access | GET `/org/:orgId`, GET `/org/:orgId/user/:userId/check` | Org/user path | Membership, tenant active policy, and user scope |
| Device enrollment | PUT `/user/:userId/olm` | `{name}` | Authenticated owner; own newly provisioned ID/secret only |
| Device lookup | GET `/user/:userId/olm/:olmId` | Optional `orgId` query | Device ownership and selected organization |
| Device recovery | POST `/user/:userId/olm/recover` | `{platformFingerprint}` | Authenticated owner; recovery must not cross users/orgs |
| Client lookup | GET `/client/:clientId` | Numeric client path | Owner/org authorization; minimize returned fields |
| My device | GET `/my-device` | `olmId` query | Session user's own device and approval status |
| User policy | GET `/org/:orgId/my-client-settings` | Org path | Membership; unsupported on reviewed EE 1.26.0 pair |
| Machine policy | POST `/auth/olm/client-settings` | `{olmId,secret,orgId}` | Machine credential, no browser/user cookie; never a management proxy |
| Logout | POST `/auth/logout` | None | Current native session only |
| Reachability | GET or HEAD `/api/v1` | None | Preserve narrowly reviewed native reachability behavior; not a wildcard proxy |

Use the exact request/response structures in pinned upstream API source/types.
Native cookie name remains `p_session_token`; agent identifiers and backend
negotiation versions remain upstream-compatible. The CLI persists rotated native
session cookies for the matching account and does not import sealed console
browser cookies. Server approval cleanup stays a console responsibility.

User policy failure refuses connection, except explicit 404/501 when no policy
was previously cached. It logs that managed settings are unavailable and uses
local settings. Machine services similarly fail for auth/network/server errors;
a durable marker prevents fallback after an endpoint previously succeeded.
Do not describe this as managed-policy enforcement on EE 1.26.0.

Machines use `up client --credentials-file PATH` and
`service install client --credentials-file PATH`. JSON fields are precisely
`id`, `secret`, `endpoint`, `org`; user-token and unknown fields are rejected.
Files cannot be symlinks/group-readable, and service files must be root-owned
0600. File input conflicts with credential flags/environment. Container mounts
the same file read-only and calls attached mode directly. Enrollment, owner/review
metadata, direct grants, block/delete, quotas and tenant lifecycle are console A1
work, not a feature toggle in this repository.

Spoke remains site type `newt`, agent `cli`; config is `/etc/ztarc/site.json` or
existing `/etc/ztarc/connector.json` for migration. Blueprint/provisioning inputs
are rejected by the official CLI. HTTP and Newt WebSocket server enforcement is
still required; this client gate does not close backend bypasses.

Console distribution interfaces remain the design's `GET /api/cli-release`,
`/get-cli.sh`, and `/get-cli/:releaseId.sh`, backed by immutable artifacts under
`https://dl.ztarc.io/cli/<version>/` and a separate active CLI pin. A UI flow must
retain one validated snapshot. No latest fallback, automatic promotion, broad
`/api/v1` routing, or public Integration API access is provided here.
