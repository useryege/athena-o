#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-root}"

if [[ -z "${REMOTE_HOST}" ]]; then
  echo "REMOTE_HOST is required. Pass it to make, for example: make install-docker-vps REMOTE_HOST=your-server-ip" >&2
  exit 1
fi

if [[ -z "${REMOTE_USER}" ]]; then
  echo "REMOTE_USER must not be empty." >&2
  exit 1
fi

command -v ssh >/dev/null

remote="${REMOTE_USER}@${REMOTE_HOST}"

echo "Checking Docker installation state on ${remote}..."
ssh "${remote}" 'bash -s' <<'REMOTE_SCRIPT'
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Docker installation requires a direct root SSH session; interactive sudo is not supported." >&2
  exit 1
fi

if [[ ! -r /etc/os-release ]]; then
  echo "Unable to identify the remote operating system: /etc/os-release is missing." >&2
  exit 1
fi

# shellcheck disable=SC1091
source /etc/os-release
if [[ "${ID:-}" != "ubuntu" ]]; then
  echo "Unsupported remote operating system: ${PRETTY_NAME:-${ID:-unknown}}. Only Ubuntu is supported." >&2
  exit 1
fi

if ! command -v apt-get >/dev/null || ! command -v dpkg >/dev/null || ! command -v dpkg-query >/dev/null; then
  echo "The remote Ubuntu host must provide apt-get, dpkg, and dpkg-query." >&2
  exit 1
fi

ubuntu_codename="${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}"
if [[ -z "${ubuntu_codename}" ]]; then
  echo "Unable to determine the Ubuntu release codename." >&2
  exit 1
fi
dpkg_architecture="$(dpkg --print-architecture)"

docker_ready() {
  command -v docker >/dev/null 2>&1 &&
    docker info >/dev/null 2>&1 &&
    docker compose version >/dev/null 2>&1 &&
    docker compose up --help 2>&1 | grep -q -- '--wait' &&
    systemctl is-active --quiet docker &&
    systemctl is-enabled --quiet docker
}

if docker_ready; then
  echo "Docker is already installed and ready; no packages were changed."
  docker --version
  docker compose version
  exit 0
fi

package_installed() {
  local package_name="$1"
  [[ "$(dpkg-query -W -f='${Status}' "${package_name}" 2>/dev/null || true)" == "install ok installed" ]]
}

existing_docker_components=()
if command -v docker >/dev/null 2>&1; then
  existing_docker_components+=("docker CLI")
fi
for package_name in docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker-ce-rootless-extras; do
  if package_installed "${package_name}"; then
    existing_docker_components+=("${package_name}")
  fi
done
if systemctl list-unit-files docker.service >/dev/null 2>&1; then
  existing_docker_components+=("docker.service")
fi

if ((${#existing_docker_components[@]} > 0)); then
  echo "An incomplete or unhealthy Docker installation already exists: ${existing_docker_components[*]}" >&2
  echo "No packages were changed. Repair or remove the existing installation before rerunning this command." >&2
  exit 1
fi

conflicting_packages=()
for package_name in docker.io docker-compose docker-compose-v2 docker-doc podman-docker containerd runc; do
  if package_installed "${package_name}"; then
    conflicting_packages+=("${package_name}")
  fi
done

if ((${#conflicting_packages[@]} > 0)); then
  echo "Docker installation stopped because conflicting packages are installed: ${conflicting_packages[*]}" >&2
  echo "No packages were removed. Review these packages manually before installing Docker CE." >&2
  exit 1
fi

echo "Installing Docker CE on ${PRETTY_NAME:-Ubuntu} (${dpkg_architecture})..."
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates curl

install -m 0755 -d /etc/apt/keyrings
docker_key_tmp="$(mktemp)"
docker_source_tmp="$(mktemp)"
cleanup() {
  rm -f "${docker_key_tmp}" "${docker_source_tmp}"
}
trap cleanup EXIT

curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o "${docker_key_tmp}"
install -m 0644 "${docker_key_tmp}" /etc/apt/keyrings/docker.asc

printf '%s\n' \
  'Types: deb' \
  'URIs: https://download.docker.com/linux/ubuntu' \
  "Suites: ${ubuntu_codename}" \
  'Components: stable' \
  "Architectures: ${dpkg_architecture}" \
  'Signed-By: /etc/apt/keyrings/docker.asc' \
  >"${docker_source_tmp}"
install -m 0644 "${docker_source_tmp}" /etc/apt/sources.list.d/docker.sources

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable --now docker

if ! docker_ready; then
  echo "Docker packages were installed, but the required Engine or Compose capabilities are not ready." >&2
  systemctl status docker --no-pager >&2 || true
  exit 1
fi

echo "Docker installed successfully."
docker --version
docker compose version
REMOTE_SCRIPT

echo
echo "Remote Docker dependency is ready."
echo "Deploy the BSC transaction indexer with:"
echo "make deploy-bsc-transaction-indexer-vps REMOTE_HOST=${REMOTE_HOST} BSC_INDEXER_ENV_FILE=.env.bsc-transaction-indexer"
