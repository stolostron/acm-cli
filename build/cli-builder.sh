#! /bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
BUILD_DIR=${BUILD_DIR:-"build/_output"}
BUILD_INPUT=${BUILD_INPUT:-"cli_map.csv"}
current_branch=$(git -C "${SCRIPT_DIR}/../" branch --show-current)

if ! [[ -d "${SCRIPT_DIR}/../${BUILD_DIR}" ]]; then
  echo "="
  echo "* error: build directory ${BUILD_DIR} does not exist. Check whether the build directory is initialized:"
  echo "  make build"
  echo
  exit 1
fi

function return_submodule_error() {
  echo "="
  echo "* error: build failed. Check whether the submodule is initialized and up to date:"
  echo "  make sync-repos"
  echo
  exit 1
}

while IFS=, read -r git_url build_cmd build_dir; do
  if [[ "${git_url}" == "GIT REPO URL" ]]; then
    continue
  fi

  git_repo=${git_url##*/}

  cd "${SCRIPT_DIR}/../external/${git_repo}" || return_submodule_error
  echo "=="

  # Set branch using .gitmodules config
  parent_gitmodules="${SCRIPT_DIR}/../.gitmodules"
  if [[ -f "${parent_gitmodules}" ]]; then
    submodule_branch=$(git config --file "${parent_gitmodules}" --get "submodule.${git_repo}.branch" 2>/dev/null || echo "")
    if [[ -n "${submodule_branch}" ]]; then
      echo "* Creating branch '${submodule_branch}' for version generation"
      git checkout -b "${submodule_branch}" 2>/dev/null || echo "* Branch '${submodule_branch}' already exists or cannot be created"
    fi
  fi

  if { [[ "${current_branch}" == "release-"* ]] && [[ "${submodule_branch}" != "${current_branch}" ]]; } ||
     { [[ -n "${previous_branch}" ]] && [[ "${submodule_branch}" != "${previous_branch}" ]]; }; then
    echo "* Branch '${submodule_branch}' for '${git_repo}' does not match the current branch '${current_branch}' or the previous submodule branch '${previous_branch}'."
    exit 1
  fi

  echo "* Building binaries from ${git_url}"
  echo "* Executing build command: ${build_cmd}"
  ${build_cmd} || return_submodule_error
  echo "* Moving binaries from repo directory <repo>/${build_dir}/ to: ./${BUILD_DIR}/"
  if [[ -f "${build_dir}" ]]; then
    mv "${build_dir}" "${SCRIPT_DIR}/../${BUILD_DIR}"
  else
  mv "${build_dir}"/* "${SCRIPT_DIR}/../${BUILD_DIR}"
  fi
  previous_branch=${submodule_branch}

  cd - 1>/dev/null
done <"${SCRIPT_DIR}/${BUILD_INPUT}"
