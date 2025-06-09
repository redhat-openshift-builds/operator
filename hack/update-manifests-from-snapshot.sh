#!/bin/bash

set -e

# Set SED_INPLACE for cross-platform compatibility (Linux/macOS)
if [[ "$(uname)" == "Darwin" ]]; then
  SED_INPLACE=(-i '')
else
  SED_INPLACE=(-i)
fi

# Files for updating
CSV_FILE="./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml"
MANIFEST_BAESES_FILE="./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
MANAGER_FILE="./config/manager/manager.yaml"
MANAGER_KUSTOMIZATION_FILE="./config/manager/kustomization.yaml"
SHARED_RESOURCE_DAEMONSET_FILE="./config/sharedresource/node_daemonset.yaml"
SHARED_RESOURCE_WEBHOOK_FILE="./config/sharedresource/webhook_deployment.yaml"

# Variable for the version (can be set by -V or VERSION env var)
VERSION=""
# RHEL image name suffix that is in the csv file
RHEL_SUFFIX="-rhel9"

# Snapshot name to fetch from OpenShift
SNAPSHOT_NAME=""

# Parse command line arguments early to set RHEL_SUFFIX if provided
while getopts "V:R:hS:" opt; do
    case $opt in
        R) RHEL_SUFFIX="$OPTARG";;
    esac
done
OPTIND=1

# Define the mapping between internal names
declare -A IMAGE_MAPPING=(
    ["operator"]="openshift-builds/openshift-builds${RHEL_SUFFIX}-operator"
    ["controller"]="openshift-builds/openshift-builds-controller${RHEL_SUFFIX}"
    ["git-cloner"]="openshift-builds/openshift-builds-git-cloner${RHEL_SUFFIX}"
    ["image-processing"]="openshift-builds/openshift-builds-image-processing${RHEL_SUFFIX}"
    ["image-bundler"]="openshift-builds/openshift-builds-image-bundler${RHEL_SUFFIX}"
    ["waiters"]="openshift-builds/openshift-builds-waiters${RHEL_SUFFIX}"
    ["webhook"]="openshift-builds/openshift-builds-webhook${RHEL_SUFFIX}"
    ["shared-resource-webhook"]="openshift-builds/openshift-builds-shared-resource-webhook${RHEL_SUFFIX}"
    ["shared-resource"]="openshift-builds/openshift-builds-shared-resource${RHEL_SUFFIX}"
)

# Define a separate mapping for Konflux Snapshot component names.
# This explicitly tells the script what to look for in the Konflux snapshot's 'name' field
# given the internal name used in IMAGE_MAPPING.
declare -A KONFLUX_COMPONENT_NAME_MAPPING=(
    ["operator"]="openshift-builds-operator-bundle"
    ["controller"]="openshift-builds-controller"
    ["git-cloner"]="openshift-builds-git-cloner"
    ["image-processing"]="openshift-builds-image-processing"
    ["image-bundler"]="openshift-builds-image-bundler"
    ["waiters"]="openshift-builds-waiter" # Snapshot has 'waiter' (singular)
    ["webhook"]="openshift-builds-webhook"
    ["shared-resource-webhook"]="openshift-builds-shared-resource-webhook"
    ["shared-resource"]="openshift-builds-shared-resource"
)


# Initialize digest variables to be empty. They will be populated by fetch_digests_from_snapshot.
OPERATOR_DIGEST=""
CONTROLLER_DIGEST=""
GIT_CLONER_DIGEST=""
IMAGE_PROCESSING_DIGEST=""
IMAGE_BUNDLER_DIGEST=""
WAITER_DIGEST=""
WEBHOOK_DIGEST=""
SHARED_RESOURCE_WEBHOOK_DIGEST=""
SHARED_RESOURCE_DIGEST=""


# Help message
function show_help {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Updates image digests in the ClusterServiceVersion (CSV) and manager.yaml files."
    echo "Digests are fetched from a Konflux Snapshot in OpenShift."
    echo ""
    echo "Options:"
    echo "  -S <name>       Name of the Konflux Snapshot in OpenShift. REQUIRED."
    echo "  -V <version>    Image version to update in CSV name (e.g., '1.2.2'). REQUIRED."
    echo "  -R <suffix>     RHEL image name suffix (e.g., '-rhel9'). Default: $RHEL_SUFFIX"
    echo "  -h              Show this help message"
    echo ""
    echo "Environment variables can also be used to set VERSION, RHEL_SUFFIX, or SNAPSHOT_NAME directly."
    exit 1
}

