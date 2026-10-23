# Telegram integration

Telegram is optional. It is enabled only when `TELEGRAM_WEBHOOK_SECRET` is non-empty; enabling it also requires at least one `TELEGRAM_ALLOWED_CHAT_IDS` value.

## Configure the webhook

Generate a dedicated secret of 32–256 URL-safe characters, set it in the service environment, and pass the same value as Telegram's `secret_token` when registering the webhook:

```bash
curl --fail-with-body --request POST \
  "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  --header 'Content-Type: application/json' \
  --data "{\"url\":\"https://padelo.example.com/telegram/webhook\",\"secret_token\":\"${TELEGRAM_WEBHOOK_SECRET}\"}"
```

`TELEGRAM_BOT_TOKEN` is used only to configure Telegram and is not needed by the Padelo process. Telegram receives replies through the webhook response body.

## Commands

| Input | Result |
| --- | --- |
| `/start` or `/help` | Show brief usage help |
| `leaderboard` | Show all current rankings |
| `history` | Show the ten most recent matches |
| `add Alex` | Add a player at 1200 Elo |
| `add Alex 1350` | Add a player at a chosen Elo |
| `adjust Alex +25` | Apply a manual Elo delta |
| `remove Alex` | Remove a player with no match history |
| `rollback` | Restore ratings and delete the latest match |

To record results, first send four players as two teams:

```text
Alex, Blair - Casey, Drew
```

Then send one or more score lines:

```text
6-3
4-6
10-8
```

The lineup is stored per chat until replaced. Every score line is a separate Elo-rated match and receives an ID derived from the Telegram update ID and line number.

## Request controls

The webhook verifies `X-Telegram-Bot-Api-Secret-Token` using constant-time comparison before decoding the body. Updates from chats outside the allowlist are acknowledged without execution. Each valid update ID is stored before its command runs; repeated delivery is acknowledged without running it again.

Because the update ID is claimed first, an internal failure while processing a valid update is not retried automatically. Resolve the underlying issue and apply the intended operation manually if necessary.

Changing or removing the webhook secret requires registering the corresponding secret with Telegram. To disable Telegram completely, unset `TELEGRAM_WEBHOOK_SECRET` and redeploy; the webhook route then returns 404.
