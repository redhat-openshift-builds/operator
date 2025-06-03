#!/bin/bash

set -e

# Files for updating
CSV_FILE="./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml"
MANIFEST_BAESES_FILE="./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
MANAGER_FILE="./config/manager/manager.yaml"
MANAGER_KUSTOMIZATION_FILE="./config/manager/kustomization.yaml"
SHARED_RESOURCE_DAEMONSET_FILE="./config/sharedresource/node_daemonset.yaml"
SHARED_RESOURCE_WEBHOOK_FILE="./config/sharedresource/webhook_deployment.yaml"

# Registry to use for fetching digests
REGISTRY="registry.redhat.io"

# Variable for the version to fetch digests (can be set by -V or VERSION env var)
VERSION=""
# RHEL image name suffix (can be set by -R or RHEL_SUFFIX env var)
# This MUST be defined before IMAGE_MAPPING uses it.
RHEL_SUFFIX="-rhel9" 

# Parse command line arguments early to set RHEL_SUFFIX if provided
# This is crucial because IMAGE_MAPPING needs RHEL_SUFFIX at declaration time.
# We'll re-parse later for other options after RHEL_SUFFIX is potentially updated.
while getopts "s:V:R:h" opt; do 
    case $opt in
        R) RHEL_SUFFIX="$OPTARG";; # Capture RHEL_SUFFIX here
        # Other options are handled later in the main parsing loop
        # We need to capture RHEL_SUFFIX first to make IMAGE_MAPPING dynamic
    esac
done
# Reset getopts to allow re-parsing from the beginning
OPTIND=1


# Define the mapping between internal names and full image names in CSV
declare -A IMAGE_MAPPING=(
    ["operator"]="openshift-builds/openshift-builds${RHEL_SUFFIX}-operator"
    ["controller"]="openshift-builds/openshift-builds-controller${RHEL_SUFFIX}"
    ["git-cloner"]="openshift-builds/openshift-builds-git-cloner${RHEL_SUFFIX}"
    ["image-processing"]="openshift-builds/openshift-builds-image-processing${RHEL_SUFFIX}"
    ["image-bundler"]="openshift-builds/openshift-builds-image-bundler${RHEL_SUFFIX}"
    ["waiter"]="openshift-builds/openshift-builds-waiters${RHEL_SUFFIX}"
    ["webhook"]="openshift-builds/openshift-builds-webhook${RHEL_SUFFIX}"
    ["shared-resource-webhook"]="openshift-builds/openshift-builds-shared-resource-webhook${RHEL_SUFFIX}"
    ["shared-resource"]="openshift-builds/openshift-builds-shared-resource${RHEL_SUFFIX}"
)

# Initialize digest variables to be empty. They will be populated by fetch_digests_from_registry.
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
    echo "Digests are fetched automatically from the registry based on a provided version."
    echo ""
    echo "Options:"
    echo "  -V <version>  Image version to fetch digests for (e.g., '1.2.2'). REQUIRED."
    echo "                (Requires 'skopeo' and 'jq' to be installed and accessible)."
    echo "  -s <registry> Staging registry (default: $REGISTRY or REGISTRY env var)"
    echo "  -R <suffix>   RHEL image name suffix (e.g., '-rhel9'). Default: $RHEL_SUFFIX"
    echo "  -h            Show this help message"
    echo ""
    echo "Environment variables can also be used to set VERSION, REGISTRY, or RHEL_SUFFIX directly."
    exit 1
}

# Parse command line arguments
while getopts "s:V:R:h" opt; do
    case $opt in
        s) REGISTRY="$OPTARG";;
        V) VERSION="$OPTARG";;
        R) RHEL_SUFFIX="$OPTARG";;
        h) show_help;;
        ?) show_help;; # Catches unknown options
    esac
done

# Fetch digests from registry based on version
function fetch_digests_from_registry {
    local version=$1
    local registry=$2

    echo "Attempting to fetch digests for version '${version}' from '${registry}' (RHEL suffix: '${RHEL_SUFFIX}')..."

    # Check for skopeo and jq
    if ! command -v skopeo &> /dev/null; then
        echo "Error: 'skopeo' not found. It is required to fetch digests by version. Please install it (e.g., 'dnf install skopeo')."
        exit 1
    fi
    if ! command -v jq &> /dev/null; then
        echo "Error: 'jq' not found. It is required to parse skopeo output. Please install it (e.g., 'dnf install jq')."
        exit 1
    fi

    local found_any_digest=false

    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        # full_image_path_after_registry already includes the RHEL_SUFFIX if applicable
        local full_image_path_after_registry="${IMAGE_MAPPING[$internal_name]}"
        
        # Construct the full image reference with the provided version
        local image_ref="${registry}/${full_image_path_after_registry}:${version}"
        local digest=""

        echo "  - Fetching digest for ${internal_name} (${image_ref})..."
        # Try to get the digest using skopeo
        # Use a subshell to capture output and silence errors
        digest=$(skopeo inspect --override-os=linux "docker://${image_ref}" 2>/dev/null | jq -r '.Digest' 2>/dev/null)

        if [ -n "$digest" ]; then
            # Set the corresponding global digest variable (e.g., OPERATOR_DIGEST)
            local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
            eval "${digest_var_name}='${digest}'" # Use eval for indirect variable assignment
            found_any_digest=true
            echo "    -> Found: ${digest}"
        else
            echo "    -> Warning: Could not fetch digest for ${image_ref}. This image might not exist with this version or there was a registry error."
        fi
    done
    echo "Digest fetching complete."

    if ! $found_any_digest; then
        echo "Error: No digests could be fetched for version '${version}'. Please check the version and registry access."
        exit 1
    fi
}

