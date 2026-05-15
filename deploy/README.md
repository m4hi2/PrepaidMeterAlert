# Deployment Guide

Production deployment targets a bare-metal Ubuntu 24.04 VPS.
The app consists of two processes managed by systemd, a PostgreSQL 18 database, and Grafana Alloy shipping telemetry to Grafana Cloud.

*There might be some mistakes since I did some experimentation while setting up, please open an issue if you find something not working*

## Architecture

```mermaid
flowchart TD
    dev["Local Machine\n(git tag + push)"]
    ci["GitHub Actions\nbuilds linux/amd64 binary"]
    dev -->|git push tag| ci

    subgraph vps["Ubuntu 24.04 VPS"]
        serve["meterbot-serve.service\nTelegram bot · always-on"]
        alert["meterbot-alert.service\nbalance checker · oneshot"]
        timer["meterbot-alert.timer\ndaily at 01:00 UTC"]
        alloy["alloy.service\nGrafana Alloy"]
        pg["postgresql-18"]

        timer -->|triggers| alert
        serve --> pg
        alert --> pg
        serve -->|OTLP gRPC| alloy
        alert -->|OTLP gRPC| alloy
    end

    ci -->|scp + ssh| serve

    subgraph grafana["Grafana Cloud"]
        tempo["Tempo\ntraces"]
        loki["Loki\nlogs"]
        mimir["Mimir\nmetrics"]
    end

    alloy -->|OTLP HTTP| tempo
    alloy -->|Loki push| loki
    alloy -->|remote_write| mimir
```

---

## One-Time VPS Setup

### 1. Create login user

Run as root immediately after provisioning — before locking root out:

```bash
useradd -m -s /bin/bash m4hi2
usermod -aG sudo m4hi2
mkdir -p /home/m4hi2/.ssh
cp /root/.ssh/authorized_keys /home/m4hi2/.ssh/authorized_keys
chmod 700 /home/m4hi2/.ssh
chmod 600 /home/m4hi2/.ssh/authorized_keys
chown -R m4hi2:m4hi2 /home/m4hi2/.ssh
```

**Open a new terminal and confirm `ssh m4hi2@<vps-ip>` works before continuing.**

### 2. Harden SSH and firewall

```bash
sudo sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sudo sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo sed -i 's/^#\?X11Forwarding.*/X11Forwarding no/' /etc/ssh/sshd_config
echo 'MaxAuthTries 3' | sudo tee -a /etc/ssh/sshd_config
sudo systemctl restart sshd

sudo apt install -y ufw fail2ban unattended-upgrades
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp comment 'SSH'
sudo ufw enable
sudo systemctl enable --now fail2ban
sudo dpkg-reconfigure -plow unattended-upgrades   # answer yes
```

### 3. Create deploy user (for GitHub Actions)

Generate a key pair on your **local machine**:

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/meterbot_deploy
# leave passphrase empty
```

- `~/.ssh/meterbot_deploy` → add as GitHub secret `VPS_SSH_KEY`
- `~/.ssh/meterbot_deploy.pub` → paste into the authorized_keys below

On the VPS:

```bash
sudo useradd -m -s /bin/bash deploy
sudo mkdir -p /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
sudo tee /home/deploy/.ssh/authorized_keys <<'EOF'
<paste meterbot_deploy.pub contents here>
EOF
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh

sudo tee /etc/sudoers.d/deploy-meterbot <<'EOF'
deploy ALL=(ALL) NOPASSWD: /usr/bin/mv /tmp/meterbot.new /opt/meterbot/meterbot, /usr/bin/chmod +x /opt/meterbot/meterbot, /opt/meterbot/run-migrate.sh, /usr/bin/systemctl restart meterbot-serve
EOF
```

### 4. Create app user and directories

```bash
sudo useradd -r -s /bin/false meterbot
sudo mkdir -p /opt/meterbot /etc/meterbot
sudo chown meterbot:meterbot /opt/meterbot
```

### 5. PostgreSQL 18

Ubuntu 24.04's default apt ships PG16. Install from the official PGDG repo:

```bash
sudo apt install -y curl ca-certificates
sudo install -d /usr/share/postgresql-common/pgdg
curl -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc --fail \
  https://www.postgresql.org/media/keys/ACCC4CF8.asc
sudo sh -c 'echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] \
  https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  > /etc/apt/sources.list.d/pgdg.list'
sudo apt update
sudo apt install -y postgresql-18 postgresql-client-18

sudo -u postgres createuser meterbot
sudo -u postgres createdb meterbot -O meterbot
sudo -u postgres psql -c "ALTER USER meterbot PASSWORD 'your-password';"
```

### 6. DESCO TLS Certificate

The DESCO prepaid portal (`prepaid.desco.org.bd`) uses a certificate not trusted by Ubuntu's default CA store. Install it manually:

```bash
# Export the certificate from your browser (visit the site → view certificate → export as PEM)
# or fetch it with openssl:
openssl s_client -connect prepaid.desco.org.bd:443 -showcerts </dev/null 2>/dev/null \
  | openssl x509 -outform PEM > desco.crt

