#!/usr/bin/env bash
set -euo pipefail
if [[ $EUID -ne 0 || $# -ne 1 ]]; then
  printf 'Usage: sudo bash deploy/setup-deploy.sh <deploy-public-key.pub>\n' >&2; exit 64
fi
public_key=$(realpath "$1")
scripts=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ssh-keygen -l -f "$public_key" >/dev/null
if [[ $(wc -l < "$public_key") -ne 1 ]] || ! grep -q '^ssh-ed25519 ' "$public_key"; then
  printf 'Expected one Ed25519 public key.\n' >&2; exit 64
fi
test -x /opt/aicalendar/bin/aicalendar
if id aicalendar-deploy >/dev/null 2>&1 || [[ -e /etc/sudoers.d/aicalendar-deploy ]]; then
  printf 'Deployment user already exists; rotate its key explicitly instead of rerunning setup.\n' >&2; exit 1
fi
useradd --system --home-dir /opt/aicalendar/deploy-user --shell /bin/bash aicalendar-deploy
# 用户不能改 home、authorized_keys、强制命令或 root 发布脚本。
install -d -m 0755 /opt/aicalendar/deploy-user /opt/aicalendar/deploy-user/.ssh
install -m 0755 "$scripts/deploy-ssh.sh" /opt/aicalendar/bin/deploy-ssh.sh
install -m 0755 "$scripts/deploy-release.sh" /opt/aicalendar/bin/deploy-release.sh
printf 'restrict,command="/opt/aicalendar/bin/deploy-ssh.sh" %s\n' "$(< "$public_key")" > /opt/aicalendar/deploy-user/.ssh/authorized_keys
chmod 0644 /opt/aicalendar/deploy-user/.ssh/authorized_keys
printf 'aicalendar-deploy ALL=(root) NOPASSWD: /opt/aicalendar/bin/deploy-release.sh\n' > /etc/sudoers.d/aicalendar-deploy
chmod 0440 /etc/sudoers.d/aicalendar-deploy
visudo -cf /etc/sudoers.d/aicalendar-deploy
printf 'Restricted deployment user configured.\n'
