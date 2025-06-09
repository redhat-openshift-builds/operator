#!/bin/bash

set -e

# Files for updating
CSV_FILE="./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml"
MANIFEST_BAESES_FILE="./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
MANAGER_FILE="./config/manager/manager.yaml"
MANAGER_KUSTOMIZATION_FILE="./config/manager/kustomization.yaml"
SHARED_RESOURCE_DAEMONSET_FILE="./config/sharedresource/node_daemonset.yaml"
SHARED_RESOURCE_WEBHOOK_FILE="./config/sharedresource/webhook_deployment.yaml"

# Registry to use for fetching digests (still needed for constructing full image paths)
REGISTRY="registry.redhat.io"

# Variable for the version (can be set by -V or VERSION env var)
VERSION=""
# RHEL image name suffix that is in the csv file
RHEL_SUFFIX="-rhel9"

# Path to the Konflux Snapshot JSON file
SNAPSHOT_FILE=""

# Parse command line arguments early to set RHEL_SUFFIX if provided
while getopts "s:V:R:hS:" opt; do
    case $opt in
        R) RHEL_SUFFIX="$OPTARG";;
        # We need to capture RHEL_SUFFIX first to make IMAGE_MAPPING dynamic
    esac
done
# Reset getopts to allow re-parsing from the beginning
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
    echo "Digests are fetched from a Konflux Snapshot JSON file."
    echo ""
    echo "Options:"
    echo "  -S <file>       Path to the Konflux Snapshot JSON file. REQUIRED."
    echo "  -V <version>    Image version to update in CSV name (e.g., '1.2.2'). REQUIRED."
    echo "  -s <registry>   Registry to match for in the csv file (default: $REGISTRY or REGISTRY env var)"
    echo "  -R <suffix>     RHEL image name suffix (e.g., '-rhel9'). Default: $RHEL_SUFFIX"
    echo "  -h              Show this help message"
    echo ""
    echo "Environment variables can also be used to set VERSION, REGISTRY, RHEL_SUFFIX, or SNAPSHOT_FILE directly."
    exit 1
}

# Parse command line arguments
while getopts "s:V:R:hS:" opt; do
    case $opt in
        s) REGISTRY="$OPTARG";;
        V) VERSION="$OPTARG";;
        R) RHEL_SUFFIX="$OPTARG";;
        S) SNAPSHOT_FILE="$OPTARG";;
        h) show_help;;
        ?) show_help;; # Catches unknown options
    esac
done

# Check if VERSION is set from env var if not from opt
if [ -z "$VERSION" ] && [ -n "$_VERSION_ENV" ]; then
    VERSION="$_VERSION_ENV"
fi
# Check if REGISTRY is set from env var if not from opt
if [ -z "$REGISTRY" ] && [ -n "$_REGISTRY_ENV" ]; then
    REGISTRY="$_REGISTRY_ENV"
fi
# Check if RHEL_SUFFIX is set from env var if not from opt
if [ -z "$RHEL_SUFFIX" ] && [ -n "$_RHEL_SUFFIX_ENV" ]; then
    RHEL_SUFFIX="$_RHEL_SUFFIX_ENV"
fi
# Check if SNAPSHOT_FILE is set from env var if not from opt
if [ -z "$SNAPSHOT_FILE" ] && [ -n "$_SNAPSHOT_FILE_ENV" ]; then
    SNAPSHOT_FILE="$_SNAPSHOT_FILE_ENV"
fi


