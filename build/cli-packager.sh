#! /bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

BUILD_DIR=${BUILD_DIR:-"${SCRIPT_DIR}/_output"}

cp "${SCRIPT_DIR}"/../LICENSE "${BUILD_DIR}"

cd "${BUILD_DIR}"

echo "# Packaging Windows binaries into zip archives"
find . -type f -maxdepth 1 \
  -name "*.exe" \
  -exec bash -c 'f="${1}"; mkdir $(basename ${f} ".exe")' - '{}' \; \
  -exec bash -c 'f="${1}"; mv ${f} $(basename ${f} ".exe")/${f##*-}' - '{}' \; \
  -exec bash -c 'f="${1}"; cp LICENSE $(basename ${f} ".exe")/' - '{}' \; \
  -exec bash -c 'f="${1}"; zip -rv $(basename ${f} ".exe").zip $(basename ${f} ".exe")' - '{}' \; \
  -exec bash -c 'f="${1}"; rm -r $(basename ${f} ".exe")' - '{}' \;

echo "# Packaging Linux/Darwin binaries into tarballs"
find . -type f -maxdepth 1 \
  -not -name ".*" \
  -not -name "acm-cli-server" \
  -not -name "*.zip" \
  -not -name "*.tar.gz" \
  -not -name "LICENSE" \
  -exec bash -c 'f="${1}"; mkdir tmp-$(basename ${f})' - '{}' \; \
  -exec bash -c 'f="${1}"; cp LICENSE tmp-$(basename ${f})/' - '{}' \; \
  -exec bash -c 'f="${1}"; mv ${f} tmp-$(basename ${f})/${f##*-}' - '{}' \; \
  -exec bash -c 'f="${1}"; tar -C tmp-$(basename ${f}) -zvcf $(basename ${f}).tar.gz LICENSE ${f##*-}' - '{}' \; \
  -exec bash -c 'f="${1}"; rm -r tmp-$(basename ${f})' - '{}' \; \

rm LICENSE
