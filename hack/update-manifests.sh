#! /usr/bin/env bash
set -x
set -o errexit
set -o nounset
set -o pipefail

SRCROOT="$( CDPATH='' cd -- "$(dirname "$0")/.." && pwd -P )"
AUTOGENMSG="# This is an auto-generated file. DO NOT EDIT"

KUSTOMIZE=kustomize
[ -f "$SRCROOT/dist/kustomize" ] && KUSTOMIZE="$SRCROOT/dist/kustomize"

cd "${SRCROOT}/manifests/ha/base/redis-ha" && ./generate.sh

# Image repository configuration - can be overridden in forks
IMAGE_REGISTRY="${IMAGE_REGISTRY:-quay.io}"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-useryege}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-athena}"
IMAGE_TAG="${IMAGE_TAG:-}"

# Construct full image name
FULL_IMAGE_NAME="${IMAGE_REGISTRY}/${IMAGE_NAMESPACE}/${IMAGE_REPOSITORY}"

# Auto-detect current image in manifests for release workflows
detect_current_image() {
  local manifest_file="$1"
  if [ -f "$manifest_file" ]; then
    # Look for the current image name in kustomization.yaml images section
    awk '/^images:/,/^[a-zA-Z]/ { if (/- name:/ && /athena/) { gsub(/.*name: */, ""); gsub(/ *$/, ""); print; exit } }' "$manifest_file"
  fi
}

# Determine source image (what to replace)
DETECTED_IMAGE=$(detect_current_image "${SRCROOT}/manifests/base/kustomization.yaml")
if [ -n "$DETECTED_IMAGE" ] && [ "$DETECTED_IMAGE" != "quay.io/useryege/athena" ]; then
  # Found a custom image in manifests (subsequent release scenario)
  SOURCE_IMAGE_NAME="$DETECTED_IMAGE"
  echo "Detected existing custom image in manifests: $SOURCE_IMAGE_NAME"
else
  # Use default source image (fresh fork or manual override)
  SOURCE_IMAGE_NAME="quay.io/useryege/athena"
  echo "Using default source image: $SOURCE_IMAGE_NAME"
fi

# if the tag has not been declared, and we are on a release branch, use the VERSION file.
if [ "$IMAGE_TAG" = "" ]; then
  branch=$(git rev-parse --abbrev-ref HEAD)
  # In GitHub Actions PRs, HEAD is detached; use GITHUB_BASE_REF (the target branch) instead
  if [ "$branch" = "HEAD" ] && [ -n "${GITHUB_BASE_REF:-}" ]; then
    branch="$GITHUB_BASE_REF"
  fi
  if [[ $branch = release-* ]]; then
    pwd
    IMAGE_TAG=v$(cat "$SRCROOT/VERSION")
  fi
fi
# otherwise, use latest
if [ "$IMAGE_TAG" = "" ]; then
  IMAGE_TAG=latest
fi

$KUSTOMIZE version
which "$KUSTOMIZE"

echo "=== Manifest Generation Configuration ==="
echo "Source image (to replace): ${SOURCE_IMAGE_NAME}"
echo "Target image (replace with): ${FULL_IMAGE_NAME}:${IMAGE_TAG}"
if [ "$DETECTED_IMAGE" != "quay.io/useryege/athena" ] && [ -n "$DETECTED_IMAGE" ]; then
  echo "Scenario: Subsequent release (updating existing custom image)"
else
  echo "Scenario: First release or local development"
fi
echo "========================================"

cd "${SRCROOT}/manifests/base" && $KUSTOMIZE edit set image "${SOURCE_IMAGE_NAME}=${FULL_IMAGE_NAME}:${IMAGE_TAG}"
cd "${SRCROOT}/manifests/ha/base" && $KUSTOMIZE edit set image "${SOURCE_IMAGE_NAME}=${FULL_IMAGE_NAME}:${IMAGE_TAG}"

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/install.yaml"
$KUSTOMIZE build "${SRCROOT}/manifests/cluster-install" >> "${SRCROOT}/manifests/install.yaml"

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/ha/install.yaml"
$KUSTOMIZE build "${SRCROOT}/manifests/ha/cluster-install" >> "${SRCROOT}/manifests/ha/install.yaml"

