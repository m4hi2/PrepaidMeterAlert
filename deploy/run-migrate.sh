#!/bin/bash
set -euo pipefail
set -a
source /etc/meterbot/.env
set +a
exec /opt/meterbot/meterbot migrate up
