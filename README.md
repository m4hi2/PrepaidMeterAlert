# PrepaidMeter Alert Bot

[![Deploy](https://github.com/m4hi2/PrepaidMeterAlert/actions/workflows/deploy.yml/badge.svg)](https://github.com/m4hi2/PrepaidMeterAlert/actions/workflows/deploy.yml)

![PrepaidMeter Alert Bot icon](icon/meteralert-icon.png)

A Telegram bot that monitors your Bangladesh prepaid electricity meter balance and alerts you when it drops below a threshold you set.

[Try it now → @BDPrepaidMeterBot](https://t.me/BDPrepaidMeterBot)

## What it does

Register one or more prepaid meters through the bot. Every day the bot fetches the current balance from the utility provider's API and sends you a Telegram message if the balance is running low — before you lose power unexpectedly.

### Supported providers

| Provider | Status |
| --- | --- |
| DESCO | ✅ Live |
| NESCO | ✅ Live |
| DPDC | Maybe (If a user contributes) |

## How it works

The project is a single Go binary with two runtime modes:

- **`serve`** — runs the Telegram bot, handles user commands, lets users register meters, set thresholds, and choose a notification mode (*single*: alert once per low-balance episode, or *daily*: alert every day until recharged).
- **`alert`** — a one-shot command that fetches balances for all registered meters and sends notifications. Intended to be triggered by a cron job or systemd timer once a day.
- **`migrate`** — applies database migrations.

User data and meter state are stored in PostgreSQL.

## Running locally

### Prerequisites

- Go 1.26+
- GNU Make
- PostgreSQL 18
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- Docker (I've provide dev infra)

### Setup

```bash
# Clone the repo
git clone https://github.com/m4hi2/PrepaidMeterAlert
cd PrepaidMeterAlert

# Copy and fill in the environment file
cp .env.example .env
# Edit .env — at minimum set MA_DATABASE_URL and MA_TELEGRAM_TOKEN
# Other default values are fine for development. If you want local telemetry,
# Then you can set MA_OTEL_ENABLED=true

# Source the env if you don't use automatic env loaders like `direnv`
source .env

# If you don't have postgres 18 installed, you can use the docker compose
make infra

# Run migrations
make migrate

# Start the bot
make serve
```

To trigger a one-off balance check and alert cycle:

```bash
go run . alert
```

### Environment variables

See [.env.example](.env.example) for all available options.

## Deployment

See [deploy/README.md](deploy/README.md) for a full guide to running this on a Ubuntu VPS with systemd.

## Contributing

Contributions are welcome! Some areas where help would be great:

- **New providers** — if you're on DPDC or another Bangladeshi utility, help add support for it. The `datasources.DataFetcher` interface is small and easy to implement.
- **Bug reports** — open an issue if the bot misbehaves or a provider's API changes.
- **Features** — weekly summary messages, multiple threshold levels, a web dashboard, etc.
- **[Check for open issues](https://github.com/m4hi2/PrepaidMeterAlert/issues)**

To contribute, fork the repo, make your changes on a branch, and open a pull request. Please open an issue first for larger features so we can discuss the approach before you write the code.

## License

[MIT](LICENSE)
