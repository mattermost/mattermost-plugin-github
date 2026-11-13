#!/usr/bin/env bash
# Tests build/next-version.sh against throwaway local repos. Fixtures have no
# remotes, so nothing here can push.
set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/next-version.sh"

while IFS= read -r var; do unset "$var"; done < <(env | sed -n 's/^\(GIT_[A-Za-z0-9_]*\)=.*/\1/p')
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

TMP_ROOT=$(mktemp -d)
trap 'rm -rf "$TMP_ROOT"' EXIT

failures=0

new_repo() {
	local dir
	dir=$(mktemp -d "$TMP_ROOT/repo.XXXXXX")
	cd "$dir"
	case "$PWD" in
	"$TMP_ROOT"/* | /private"$TMP_ROOT"/*) ;;
	*)
		echo "Refusing to run outside $TMP_ROOT (in $PWD)" >&2
		exit 1
		;;
	esac
	git init -q -b master
}

# Unique messages keep empty commits on different branches from hashing identically.
commit_count=0
commit() {
	commit_count=$((commit_count + 1))
	git commit -q --allow-empty -m "commit $commit_count"
}
tag() { for t in "$@"; do git tag -a -m "$t" "$t"; done; }

# Mirrors mattermost-plugin-jira: each final is tagged on the same commit as its last RC.
jira_fixture() {
	new_repo
	commit; tag v4.8.0
	git branch release-4.8
	commit; tag v5.0.0-rc1
	commit; tag v5.0.0-rc2 v5.0.0
	commit; tag v5.1.0-rc1 v5.1.0
	git checkout -q release-4.8
	commit; tag v4.8.2-rc1 v4.8.2
	commit; tag v4.8.3-rc1 v4.8.3
	git checkout -q master
}

expect() {
	local ref=$1 kind=$2 want=$3 got
	if got=$("$SCRIPT" "$kind" "$ref" 2>&1) && [ "$got" = "$want" ]; then
		echo "ok   $ref $kind -> $got"
	else
		echo "FAIL $ref $kind: want $want, got '$got'"
		failures=$((failures + 1))
	fi
}

expect_error() {
	local ref=$1 kind=$2 got
	if got=$("$SCRIPT" "$kind" "$ref" 2>&1); then
		echo "FAIL $ref $kind: want error, got '$got'"
		failures=$((failures + 1))
	else
		echo "ok   $ref $kind -> error: $got"
	fi
}

echo "# Jira-like history, master"
jira_fixture
expect master patch v5.1.1
expect master minor v5.2.0
expect master major v6.0.0
expect master patch-rc v5.1.1-rc1
expect master minor-rc v5.2.0-rc1
expect master major-rc v6.0.0-rc1
expect HEAD patch-rc v5.1.1-rc1

echo "# Jira-like history, release-4.8"
expect release-4.8 patch v4.8.4
expect release-4.8 minor v4.9.0
expect_error release-4.8 major
expect release-4.8 patch-rc v4.8.4-rc1
expect release-4.8 minor-rc v4.9.0-rc1
expect_error release-4.8 major-rc

echo "# Unfinished RC on master"
jira_fixture
commit; tag v5.2.0-rc1
expect master patch-rc v5.2.0-rc2
expect master minor-rc v5.2.0-rc2
expect master minor v5.2.0
expect master patch v5.1.1
expect master major-rc v6.0.0-rc1
tag v5.2.0-rc2
expect master patch-rc v5.2.0-rc3

echo "# RC already cut from an unmerged branch"
jira_fixture
git checkout -q -b hotfix release-4.8
commit; tag v4.8.4-rc1
git checkout -q master
expect release-4.8 patch-rc v4.8.4-rc2
expect release-4.8 patch v4.8.4

echo "# Reachable RC whose final was tagged elsewhere"
jira_fixture
git checkout -q -b rc-branch master
commit; tag v5.2.0-rc1
git checkout -q -b final-branch master
commit; tag v5.2.0
git checkout -q master
expect_error rc-branch patch-rc

echo "# No tags"
new_repo
commit
expect HEAD patch v0.0.1
expect HEAD minor v0.1.0
expect HEAD major v1.0.0
expect HEAD patch-rc v0.0.1-rc1

echo "# Non-semver tags are ignored"
new_repo
commit; tag foo v1.2 v1.2.3-beta 1.0.0
expect HEAD patch v0.0.1
tag v1.2.3
expect HEAD patch v1.2.4

echo "# Invalid input"
expect_error HEAD bogus
expect_error does-not-exist patch

if [ "$failures" -ne 0 ]; then
	echo "$failures test(s) failed"
	exit 1
fi
echo "All tests passed"
