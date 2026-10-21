# syntax=docker/dockerfile:1

# Web interface of Asternic Call Center Stats Lite (PHP 8.5 + Apache).
# Build from the repository root: docker build -f docker/web/Dockerfile .
#
# The floating php:8.5-apache tag and the apt-get upgrade below are on
# purpose: every build (CI uses --pull --no-cache) takes the newest PHP 8.5
# patch release and the current Debian security updates.
FROM php:8.5-apache

RUN apt-get update \
	&& apt-get upgrade -y \
	&& apt-get install -y --no-install-recommends \
		libfreetype-dev \
		libjpeg62-turbo-dev \
		libpng-dev \
	&& docker-php-ext-configure gd --with-freetype --with-jpeg \
	&& docker-php-ext-install -j$(nproc) gd mysqli \
	&& rm -rf /var/lib/apt/lists/*

# Production defaults: errors go to the log (docker logs), not into the HTML.
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"
COPY docker/web/php.ini "$PHP_INI_DIR/conf.d/zz-asternic.ini"

# Real client IP behind Cloudflare (cloudflared tunnel on the compose network):
# mod_remoteip rewrites the client address (REMOTE_ADDR in PHP) and the
# access log uses %a instead of %h. See remoteip.conf for details.
COPY docker/web/remoteip.conf /etc/apache2/conf-available/remoteip.conf
RUN a2enmod remoteip && a2enconf remoteip

COPY html/ /var/www/html/
# Configuration from environment variables (dbhost, dbname, ...). A config.php
# mounted over it still works.
COPY docker/web/config.php /var/www/html/config.php
# Package manifests of the vendored libraries (Chart.js, FPDF) so that the
# SBOM lists them; html/ itself has no package manager files.
COPY docker/sbom/ /usr/local/share/asternic/sbom/

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s \
	CMD curl -fsS -o /dev/null http://localhost/ || exit 1
