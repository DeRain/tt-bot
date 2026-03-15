#!/usr/bin/env bash
set -e
QBT_URL="${QBITTORRENT_URL:-http://localhost:18080}"
echo "Waiting for qBittorrent at $QBT_URL..."
for i in $(seq 1 30); do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$QBT_URL/api/v2/app/version" 2>/dev/null || echo "000")
  if [ "$status" = "200" ] || [ "$status" = "401" ] || [ "$status" = "403" ]; then
    echo "qBittorrent is ready"
    exit 0
  fi
  sleep 2
done
echo "qBittorrent did not start in time"
exit 1
