# Explicit host migration and rollback

Never create a new site to replace an existing connector. Site ID, credentials,
resources, tenant and quotas remain unchanged. Cutover may interrupt connections.

For a standard systemd unit that runs only
`/usr/local/bin/ztarc-connector --config-file /etc/ztarc/connector.json`, the
journaled tool is `scripts/migrate-systemd.py`. It rejects extra flags, unit hooks,
environment directives or nonstandard layouts; preserve those settings through
a reviewed host-specific runbook instead of silently dropping them. Inventory
other processes, containers, architecture, networking and credential sources
before using the tool. Never run another instance with the same site identity.

Verify the candidate signatures and compatibility first, install its binary
alongside the existing connector, retain the old binary, and prepare a root-only
executable health-check script. The check must prove the same site identity and
actual resource connectivity, DNS and reconnect; an online heartbeat is insufficient.

Run the tool as root with `plan`, then explicitly `apply`, supplying
`--sha256 <verified-binary-sha256> --health-check /root/check-ztarc-dataplane`.
`plan` validates the config without opening a tunnel. The tool preserves the unit,
config and binary inventory in root-only `/var/lib/ztarc-cli/migration/`, takes a
lock, stops the old service, confirms its MainPID is zero, replaces ExecStart,
restarts and runs the dataplane check. Existing `ztarc-connector.service` and
`/etc/ztarc/connector.json` names are retained.

After interruption use `resume` with the same inputs. After failure inspect the
journal and use `rollback`; no automatic switching loop runs. Rollback verifies
the retained old binary, stops CLI, restores the original unit and runs the
health check. Config is not modified; unexpected config drift requires manual
recovery. Retain backups through observation. CLI tooling recognizes migrated
units through `service status|logs|uninstall site --unit ztarc-connector`.

For Docker: pre-pull the verified digest, save the old deployment definition,
digest, restart policy, network, mounts and secret source. Disable any controller
that could restart the old instance, stop it, then start the CLI replacement with
the same site config and networking. Validate resource dataplane. Rollback stops
the CLI container before restoring the old definition. Do not put secret values
in command arguments. Kubernetes, NAS and other orchestrators need separate
runbooks and are not claimed supported by these scripts.
