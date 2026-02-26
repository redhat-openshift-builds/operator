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

# Snapshot name to fetch from OpenShift
SNAPSHOT_NAME=""

# Add at the top, after RHEL_SUFFIX and before IMAGE_MAPPING
IMAGE_PREFIX="${IMAGE_PREFIX:-quay.io/redhat-user-workloads/rh-openshift-builds-tenant/}"

# Add option to use snapshot image names
USE_SNAPSHOT_NAMES=false

# New associative array to store snapshot image names (without digest)
declare -A SNAPSHOT_IMAGE_NAME=()

# Environment variables for image names (with defaults matching current CSV at this time)
# These can be overridden if image names change in the future
IMAGE_OPERATOR="${IMAGE_OPERATOR:-openshift-builds-rhel9-operator}"
IMAGE_CONTROLLER="${IMAGE_CONTROLLER:-openshift-builds-controller-rhel9}"
IMAGE_GIT_CLONER="${IMAGE_GIT_CLONER:-openshift-builds-git-cloner-rhel9}"
IMAGE_PROCESSING="${IMAGE_PROCESSING:-openshift-builds-image-processing-rhel9}"
IMAGE_BUNDLER="${IMAGE_BUNDLER:-openshift-builds-image-bundler-rhel9}"
IMAGE_WAITERS="${IMAGE_WAITERS:-openshift-builds-waiters-rhel9}"
IMAGE_WEBHOOK="${IMAGE_WEBHOOK:-openshift-builds-webhook-rhel9}"
IMAGE_SHARED_RESOURCE_WEBHOOK="${IMAGE_SHARED_RESOURCE_WEBHOOK:-openshift-builds-shared-resource-webhook-rhel9}"
IMAGE_SHARED_RESOURCE="${IMAGE_SHARED_RESOURCE:-openshift-builds-shared-resource-rhel9}"

# Define a mapping from internal names to manager.yaml environment variable names
# This is used to ensure correct updates regardless of how digests are fetched
declare -A INTERNAL_TO_MANAGER_ENV=(
    [controller]="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD"
    [git-cloner]="IMAGE_SHIPWRIGHT_GIT_CONTAINER_IMAGE"
    [image-processing]="IMAGE_SHIPWRIGHT_IMAGE_PROCESSING_CONTAINER_IMAGE"
    [image-bundler]="IMAGE_SHIPWRIGHT_BUNDLE_CONTAINER_IMAGE"
    [waiters]="IMAGE_SHIPWRIGHT_WAITER_CONTAINER_IMAGE"
    [webhook]="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD_WEBHOOK"
)

# Parse command line arguments early to set IMAGE_PREFIX if provided
while getopts "hS:P:N" opt; do
    case $opt in
        P) IMAGE_PREFIX="$OPTARG";;
        N) USE_SNAPSHOT_NAMES=true;;
    esac
done
OPTIND=1

# Define the mapping between internal names
# These names must match the actual image names in the CSV file (with -rhel9 suffix)
declare -A IMAGE_MAPPING=(
    ["operator"]="$IMAGE_OPERATOR"
    ["controller"]="$IMAGE_CONTROLLER"
    ["git-cloner"]="$IMAGE_GIT_CLONER"
    ["image-processing"]="$IMAGE_PROCESSING"
    ["image-bundler"]="$IMAGE_BUNDLER"
    ["waiters"]="$IMAGE_WAITERS"
    ["webhook"]="$IMAGE_WEBHOOK"
    ["shared-resource-webhook"]="$IMAGE_SHARED_RESOURCE_WEBHOOK"
    ["shared-resource"]="$IMAGE_SHARED_RESOURCE"
)

# Define a separate mapping for Konflux Snapshot component names.
# This explicitly tells the script what to look for in the Konflux snapshot's 'name' field
# given the internal name used in IMAGE_MAPPING.
declare -A KONFLUX_COMPONENT_NAME_MAPPING=(
    ["operator"]="openshift-builds-operator"
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
    echo "  -P <prefix>     Image prefix (e.g., 'quay.io/org/'). Default: (no prefix, use as in mapping)"
    echo "  -N              Use snapshot image names instead of the ones from the CSV"
    echo "  -h              Show this help message"
    echo ""
    echo "Environment variables can also be used to set IMAGE_PREFIX or SNAPSHOT_NAME directly."
    exit 1
}