# Fetch digests from the snapshot JSON file
function fetch_digests_from_snapshot {
    local snapshot_path=$1

    echo "Attempting to fetch digests from Konflux Snapshot file: '${snapshot_path}' (RHEL suffix: '${RHEL_SUFFIX}')..."

    # Check for jq
    if ! command -v jq &> /dev/null; then
        echo "Error: 'jq' not found. It is required to parse the snapshot JSON. Please install it (e.g., 'dnf install jq')."
        exit 1
    fi

    if [ ! -f "$snapshot_path" ]; then
        echo "Error: Snapshot file '$snapshot_path' not found."
        exit 1
    fi

    local snapshot_json=$(cat "$snapshot_path")
    local found_any_digest=false

    # Iterate over the keys of IMAGE_MAPPING (which are our internal names)
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local current_digest="" # Ensures current_digest is reset for each iteration

        # Get the corresponding component name from the KONFLUX_COMPONENT_NAME_MAPPING
        local component_name_in_snapshot="${KONFLUX_COMPONENT_NAME_MAPPING[$internal_name]}"

        if [ -z "$component_name_in_snapshot" ]; then
            echo "  - Warning: No Konflux snapshot component name mapping found for internal name: '${internal_name}'. Skipping."
            continue
        fi

        echo "  - Searching for digest for internal component: '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}')..."

        # Corrected jq query:
        # 1. select(.name | contains($name_part)) finds components whose name includes the search string.
        # 2. sort_by(.name | length) | .[0] ensures we pick the most specific match if multiple exist.
        local full_container_image=$(echo "$snapshot_json" | jq -r --arg name_part "$component_name_in_snapshot" \
            '([.spec.components[] | select(.name | contains($name_part))] | if length > 0 then (sort_by(.name | length) | .[0].containerImage) else empty end)' 2>/dev/null)
            
        if [ -n "$full_container_image" ]; then
            # Extract the digest from the full_container_image string.
            # Example: quay.io/.../image@sha256:digest -> sha256:digest
            current_digest=$(echo "$full_container_image" | awk -F'@' '{print $2}')
        fi

        if [ -n "$current_digest" ]; then
            local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
            eval "${digest_var_name}='${current_digest}'" # Use eval for indirect variable assignment
            found_any_digest=true
            echo "    -> Found '${internal_name}' digest: ${current_digest}"
        else
            echo "    -> Warning: Could not find or extract digest for '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}') in the snapshot. Skipping."
        fi
    done
    echo "Digest fetching from snapshot complete."

    if ! $found_any_digest; then
        echo "Error: No digests could be fetched from the snapshot file '${snapshot_path}'. Please check the file content and component names."
        exit 1
    fi
}

# --- Main validation and digest fetching ---
if [ -z "$VERSION" ]; then
    echo "Error: Version (-V or VERSION env var) is required to update the CSV name."
    show_help
fi

if [ -z "$SNAPSHOT_FILE" ]; then
    echo "Error: Snapshot file (-S or SNAPSHOT_FILE env var) is required to fetch image digests."
    show_help
fi

# Call fetch_digests_from_snapshot to populate the global digest variables
fetch_digests_from_snapshot "$SNAPSHOT_FILE"

function update_operator_csv_version {
    local csv_file=$1
    local version=$2

    echo "Updating operator CSV version in $csv_file to $version..."

    # Replace the version in the 'name' field.
    # Example: name: openshift-builds-operator.v0.0.1 -> name: openshift-builds-operator.v1.2.3
    sed -i "s/^\(\s*name: openshift-builds-operator\.v\)[^ ]*/\1${version}/" "$csv_file"
    echo "  - Updated operator name to openshift-builds-operator.v${version}"
}

# --- Update relatedImages in CSV ---
function update_related_images {
    local csv_file=$1
    local registry=$2

    echo "Updating images in $csv_file..."

    # Explicitly check if the 'relatedImages' block exists
    if ! grep -qE '^\s*relatedImages:' "$csv_file"; then
        echo "Warning: 'relatedImages' block not found in $csv_file. Skipping CSV update."
        return 0 # Exit function gracefully if block isn't found
    fi

    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        # full_image_path_after_registry already includes the RHEL_SUFFIX if applicable
        local full_image_path_after_registry="${IMAGE_MAPPING[$internal_name]}"

        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"

        if [ -n "$current_digest" ]; then
            local new_image_full_ref="${registry}/${full_image_path_after_registry}@${current_digest}"

            # Escape '/' characters in the image path for sed
            local escaped_full_image_path_after_registry=$(echo "$full_image_path_after_registry" | sed 's/\//\\\//g')
            local escaped_new_image_full_ref=$(echo "$new_image_full_ref" | sed 's/\//\\\//g')
            local escaped_registry=$(echo "$registry" | sed 's/\//\\\//g')

            # matches the specific image basename and name tag within the relatedImages block
            sed -i "/^  relatedImages:/,/^[^ ]/s|\(^\s*- image: \)${escaped_registry}\/${escaped_full_image_path_after_registry}@.*|\1${escaped_new_image_full_ref}|" "$csv_file"
            echo "  - Updated ${internal_name} in CSV: to ${new_image_full_ref}"
        fi
    done
}

