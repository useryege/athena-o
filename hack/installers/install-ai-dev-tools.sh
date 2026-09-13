#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
# shellcheck source=hack/tool-versions.sh
. "$root/hack/tool-versions.sh"

if [[ $(uname -s) != Linux ]]; then
  echo "AI development tools installer supports Ubuntu Linux only (got $(uname -s))." >&2
  exit 1
fi
# shellcheck source=/dev/null
. /etc/os-release
if [[ ${ID:-} != ubuntu ]]; then
  echo "PostgreSQL client installation requires Ubuntu (got ${ID:-unknown})." >&2
  exit 1
fi
case $(uname -m) in
  x86_64)
    shellcheck_arch=x86_64
    grpcurl_arch=x86_64
    ;;
  aarch64|arm64)
    shellcheck_arch=aarch64
    grpcurl_arch=arm64
    ;;
  *)
    echo "Unsupported AI development tools architecture: $(uname -m). Supported: x86_64, aarch64." >&2
    exit 1
    ;;
esac

downloads=${DOWNLOADS:-/tmp/dl}
mkdir -p "$downloads" "$root/dist"
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT

download_verified() {
  local archive=$1 url=$2 checksum=$3
  if [[ ! -f "$downloads/$archive" ]]; then
    curl -fsSL --retry 3 --retry-delay 2 -o "$downloads/$archive" "$url"
  fi
  (cd "$downloads" && sha256sum -c "$checksum") || {
    echo "SHA256 verification failed for $archive; remove $downloads/$archive and retry." >&2
    return 1
  }
}

shellcheck_archive="shellcheck-v${shellcheck_version}.linux.${shellcheck_arch}.tar.gz"
if [[ -x "$root/dist/shellcheck" ]] &&
   "$root/dist/shellcheck" --version | grep -Fq "version: $shellcheck_version"; then
  echo "ShellCheck $shellcheck_version already installed."
else
  download_verified "$shellcheck_archive" \
    "https://github.com/koalaman/shellcheck/releases/download/v${shellcheck_version}/${shellcheck_archive}" \
    "$root/hack/installers/checksums/${shellcheck_archive}.sha256"
  tar -xzf "$downloads/$shellcheck_archive" -C "$staging" \
    "shellcheck-v${shellcheck_version}/shellcheck"
  install -m 0755 "$staging/shellcheck-v${shellcheck_version}/shellcheck" "$root/dist/shellcheck"
fi

grpcurl_archive="grpcurl_${grpcurl_version}_linux_${grpcurl_arch}.tar.gz"
if [[ -x "$root/dist/grpcurl" ]] &&
   "$root/dist/grpcurl" -version 2>&1 | grep -Fq "v$grpcurl_version"; then
  echo "grpcurl $grpcurl_version already installed."
else
  download_verified "$grpcurl_archive" \
    "https://github.com/fullstorydev/grpcurl/releases/download/v${grpcurl_version}/${grpcurl_archive}" \
    "$root/hack/installers/checksums/${grpcurl_archive}.sha256"
  tar -xzf "$downloads/$grpcurl_archive" -C "$staging" grpcurl
  install -m 0755 "$staging/grpcurl" "$root/dist/grpcurl"
fi

required_go=$(awk '$1 == "go" { print "go" $2; exit }' "$root/go.mod")
if ! command -v go >/dev/null 2>&1; then
  echo "govulncheck $govulncheck_version requires the project Go toolchain $required_go; install it and retry." >&2
  exit 1
fi
host_go=$(GOTOOLCHAIN=local go env GOVERSION)
if [[ $host_go != "$required_go" ]]; then
  echo "govulncheck $govulncheck_version requires the project Go toolchain $required_go; found $host_go. Install $required_go and retry." >&2
  exit 1
fi
if [[ -x "$root/dist/govulncheck" ]] &&
   "$root/dist/govulncheck" -version 2>&1 | grep -Fxq "Scanner: govulncheck@v$govulncheck_version" &&
   go_build_info=$(GOTOOLCHAIN=local go version -m "$root/dist/govulncheck" 2>/dev/null) &&
   [[ $(awk 'NR == 1 { print $NF }' <<<"$go_build_info") == "$required_go" ]]; then
  echo "govulncheck $govulncheck_version already installed (built with $required_go)."
else
  GOTOOLCHAIN=local GOWORK=off GOFLAGS=-mod=mod GOBIN="$root/dist" \
    go install "golang.org/x/vuln/cmd/govulncheck@v${govulncheck_version}"
fi

if ! command -v node >/dev/null 2>&1 || ! command -v yarn >/dev/null 2>&1 ||
   ! node -e 'const version = process.versions.node; if (!/^\d+\.\d+\.\d+$/.test(version)) process.exit(1); const [major, minor, patch] = version.split(".").map(Number); if (major !== 24 || minor < 14 || (minor === 14 && patch < 1)) process.exit(1)' ||
   [[ $(yarn --version) != 1.22.22 ]]; then
  echo "Activate Node >=24.14.1 <25 and Yarn 1.22.22 before installing UI tools." >&2
  exit 1
fi
if [[ $(cd "$root/ui" && node -p 'require("./package.json").devDependencies["@axe-core/playwright"]') != "$axe_core_playwright_version" ]]; then
  echo "ui/package.json must pin @axe-core/playwright $axe_core_playwright_version exactly." >&2
  exit 1
fi
(cd "$root/ui" && yarn install --frozen-lockfile)

if ! dpkg-query -W -f='${Status}' "postgresql-client-${postgresql_client_major}" 2>/dev/null | grep -Fq 'install ok installed'; then
  if ! sudo -n true 2>/dev/null; then
    echo "PostgreSQL client $postgresql_client_major needs system installation. Run: sudo apt-get update && sudo apt-get install --no-install-recommends postgresql-client-${postgresql_client_major}; then retry." >&2
    exit 1
  fi
  sudo -n apt-get update
  sudo -n apt-get install -y --no-install-recommends "postgresql-client-${postgresql_client_major}"
fi

"$root/hack/ai-dev-tools.sh" check