# Parse command line arguments
while getopts "V:R:hS:" opt; do
    case $opt in
        V) VERSION="$OPTARG";;
        R) RHEL_SUFFIX="$OPTARG";;
        S) SNAPSHOT_NAME="$OPTARG";;
        h) show_help;;
        ?) show_help;; # Catches unknown options
    esac
done

# Check if VERSION is set from env var if not from opt
if [ -z "$VERSION" ] && [ -n "$_VERSION_ENV" ]; then
    VERSION="$_VERSION_ENV"
fi
# Check if RHEL_SUFFIX is set from env var if not from opt
if [ -z "$RHEL_SUFFIX" ] && [ -n "_RHEL_SUFFIX_ENV" ]; then
    RHEL_SUFFIX="$_RHEL_SUFFIX_ENV"
fi
# Check if SNAPSHOT_NAME is set from env var if not from opt
if [ -z "$SNAPSHOT_NAME" ] && [ -n "$_SNAPSHOT_NAME_ENV" ]; then
    SNAPSHOT_NAME="$_SNAPSHOT_NAME_ENV"
fi

# Fetch digests from the snapshot in OpenShift
function fetch_digests_from_snapshot {
    local snapshot_name=$1

    echo "Attempting to fetch digests from Konflux Snapshot: '${snapshot_name}' (RHEL suffix: '${RHEL_SUFFIX}')..."

    # Check for oc
    if ! command -v oc &> /dev/null; then
        echo "Error: 'oc' not found. It is required to fetch the snapshot from OpenShift. Please install the OpenShift CLI."
        exit 1
    fi

    # Check if user is logged in to OpenShift
    if ! oc whoami &> /dev/null; then
        echo "Error: Not logged in to OpenShift. Please run 'oc login' first."
        exit 1
    fi

    # Fetch the snapshot JSON from OpenShift
    local snapshot_json
    if ! snapshot_json=$(oc get snapshot "$snapshot_name" -o json 2>/dev/null); then
        echo "Error: Failed to fetch snapshot '$snapshot_name' from OpenShift. Please check the snapshot name and your permissions."
        exit 1
    fi

    local found_any_digest=false

    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local current_digest=""
        local component_name_in_snapshot="${KONFLUX_COMPONENT_NAME_MAPPING[$internal_name]}"
        if [ -z "$component_name_in_snapshot" ]; then
            echo "  - Warning: No Konflux snapshot component name mapping found for internal name: '${internal_name}'. Skipping."
            continue
        fi
        echo "  - Searching for digest for internal component: '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}')..."
        local full_container_image=$(echo "$snapshot_json" | jq -r --arg name_part "$component_name_in_snapshot" \
            '([.spec.components[] | select(.name | contains($name_part))] | if length > 0 then (sort_by(.name | length) | .[0].containerImage) else empty end)' 2>/dev/null)
        if [ -n "$full_container_image" ]; then
            current_digest=$(echo "$full_container_image" | awk -F'@' '{print $2}')
        fi
        if [ -n "$current_digest" ]; then
            local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
            eval "${digest_var_name}='${current_digest}'"
            found_any_digest=true
            echo "    -> Found '${internal_name}' digest: ${current_digest}"
        else
            echo "    -> Warning: Could not find or extract digest for '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}') in the snapshot. Skipping."
        fi
    done
    echo "Digest fetching from snapshot complete."
    if ! $found_any_digest; then
        echo "Error: No digests could be fetched from the snapshot '$snapshot_name'. Please check the snapshot content and component names."
        exit 1
    fi
}

# --- Main validation and digest fetching ---
if [ -z "$VERSION" ]; then
    echo "Error: Version (-V or VERSION env var) is required to update the CSV name."
    show_help
fi

if [ -z "$SNAPSHOT_NAME" ]; then
    echo "Error: Snapshot name (-S or SNAPSHOT_NAME env var) is required to fetch image digests."
    show_help
fi

# Call fetch_digests_from_snapshot to populate the global digest variables
fetch_digests_from_snapshot "$SNAPSHOT_NAME"

