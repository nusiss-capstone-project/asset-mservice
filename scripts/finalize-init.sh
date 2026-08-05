#!/usr/bin/env bash
# Post-init cleanup for repos created from the template.
set -euo pipefail

CANONICAL_TEMPLATE_REPO="${1:-lianjin/go-mservice-template}"
CURRENT_REPO="${2:-}"

if [ -z "${CURRENT_REPO}" ]; then
  if [ -n "${GITHUB_REPOSITORY:-}" ]; then
    CURRENT_REPO="${GITHUB_REPOSITORY}"
  else
    echo "Usage: $0 [canonical_template_repo] <current_repo>"
    exit 1
  fi
fi

if [ "${CURRENT_REPO}" = "${CANONICAL_TEMPLATE_REPO}" ]; then
  echo "Canonical template repo; keep template-only files."
  exit 0
fi

if [ -f README.md ] && grep -q '__TEMPLATE_' README.md; then
  echo "README still contains placeholders; run replace-template-vars.sh first."
  exit 1
fi

echo "Removing template-only files..."

# Documentation
rm -f TEMPLATE.md

# Init / provision workflows (not needed after bootstrap)
rm -f .github/workflows/init-from-template.yml
rm -f .github/workflows/provision-microservice.yml
rm -f .github/workflows/snyk.yml

# Init-only scripts (gotest-summary.sh is kept for CI)
rm -f scripts/replace-template-vars.sh
rm -f scripts/publish-common-module.sh
rm -f scripts/link-common-module.sh

# Snyk policy files
rm -f client/.snyk
rm -f server/.snyk

# Remove self last
rm -f scripts/finalize-init.sh

echo "Finalize complete."