# Parse command line arguments
while getopts "hS:P:N" opt; do
    case $opt in
        S) SNAPSHOT_NAME="$OPTARG";;
        P) IMAGE_PREFIX="$OPTARG";;
        N) USE_SNAPSHOT_NAMES=true;;
        h) show_help;;
        ?) show_help;; # Catches unknown options
    esac
done

# Check if IMAGE_PREFIX is set from env var if not from opt
if [ -z "$IMAGE_PREFIX" ] && [ -n "$_IMAGE_PREFIX_ENV" ]; then
    IMAGE_PREFIX="$_IMAGE_PREFIX_ENV"
fi

# Fetch digests from the snapshot in OpenShift
function fetch_digests_from_snapshot {
    local snapshot_name=$1

    echo "Attempting to fetch digests from Konflux Snapshot: '${snapshot_name}'..."

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

    # Debug: Print all component names in the snapshot
    echo "Snapshot component names found:" >&2
    echo "$snapshot_json" | jq -r '.spec.components[].name' >&2

    local found_any_digest=false

    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local current_digest=""
        local component_name_in_snapshot="${KONFLUX_COMPONENT_NAME_MAPPING[$internal_name]}"
        if [ -z "$component_name_in_snapshot" ]; then
            echo "  - Warning: No Konflux snapshot component name mapping found for internal name: '${internal_name}'. Skipping."
            continue
        fi
        echo "  - Searching for digest for internal component: '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}')..."
        # Improved: match names that start with the mapping string (for versioned names)
        local full_container_image=$(echo "$snapshot_json" | jq -r --arg name_part "$component_name_in_snapshot" \
            '([.spec.components[] | select(.name | test("^"+$name_part))] | if length > 0 then (sort_by(.name | length) | .[0].containerImage) else empty end)' 2>/dev/null)
        if [ -n "$full_container_image" ]; then
            current_digest=$(echo "$full_container_image" | awk -F'@' '{print $2}')
            # Save the image name (without digest)
            local image_name_no_digest=$(echo "$full_container_image" | awk -F'@' '{print $1}')
            SNAPSHOT_IMAGE_NAME[$internal_name]="$image_name_no_digest"
        fi
        if [ -n "$current_digest" ]; then
            local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
            eval "${digest_var_name}='${current_digest}'"
            found_any_digest=true
            echo "    -> Found '${internal_name}' digest: ${current_digest}"
            if [ -n "$image_name_no_digest" ]; then
                echo "    -> Found '${internal_name}' image name from snapshot: ${image_name_no_digest}"
            fi
        else
            echo "    -> Warning: Could not find or extract digest for '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}') in the snapshot. Skipping."
            echo "    -> (Debug) No match for '${component_name_in_snapshot}' in snapshot component names."
        fi
    done
    # Debug: Print all found digests and image names
    echo "--- Debug: Final digest and image name mapping after snapshot fetch ---" >&2
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        local image_name="${SNAPSHOT_IMAGE_NAME[$internal_name]}"
        echo "  $internal_name: digest=$current_digest, image_name=$image_name" >&2
    done
    echo "Digest fetching from snapshot complete."
    if ! $found_any_digest; then
        echo "Error: No digests could be fetched from the snapshot '$snapshot_name'. Please check the snapshot content and component names."
        exit 1
    fi
}

