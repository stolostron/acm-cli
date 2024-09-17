#! /bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

REMOTE_SOURCES_DIR=${REMOTE_SOURCES_DIR:-${SCRIPT_DIR}/../external}
REMOTE_SOURCES_SUBDIR=${REMOTE_SOURCES_SUBDIR:-}
BUILD_DIR=${BUILD_DIR:-${SCRIPT_DIR}}

while IFS=, read -r git_url build_cmd build_dir; do
  git_repo=${git_url##*/}

  (
    # Source relevant cachito.env for downstream builds
    cachito_path="${REMOTE_SOURCES_DIR}/${git_repo}"/cachito.env
    if [[ -f "${cachito_path}" ]]; then
      source "${cachito_path}"
    fi

    # Set branch using downstream UPSTREAM_BRANCH variable
    # ref: https://cpaas.pages.redhat.com/documentation/users/midstream/generating_files_from_templates.html#_environment_variables
    repo_upper=$(echo ${git_repo} | tr '[:lower:]' '[:upper:]')
    branch_var=CI_${repo_upper//-/_}_UPSTREAM_BRANCH
    if [[ -z "$(git branch --show-current)" ]] && [[ -n "${!branch_var}" ]]; then
      git checkout -b ${!branch_var}
    fi

    echo "* Building binaries from ${git_url}"
    cd "${REMOTE_SOURCES_DIR}/${git_repo}/${REMOTE_SOURCES_SUBDIR}"
    echo "* Executing build command: ${build_cmd}"
    ${build_cmd}
    echo "* Moving binaries from repo directory <repo>/${build_dir}/ to: ./${BUILD_DIR}/"
    mv ${build_dir}/* ${SCRIPT_DIR}/../${BUILD_DIR}
  )
done <${SCRIPT_DIR}/cli_map.csv
