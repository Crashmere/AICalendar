#!/usr/bin/env bash
set -euo pipefail
umask 077
if [[ $EUID -ne 0 || $# -ne 4 ]]; then
  echo 'Usage: install.sh <linux-binary> <https-origin> <token-hash-file> <deploy-public-key>' >&2
  exit 64
fi
app=/opt/aicalendar
binary=$(realpath "$1")
origin=$2
hash_file=$(realpath "$3")
public_key=$(realpath "$4")
scripts=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
[[ $origin =~ ^https://[a-zA-Z0-9.:-]+$ ]] || { echo 'Invalid HTTPS origin' >&2; exit 64; }
[[ ! -e $app && ! -e /etc/systemd/system/aicalendar.service && ! -e /etc/nginx/app-locations/aicalendar.conf ]] || { echo 'Installation already exists' >&2; exit 1; }
if id aicalendar >/dev/null 2>&1 || id aicalendar-deploy >/dev/null 2>&1; then echo 'Account already exists' >&2; exit 1; fi
[[ -f /run/lock/ali-release-retention.lock && -x /opt/serverportal/bin/portal ]]
grep -Eq '^[0-9a-f]{64}$' "$hash_file"
ssh-keygen -l -f "$public_key" >/dev/null
if ss -H -ltn 'sport = :18086' | grep -q .; then echo 'Port 18086 is in use' >&2; exit 1; fi
useradd --system --home-dir "$app" --shell /usr/sbin/nologin aicalendar
install -d -m 0755 "$app" "$app/bin" "$app/config"
install -d -m 0700 -o aicalendar -g aicalendar "$app/data" "$app/backups"
install -m 0755 "$binary" "$app/bin/aicalendar"
install -m 0755 "$scripts/backup.sh" "$app/bin/backup.sh"
install -m 0640 -o root -g aicalendar "$hash_file" "$app/config/import-token.sha256"
sed "s|https://YOUR_SERVER|$origin|" "$scripts/aicalendar.env.example" > "$app/config/aicalendar.env"
chmod 0640 "$app/config/aicalendar.env"
chown root:aicalendar "$app/config/aicalendar.env"
install -m 0644 "$scripts/nginx-location.conf" "$scripts/aicalendar.service" "$scripts/aicalendar-backup.service" "$scripts/aicalendar-backup.timer" "$app/config/"
runuser -u aicalendar -- "$app/bin/aicalendar" init --db "$app/data/aicalendar.sqlite"
bash "$scripts/setup-deploy.sh" "$public_key"
systemctl link "$app/config/aicalendar.service" "$app/config/aicalendar-backup.service" "$app/config/aicalendar-backup.timer"
systemctl daemon-reload
systemctl enable --now aicalendar.service aicalendar-backup.timer
curl --fail --silent --retry 10 --retry-delay 1 --retry-connrefused http://127.0.0.1:18086/healthz
systemctl start aicalendar-backup.service
ln -s "$app/config/nginx-location.conf" /etc/nginx/app-locations/aicalendar.conf
if ! nginx -t; then rm /etc/nginx/app-locations/aicalendar.conf; echo 'Nginx validation failed; new link removed' >&2; exit 1; fi
systemctl reload nginx
echo 'AICalendar installed. Publish its portal declaration and synchronize documents next.'