# Fetch digests from a JSON string in the SNAPSHOT env var
function fetch_digests_from_json_snapshot {
    local snapshot_json="$1"
    echo "Attempting to fetch digests from SNAPSHOT JSON string..."
    local found_any_digest=false

    for internal_name in "${!KONFLUX_COMPONENT_NAME_MAPPING[@]}"; do
        local component_name_in_snapshot="${KONFLUX_COMPONENT_NAME_MAPPING[$internal_name]}"

        if [ -z "$component_name_in_snapshot" ]; then
            echo "  - Warning: No Konflux snapshot component name mapping found for internal name: '${internal_name}'. Skipping."
            continue
        fi
        echo "  - Searching for digest for internal component: '${internal_name}' (Konflux component search string: '${component_name_in_snapshot}')..."

        # Find the component in the JSON with a matching name (case-insensitive, dashes/underscores normalized, prefix match)
        local full_container_image=$(echo "$snapshot_json" | jq -r --arg name_part "$component_name_in_snapshot" '
            ([.components[] | select(.name | ascii_downcase | gsub("_"; "-") | test("^"+$name_part))] | if length > 0 then (sort_by(.name | length) | .[0].containerImage) else empty end)' 2>/dev/null)

        if [ -n "$full_container_image" ]; then
            local current_digest=$(echo "$full_container_image" | awk -F'@' '{print $2}')
            # Save the image name (without digest)
            local image_name_no_digest=$(echo "$full_container_image" | awk -F'@' '{print $1}')
            
            if [ -n "$image_name_no_digest" ]; then
                SNAPSHOT_IMAGE_NAME[$internal_name]="$image_name_no_digest"
            fi
            
            if [ -n "$current_digest" ]; then
                local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
                eval "${digest_var_name}='${current_digest}'"
                found_any_digest=true
                echo "    -> Found '$internal_name' digest: $current_digest (set $digest_var_name)"
                if [ -n "$image_name_no_digest" ]; then
                    echo "    -> Found '$internal_name' image name from snapshot: $image_name_no_digest (set SNAPSHOT_IMAGE_NAME[$internal_name])"
                fi
            else
                echo "    -> Warning: Could not extract digest for '$internal_name' in the JSON. Skipping."
            fi
        else
            echo "    -> Warning: Could not find component for '$internal_name' (normalized: $component_name_in_snapshot) in the JSON. Skipping."
        fi
    done
    
    echo "Digest fetching from SNAPSHOT JSON complete."
    if ! $found_any_digest; then
        echo "Error: No digests could be fetched from the SNAPSHOT JSON. Please check the content and component names."
        exit 1
    fi
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
            local image_path="${IMAGE_MAPPING[$internal_name]}"
            local full_image_path="$image_path"
            if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[$internal_name]}" ]; then
                full_image_path="${SNAPSHOT_IMAGE_NAME[$internal_name]}"
            elif [ -n "$IMAGE_PREFIX" ]; then
                full_image_path="${IMAGE_PREFIX}$(echo "$image_path" | sed 's#^[^/]*/##')"
            fi
            # Extract just the image name (after last /)
            image_name=$(basename "$image_path")
            # Regex: match any registry/org, then the image name, then @sha256
            sed "${SED_INPLACE[@]}" -E "/^  relatedImages:/,/^[^ ]/s#(image: )([^@]*/)?${image_name}@sha256:[a-f0-9]+#\\1${full_image_path}@${current_digest}#g" "$csv_file"
            echo "  - Updated ${internal_name} image and digest in CSV to ${full_image_path}@${current_digest}"
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
        local full_operator_image_path="$operator_image_path"
        if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[operator]}" ]; then
            full_operator_image_path="${SNAPSHOT_IMAGE_NAME[operator]}"
        fi
        local escaped_operator_image_path=$(echo "$full_operator_image_path" | sed 's/\//\\\//g')
        sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*${escaped_operator_image_path})@sha256:[a-f0-9]+#\1@${operator_digest}#g" "$manager_file"
        echo "  - Updated operator image in manager.yaml to digest ${operator_digest}"
    else
        echo "  - Info: No digest found for 'operator'. Skipping direct image update in $manager_file."
    fi
    # Update images specified in environment variables (IMAGE_SHIPWRIGHT_*)
    for internal_name in "${!INTERNAL_TO_MANAGER_ENV[@]}"; do
        env_var_name="${INTERNAL_TO_MANAGER_ENV[$internal_name]}"
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        if [ -z "$env_var_name" ]; then
            echo "  - Info: Internal name '${internal_name}' does not map to an environment variable in manager.yaml. Skipping."
            continue
        fi
        if [ -n "$current_digest" ]; then
            local image_path="${IMAGE_MAPPING[$internal_name]}"
            local full_image_path="$image_path"
            if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[$internal_name]}" ]; then
                full_image_path="${SNAPSHOT_IMAGE_NAME[$internal_name]}"
            elif [ -n "$IMAGE_PREFIX" ]; then
                full_image_path="${IMAGE_PREFIX}$(echo "$image_path" | sed 's#^[^/]*/##')"
            fi
            if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[$internal_name]}" ]; then
                # Replace the entire value line with the new image@digest
                sed "${SED_INPLACE[@]}" -E "/- name: ${env_var_name}/ {n; s#(^[[:space:]]*value:[[:space:]]*).*#\\1${full_image_path}@${current_digest}#;}" "$manager_file"
                echo "  - Updated ${env_var_name} image and digest in manager.yaml to ${full_image_path}@${current_digest}"
            else
                # Always replace the entire value line regardless of registry/image name
                sed "${SED_INPLACE[@]}" -E "/- name: ${env_var_name}/ {n; s#(^[[:space:]]*value:[[:space:]]*).*#\\1${full_image_path}@${current_digest}#;}" "$manager_file"
                echo "  - Updated ${env_var_name} image and digest in manager.yaml to ${full_image_path}@${current_digest}"
            fi
        else
            echo "  - Info: No digest found for ${internal_name}. Skipping update for this image in $manager_file."
        fi
    done
}

