#!/bin/bash

exit_code=0

echo "=== Validate Dockerfiles"
builder_image_regex="^(FROM ).*([0-9]+\.[0-9]+).*( AS builder)$"
base_image_regex="^(FROM ).*/(ubi[0-9]+)/.*(:latest)$"
base_dockerfile=$(sed -E '
  s/'"${builder_image_regex}"'/\1\2\3/g
  s%'"${base_image_regex}"'%\1\2\3%g
  s/--chown=1001:1001 //' \
  Dockerfile)
rhtap_dockerfile=$(sed -E '
  s/'"${builder_image_regex}"'/\1\2\3/g
  s%'"${base_image_regex}"'%\1\2\3%g
  s/GOEXPERIMENT=strictfipsruntime //' \
  Dockerfile.rhtap)

if ! diff \
  <(echo "${base_dockerfile}") \
  <(echo "${rhtap_dockerfile}"); then
  echo "  ❌ Dockerfile and Dockerfile.rhtap are not in sync"
  exit_code=1
else
  echo "  ✅ Dockerfile and Dockerfile.rhtap are in sync"
fi

echo "=== Validate git submodule branches"
version="${VERSION_REF}"
current_version=$(curl -s --fail https://raw.githubusercontent.com/stolostron/governance-policy-framework/refs/heads/main/CURRENT_VERSION | head -n 1)
if [[ -z "${current_version}" ]] || ! [[ "${current_version}" =~ ^[0-9]+\.[0-9]+$ ]]; then
  echo "* error: failed to fetch valid CURRENT_VERSION (got: '${current_version}')"
  exit 1
fi

if [[ ${version} =~ ^release-[0-9]+.[0-9]+$ ]]; then
  version=$(printf "%s\n%s\n" "${version#release-}" "${current_version}" | sort -V | head -n 1)
else
  version="${current_version}"
fi

for submodule in policy-cli policy-generator-plugin; do
  if [[ $(git config --file ".gitmodules" --get "submodule.${submodule}.branch") != "release-${version}" ]]; then
    echo "  ❌ submodule '${submodule}' is not on the correct branch. Expected 'release-${version}' :"
    git config --file ".gitmodules" --get "submodule.${submodule}.branch"
    exit_code=1
  else
    echo "  ✅ Submodule '${submodule}' is on the correct branch"
  fi
done

exit ${exit_code}
