#!/usr/bin/env bash
set -euo pipefail

matrix=${PROFILE_MATRIX_PATH:-build/harness/profile-matrix.json}
image_repo=${HARNESS_IMAGE:-kocao/harness-runtime}
image_tag=${IMAGE_TAG:-dev}
build_context=${BUILD_CONTEXT:-.}

profile_image_ref() {
  local suffix="$1"
  printf '%s:%s-%s' "${image_repo}" "${image_tag}" "${suffix}"
}

dockerfile=$(jq -r '.dockerfile' "${matrix}")
compatibility_profile=$(jq -r '.compatibilityProfile' "${matrix}")

mapfile -t profiles < <(jq -r '.buildOrder[]' "${matrix}")

for profile in "${profiles[@]}"; do
  target=$(jq -r --arg profile "${profile}" '.profiles[$profile].buildTarget' "${matrix}")
  suffix=$(jq -r --arg profile "${profile}" '.profiles[$profile].imageSuffix' "${matrix}")
  image_ref=$(profile_image_ref "${suffix}")

  echo "==> Building ${profile} profile as ${image_ref}"
  docker build -f "${dockerfile}" --target "${target}" -t "${image_ref}" "${build_context}"
done

compatibility_suffix=$(jq -r --arg profile "${compatibility_profile}" '.profiles[$profile].imageSuffix' "${matrix}")
docker tag "$(profile_image_ref "${compatibility_suffix}")" "${image_repo}:${image_tag}"
echo "==> Tagged compatibility profile ${compatibility_profile} as ${image_repo}:${image_tag}"