function update_operator_csv_version {
    local csv_file=$1
    local version=$2

    echo "Updating operator CSV version in $csv_file to $version..."

    # Replace the version in the 'name' field.
    # Example: name: openshift-builds-operator.v0.0.1 -> name: openshift-builds-operator.v1.2.3
    sed "${SED_INPLACE[@]}" "s/^\(\s*name: openshift-builds-operator\.v\)[^ ]*/\1${version}/" "$csv_file"
    echo "  - Updated operator name to openshift-builds-operator.v${version}"
}

# --- Update relatedImages in CSV ---
function update_related_images {
    local csv_file=$1
    echo "Updating images in $csv_file..."
    if ! grep -qE '^\s*relatedImages:' "$csv_file"; then
        echo "Warning: 'relatedImages' block not found in $csv_file. Skipping CSV update."
        return 0
    fi
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        if [ -n "$current_digest" ]; then
            # Only update the digest part, preserving the registry and image path
            local image_path="${IMAGE_MAPPING[$internal_name]}"
            local escaped_image_path=$(echo "$image_path" | sed 's/\//\\\//g')
            sed "${SED_INPLACE[@]}" -E "/^  relatedImages:/,/^[^ ]/s|(image: [^@]*${escaped_image_path})@sha256:[a-f0-9]+|\1@${current_digest}|g" "$csv_file"
            echo "  - Updated ${internal_name} digest in CSV to ${current_digest}"
        fi
    done
}

# Update manager.yaml
function update_manager_yaml {
    local manager_file=$1
    echo "Updating images in $manager_file..."
    if [ ! -f "$manager_file" ]; then
        echo "Warning: $manager_file not found. Skipping..."
        return
    fi
    # Handle the 'operator' image specifically
    local operator_digest="${OPERATOR_DIGEST}"
    if [ -n "$operator_digest" ]; then
        local operator_image_path="${IMAGE_MAPPING["operator"]}"
        local escaped_operator_image_path=$(echo "$operator_image_path" | sed 's/\//\\\//g')
        sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*${escaped_operator_image_path})@sha256:[a-f0-9]+#\1@${operator_digest}#g" "$manager_file"
        echo "  - Updated operator image in manager.yaml to digest ${operator_digest}"
    else
        echo "  - Info: No digest found for 'operator'. Skipping direct image update in $manager_file."
    fi
    # Update images specified in environment variables (IMAGE_SHIPWRIGHT_*)
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        if [[ "$internal_name" == "operator" ]]; then
            continue
        fi
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        local env_var_name
        case "$internal_name" in
            "controller") env_var_name="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD";;
            "git-cloner") env_var_name="IMAGE_SHIPWRIGHT_GIT_CONTAINER_IMAGE";;
            "image-processing") env_var_name="IMAGE_SHIPWRIGHT_IMAGE_PROCESSING_CONTAINER_IMAGE";;
            "image-bundler") env_var_name="IMAGE_SHIPWRIGHT_BUNDLE_CONTAINER_IMAGE";;
            "waiters") env_var_name="IMAGE_SHIPWRIGHT_WAITER_CONTAINER_IMAGE";;
            "webhook") env_var_name="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD_WEBHOOK";;
            *)
                echo "  - Info: Internal name '${internal_name}' does not map to an environment variable in manager.yaml. Skipping."
                continue
            ;;
        esac
        if [ -n "$current_digest" ]; then
            local image_path="${IMAGE_MAPPING[$internal_name]}"
            local escaped_image_path=$(echo "$image_path" | sed 's/\//\\\//g')
            sed "${SED_INPLACE[@]}" -E "/- name: ${env_var_name}/{n;s#(value: [^@]*${escaped_image_path})@sha256:[a-f0-9]+#\1@${current_digest}#;}" "$manager_file"
            echo "  - Updated ${env_var_name} digest in manager.yaml to ${current_digest}"
        else
            echo "  - Info: No digest found for ${internal_name}. Skipping update for this image in $manager_file."
        fi
    done
}

