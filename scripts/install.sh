#!/bin/sh

set -eu

repository="InsonusK/go-ai-skill-manager"
install_dir="${AISM_INSTALL_DIR:-/usr/local/bin}"

fail() {
	printf 'aism installer: %s\n' "$*" >&2
	exit 1
}

for dependency in curl sha256sum awk mktemp install; do
	command -v "$dependency" >/dev/null 2>&1 || fail "required command not found: $dependency"
done

[ "$(uname -s)" = "Linux" ] || fail "this installer supports Linux only"
case "$(uname -m)" in
	x86_64 | amd64) ;;
	*) fail "unsupported architecture: $(uname -m) (only amd64 is published)" ;;
esac

latest_url=$(curl -fsSIL -o /dev/null -w '%{url_effective}' \
	"https://github.com/${repository}/releases/latest")
latest_url=${latest_url%/}
tag=${latest_url##*/}
case "$tag" in
	v?*) ;;
	*) fail "could not determine the latest release from: $latest_url" ;;
esac

version=${tag#v}
asset="ai-skill-manager_${version}_linux_amd64"
checksums="ai-skill-manager_${version}_checksums.txt"
download_base="https://github.com/${repository}/releases/download/${tag}"

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
trap 'exit 1' HUP INT TERM

printf 'Downloading ai-skill-manager %s for linux/amd64...\n' "$version"
curl -fsSLo "$temp_dir/$asset" "$download_base/$asset"
curl -fsSLo "$temp_dir/$checksums" "$download_base/$checksums"

expected_hash=$(awk -v name="$asset" '$2 == name || $2 == "*" name { print $1; exit }' \
	"$temp_dir/$checksums")
[ -n "$expected_hash" ] || fail "checksum for $asset is missing"
actual_hash=$(sha256sum "$temp_dir/$asset" | awk '{ print $1 }')
[ "$actual_hash" = "$expected_hash" ] || fail "checksum verification failed for $asset"

if mkdir -p "$install_dir" 2>/dev/null && [ -w "$install_dir" ]; then
	install -m 0755 "$temp_dir/$asset" "$install_dir/aism"
else
	command -v sudo >/dev/null 2>&1 || \
		fail "cannot write to $install_dir; set AISM_INSTALL_DIR or install sudo"
	sudo mkdir -p "$install_dir"
	sudo install -m 0755 "$temp_dir/$asset" "$install_dir/aism"
fi

printf 'Installed %s\n' "$install_dir/aism"
"$install_dir/aism" --version