# --- Main validation and digest fetching ---
if [ -z "$VERSION" ]; then
    echo "Error: Version (-V or VERSION env var) is required to fetch image digests."
    show_help
fi

# Call fetch_digests_from_registry to populate the global digest variables
fetch_digests_from_registry "$VERSION" "$REGISTRY"

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
        local current_digest="${!digest_var_name}" # Indirect expansion to get the value of the variable

        if [ -n "$current_digest" ]; then
            local new_image_full_ref="${registry}/${full_image_path_after_registry}@${current_digest}"

            # Escape '/' characters in the image path for sed
            local escaped_full_image_path_after_registry=$(echo "$full_image_path_after_registry" | sed 's/\//\\\//g')
            local escaped_new_image_full_ref=$(echo "$new_image_full_ref" | sed 's/\//\\\//g')
            local escaped_registry=$(echo "$registry" | sed 's/\//\\\//g')

            # The sed pattern now matches the specific image basename and name tag within the relatedImages block
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

    # Update images specified in environment variables (IMAGE_SHIPWRIGHT_*)
    # These are nested under 'env:' blocks, with 'value:' holding the image reference.
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        # full_image_path_after_registry already includes the RHEL_SUFFIX if applicable
        local full_image_path_after_registry="${IMAGE_MAPPING[$internal_name]}"

        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}" # Get the digest value

        # Define the expected environment variable name for the target image
        local env_var_name
        case "$internal_name" in
            "controller") env_var_name="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD";;
            "git-cloner") env_var_name="IMAGE_SHIPWRIGHT_GIT_CONTAINER_IMAGE";;
            "image-processing") env_var_name="IMAGE_SHIPWRIGHT_IMAGE_PROCESSING_CONTAINER_IMAGE";;
            "image-bundler") env_var_name="IMAGE_SHIPWRIGHT_BUNDLE_CONTAINER_IMAGE";;
            "waiter") env_var_name="IMAGE_SHIPWRIGHT_WAITER_CONTAINER_IMAGE";;
            "webhook") env_var_name="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD_WEBHOOK";;
            # All other images are explicitly excluded as they are not in the target env var list in manager.yaml.
            *)
                continue
            ;;
        esac

        # Proceed only if a digest is provided for this specific image
        if [ -n "$current_digest" ]; then
            local new_env_value="${registry}/${full_image_path_after_registry}@${current_digest}"

            # Escape '/' characters for sed
            local escaped_full_image_path_after_registry=$(echo "$full_image_path_after_registry" | sed 's/\//\\\//g')
            local escaped_new_env_value=$(echo "$new_env_value" | sed 's/\//\\\//g')
            local escaped_registry=$(echo "$registry" | sed 's/\//\\\//g')

            # This sed command searches for the specific environment variable name,
            # then replaces the 'value:' line that follows it.
            # The range is defined to correctly capture the value of the specific env var.
            sed -i "/- name: ${env_var_name}/,/^- name:\|^        securityContext:\|^        livenessProbe:/s|\(^\s*value: \)${escaped_registry}\/${escaped_full_image_path_after_registry}@.*|\1${escaped_new_env_value}|" "$manager_file"
            echo "  - Updated ${env_var_name} in manager.yaml: to ${new_env_value}"
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

        local full_image_path_after_registry="${IMAGE_MAPPING[$internal_image_name]}"
        local digest_var_name=$(echo "$internal_image_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}" # Indirect expansion

        if [ -n "$current_digest" ]; then
            # The new reference will always be @digest
            local new_image_full_ref="${registry}/${full_image_path_after_registry}@${current_digest}"

            # Escape for sed. Using # as delimiter to reduce need for escaping slashes.
            # Using simple sed 's/\//\\\//g' for slashes is fine too if # is not in data
            local escaped_registry=$(echo "$registry" | sed 's/[#]/\\#/g')
            local escaped_full_image_path_after_registry=$(echo "$full_image_path_after_registry" | sed 's/[#]/\\#/g')
            local escaped_new_image_full_ref=$(echo "$new_image_full_ref" | sed 's/[#]/\\#/g')
            
            # This sed command targets lines that:
            # 1. Start with any indentation (\s*)
            # 2. Optionally have '- ' for a list item (\( -\s*\)?)
            # 3. Contain "image: " (with optional spaces after colon)
            # 4. Contain the specific registry and image path
            # 5. Then match EITHER a colon followed by non-space chars OR an @ followed by anything (tag or digest)
            #    and replace that entire part with the new @digest reference.
            sed -i -E "s#^(\s*(-\s+)?image:\s*)${escaped_registry}/${escaped_full_image_path_after_registry}(:.*|@.*)?#\1${escaped_new_image_full_ref}#g" "$resource_file"
            
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
            yq eval '(.images[] | select(.name == "operator")).newTag = env(VERSION)' -i "$kustomization_file"
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
        yq eval '(.images[] | select(.name == "operator")).digest = env(OPERATOR_DIGEST)' -i "$kustomization_file"
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
