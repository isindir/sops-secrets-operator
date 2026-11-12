#!/usr/bin/env bash
# Chart release helper used by CI workflows.
#
#   verify [version]  packaged chart archive and chart index entry exist in docs/
#   push   [version]  verify, then push the packaged archive to the OCI repository
#
# version defaults to the chart version in Chart.yaml.
# push does not log in, registry authentication is up to the caller ('helm registry login').
#
# Environment:
#   CHART_OCI_REPO        target OCI repository (default: oci://quay.io/isindir/charts)
#   CHART_OCI_PLAIN_HTTP  "true" to push over plain HTTP, for local test registries only
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="${REPO_ROOT}/chart/helm4/sops-secrets-operator"
PACKAGE_DIR="${REPO_ROOT}/docs"
INDEX_FILE="${PACKAGE_DIR}/index.yaml"
CHART_OCI_REPO="${CHART_OCI_REPO:-oci://quay.io/isindir/charts}"
CHART_OCI_PLAIN_HTTP="${CHART_OCI_PLAIN_HTTP:-false}"

usage() {
    echo "usage: $(basename "$0") verify|push [version]" >&2
    exit 2
}

chart_field() {
    awk -v key="$1" '$0 ~ "^" key ":" { gsub(/["'\'']/, "", $2); print $2; exit }' "${CHART_DIR}/Chart.yaml"
}

chart_archive() {
    echo "${PACKAGE_DIR}/$(chart_field name)-$1.tgz"
}

verify() {
    local version="$1" archive
    archive="$(chart_archive "${version}")"

    if [[ ! -f "${archive}" ]]; then
        echo "ERROR: chart ${version} is not packaged: ${archive#"${REPO_ROOT}"/} is missing, run 'make package-helm'" >&2
        return 1
    fi
    # chart entries in index.yaml written by 'helm repo index' carry their version at 4 spaces indentation
    if ! grep -qxF "    version: ${version}" "${INDEX_FILE}"; then
        echo "ERROR: chart ${version} has no entry in ${INDEX_FILE#"${REPO_ROOT}"/}, run 'make package-helm'" >&2
        return 1
    fi
    echo "chart ${version} is packaged: ${archive#"${REPO_ROOT}"/}"
}

push() {
    local version="$1" archive
    local -a flags=()
    verify "${version}"
    archive="$(chart_archive "${version}")"
    [[ "${CHART_OCI_PLAIN_HTTP}" == "true" ]] && flags+=(--plain-http)

    # always push: re-pushing a re-packaged archive keeps the OCI tag identical to docs/
    helm push "${archive}" "${CHART_OCI_REPO}" ${flags[@]+"${flags[@]}"}
}

main() {
    [[ $# -ge 1 && $# -le 2 ]] || usage
    local command="$1" version="${2:-}"
    case "${command}" in
        verify | push) ;;
        *) usage ;;
    esac

    [[ -n "${version}" ]] || version="$(chart_field version)"
    if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$ ]]; then
        echo "ERROR: invalid chart version '${version}'" >&2
        exit 2
    fi

    "${command}" "${version}"
}

main "$@"
