#!/usr/bin/env bash
# Replace template placeholders across the repository.
# Used by init-from-template GitHub Action and local bootstrap.
set -euo pipefail

TEMPLATE_ORG="${1:?Usage: $0 <org> <repo> <go_version> [service_slug] [db_name]}"
TEMPLATE_REPO="${2:?}"
GO_VERSION="${3:?}"
if [ -n "${4:-}" ]; then
  SERVICE_SLUG="$4"
else
  base="${TEMPLATE_REPO%-mservice}"
  base="${base%-api}"
  if [[ "${base}" == *-ms ]]; then
    SERVICE_SLUG="${base}"
  else
    SERVICE_SLUG="${base}-ms"
  fi
fi

DB_NAME="${5:-${SERVICE_SLUG//-/_}_db}"
SONAR_PROJECT_KEY="${TEMPLATE_ORG}_${TEMPLATE_REPO}"

# Repo is expected to be <domain>-mservice, e.g. order-mservice → domain=order
if [[ "${TEMPLATE_REPO}" == *-mservice ]]; then
  DOMAIN="${TEMPLATE_REPO%-mservice}"
else
  DOMAIN="${TEMPLATE_REPO%-api}"
  DOMAIN="${DOMAIN%-template}"
fi

PROTO_FILE="${DOMAIN}"
PROTO_PACKAGE="$(echo "${DOMAIN}" | tr -d '-')pb"
GRPC_SERVICE="$(perl -e '$d=shift; $d =~ s/(?:^|-)(.)/\U$1/g; print "${d}Service"' "$DOMAIN")"

echo "Replacing template variables:"
echo "  org:              ${TEMPLATE_ORG}"
echo "  repo:             ${TEMPLATE_REPO}"
echo "  go_version:       ${GO_VERSION}"
echo "  service_slug:     ${SERVICE_SLUG}"
echo "  db_name:          ${DB_NAME}"
echo "  sonar_project_key: ${SONAR_PROJECT_KEY}"
echo "  domain:           ${DOMAIN}"
echo "  proto_file:       ${PROTO_FILE}"
echo "  proto_package:    ${PROTO_PACKAGE}"
echo "  grpc_service:     ${GRPC_SERVICE}"

replace_in_file() {
  local file="$1"
  perl -pi -e "
    s/__TEMPLATE_ORG__/${TEMPLATE_ORG}/g;
    s/__TEMPLATE_REPO__/${TEMPLATE_REPO}/g;
    s/__GO_VERSION__/${GO_VERSION}/g;
    s/__SERVICE_SLUG__/${SERVICE_SLUG}/g;
    s/__DB_NAME__/${DB_NAME}/g;
    s/__SONAR_PROJECT_KEY__/${SONAR_PROJECT_KEY}/g;
    s/__PROTO_FILE__/${PROTO_FILE}/g;
    s/__PROTO_PACKAGE__/${PROTO_PACKAGE}/g;
    s/__GRPC_SERVICE__/${GRPC_SERVICE}/g;
    s/X_GRPC_SERVICE__/${GRPC_SERVICE}/g;
  " "${file}"
}

# Workflow YAML is excluded: GITHUB_TOKEN cannot push modified workflow files.
# CI workflows read Go version from server/go.mod and repo metadata at runtime.
while IFS= read -r -d '' file; do
  case "${file}" in
    ./scripts/replace-template-vars.sh|./scripts/finalize-init.sh|./TEMPLATE.md)
      continue
      ;;
  esac
  replace_in_file "${file}"
done < <(find . -type f \
  ! -path "./.git/*" \
  ! -path "./.github/workflows/*" \
  \( -name "*.go" -o -name "*.mod" -o -name "*.sum" -o -name "*.yml" -o -name "*.yaml" \
     -o -name "*.json" -o -name "*.properties" -o -name "Dockerfile" -o -name "*.sh" \
     -o -name "*.proto" -o -name "README.md" \) \
  -print0)

rename_if_exists() {
  local src="$1"
  local dest="$2"
  if [ -e "${src}" ]; then
    mv "${src}" "${dest}"
  fi
}

rename_if_exists "common/proto/__PROTO_FILE__.proto" "common/proto/${PROTO_FILE}.proto"
rename_if_exists "common/__PROTO_PACKAGE__" "common/${PROTO_PACKAGE}"
rename_if_exists "common/${PROTO_PACKAGE}/__PROTO_FILE__.pb.go" "common/${PROTO_PACKAGE}/${PROTO_FILE}.pb.go"
rename_if_exists "common/${PROTO_PACKAGE}/__PROTO_FILE___grpc.pb.go" "common/${PROTO_PACKAGE}/${PROTO_FILE}_grpc.pb.go"

if command -v protoc >/dev/null 2>&1; then
  echo "Regenerating protobuf files..."
  (cd common && ./scripts/compile_proto.sh)
else
  echo "protoc not found; skip protobuf regeneration"
fi

echo "Template variable replacement complete."
