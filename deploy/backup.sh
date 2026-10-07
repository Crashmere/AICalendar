#!/usr/bin/env bash
set -euo pipefail
umask 077
aicalendar_binary="${AICALENDAR_BINARY:-/opt/aicalendar/bin/aicalendar}"
aicalendar_database="${AICALENDAR_DB:-/opt/aicalendar/data/aicalendar.sqlite}"
aicalendar_backups="${AICALENDAR_BACKUP_DIR:-/opt/aicalendar/backups}"
mkdir -p "$aicalendar_backups"
backup_path="$aicalendar_backups/daily-$(date -u +%Y%m%dT%H%M%S)-$$.sqlite"
"$aicalendar_binary" backup --db "$aicalendar_database" --out "$backup_path"

# 只有新备份成功并通过完整性检查后才轮换；手工/升级/恢复前备份不参与。
mapfile -t daily_backups < <(find "$aicalendar_backups" -maxdepth 1 -type f -name 'daily-*.sqlite' -printf '%f\n' | LC_ALL=C sort -r)
for ((index=14; index<${#daily_backups[@]}; index++)); do
  rm -- "$aicalendar_backups/${daily_backups[index]}"
done
printf 'Backup created: %s\n' "$backup_path"