# Update images in generic Kubernetes resource files
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
            local full_image_path="$image_path"
            # IMAGE_PREFIX takes priority over snapshot image names
            if [ -n "$IMAGE_PREFIX" ]; then
                full_image_path="${IMAGE_PREFIX}${image_path}"
            elif [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[$internal_image_name]}" ]; then
                full_image_path="${SNAPSHOT_IMAGE_NAME[$internal_image_name]}"
            fi
            # Only replace image lines that contain the target image name pattern
            # Match images ending with the image name (e.g., shared-resource-rhel9) before @sha256
            sed "${SED_INPLACE[@]}" -E "s#(image: )[^@]*${image_path}@sha256:[a-f0-9]+#\\1${full_image_path}@${current_digest}#g" "$resource_file"
            echo "  - Updated ${internal_image_name} image and digest in $resource_file to ${full_image_path}@${current_digest}"
        else
            echo "  - Info: No digest found for ${internal_image_name}. Skipping update for this image in $resource_file."
        fi
    done
}

# Update manager kustomization
function update_manager_kustomization {
    local kustomization_file=$1

    if [ -z "$OPERATOR_DIGEST" ]; then
        echo "No OPERATOR_DIGEST set, skipping digest update in $kustomization_file."
        return 0
    fi

    # Use yq if available for robust YAML editing
    if command -v yq >/dev/null 2>&1; then
        # Remove newTag for operator
        yq eval 'del(.images[] | select(.name == "operator").newTag)' -i "$kustomization_file"
        # Set digest for operator
        OPERATOR_DIGEST="$OPERATOR_DIGEST" yq eval '(.images[] | select(.name == "operator")).digest = env(OPERATOR_DIGEST)' -i "$kustomization_file"
        # Always set newName based on -N and -P flags
        local operator_image_path="${IMAGE_MAPPING["operator"]}"
        local full_operator_image_path="$operator_image_path"
        if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[operator]}" ]; then
            full_operator_image_path="${SNAPSHOT_IMAGE_NAME[operator]}"
        elif [ -n "$IMAGE_PREFIX" ]; then
            full_operator_image_path="${IMAGE_PREFIX}$(echo "$operator_image_path" | sed 's#^[^/]*/##')"
        fi
        OPERATOR_IMAGE_NAME="$full_operator_image_path" yq eval '(.images[] | select(.name == "operator")).newName = env(OPERATOR_IMAGE_NAME)' -i "$kustomization_file"
        echo "  - Removed newTag and set digest and newName for operator image in $kustomization_file (using yq)"
    else
        # Fallback: Use awk/sed (less robust, assumes standard structure)
        # Remove any existing digest line for operator
        sed "${SED_INPLACE[@]}" "/- name: operator/{n; n; n; /  digest:/d;}" "$kustomization_file"
        # Remove newTag for operator
        awk '
        BEGIN {in_operator=0}
        /- name: operator/ {in_operator=1; print; next}
        in_operator==1 && /newTag:/ {next}
        in_operator==1 && /digest:/ {in_operator=0}
        {print}
        ' "$kustomization_file" > "$kustomization_file.tmp" && mv "$kustomization_file.tmp" "$kustomization_file"
        # Insert digest after name for operator
        awk -v digest="$OPERATOR_DIGEST" '
        BEGIN {in_operator=0}
        /- name: operator/ {in_operator=1; print; next}
        in_operator==1 {
            print "  digest: " digest;
            in_operator=0;
            next
        }
        {print}
        ' "$kustomization_file" > "$kustomization_file.tmp2" && mv "$kustomization_file.tmp2" "$kustomization_file"
        # Always update newName based on -N and -P flags
        local operator_image_path="${IMAGE_MAPPING["operator"]}"
        local full_operator_image_path="$operator_image_path"
        if [ "$USE_SNAPSHOT_NAMES" = true ] && [ -n "${SNAPSHOT_IMAGE_NAME[operator]}" ]; then
            full_operator_image_path="${SNAPSHOT_IMAGE_NAME[operator]}"
        elif [ -n "$IMAGE_PREFIX" ]; then
            full_operator_image_path="${IMAGE_PREFIX}$(echo "$operator_image_path" | sed 's#^[^/]*/##')"
        fi
        awk -v newname="$full_operator_image_path" '
        BEGIN {in_operator=0}
        /- name: operator/ {in_operator=1; print; next}
        in_operator==1 && /newName:/ {print "  newName: " newname; in_operator=0; next}
        in_operator==1 && !/newName:/ {print "  newName: " newname; in_operator=0; next}
        {print}
        ' "$kustomization_file" > "$kustomization_file.tmp3" && mv "$kustomization_file.tmp3" "$kustomization_file"
        echo "  - Removed newTag and set digest and newName for operator image in $kustomization_file (using awk/sed)"
    fi
}

