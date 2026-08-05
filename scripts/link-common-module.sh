#!/usr/bin/env bash
# Point client/server at the published common module (remote go get).
set -euo pipefail

TEMPLATE_ORG="${1:?Usage: $0 <org> <repo> [version]}"
TEMPLATE_REPO="${2:?}"
COMMON_VERSION="${3:-v0.0.1}"
COMMON_MODULE="github.com/${TEMPLATE_ORG}/${TEMPLATE_REPO}/common"
COMMON_TAG="common/${COMMON_VERSION}"

remove_common_replace() {
  local mod_dir="$1"
  perl -pi -e '
    if (/^replace github.com\/.*\/common => \.\.\/common\s*$/) {
      $_ = "";
    }
  ' "${mod_dir}/go.mod"
}

# go.sum from the template was generated with local ../common; must drop before remote go get.
purge_common_from_sum() {
  local mod_dir="$1"
  local sum_file="${mod_dir}/go.sum"
  if [ ! -f "${sum_file}" ]; then
    return 0
  fi
  if grep -q "${COMMON_MODULE}" "${sum_file}"; then
    grep -v "${COMMON_MODULE}" "${sum_file}" > "${sum_file}.tmp" || true
    if [ -s "${sum_file}.tmp" ]; then
      mv "${sum_file}.tmp" "${sum_file}"
    else
      rm -f "${sum_file}" "${sum_file}.tmp"
    fi
    echo "Cleared stale ${COMMON_MODULE} checksums from ${sum_file}"
  fi
}

echo "Linking client/server to ${COMMON_MODULE}@${COMMON_TAG}"

for module in client server; do
  remove_common_replace "${module}"
  purge_common_from_sum "${module}"
  (
    cd "${module}"
    GOPROXY=direct go get "${COMMON_MODULE}@${COMMON_TAG}"
    go mod tidy
  )
done

echo "Client/server linked to remote common module."
