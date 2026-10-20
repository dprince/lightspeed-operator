#!/usr/bin/env bash

set -euo pipefail
if [[ "$HAS_TOKEN" != true ]]; then
  echo "::error::Set the COMMUNITY_OPERATORS_TOKEN repository secret."
  exit 1
fi
: "${AUTHOR_NAME:?Set the RELEASE_AUTHOR_NAME repository variable.}"
: "${AUTHOR_EMAIL:?Set the RELEASE_AUTHOR_EMAIL repository variable.}"
version_pattern='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
if [[ ! "$NEW_VERSION" =~ $version_pattern ]]; then
  echo "::error::new_version must have the form X.Y.Z."
  exit 1
fi
if [[ ! "$FORK_REPOSITORY" =~ ^[a-zA-Z0-9-]+/[a-zA-Z0-9_.-]+$ ]]; then
  echo "::error::Set the COMMUNITY_OPERATORS_FORK repository variable to owner/repository."
  exit 1
fi
if [[ ! "$CONTAINER_IMAGE_TAG" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]]; then
  echo "::error::container_image_tag must be a valid container image tag."
  exit 1
fi

operator_dir=operators/openstack-lightspeed-operator
release_dir="$operator_dir/$NEW_VERSION"
if [[ -e "$release_dir" ]]; then
  echo "::error::Version $NEW_VERSION already exists on upstream main."
  exit 1
fi
PREVIOUS_VERSION=$(
  { find "$operator_dir" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'; printf '%s\n' "$NEW_VERSION"; } |
    grep -E "$version_pattern" | sort -Vu |
    awk -v version="$NEW_VERSION" '$0 == version { print previous } { previous = $0 }'
)
: "${PREVIOUS_VERSION:?No existing community release precedes this version.}"

declare -A image_digests
for repository in operator operator-bundle; do
  digest=$(curl --fail --silent --show-error --max-time 30 \
    --get --data-urlencode "specificTag=$CONTAINER_IMAGE_TAG" \
    --data-urlencode 'onlyActiveTags=true' \
    "https://quay.io/api/v1/repository/openstack-lightspeed/$repository/tag/" |
    jq -r --arg tag "$CONTAINER_IMAGE_TAG" \
      '[.tags[]? | select(.name == $tag) | .manifest_digest][0] // empty')
  if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "::error::No active $repository image with a valid digest for $CONTAINER_IMAGE_TAG."
    exit 1
  fi
  image_digests[$repository]="$digest"
done
OPERATOR_IMAGE_DIGEST=${image_digests[operator]}
BUNDLE_IMAGE_DIGEST=${image_digests[operator-bundle]}
export PREVIOUS_VERSION OPERATOR_IMAGE_DIGEST

image="quay.io/openstack-lightspeed/operator-bundle@$BUNDLE_IMAGE_DIGEST"
podman pull "$image"
image_id=$(podman image inspect --format '{{.Id}}' "$image")
# Podman 4 requires a command even though this container is never started.
container_id=$(podman create "$image_id" /bin/true)
trap 'podman rm "$container_id" >/dev/null || true' EXIT
mkdir -p "$release_dir"
for directory in manifests metadata tests; do
  podman cp "$container_id:/$directory" "$release_dir/$directory"
done

yq -i '
  .metadata.name = ("openstack-lightspeed-operator.v" + strenv(NEW_VERSION)) |
  .metadata.annotations.containerImage = ("quay.io/openstack-lightspeed/operator@" + strenv(OPERATOR_IMAGE_DIGEST)) |
  .metadata.annotations.support = "Community" |
  .spec.displayName = "OpenStack Lightspeed (Community)" |
  .spec.replaces = ("openstack-lightspeed-operator.v" + strenv(PREVIOUS_VERSION)) |
  .spec.version = strenv(NEW_VERSION)
' "$release_dir/manifests/openstack-lightspeed-operator.clusterserviceversion.yaml"

cat > "$release_dir/release-config.yaml" <<EOF
catalog_templates:
  - template_name: v4.16.yaml
    channels: [alpha]
    replaces: openstack-lightspeed-operator.v$PREVIOUS_VERSION
  - template_name: v4.18.yaml
    channels: [alpha]
    replaces: openstack-lightspeed-operator.v$PREVIOUS_VERSION
  - template_name: v4.22.yaml
    channels: [alpha]
    replaces: openstack-lightspeed-operator.v$PREVIOUS_VERSION
EOF
echo "operator_digest=$OPERATOR_IMAGE_DIGEST" >> "$GITHUB_OUTPUT"
echo "bundle_digest=$BUNDLE_IMAGE_DIGEST" >> "$GITHUB_OUTPUT"
