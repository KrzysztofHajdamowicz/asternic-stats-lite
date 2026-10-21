# parselogd

`parselogd` imports the Asterisk `queue_log` into the Asternic Call Center
Stats Lite database. It replaces `parselog/parselog.php` and writes to the
same tables (`qname`, `qagent`, `queue_stats`), so the web application does
not need any change.

What is different from `parselog.php` run from cron:

- It runs all the time and follows the file like `tail -F`; new calls show up
  in the reports within a second instead of every 5 minutes.
- It survives log rotation (`create` and `copytruncate`), including rotation
  while it is not running: on start it finishes the rotated file
  (`queue_log.1`, `queue_log-20241021`, ...) before reading the new one.
- The read position is saved in the table `parselog_state`, in the same
  transaction as the events, so every line is imported exactly once. The
  table is part of `sql/qstats.sql`; on an existing installation parselogd
  creates it on first start if the database user may create tables,
  otherwise run that `CREATE TABLE` statement from `sql/qstats.sql` once.
- All queries are parameterised. Queue and agent names with `'` or `%` are
  stored correctly; `parselog.php` lost those events.
- Values longer than the `varchar(40)` columns are cut instead of making
  MySQL in strict mode reject the row.

## Configuration

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `-queue-log` | `QUEUE_LOG` | `/var/log/asterisk/queue_log` | file to follow |
| `-dsn` | `PARSELOG_DSN` | built from `dbhost`, `dbuser`, `dbpass`, `dbname` | [MySQL DSN](https://github.com/go-sql-driver/mysql#dsn-data-source-name), e.g. `user:pass@tcp(db:3306)/qstatslite` |
| `-convert-local` | `CONVERT_LOCAL` | `false` | store `Local/101@context` agents as `Agent/101` (same as `parselog.php convertlocal`) |
| `-poll-interval` | `POLL_INTERVAL` | `1s` | how often to look for new lines |
| `-rotate-grace` | `ROTATE_GRACE` | `5s` | how long a rotated file is still read for late writes |
| `-verbose` | `VERBOSE` | `false` | log every batch |
| `-once` | | | import what is in the file and exit (cron style) |
| `-purge` | | | delete all imported data and the saved position |

`dbhost`, `dbuser`, `dbpass` and `dbname` are the variables the Docker Compose
setup already uses. `dbhost` may include a port (`db:3307`).

**Time zone.** `queue_stats.datetime` is written in local time, like the PHP
version did. Set `TZ` (e.g. `TZ=Europe/Warsaw`) to the zone the reports
should use; the image contains the time zone database. Without `TZ` the
container uses UTC.

## Docker

The image is published to `ghcr.io/krzysztofhajdamowicz/asternic-parselogd`
(`latest` follows `master`, `vX.Y.Z` tags give `X.Y.Z`, `X.Y`, `X`) for
`linux/amd64` and `linux/arm64`.

```yaml
services:
  parselog:
    image: ghcr.io/krzysztofhajdamowicz/asternic-parselogd:latest
    restart: always
    environment:
      TZ: Europe/Warsaw
      CONVERT_LOCAL: "true"
    env_file: .env          # dbhost, dbuser, dbpass, dbname
    volumes:
      # Mount the directory, not the file, or rotation is not visible.
      - /var/log/asterisk:/var/log/asterisk:ro
    # The image runs as an unprivileged user. If queue_log is not readable
    # by everyone, run as root with only the right to read any file:
    user: "0:0"
    cap_drop: [ALL]
    cap_add: [DAC_READ_SEARCH]
    read_only: true
```

## Moving from parselog.php

Stop the cron job and start `parselogd`. On the first start there is no saved
position yet, so it skips events older than the newest one in `queue_stats`
(minus 10 seconds, as `parselog.php` did) and continues from there.
Duplicates are ignored by the unique key of `queue_stats` either way.

## Development

```sh
go test ./...                       # unit tests
docker run -d --rm -p 3306:3306 -e MARIADB_ROOT_PASSWORD=root mariadb:11.4
PARSELOG_TEST_DSN='root:root@tcp(127.0.0.1:3306)/' go test ./...   # with database tests
golangci-lint run
```

The database tests load `../sql/qstats.sql` into a throwaway database and
compare the import of `testdata/queue_log` with `testdata/queue_stats.golden`,
which was produced by `parselog.php` (with the `'` and `%` rows corrected).
