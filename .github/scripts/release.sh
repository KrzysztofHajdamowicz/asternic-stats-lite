#!/usr/bin/env bash
# Creates or updates the GitHub release for a version tag (run by
# .github/workflows/release.yml). Waits until images.yml and parselogd.yml
# have pushed the tag's images, writes CycloneDX SBOMs of every image and
# platform plus one of the source tree, and publishes them with release notes.
#
# Needs: TAG (v1.2.3), GITHUB_REPOSITORY, GH_TOKEN, SYFT (syft binary),
# docker logged in to ghcr.io. Optional: SOURCE_DIR (checkout of the tag,
# default .), WAIT_MINUTES (default 60), DRY_RUN=1
# (write dist/ and notes.md, do not touch the release).
set -euo pipefail

: "${TAG:?}" "${GITHUB_REPOSITORY:?}" "${SYFT:=syft}"
[[ $TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]] || { echo "not a version tag: $TAG" >&2; exit 1; }
version=${TAG#v}
owner=${GITHUB_REPOSITORY%%/*}
registry=ghcr.io/${owner,,}
images=(asternic-web asternic-parselog asternic-parselogd)
platforms=(linux/amd64 linux/arm64)

rm -rf dist && mkdir dist

declare -A digest
deadline=$(( $(date +%s) + ${WAIT_MINUTES:-60} * 60 ))
for image in "${images[@]}"; do
	ref=$registry/$image:$version
	# Plain output: --format would also download every config and attestation blob.
	until out=$(docker buildx imagetools inspect "$ref" 2>&1) &&
		digest[$image]=$(awk '$1 == "Digest:" { print $2; exit }' <<<"$out") && [[ -n ${digest[$image]} ]]; do
		if (( $(date +%s) > deadline )); then
			echo "$ref did not appear; did images.yml and parselogd.yml succeed for $TAG?" >&2
			exit 1
		fi
		# Not pushed yet, or a registry error: show which, then retry.
		echo "waiting for $ref: $(tail -n1 <<<"$out")"
		sleep 30
	done
	echo "$ref = ${digest[$image]}"

	for platform in "${platforms[@]}"; do
		out=dist/SBOM-$image-$version-${platform#linux/}.cyclonedx.json
		"$SYFT" scan "registry:$registry/$image@${digest[$image]}" --platform "$platform" \
			--source-name "$image" --source-version "$version" -q -o "cyclonedx-json=$out"
	done
done

# Source tree: Go modules, GitHub Actions, and the vendored Chart.js and FPDF
# described in docker/sbom/ (package.json and installed.json manifests are
# only picked up by the "installed package" catalogers).
"$SYFT" scan "dir:${SOURCE_DIR:-.}" --source-name asternic-stats-lite --source-version "$version" \
	--select-catalogers +javascript-package-cataloger,+php-composer-installed-cataloger \
	-q -o "cyclonedx-json=dist/SBOM-source-$version.cyclonedx.json"

changes=$(gh api "repos/$GITHUB_REPOSITORY/releases/generate-notes" -f tag_name="$TAG" --jq .body)

{
	echo "## Container images"
	echo
	echo '```sh'
	for image in "${images[@]}"; do
		echo "docker pull $registry/$image:$version"
	done
	echo '```'
	echo
	echo "| Image | Digest (multi-arch index, linux/amd64 + linux/arm64) |"
	echo "|---|---|"
	for image in "${images[@]}"; do
		echo "| \`$registry/$image:$version\` | \`${digest[$image]}\` |"
	done
	echo
	echo "Verify the build provenance and the SBOM attestation:"
	echo
	echo '```sh'
	echo "gh attestation verify oci://$registry/asternic-web:$version --owner $owner"
	echo "gh attestation verify oci://$registry/asternic-web:$version --owner $owner \\"
	echo "  --predicate-type https://cyclonedx.org/bom"
	echo '```'
	echo
	echo "## SBOM"
	echo
	echo "CycloneDX files attached below: \`SBOM-<image>-$version-<arch>.cyclonedx.json\` for every"
	echo "image and platform (Debian packages, PHP, Go modules, vendored Chart.js and FPDF),"
	echo "and \`SBOM-source-$version.cyclonedx.json\` for the source tree."
	echo
	echo "$changes"
} > notes.md

if [[ ${DRY_RUN:-} == 1 ]]; then
	cat notes.md
	ls -l dist
	exit 0
fi

prerelease=()
[[ $version == *-* ]] && prerelease=(--prerelease)

if gh release view "$TAG" >/dev/null 2>&1; then
	gh release edit "$TAG" --notes-file notes.md
	gh release upload "$TAG" dist/* --clobber
else
	gh release create "$TAG" --verify-tag --title "$TAG" --notes-file notes.md "${prerelease[@]}" dist/*
fi