# --- Main validation and digest fetching ---
if [ -z "$SNAPSHOT_NAME" ] && [ -z "$SNAPSHOT" ]; then
    echo "Error: Either Snapshot name (-S or SNAPSHOT_NAME env var) or SNAPSHOT env var (JSON string) is required to fetch image digests."
    show_help
fi

# If SNAPSHOT env var is set and non-empty, use it as a JSON string; otherwise, fetch from OpenShift
if [ -n "$SNAPSHOT" ]; then
    if ! command -v jq &> /dev/null; then
        echo "Error: 'jq' not found. It is required to parse the SNAPSHOT JSON string. Please install jq."
        exit 1
    fi
    fetch_digests_from_json_snapshot "$SNAPSHOT"
    # Also update these files after fetching from JSON
    update_manager_kustomization "$MANAGER_KUSTOMIZATION_FILE"
    update_generic_k8s_resource_images "$SHARED_RESOURCE_DAEMONSET_FILE" "shared-resource"
    update_generic_k8s_resource_images "$SHARED_RESOURCE_WEBHOOK_FILE" "shared-resource-webhook"
    update_manager_yaml "$MANAGER_FILE"
else
    fetch_digests_from_snapshot "$SNAPSHOT_NAME"
    update_manager_kustomization "$MANAGER_KUSTOMIZATION_FILE"
    update_generic_k8s_resource_images "$SHARED_RESOURCE_DAEMONSET_FILE" "shared-resource"
    update_generic_k8s_resource_images "$SHARED_RESOURCE_WEBHOOK_FILE" "shared-resource-webhook"
    update_manager_yaml "$MANAGER_FILE"
fi

# Main execution
echo "Starting image reference update..."

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