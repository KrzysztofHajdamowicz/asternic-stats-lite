# AGENTS.md

Asterisk queue reports. PHP 8.5 web app (`html/`) reads MariaDB tables filled
from Asterisk `queue_log` by `parselogd/` (Go, preferred) or `parselog/`
(PHP, legacy cron). Fork of Asternic Stats Lite 1.8. User docs: `README.md`.

## Map

| Path | What |
|---|---|
| `html/index.php` | report form (queues, agents, date range), posts to the reports |
| `html/answered.php`, `unanswered.php`, `distribution.php` | the three reports (SQL + tables + charts) |
| `html/realtime*.php`, `asmanager.php` | realtime tab via Asterisk AMI |
| `html/sesvars.php` | included by reports: takes the form POST, keeps it in `$_SESSION['QSTATS']` |
| `html/set_sesvar.php` | AJAX setter for realtime view options (allow-list) |
| `html/misc.php` | helpers: `h()` escape, `chart_bar()` chart JSON, `sql_in_list()`, `print_exports()` |
| `html/dblib.php` | mysqli wrapper (`$midb`); vendored, large, rarely needs changes |
| `html/export.php`, `fpdf.php`, `font/` | CSV/PDF export; FPDF 1.9 vendored, do not edit |
| `html/header.php`, `footer.php`, `menu.php` | page shell |
| `html/config.php` | DB/AMI credentials, `$language` (manual installs; images use `docker/web/config.php` with `getenv()`) |
| `html/lang/*.php` | UI strings (`$lang[...]`); add new keys to all files, `en.php` first |
| `html/css/app.css` | the only stylesheet; colours = CSS custom properties, dark mode via `prefers-color-scheme` |
| `html/js/` | `charts.js` (renders `<canvas data-chart>`), `index.js`, `realtime.js`, `sortable.js` |
| `html/js/vendor/chart.umd.min.js` | Chart.js 4, vendored, do not edit |
| `parselog/` | legacy importer `parselog.php` (+ own copy of `dblib.php`, `misc.php`) |
| `parselogd/` | Go importer; own `go.mod`, `README.md`, `Dockerfile`; `internal/{queuelog,tail,store,testdb}` |
| `parselogd/testdata/` | sample `queue_log` (21 Oct 2024) and expected `queue_stats.golden` |
| `sql/qstats.sql` | schema (creates DB `qstatslite`); tables `qname`, `qagent`, `qevent`, `queue_stats`, `parselog_state` |
| `docker/web/`, `docker/parselog/` | Dockerfiles + config of the PHP images (build context = repo root, `.dockerignore`) |
| `docker/sbom/` | SBOM manifests for vendored Chart.js/FPDF; CI fails if they do not match |
| `docker-compose.yaml`, `.env.example` | quick start: web + MariaDB + parselogd from GHCR |
| `.github/workflows/` | `images.yml` (web, parselog), `parselogd.yml`; `.github/actions/publish-image` (GHCR, SBOM, attestations); `dependabot.yml` |

Data flow: `queue_log` line → `queue_stats` row (`qname`/`qagent` are lookup
ids, `qevent` is a fixed event list) → report SQL in each `html/*.php`.

## Conventions

- No npm, bundler, framework or CDN. Plain PHP pages, vanilla JS, one CSS file.
- Escape all output with `h()`. Build SQL lists with `sql_in_list()` or
  `mysqli_real_escape_string`; never interpolate request data.
- Charts: call `chart_bar($id, $title, $labels, $series, $options)` in PHP;
  do not write Chart.js config inline.
- Go: standard library + `go-sql-driver/mysql` only; parameterised queries;
  `gofmt`, `golangci-lint` clean.
- New files start with a short header:
  `Copyright 2026 Krzysztof Hajdamowicz` / `Part of Asternic Call Center Stats Lite, GPL-3.0-or-later (see LICENSE).`
  Keep the original Nicolás Gudiño headers in existing files.
- Schema changes go to `sql/qstats.sql` and must be additive (`CREATE TABLE IF
  NOT EXISTS`); `parselog.php` and `parselogd` write the same tables.
- Upgrading vendored Chart.js or FPDF: update the matching file in `docker/sbom/`.
- New env var for an image: document it in `README.md` (and `.env.example`
  if the quick start uses it).
- Keep `README.md` in sync when behaviour, env vars or images change.
- Commits: imperative subject, body explains why. One topic per PR.

## Test

```sh
cd parselogd && go vet ./... && go test ./...      # unit tests
docker run -d --rm -p 3306:3306 -e MARIADB_ROOT_PASSWORD=root mariadb:lts
PARSELOG_TEST_DSN='root:root@tcp(127.0.0.1:3306)/' go test ./...   # + DB golden test
for f in html/*.php html/lang/*.php parselog/*.php; do php -l "$f" >/dev/null || echo "$f"; done
```

Web app end to end: `docker compose up -d` with `ASTERISK_LOG_DIR` pointing at
a dir holding `parselogd/testdata/queue_log`, open the reports for
2024-10-21, check PHP warnings in `docker compose logs web`. Test light and
dark mode and a narrow (phone) viewport after frontend changes. Docker Hub
may rate-limit; `mirror.gcr.io/library/<image>` works as a mirror.

## Do not touch

- Vendored code: `html/fpdf.php`, `html/font/`, `html/js/vendor/`, `html/dblib.php`.
- Meaning of existing columns in `queue_stats` and existing `qevent` ids;
  other tools read the same database.
- The "About this fork" and copyright sections of `README.md` and the footer.
- `LICENSE`, `INSTALL` (original upstream text).
