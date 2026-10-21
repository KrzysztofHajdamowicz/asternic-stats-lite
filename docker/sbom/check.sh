#!/bin/sh
# Fails when the manifests in docker/sbom/ no longer match the vendored files,
# so the SBOM cannot silently report an old Chart.js or FPDF version.
set -eu
cd "$(dirname "$0")/../.."

chartjs=$(sed -n 's/^ \* Chart\.js v\([0-9.]*\)$/\1/p' html/js/vendor/chart.umd.min.js | head -n1)
fpdf=$(sed -n "s/^const VERSION = '\([0-9.]*\)';/\1/p" html/fpdf.php | head -n1)

want_chartjs=$(sed -n 's/^  "version": "\(.*\)",$/\1/p' docker/sbom/chart.js/package.json)
want_fpdf=$(sed -n 's/^            "version": "\(.*\)",$/\1/p' docker/sbom/composer/installed.json)

status=0
if [ "$chartjs" != "$want_chartjs" ]; then
	echo "Chart.js is $chartjs, docker/sbom/chart.js/package.json says $want_chartjs" >&2
	status=1
fi
# FPDF calls 1.9 what Packagist calls 1.9.0.
if [ "$fpdf" != "$want_fpdf" ] && [ "$fpdf.0" != "$want_fpdf" ]; then
	echo "FPDF is $fpdf, docker/sbom/composer/installed.json says $want_fpdf" >&2
	status=1
fi
[ $status -eq 0 ] && echo "Vendored library manifests up to date (Chart.js $chartjs, FPDF $fpdf)"
exit $status
