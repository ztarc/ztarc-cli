#!/bin/sh
# Run only in a disposable container: this intentionally removes /usr/local.
set -eu
mkdir -m 0755 /test-bin
cat > /test-bin/curl <<'CURL'
#!/bin/sh
set -eu
source=/fixtures/binary
dest=
while [ "$#" -gt 0 ]; do
 case "$1" in
  *licenses.tar.gz) source=/fixtures/licenses.tar.gz;;
  -o) shift; dest=$1;;
 esac
 shift
done
cp "$source" "$dest"
CURL
chmod 0755 /test-bin/curl
export PATH=/test-bin:$PATH
for scenario in missing-parent missing-bin existing; do
 rm -rf /usr/local
 case "$scenario" in
  missing-parent) :;;
  missing-bin) mkdir -m 0755 /usr/local;;
  existing) mkdir -m 0751 /usr/local; mkdir -m 0755 /usr/local/bin;;
 esac
 sh /fixtures/install.sh
 [ "$(stat -c '%a' /usr/local/bin)" = 755 ]
 if [ "$scenario" = existing ]; then
  [ "$(stat -c '%a' /usr/local)" = 751 ]
 else
  [ "$(stat -c '%a' /usr/local)" = 755 ]
 fi
 [ "$(stat -c '%u:%a' /run/ztarc-cli-install)" = 0:700 ]
 [ "$(setpriv --reuid=1000 --regid=1000 --clear-groups /usr/local/bin/ztarc-cli)" = fixture-cli-ok ]
done
# Keep rejecting unsafe existing parents before downloading or installing.
for scenario in writable symlink; do
 rm -rf /usr/local
 case "$scenario" in
  writable) mkdir -m 0777 /usr/local;;
  symlink) ln -s /tmp /usr/local;;
 esac
 if sh /fixtures/install.sh; then
  echo "Accepted unsafe $scenario destination" >&2
  exit 1
 fi
done
printf 'Fresh install, user execution, existing modes, private lock and unsafe destinations passed.\n'
