# dbbackup

A simple CLI tool to **back up, restore, and test connections** to **MySQL** and **PostgreSQL** databases, with **Telegram** notifications.

- Backups are gzip-compressed by default (`.sql.gz`)
- Backups are written to `*.tmp` first and renamed only on success — no half-written files
- Passwords are passed via environment variables, so they never show up in the process list (`ps`)
- Configuration is read from a `.env` file
- Logs to stdout and (optionally) to a file
- Telegram notification after every backup/restore, whether it succeeds or fails

> Note: the tool's log and notification messages are in Indonesian (e.g. `backup sukses` = backup succeeded, `GAGAL` = failed).

---

## Requirements

| Requirement | Notes |
|---|---|
| Go 1.24+ | build only |
| `mysqldump` & `mysql` | for MySQL databases |
| `pg_dump` & `psql` | for PostgreSQL databases |

Make sure the database client tools are on your `PATH`:

```sh
which mysqldump mysql      # MySQL
which pg_dump psql         # PostgreSQL
```

Installing on macOS (Homebrew):

```sh
brew install mysql-client      # or: brew install mysql
brew install libpq && brew link --force libpq   # pg_dump & psql
```

---

## Build

```sh
go build -o dbbackup .
```

The result is a single `dbbackup` binary with no extra dependencies.

---

## Configuration (`.env`)

Copy the template and fill it in:

```sh
cp .env.example .env
chmod 600 .env
```

Example `.env`:

```sh
# Database
DB_TYPE=mysql             # postgres | mysql
DB_HOST=localhost
DB_PORT=                  # empty = 5432 (postgres) / 3306 (mysql)
DB_USER=root
DB_PASSWORD=secret
DB_NAME=my_database

# Output
BACKUP_DIR=./backups
LOG_FILE=./backup.log

# Telegram notifications (leave empty to disable)
TELEGRAM_BOT_TOKEN=123456789:ABCdefGhIJKlmNoPQRstuVWxyz
TELEGRAM_CHAT_ID=-1001234567890
```

**Where `.env` is loaded from** — searched in this order:
1. the current working directory
2. the directory containing the `dbbackup` binary (handy for cron)

**Precedence** (highest first):
1. command-line flags (e.g. `-db other_db`)
2. environment variables already set in the system
3. values from `.env`
4. built-in defaults

Supported `.env` syntax: `KEY=VALUE`, `#` comment lines, trailing comments (` # ...`), quoted values (`"..."` / `'...'`), and an optional `export` prefix.

> ⚠️ `.env` contains passwords and tokens. Never commit it — it is already listed in `.gitignore`.

---

## Usage

```
dbbackup <command> [flags]
```

| Command | Description |
|---|---|
| `test` | test the database connection |
| `backup` | create a backup file |
| `restore` | restore a database from a backup file |

Show all flags: `./dbbackup <command> -h`

### 1. Test connection

```sh
./dbbackup test
```

```
2026/11/22 20:24:50 [TEST] mysql/db_shop@localhost:3306
2026/11/22 20:24:50 [OK] koneksi OK: mysql/db_shop@localhost:3306
```

### 2. Backup

```sh
./dbbackup backup
```

```
2026/11/22 20:24:53 [BACKUP] mulai mysql/db_shop@localhost:3306
2026/11/22 20:24:53 [OK] backup sukses: mysql/db_shop@localhost:3306 -> backups/mysql_db_shop_20261122_202453.sql.gz (2278 bytes, 183ms)
```

File name format: `<type>_<database>_<YYYYMMDD_HHMMSS>.sql.gz`

More examples:

```sh
# back up a different database without editing .env
./dbbackup backup -db other_db

# write to a different folder
./dbbackup backup -out /mnt/backup/mysql

# no compression (plain .sql)
./dbbackup backup -compress=false
```

### 3. Restore

```sh
./dbbackup restore -file backups/mysql_db_shop_20261122_202453.sql.gz
```

Both `.sql` and `.sql.gz` files are supported (detected by file extension).

> ⚠️ **Restore overwrites data in the target database.** PostgreSQL backups are created with `--clean --if-exists`, so existing tables are dropped before being recreated. Double-check that `DB_NAME` / `-db` points to the right database.

The target database must already exist. To restore into a new database:

```sh
mysql -u root -p -e "CREATE DATABASE db_shop_copy"
./dbbackup restore -db db_shop_copy -file backups/mysql_db_shop_20261122_202453.sql.gz
```

### Flags

Common flags (all commands):

| Flag | Env | Default | Description |
|---|---|---|---|
| `-type` | `DB_TYPE` | — | `postgres` or `mysql` (required) |
| `-host` | `DB_HOST` | `localhost` | database host |
| `-port` | `DB_PORT` | 5432 / 3306 | database port |
| `-user` | `DB_USER` | — | database user (required) |
| `-password` | `DB_PASSWORD` | — | database password |
| `-db` | `DB_NAME` | — | database name (required) |
| `-log` | `LOG_FILE` | — | log file path (optional) |
| `-tg-token` | `TELEGRAM_BOT_TOKEN` | — | Telegram bot token |
| `-tg-chat` | `TELEGRAM_CHAT_ID` | — | Telegram chat ID |

