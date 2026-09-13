#!/bin/sh
###############################################################################
# This file defines the versions of the tools that are installed in the CI
# toolchain and the Docker image.
#
# Updating a tool's version here is not enough, you will need to create a
# checksum file in ./hack/installers/checksums matching the name of the
# downloaded binary with a ".sha256" suffix appended, containing the proper
# SHA256 sum of the binary.
#
# Use helper scripts under ./hack/installers/checksums to help download checksums.
###############################################################################
protoc_version=29.3
oras_version=1.2.0
export shellcheck_version=0.11.0
export grpcurl_version=1.9.4
export govulncheck_version=1.7.0
export axe_core_playwright_version=4.13.0
export postgresql_client_major=16
