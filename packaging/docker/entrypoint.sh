#!/bin/sh
set -eu
umask 077
case "${1:-}" in
 site) shift; exec /usr/local/bin/ztarc-cli up site --config-file /etc/ztarc/site.json "$@";;
 client) shift; exec /usr/local/bin/ztarc-cli up client --attach --credentials-file /etc/ztarc/client.json "$@";;
 *) echo 'Choose site or client; mount the root-owned 0600 config read-only at /etc/ztarc.' >&2; exit 2;;
esac
