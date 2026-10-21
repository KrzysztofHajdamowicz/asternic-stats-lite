#!/bin/sh
set -e

# cron starts jobs with an empty environment; pam_env loads /etc/environment,
# which is how parselog gets dbhost, dbname, ... from the container.
env >> /etc/environment

case "$(echo "$CONVERT_LOCAL" | tr '[:upper:]' '[:lower:]')" in
	1|t|true|yes|on) arg=convertlocal ;;
	*) arg= ;;
esac

cat > /etc/cron.d/parselog <<CRON
$PARSELOG_SCHEDULE root (echo parselog.php $arg; /usr/local/bin/php -f /usr/local/bin/parselog/parselog.php $arg) >/proc/1/fd/1 2>/proc/1/fd/2

CRON
chmod 0644 /etc/cron.d/parselog

exec "$@"