sudo cp desco.crt /usr/local/share/ca-certificates/desco.crt
sudo update-ca-certificates
```

Verify Go picks it up (Go uses the system cert pool):

```bash
curl -v https://prepaid.desco.org.bd
# should complete without certificate errors
```

### 7. Environment file

Create `/etc/meterbot/.env` — never commit this file:

```ini
MA_LOG_LEVEL=info
MA_LOG_FORMAT=json
MA_DATABASE_URL=postgres://meterbot:your-password@localhost:5432/meterbot?sslmode=disable
MA_TELEGRAM_TOKEN=<from BotFather>
MA_TELEGRAM_RATE_LIMIT=30
MA_DESCO_BASE_PATH=https://prepaid.desco.org.bd
MA_DESCO_TIMEOUT=10s
MA_DESCO_RETRY=3
MA_DESCO_RETRY_DELAY=1s
MA_DESCO_RATE_LIMIT=5
MA_OTEL_ENABLED=true
MA_OTLP_ENDPOINT=localhost:4317
MA_SERVICE_NAME=meterbot
MA_ENVIRONMENT=production
MA_NESCO_BASE_PATH=https://customer.nesco.gov.bd
MA_NESCO_TIMEOUT=10s
MA_NESCO_RETRY=3
MA_NESCO_RETRY_DELAY=1s
MA_NESCO_RATE_LIMIT=2
```

```bash
sudo chmod 600 /etc/meterbot/.env
sudo chown meterbot:meterbot /etc/meterbot/.env
```

### 8. Grafana Alloy

```bash
sudo mkdir -p /etc/apt/keyrings
wget -q -O - https://apt.grafana.com/gpg.key \
  | gpg --dearmor \
  | sudo tee /etc/apt/keyrings/grafana.gpg > /dev/null
echo "deb [signed-by=/etc/apt/keyrings/grafana.gpg] https://apt.grafana.com stable main" \
  | sudo tee /etc/apt/sources.list.d/grafana.list
sudo apt update
sudo apt install -y alloy
```

Copy `alloy-config.alloy` from this directory to `/etc/alloy/config.alloy` and fill in the credentials from Grafana Cloud:

1. Create a free stack at [grafana.com](https://grafana.com)
2. **My Account → Stack → Details** — note the instance IDs for Tempo, Prometheus, and Loki
3. **Security → API Keys** — create a key with `MetricsPublisher`, `LogsPublisher`, `TracesPublisher` scopes
4. Fill all `<placeholder>` values in the config

```bash
sudo systemctl enable --now alloy
```

### 9. Systemd service files

Copy the three service files from this directory to `/etc/systemd/system/`:

```bash
sudo cp meterbot-serve.service meterbot-alert.service meterbot-alert.timer \
  /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now meterbot-serve meterbot-alert.timer
```

### 10. Migration wrapper script

Copy `run-migrate.sh` to the VPS and lock it down so only root can modify it:

```bash
sudo cp run-migrate.sh /opt/meterbot/run-migrate.sh
sudo chown root:root /opt/meterbot/run-migrate.sh
sudo chmod 755 /opt/meterbot/run-migrate.sh
```

---

## First Manual Deploy

```bash
# On your local machine
GOOS=linux GOARCH=amd64 go build -o meterbot .
scp meterbot m4hi2@<vps-ip>:/tmp/meterbot.new
ssh m4hi2@<vps-ip> \
  "sudo mv /tmp/meterbot.new /opt/meterbot/meterbot && \
   sudo chmod +x /opt/meterbot/meterbot && \
   sudo /opt/meterbot/run-migrate.sh && \
   sudo systemctl start meterbot-serve meterbot-alert.timer"
```

---

## Ongoing Deploys (GitHub Actions)

Deployments trigger on a new tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

The workflow (`.github/workflows/deploy.yml`) will:

1. Build a `linux/amd64` binary
2. Upload it to `/tmp/meterbot.new` on the VPS via scp
3. Move it into place, run migrations, restart the bot

### Required GitHub Secrets

| Secret | Value |
| --- | --- |
| `VPS_SSH_KEY` | Contents of `~/.ssh/meterbot_deploy` (private key) |
| `VPS_HOST` | VPS IP or hostname |

---

## Verification

```bash
sudo systemctl status meterbot-serve        # active (running)
sudo systemctl list-timers meterbot-alert   # shows next 01:00 UTC fire
sudo journalctl -u meterbot-serve -f        # live JSON logs
sudo systemctl status alloy                 # Alloy shipping telemetry
```

Grafana Cloud:

- **Explore → Logs** — filter `{app="meterbot"}` for live log stream
- **Explore → Metrics** — query `alerter_fetch_success_total`
- **Explore → Traces** — look for `alerter.run` spans after 01:00 UTC
