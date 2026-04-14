#!/usr/bin/env bash
set -euo pipefail

cluster_type=${1:-kind}
matrix=${PROFILE_MATRIX_PATH:-build/harness/profile-matrix.json}
image_repo=${HARNESS_IMAGE:-kocao/harness-runtime}
image_tag=${IMAGE_TAG:-dev}
kind_bin=${KIND_BIN:-kind}
kubectl_bin=${KUBECTL_BIN:-kubectl}
prepull_namespace=${PREPULL_NAMESPACE:-kocao-system}
prepull_name=${PREPULL_NAME:-harness-profile-prepull}
prepull_timeout=${PREPULL_TIMEOUT:-10m}
prepull_context=${PREPULL_CONTEXT:-}
prepull_keep=${KEEP_PREPULL_DAEMONSET:-0}
prepull_dry_run=${PREPULL_DRY_RUN:-0}
image_pull_secrets=${IMAGE_PULL_SECRETS:-}
prepull_cleanup_armed=0

usage() {
  cat <<'EOF'
usage: hack/dev/prepull-harness-images.sh [kind|microk8s|registry]

Modes:
  kind      Load locally built harness profile images into a Kind cluster.
  microk8s  Pre-pull registry-backed harness profile images via a DaemonSet.
  registry  Same as microk8s, but without the default microk8s context.

Environment:
  HARNESS_IMAGE           Image repository (default: kocao/harness-runtime)
  IMAGE_TAG               Image tag prefix (default: dev)
  HARNESS_PREPULL_PROFILES Comma-separated profile list (default: matrix buildOrder)
  PREPULL_CONTEXT         Optional kubectl context override
  PREPULL_NAMESPACE       Namespace for registry-backed pre-pull jobs
  IMAGE_PULL_SECRETS      Comma-separated image pull secrets for registry-backed mode
  KEEP_PREPULL_DAEMONSET  Set to 1 to leave the DaemonSet in place
  PREPULL_DRY_RUN         Set to 1 to print actions/manifest without applying them
EOF
}

require_tool() {
  local tool=$1
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "required tool not found: ${tool}" >&2
    exit 1
  fi
}

matrix_profiles() {
  jq -r '.buildOrder[]' "${matrix}"
}

profile_suffix() {
  local profile=$1
  jq -r --arg profile "${profile}" '.profiles[$profile].imageSuffix // empty' "${matrix}"
}

compatibility_profile() {
  jq -r '.compatibilityProfile' "${matrix}"
}

image_ref_for_profile() {
  local profile=$1
  local suffix
  suffix=$(profile_suffix "${profile}")
  if [ -z "${suffix}" ]; then
    echo "unknown harness profile: ${profile}" >&2
    exit 1
  fi
  printf '%s:%s-%s\n' "${image_repo}" "${image_tag}" "${suffix}"
}

compatibility_alias_ref() {
  printf '%s:%s' "${image_repo}" "${image_tag}"
}

selected_profiles() {
  local raw=${HARNESS_PREPULL_PROFILES:-}
  if [ -n "${raw}" ]; then
    printf '%s' "${raw}" | tr ',' '\n' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed '/^$/d'
    return
  fi
  matrix_profiles
}

selected_image_refs() {
  local profile
  while IFS= read -r profile; do
    [ -n "${profile}" ] || continue
    image_ref_for_profile "${profile}"
  done < <(selected_profiles)
}

print_selected_profiles() {
  local first=1
  local profile
  while IFS= read -r profile; do
    [ -n "${profile}" ] || continue
    if [ "${first}" -eq 1 ]; then
      printf '%s' "${profile}"
      first=0
      continue
    fi
    printf ', %s' "${profile}"
  done < <(selected_profiles)
}