# Update manager.yaml
function update_manager_yaml {
    local manager_file=$1
    local registry=$2

    echo "Updating images in $manager_file..."

    # Check if the file exists
    if [ ! -f "$manager_file" ]; then
        echo "Warning: $manager_file not found. Skipping..."
        return
    fi

    # Handle the 'operator' image specifically, as it's a direct image reference, not an env var.
    local operator_image_path_after_registry="${IMAGE_MAPPING["operator"]}"
    local operator_digest="${OPERATOR_DIGEST}"

    if [ -n "$operator_digest" ]; then
        local new_operator_image_full_ref="${registry}/${operator_image_path_after_registry}@${operator_digest}"
        local escaped_operator_image_path_after_registry=$(echo "$operator_image_path_after_registry" | sed 's/\//\\\//g')
        local escaped_new_operator_image_full_ref=$(echo "$new_operator_image_full_ref" | sed 's/\//\\\//g')
        local escaped_registry=$(echo "$registry" | sed 's/\//\\\//g')

        # This sed command targets the 'image: operator:latest' line directly
        # It looks for 'image: operator:latest' or 'image: operator@sha256:...'
        sed -i -E "s#^(\\s*image: )${escaped_registry}\\/${escaped_operator_image_path_after_registry}(:.*|@.*)?#\\1${escaped_new_operator_image_full_ref}#g" "$manager_file"
        echo "  - Updated operator image in manager.yaml to ${new_operator_image_full_ref}"
    else
        echo "  - Info: No digest found for 'operator'. Skipping direct image update in $manager_file."
    fi

    # Update images specified in environment variables (IMAGE_SHIPWRIGHT_*)
    # These are nested under 'env:' blocks, with 'value:' holding the image reference.
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        # The 'operator' image was handled above.
        if [[ "$internal_name" == "operator" ]]; then
            continue
        fi

        local full_image_path_after_registry="${IMAGE_MAPPING[$internal_name]}"
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
            local new_env_value="${registry}/${full_image_path_after_registry}@${current_digest}"

            local escaped_full_image_path_after_registry=$(echo "$full_image_path_after_registry" | sed 's/\//\\\//g')
            local escaped_new_env_value=$(echo "$new_env_value" | sed 's/\//\\\//g')
            local escaped_registry=$(echo "$registry" | sed 's/\//\\\//g')
            
            sed -i -E "/- name: ${env_var_name}/{n;s#^(\\s*value: )${escaped_registry}\\/${escaped_full_image_path_after_registry}(:.*|@.*)?#\\1${escaped_new_env_value}#;}" "$manager_file"
            echo "  - Updated ${env_var_name} in manager.yaml: to ${new_env_value}"
        else
            echo "  - Info: No digest found for ${internal_name}. Skipping update for this image in $manager_file."
        fi
    done
}

# --- Function to update images in generic Kubernetes resource files ---
function update_generic_k8s_resource_images {
    local resource_file=$1
    local registry=$2
    shift 2 # Shift off resource_file and registry
    
    echo "Updating images in $resource_file..."

    if [ ! -f "$resource_file" ]; then
        echo "Warning: $resource_file not found. Skipping..."
        return
    fi

    local internal_image_name
    for internal_image_name in "$@"; do # Iterate over remaining arguments (internal image names)
        if [[ -z "${IMAGE_MAPPING[$internal_image_name]}" ]]; then
            echo "Warning: Image key '$internal_image_name' not found in IMAGE_MAPPING. Skipping for $resource_file."
            continue
        fi

        # Get the target image path with RHEL_SUFFIX from IMAGE_MAPPING for replacement
        local target_image_path_with_suffix="${IMAGE_MAPPING[$internal_image_name]}"
        local digest_var_name=$(echo "$internal_image_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}" # Indirect expansion

        if [ -n "$current_digest" ]; then
            local new_image_full_ref="${registry}/${target_image_path_with_suffix}@${current_digest}"

            # Escape characters for sed. Using '#' as delimiter.
            local escaped_registry=$(echo "$registry" | sed 's/[#]/\\#/g') # Escape # if registry could contain it
            local escaped_image_path_to_match_in_yaml=$(echo "$target_image_path_with_suffix" | sed 's/[#]/\\#/g') # Escape #
            local escaped_new_image_full_ref=$(echo "$new_image_full_ref" | sed 's/[#]/\\#/g') # Escape #

            # Match against the full image path including RHEL suffix, using '#' as delimiter
            sed -i -E "s#^(\s*(-\s+)?image:\s*)${escaped_registry}/${escaped_image_path_to_match_in_yaml}(:.*|@.*)?#\1${escaped_new_image_full_ref}#g" "$resource_file"

            echo "  - Updated ${internal_image_name} in $resource_file to ${new_image_full_ref}"
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
        sed -i "/- name: operator/{n; n; n; /  digest:/d;}" "$kustomization_file"
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
update_related_images "$CSV_FILE" "$REGISTRY"
update_related_images "$MANIFEST_BAESES_FILE" "$REGISTRY"

update_manager_kustomization "$MANAGER_KUSTOMIZATION_FILE"

# Update shared resource daemonset files
update_generic_k8s_resource_images "$SHARED_RESOURCE_DAEMONSET_FILE" "$REGISTRY" "shared-resource"
update_generic_k8s_resource_images "$SHARED_RESOURCE_WEBHOOK_FILE" "$REGISTRY" "shared-resource-webhook"

# Update manager.yaml file
update_manager_yaml "$MANAGER_FILE" "$REGISTRY"


echo "YAML files have been updated successfully!"
echo "To generate a test bundle, run: make bundle"