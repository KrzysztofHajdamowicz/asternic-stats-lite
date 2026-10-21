# syntax=docker/dockerfile:1

# Original PHP parselog, run from cron (the Go daemon in parselogd/ replaces it).
# Build from the repository root: docker build -f docker/parselog/Dockerfile .
#
# The floating php:8.5-cli tag and the apt-get upgrade below are on purpose:
# every build (CI uses --pull --no-cache) takes the newest PHP 8.5 patch
# release and the current Debian security updates.
FROM php:8.5-cli

RUN apt-get update \
	&& apt-get upgrade -y \
	&& apt-get install -y --no-install-recommends \
		cron \
		procps \
	&& docker-php-ext-install -j$(nproc) mysqli \
	&& rm -rf /var/lib/apt/lists/*

RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"

COPY parselog/ /usr/local/bin/parselog/
# Configuration from environment variables (dbhost, dbname, ...).
COPY docker/parselog/config.php /usr/local/bin/parselog/config.php
COPY --chmod=755 docker/parselog/entrypoint.sh /entrypoint.sh

ENV PARSELOG_SCHEDULE="*/5 * * * *" \
	CONVERT_LOCAL=true

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/sbin/cron", "-f", "-l", "2"]