kind_load() {
  require_tool jq
  require_tool docker
  if [ "${prepull_dry_run}" != "1" ]; then
    require_tool "${kind_bin}"
  fi

  local cluster_name=${KIND_CLUSTER_NAME:-kocao-dev}
  local image_ref

  echo "pre-pulling harness profiles into kind cluster ${cluster_name}: $(print_selected_profiles)"
  while IFS= read -r image_ref; do
    [ -n "${image_ref}" ] || continue
    if [ "${prepull_dry_run}" = "1" ]; then
      echo "${kind_bin} load docker-image --name ${cluster_name} ${image_ref}"
    else
      docker image inspect "${image_ref}" >/dev/null 2>&1 || {
        echo "local image missing: ${image_ref}" >&2
        echo "build the harness profiles first (for example: make harness-images)" >&2
        exit 1
      }
      echo "loading ${image_ref}"
      "${kind_bin}" load docker-image --name "${cluster_name}" "${image_ref}"
    fi
  done < <(selected_image_refs)

  local compat_ref
  compat_ref=$(compatibility_alias_ref)
  docker image inspect "${compat_ref}" >/dev/null 2>&1 || true
  if docker image inspect "${compat_ref}" >/dev/null 2>&1; then
    if [ "${prepull_dry_run}" = "1" ]; then
      echo "${kind_bin} load docker-image --name ${cluster_name} ${compat_ref}"
    else
      echo "loading compatibility alias ${compat_ref}"
      "${kind_bin}" load docker-image --name "${cluster_name}" "${compat_ref}"
    fi
  fi
}

kubectl_cmd() {
  if [ -n "${prepull_context}" ]; then
    "${kubectl_bin}" --context "${prepull_context}" "$@"
    return
  fi
  "${kubectl_bin}" "$@"
}

cleanup_registry_daemonset() {
  local exit_code=$?

  if [ "${prepull_cleanup_armed}" = "1" ] && [ "${prepull_keep}" != "1" ] && [ "${prepull_dry_run}" != "1" ]; then
    kubectl_cmd -n "${prepull_namespace}" delete daemonset "${prepull_name}" --ignore-not-found >/dev/null 2>&1 || true
  fi

  return "${exit_code}"
}

render_registry_manifest() {
  local image_refs=()
  local image_ref
  while IFS= read -r image_ref; do
    [ -n "${image_ref}" ] || continue
    image_refs+=("${image_ref}")
  done < <(selected_image_refs)

  cat <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: ${prepull_name}
  namespace: ${prepull_namespace}
  labels:
    app.kubernetes.io/name: ${prepull_name}
    app.kubernetes.io/part-of: kocao
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: ${prepull_name}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ${prepull_name}
        app.kubernetes.io/part-of: kocao
    spec:
      serviceAccountName: default
      tolerations:
        - operator: Exists
EOF

  if [ -n "${image_pull_secrets}" ]; then
    printf '%s\n' '      imagePullSecrets:'
    printf '%s' "${image_pull_secrets}" | tr ',' '\n' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | sed '/^$/d' | while IFS= read -r secret; do
      printf '        - name: %s\n' "${secret}"
    done
  fi

  printf '%s\n' '      initContainers:'
  local index=0
  for image_ref in "${image_refs[@]}"; do
    index=$((index + 1))
    cat <<EOF
        - name: prepull-${index}
          image: ${image_ref}
          imagePullPolicy: IfNotPresent
          command: ["/bin/sh", "-c", "echo prepulled ${image_ref}"]
EOF
  done

  cat <<'EOF'
      containers:
        - name: pause
          image: registry.k8s.io/pause:3.10
          imagePullPolicy: IfNotPresent
      terminationGracePeriodSeconds: 0
EOF
}

registry_prepull() {
  require_tool jq
  require_tool "${kubectl_bin}"

  if [ "${cluster_type}" = "microk8s" ] && [ -z "${prepull_context}" ]; then
    prepull_context=microk8s
  fi

  echo "pre-pulling harness profiles via daemonset in namespace ${prepull_namespace}: $(print_selected_profiles)"

  if [ "${prepull_dry_run}" = "1" ]; then
    render_registry_manifest
    return
  fi

  trap cleanup_registry_daemonset EXIT
  prepull_cleanup_armed=1
  kubectl_cmd get namespace "${prepull_namespace}" >/dev/null 2>&1 || kubectl_cmd create namespace "${prepull_namespace}" >/dev/null
  render_registry_manifest | kubectl_cmd apply -f - >/dev/null
  kubectl_cmd -n "${prepull_namespace}" rollout status daemonset/"${prepull_name}" --timeout "${prepull_timeout}"
}

case "${cluster_type}" in
  kind)
    kind_load
    ;;
  microk8s|registry)
    registry_prepull
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "unsupported cluster type: ${cluster_type}" >&2
    usage >&2
    exit 2
    ;;
esac
