#!/bin/sh
# Creates a GitHub release vX.Y.Z, which triggers the binary and image builds.
#
# Usage: scripts/release.sh [major|minor|patch] [version]
#   Without a version, bumps the highest vX.Y.Z tag on GitHub (default: minor).
#   The first release is v0.1.0.
set -eu

bump=${1:-minor}
version=${2:-}

latest=$(gh api 'repos/{owner}/{repo}/tags' --paginate -q '.[].name' \
	| grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1)

if [ -z "$version" ]; then
	if [ -z "$latest" ]; then
		version=v0.1.0
	else
		IFS=. read -r major minor patch <<EOF
${latest#v}
EOF
		case $bump in
		major) version=v$((major + 1)).0.0 ;;
		minor) version=v$major.$((minor + 1)).0 ;;
		patch) version=v$major.$minor.$((patch + 1)) ;;
		*)
			echo "unknown bump '$bump', use major, minor or patch" >&2
			exit 1
			;;
		esac
	fi
fi

if ! echo "$version" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "version must look like v1.2.3, got '$version'" >&2
	exit 1
fi

if [ -n "$latest" ]; then
	echo "Releasing $version (previous: $latest)"
	# Notes cover the PRs since the last real release, not since a test release
	gh release create "$version" --generate-notes --notes-start-tag "$latest"
else
	echo "Releasing $version (first release)"
	gh release create "$version" --generate-notes
fi