# --- Function to update images in generic Kubernetes resource files ---
function update_generic_k8s_resource_images {
    local resource_file=$1
    shift 1
    echo "Updating images in $resource_file..."
    if [ ! -f "$resource_file" ]; then
        echo "Warning: $resource_file not found. Skipping..."
        return
    fi
    for internal_image_name in "$@"; do
        if [[ -z "${IMAGE_MAPPING[$internal_image_name]}" ]]; then
            echo "Warning: Image key '$internal_image_name' not found in IMAGE_MAPPING. Skipping for $resource_file."
            continue
        fi
        local digest_var_name=$(echo "$internal_image_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        if [ -n "$current_digest" ]; then
            local image_path="${IMAGE_MAPPING[$internal_image_name]}"
            local escaped_image_path=$(echo "$image_path" | sed 's/\//\\\//g')
            sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*${escaped_image_path})@sha256:[a-f0-9]+#\1@${current_digest}#g" "$resource_file"
            echo "  - Updated ${internal_image_name} digest in $resource_file to ${current_digest}"
        else
            echo "  - Info: No digest found for ${internal_image_name}. Skipping update for this image in $resource_file."
        fi
    done
}

# Update manager kustomization
function update_manager_kustomization {
    local kustomization_file=$1

    # Only update newTag if it already exists for operator
    if [ -n "$VERSION" ]; then
        if command -v yq >/dev/null 2>&1; then
            VERSION="$VERSION" yq eval '(.images[] | select(.name == "operator")).newTag = env(VERSION)' -i "$kustomization_file"
            echo "  - Set newTag for operator image in $kustomization_file to $VERSION (using yq)"
        else
            # Use awk/sed to update newTag for operator
            awk -v version="$VERSION" '
            BEGIN {in_operator=0}
            /- name: operator/ {in_operator=1; print; next}
            in_operator==1 && /newTag:/ {
                print "  newTag: " version;
                in_operator=0;
                next
            }
            {print}
            ' "$kustomization_file" > "$kustomization_file.tmp" && mv "$kustomization_file.tmp" "$kustomization_file"
            echo "  - Set newTag for operator image in $kustomization_file to $VERSION (using awk/sed)"
        fi
    fi

    if [ -z "$OPERATOR_DIGEST" ]; then
        echo "No OPERATOR_DIGEST set, skipping digest update in $kustomization_file."
        return 0
    fi

    # Use yq if available for robust YAML editing
    if command -v yq >/dev/null 2>&1; then
        # Ensure OPERATOR_DIGEST is available to yq's env()
        OPERATOR_DIGEST="$OPERATOR_DIGEST" yq eval '(.images[] | select(.name == "operator")).digest = env(OPERATOR_DIGEST)' -i "$kustomization_file"
        echo "  - Set digest for operator image in $kustomization_file to $OPERATOR_DIGEST (using yq)"
    else
        # Fallback: Use awk/sed (less robust, assumes standard structure)
        # Remove any existing digest line for operator
        sed "${SED_INPLACE[@]}" "/- name: operator/{n; n; n; /  digest:/d;}" "$kustomization_file"
        # Insert digest after newTag for operator
        awk -v digest="$OPERATOR_DIGEST" '
        BEGIN {in_operator=0}
        /- name: operator/ {in_operator=1; print; next}
        in_operator==1 && /newTag:/ {
            print;
            print "  digest: " digest;
            in_operator=0;
            next
        }
        {print}
        ' "$kustomization_file" > "$kustomization_file.tmp" && mv "$kustomization_file.tmp" "$kustomization_file"
        echo "  - Set digest for operator image in $kustomization_file to $OPERATOR_DIGEST (using awk/sed)"
    fi
}


# Main execution
echo "Starting image reference update..."

# Update operator version in CSV name
update_operator_csv_version "$CSV_FILE" "$VERSION"

# Update CSV files
update_related_images "$CSV_FILE"
update_related_images "$MANIFEST_BAESES_FILE"

update_manager_kustomization "$MANAGER_KUSTOMIZATION_FILE"

# Update shared resource daemonset files
update_generic_k8s_resource_images "$SHARED_RESOURCE_DAEMONSET_FILE" "shared-resource"
update_generic_k8s_resource_images "$SHARED_RESOURCE_WEBHOOK_FILE" "shared-resource-webhook"

# Update manager.yaml file
update_manager_yaml "$MANAGER_FILE"


echo "YAML files have been updated successfully!"
echo "To generate a test bundle, run: make bundle"