`backup` only:

| Flag | Env | Default | Description |
|---|---|---|---|
| `-out` | `BACKUP_DIR` | `./backups` | output folder |
| `-compress` | — | `true` | gzip compression (`-compress=false` to disable) |

`restore` only:

| Flag | Description |
|---|---|
| `-file` | backup file `.sql` / `.sql.gz` (required) |

> Avoid passing `-password` on the command line on shared servers — it ends up in your shell history. Use `.env` instead.

---

## Telegram notifications

A notification is sent after every `backup` and `restore` (success or failure). The `test` command does not send notifications. If the token or chat ID is empty, notifications are skipped.

Example messages:

```
backup sukses: mysql/db_shop@localhost:3306 -> backups/mysql_db_shop_20261122_202453.sql.gz (2278 bytes, 183ms)
```

```
backup GAGAL (mysql/db_shop@localhost:3306): mysqldump gagal: exit status 2: Access denied for user 'root'@'localhost'
```

### Bot setup

1. Open [@BotFather](https://t.me/BotFather) in Telegram, send `/newbot`, and follow the steps.
2. Copy the token it gives you into `TELEGRAM_BOT_TOKEN`.
3. Get the chat ID:
   - **Private chat:** send any message to your bot.
   - **Group:** add the bot to the group, then send a message in the group.
4. Open this in a browser (replace `<TOKEN>`):
   ```
   https://api.telegram.org/bot<TOKEN>/getUpdates
   ```
   Find `"chat":{"id": ...}` and copy the number into `TELEGRAM_CHAT_ID`. Group chat IDs are usually negative (e.g. `-1001234567890`).
5. Test it by running `./dbbackup backup`.

If a notification fails, the backup still completes and a warning is logged:

```
[WARN] notifikasi telegram gagal: telegram status 401: Unauthorized
```

| Error | Cause |
|---|---|
| `401: Unauthorized` | wrong token |
| `400: Bad Request: chat not found` | wrong chat ID, or you never messaged the bot |
| `403: Forbidden: bot was kicked...` | the bot was removed from the group |

---

## Scheduled backups (cron)

Put the binary and `.env` in the same folder, e.g. `/opt/dbbackup/`:

```sh
/opt/dbbackup/
├── dbbackup
└── .env
```

Edit your crontab (`crontab -e`) — example daily backup at 02:00:

```cron
0 2 * * * cd /opt/dbbackup && ./dbbackup backup >> /opt/dbbackup/cron.log 2>&1
```

Tips:
- Use absolute paths for `BACKUP_DIR` and `LOG_FILE` in `.env`.
- Cron runs with a minimal `PATH`. If you see `executable file not found`, add a line at the top of your crontab, for example:
  ```cron
  PATH=/usr/local/bin:/usr/local/opt/mysql-client/bin:/usr/bin:/bin
  ```
- **Backing up multiple databases:** use one folder (with its own `.env`) per database, or override with a flag:
  ```cron
  0 2 * * * cd /opt/dbbackup && ./dbbackup backup -db db_shop
  5 2 * * * cd /opt/dbbackup && ./dbbackup backup -db db_blog
  ```
- **Deleting old backups** — the tool has no retention feature yet. Example: delete backups older than 7 days:
  ```cron
  30 2 * * * find /opt/dbbackup/backups -name '*.sql.gz' -mtime +7 -delete
  ```

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | backup / restore / connection test failed |
| `2` | invalid arguments or config (missing required value, unknown command, invalid `.env`) |

Useful in scripts or monitoring:

```sh
./dbbackup backup || echo "backup failed!"
```

---

## Project structure

```
.
├── main.go        # entry point, command & flag parsing
├── env.go         # .env loader
├── db.go          # DB config & mysqldump / pg_dump / psql / mysql commands
├── backup.go      # backup, restore, and connection test logic
├── notify.go      # logger & Telegram notifications
├── .env.example   # configuration template
└── .gitignore
```

---

## Troubleshooting

| Problem | Solution |
|---|---|
| `error: type harus 'postgres' atau 'mysql'` | set `DB_TYPE` in `.env` |
| `error: nama database (-db) wajib diisi` | set `DB_NAME` in `.env`, or `.env` was not found — check your working directory |
| `exec: "pg_dump": executable file not found` | install the PostgreSQL client / add it to `PATH` |
| `Access denied for user` | wrong user/password, or the user lacks privileges |
| `mysqldump: Couldn't execute 'SHOW ... ROUTINES'` | the user needs extra privileges; backups include `--routines --triggers` |
| `.env baris N: format harus KEY=VALUE` | a line in `.env` is missing the `=` sign |
