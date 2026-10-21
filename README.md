# Asternic Call Center Stats Lite

Queue reports for the Asterisk© PBX: answered and unanswered calls, call
distribution by hour and day, agent activity and a realtime view of the
queues. The data comes from the Asterisk `queue_log`, which is imported into
a MySQL/MariaDB database.

The application has three parts:

| Part | Directory | Image |
|---|---|---|
| web application (PHP 8.5, Apache) | `html/` | `ghcr.io/krzysztofhajdamowicz/asternic-web` |
| `parselogd`, the `queue_log` importer (Go, runs all the time) | `parselogd/` | `ghcr.io/krzysztofhajdamowicz/asternic-parselogd` |
| `parselog.php`, the original importer (PHP, run from cron) | `parselog/` | `ghcr.io/krzysztofhajdamowicz/asternic-parselog` |

Use either `parselogd` or `parselog.php`, not both. `parselogd` is the
recommended one: new calls appear in the reports within a second, it handles
log rotation and it imports every line exactly once.

## About this fork

This is a modified version of Asternic Call Center Stats Lite 1.8 by
Nicolás Gudiño ([asternic/asternic-stats-lite](https://github.com/asternic/asternic-stats-lite)),
maintained by Krzysztof Hajdamowicz since 2026. Changes made in 2026: PHP 8.5
support, a rewritten frontend (Chart.js 4, plain JavaScript, new stylesheet)
and `parselogd`, a Go replacement for `parselog.php`. The full list is in the
git history.

Copyright 2007–2026 Nicolás Gudiño; modifications copyright 2026 Krzysztof
Hajdamowicz. Distributed under the GNU General Public License version 3 or
later, see `LICENSE`.

## Security warning

The application has **no authentication** and shows call activity (phone
numbers, agents, queues) to anyone who can open it. The quick-start setup
listens on `127.0.0.1` only. Before making it reachable from the network, put
it behind a reverse proxy with authentication (HTTP Basic Auth, an SSO proxy,
Cloudflare Access or similar).

## Quick start (Docker Compose)

Requirements: Docker with the Compose plugin, on the host where Asterisk
writes `/var/log/asterisk/queue_log`.

```sh
git clone https://github.com/KrzysztofHajdamowicz/asternic-stats-lite.git
cd asternic-stats-lite
cp .env.example .env        # set DB_PASSWORD, and AMI_* for the realtime tab
docker compose up -d
```

Run `docker compose` from the repository checkout: the `db` service loads
`sql/qstats.sql` from it on the first start. If you copy only
`docker-compose.yaml` somewhere else, copy `sql/qstats.sql` next to it (same
relative path); without it compose stops with `bind source path does not exist`.

Open <http://127.0.0.1:8080/> (from another machine: `ssh -L 8080:127.0.0.1:8080 your-pbx`).

`docker-compose.yaml` starts three services:

- `db`: MariaDB (LTS). On the first start it loads `sql/qstats.sql`. Data is
  kept in the `db` volume.
- `parselogd`: follows `queue_log` from `/var/log/asterisk` (mounted read
  only) and writes new events to the database. On the first start it imports
  the whole current file.
- `web`: the reports, on port 8080.

Useful commands:

```sh
docker compose logs -f parselogd     # import progress and errors
docker compose pull && docker compose up -d   # update to the newest images
docker compose run --rm parselogd -purge      # delete all imported data
```

### Configuration

All settings are in `.env` (see `.env.example`):

| Variable | Default | Meaning |
|---|---|---|
| `DB_PASSWORD` | (required) | password of the `qstatslite` database user |
| `WEB_PORT` | `127.0.0.1:8080` | where the web interface listens; `8080` means all addresses |
| `AMI_HOST` | `host.docker.internal` (the Docker host) | Asterisk Manager Interface, used by the realtime tab only |
| `AMI_USER` | `qstats` | AMI user from `/etc/asterisk/manager.conf` |
| `AMI_PASSWORD` | empty | AMI password |
| `ASTERISK_LOG_DIR` | `/var/log/asterisk` | host directory containing `queue_log` |
| `CONVERT_LOCAL` | `true` | store `Local/101@context` agents as `Agent/101` |
| `TZ` | UTC | time zone of the reports, e.g. `Europe/Warsaw`; changing it on an existing database shifts new events against the old ones |
| `REPORT_LANGUAGE` | `en` | report language: `en`, `es`, `de`, `fr`, `ru`, `pt_BR` |

The containers themselves read these variables:

- web image: `dbhost`, `dbname`, `dbuser`, `dbpass`, `manager_host`,
  `manager_user`, `manager_pass`, `language` (the image replaces
  `html/config.php` with `docker/web/config.php`, which reads them; mounting
  your own `config.php` still works)
- `parselogd`: the same `db*` variables (or `PARSELOG_DSN`), `QUEUE_LOG`,
  `CONVERT_LOCAL`, `TZ` and a few more, see
  [parselogd/README.md](parselogd/README.md#configuration)
- `parselog` image: the same `db*` variables, `PARSELOG_SCHEDULE` (cron
  spec, default `*/5 * * * *`), `CONVERT_LOCAL` (default `true`),
  `QUEUE_LOG_DIR` (default `/var/log/asterisk`), `QUEUE_LOG_FILE` (default
  `queue_log`)

The AMI user needs `read = agent,call` (or `all`) in `manager.conf`. If AMI
listens on `127.0.0.1` only, the container cannot reach it; let it listen on
the Docker bridge address too, or leave the realtime tab unused.

### Moving from parselog.php

Stop the cron job (or the old `parselog` container), then start `parselogd`
against the same database. It continues from the newest event already in
`queue_stats`; see [parselogd/README.md](parselogd/README.md#moving-from-parselogphp).

## Container images

The images are built by GitHub Actions in this repository and published to
the GitHub Container Registry for `linux/amd64` and `linux/arm64`. Tags:
`latest` (master), `X.Y.Z`, `X.Y`, `X` (git tags `vX.Y.Z`) and `sha-<commit>`.

| Image | Dockerfile | Base | Workflow |
|---|---|---|---|
| `ghcr.io/krzysztofhajdamowicz/asternic-web` | `docker/web/Dockerfile` | `php:8.5-apache` | `images.yml` |
| `ghcr.io/krzysztofhajdamowicz/asternic-parselog` | `docker/parselog/Dockerfile` | `php:8.5-cli` | `images.yml` |
| `ghcr.io/krzysztofhajdamowicz/asternic-parselogd` | `parselogd/Dockerfile` | distroless static | `parselogd.yml` |

The PHP images are built from the repository root (see `.dockerignore`).
The web image also enables the real client IP behind Cloudflare
(`CF-Connecting-IP`) and some PHP hardening (`expose_php`, `file_uploads`
off, strict session mode).

**Fresh packages.** The PHP images are always built with `--pull --no-cache`
and `apt-get upgrade`, so every build gets the newest base image and
operating system packages and a rebuild picks up security fixes without a
code change. The images are rebuilt on every push to `master`, on version
tags and once a week (Monday morning UTC).

**Urgent rebuild.** When an important vulnerability is fixed upstream (for
example a PHP release), rebuild without waiting for the schedule:
Actions → the image's workflow → *Run workflow*, or

```sh
gh workflow run images.yml -R KrzysztofHajdamowicz/asternic-stats-lite      # web + parselog
gh workflow run parselogd.yml -R KrzysztofHajdamowicz/asternic-stats-lite   # parselogd
```

Then `docker compose pull && docker compose up -d` on the server.

**Supply chain.** Every image has a GitHub build provenance attestation and
an SBOM attestation, plus BuildKit provenance (`mode=max`) and SBOM stored in
the registry. The SBOM covers the operating system (Debian) packages, PHP, Go
modules and the vendored Chart.js and FPDF (described in `docker/sbom/`; CI
checks that these files match the vendored code). To verify:

```sh
gh attestation verify oci://ghcr.io/krzysztofhajdamowicz/asternic-web:latest \
  --owner KrzysztofHajdamowicz
gh attestation verify oci://ghcr.io/krzysztofhajdamowicz/asternic-web:latest \
  --owner KrzysztofHajdamowicz --predicate-type https://spdx.dev/Document/v2.3
docker buildx imagetools inspect ghcr.io/krzysztofhajdamowicz/asternic-web:latest \
  --format '{{ json .SBOM }}'
```

**Dependabot** opens weekly pull requests for the base images
(`docker/web`, `docker/parselog`, `parselogd`), Go modules and GitHub
Actions. Patch and minor updates are merged automatically once CI is
green, except bumps of the `php` base image, which need a review.

## Manual installation (without Docker)

Requirements: PHP 8.5 with `mysqli` and `gd`, a web server (Apache), MySQL or
MariaDB, Asterisk with queues. Works on a plain LAMP server, FreePBX and
Issabel.

1. Create the database, its tables and a user:

   ```sh
   mysql -u root -p < sql/qstats.sql
   mysql -u root -p -e "CREATE USER 'qstatsliteuser'@'localhost' IDENTIFIED BY 'somepassword'"
   mysql -u root -p -e "GRANT SELECT, INSERT, UPDATE, DELETE ON qstatslite.* TO 'qstatsliteuser'@'localhost'"
   ```

2. Copy `html/` to the web root, e.g. `/var/www/html/queue-stats`, and set
   the database and AMI credentials and the language (`$language`) in its
   `config.php`.

3. Import `queue_log`, either
   - with `parselogd` (build it with `go build` in `parselogd/`, or use the
     image), run as a service, see [parselogd/README.md](parselogd/README.md), or
   - with `parselog.php`: copy `parselog/` to e.g. `/usr/local/parseloglite`,
     set the paths and credentials in its `config.php` and run it from cron:

     ```
     */5 * * * * cd /usr/local/parseloglite && php -q parselog.php convertlocal
     ```

     `php parselog.php purge` deletes all imported data.

4. Open `http://your-server/queue-stats`.

## Frontend

The web pages need no build step and load nothing from the internet:

- `html/css/app.css` is the only stylesheet; colours are CSS custom properties
  and a dark palette is used automatically when the system is in dark mode
- charts are drawn by [Chart.js](https://www.chartjs.org) 4, vendored in
  `html/js/vendor/` (to upgrade, replace `chart.umd.min.js` with the file of
  the same name from the `chart.js` npm package); PHP code calls
  `chart_bar()` from `html/misc.php` and `html/js/charts.js` draws the result
- the other scripts in `html/js/` are small plain JavaScript files: report
  form (`index.js`), realtime polling (`realtime.js`), table sorting
  (`sortable.js`)

Any current browser works; Internet Explorer is not supported.

## Development

```sh
# parselogd
cd parselogd
go test ./...
docker run -d --rm -p 3306:3306 -e MARIADB_ROOT_PASSWORD=root mariadb:lts
PARSELOG_TEST_DSN='root:root@tcp(127.0.0.1:3306)/' go test ./...
golangci-lint run

# PHP syntax
for f in html/*.php html/lang/*.php parselog/*.php; do php -l "$f" >/dev/null || echo "$f"; done
```

To try the web application with data, start the quick-start stack with
`ASTERISK_LOG_DIR` pointing at a directory that contains a copy of
`parselogd/testdata/queue_log` and pick 21 October 2024 in the report form.

[AGENTS.md](AGENTS.md) is a short map of the repository for people and LLM
agents making changes.

## Troubleshooting

- Empty reports: check `docker compose logs parselogd`, then whether the
  date range in the form matches the data, and `TZ`.
- `permission denied` on `queue_log`: keep `user: "0:0"` and
  `cap_add: [DAC_READ_SEARCH]` for `parselogd`, or make the file readable.
- Realtime tab shows an error: check the AMI settings and that AMI is
  reachable from the container.
- `Can't read from a directory 'stdin'` in the db log (compose files from
  before this check): `sql/qstats.sql` was missing when the stack first
  started. Run from the checkout, then `docker compose down -v` (removes the
  half-initialised database volume) and start again.
