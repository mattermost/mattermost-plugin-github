#!/usr/bin/env bash
# Prints the next semver tag (vX.Y.Z or vX.Y.Z-rcN) for the given bump kind.
# Only tags reachable from <ref> are used as the base, so release branches bump
# their own line instead of whatever version is newest across the repository.
set -euo pipefail

usage() {
	echo "Usage: $0 <patch|minor|major|patch-rc|minor-rc|major-rc> [ref]" >&2
	exit 2
}

fail() {
	echo "Error: $*" >&2
	exit 1
}

kind=${1:-}
ref=${2:-HEAD}
case "$kind" in
patch | minor | major | patch-rc | minor-rc | major-rc) ;;
*) usage ;;
esac

git rev-parse -q --verify "$ref^{commit}" >/dev/null || fail "unknown ref '$ref'"

semver_tags() {
	git -c versionsort.suffix=-rc tag "$@" --sort=-v:refname |
		grep -E '^v[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$' || true
}

tag_exists() {
	git rev-parse -q --verify "refs/tags/$1" >/dev/null
}

# RC numbers are counted across all tags, not just reachable ones, to avoid
# colliding with an RC cut from another branch.
next_rc() {
	local last
	last=$(git tag -l "$1-rc*" | sed -nE "s/^${1//./\\.}-rc([0-9]+)$/\1/p" | sort -n | tail -n1)
	echo $((${last:-0} + 1))
}

reachable=$(semver_tags --merged "$ref")
latest_tag=${reachable%%$'\n'*}
latest_release=$(printf '%s\n' "$reachable" | grep -v -- '-rc' | head -n1 || true)
latest_release=${latest_release:-v0.0.0}

version=${latest_release#v}
major=${version%%.*}
minor=${version#*.}
minor=${minor%%.*}
patch=${version##*.}

case "$kind" in
patch | patch-rc) base="v$major.$minor.$((patch + 1))" ;;
minor | minor-rc) base="v$major.$((minor + 1)).0" ;;
major | major-rc) base="v$((major + 1)).0.0" ;;
esac

if [[ $kind == patch-rc && $latest_tag == *-rc* ]]; then
	base=${latest_tag%-rc*}
fi

next=$base
if [[ $kind == *-rc ]]; then
	tag_exists "$base" && fail "$base is already released; cannot cut a release candidate for it"
	next="$base-rc$(next_rc "$base")"
fi

tag_exists "$next" && fail "tag $next already exists"

echo "$next"
