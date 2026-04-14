#!/usr/bin/env bash
set -euo pipefail

matrix=${PROFILE_MATRIX_PATH:-build/harness/profile-matrix.json}
image_repo=${HARNESS_IMAGE:-kocao/harness-runtime}
image_tag=${IMAGE_TAG:-dev}
smoke_command=${HARNESS_SMOKE_COMMAND:-/usr/local/bin/kocao-harness-smoke}

mapfile -t profiles < <(jq -r '.buildOrder[]' "${matrix}")

for profile in "${profiles[@]}"; do
  suffix=$(jq -r --arg profile "${profile}" '.profiles[$profile].imageSuffix' "${matrix}")
  image_ref="${image_repo}:${image_tag}-${suffix}"

  echo "==> Running smoke for ${profile} profile (${image_ref})"
  docker run --rm "${image_ref}" "${smoke_command}"
done
