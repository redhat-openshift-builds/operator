#!/opt/homebrew/bin/bash

set -e

# Timezone configuration - Update this for your local timezone
# Format: "+HHMM" or "-HHMM" (e.g., "+0530" for GMT+5:30, "-0500" for GMT-5)
TIMEZONE_OFFSET="+0530"  # GMT+5:30 (India Standard Time)

# Set SED_INPLACE for cross-platform compatibility (Linux/macOS)
if [[ "$(uname)" == "Darwin" ]]; then
  SED_INPLACE=(-i '')
else
  SED_INPLACE=(-i)
fi

# Files for updating
MANAGER_FILE="./config/manager/manager.yaml"
CSV_FILE="./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml"
MANIFEST_BASES_FILE="./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
SHARED_RESOURCE_DAEMONSET_FILE="./config/sharedresource/node_daemonset.yaml"
SHARED_RESOURCE_WEBHOOK_FILE="./config/sharedresource/webhook_deployment.yaml"

# Environment variables for image names (with defaults matching current CSV at this time)
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
declare -A INTERNAL_TO_MANAGER_ENV=(
    [controller]="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD"
    [git-cloner]="IMAGE_SHIPWRIGHT_GIT_CONTAINER_IMAGE"
    [image-processing]="IMAGE_SHIPWRIGHT_IMAGE_PROCESSING_CONTAINER_IMAGE"
    [image-bundler]="IMAGE_SHIPWRIGHT_BUNDLE_CONTAINER_IMAGE"
    [waiters]="IMAGE_SHIPWRIGHT_WAITER_CONTAINER_IMAGE"
    [webhook]="IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD_WEBHOOK"
)

# Define the mapping between internal names
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
    ["bundle"]="$IMAGE_OPERATOR-bundle"
)

# Define a mapping for Konflux Component names
declare -A KONFLUX_COMPONENT_NAME_MAPPING=(
    ["operator"]="openshift-builds-operator"
    ["controller"]="openshift-builds-controller"
    ["git-cloner"]="openshift-builds-git-cloner"
    ["image-processing"]="openshift-builds-image-processing"
    ["image-bundler"]="openshift-builds-image-bundler"
    ["waiters"]="openshift-builds-waiters"
    ["webhook"]="openshift-builds-webhook"
    ["shared-resource-webhook"]="openshift-builds-shared-resource-webhook"
    ["shared-resource"]="openshift-builds-shared-resource"
    ["bundle"]="openshift-builds-operator-bundle"
)

# Function to check if OpenShift CLI is installed
function check_oc_installed {
    echo "Checking OpenShift CLI installation..." >&2
    
    # Check for oc command
    if ! command -v oc &> /dev/null; then
        echo "Error: 'oc' not found. Please install the OpenShift CLI." >&2
        exit 1
    fi
    echo "✓ oc command found" >&2
}

# Function to check if we're in the correct repository for pipeline-operator operations
function check_operator_repo {
    echo "Checking if current repository is the operator repository..." >&2
    
    # Check if go.mod exists
    if [ ! -f "go.mod" ]; then
        echo "Error: go.mod file not found. The --pipeline-operator flag can only be used in the operator repository." >&2
        exit 1
    fi
    
    # Check if the module name matches the expected operator repository
    local module_name=$(grep -E "^module " go.mod | awk '{print $2}')
    if [ "$module_name" != "github.com/redhat-openshift-builds/operator" ]; then
        echo "Error: Current repository module is '$module_name', but --pipeline-operator flag requires 'github.com/redhat-openshift-builds/operator'." >&2
        echo "This flag can only be used in the operator repository." >&2
        exit 1
    fi
    
    echo "✓ Confirmed: Running in operator repository" >&2
}

# Function to check OpenShift login status
function check_oc_login {
    echo "Checking OpenShift login status..." >&2
    if ! oc whoami &> /dev/null; then
        echo "Error: Not logged in to OpenShift. Please run 'oc login' first." >&2
        exit 1
    fi
    echo "✓ Logged in to OpenShift" >&2
}

# Function to check if yq is installed
function check_yq_installed {
    echo "Checking yq installation..." >&2
    
    if ! command -v yq &> /dev/null; then
        echo "Error: 'yq' not found. Please install yq to fetch component information." >&2
        exit 1
    fi
    echo "✓ yq command found" >&2
}

# Fetch digests from Konflux Components
function fetch_digests_from_components {
    local version_pattern=$1
    local mode=$2  # "operator", "csi", "bundle", or "bundle-url"

    echo "Attempting to fetch digests from Konflux Components with version pattern: '${version_pattern}'..."
    
    # Check OpenShift login status
    check_oc_login
    
    # Check if yq is installed
    check_yq_installed

    local found_any_digest=false
    local components_to_process=()

    # Determine which components to process based on mode
    case "$mode" in
        "bundle")
            components_to_process=("operator")
            echo "Bundle mode: Only fetching operator digest"
            ;;
        "bundle-url")
            components_to_process=("bundle")
            echo "Bundle URL mode: Only fetching bundle URL"
            ;;
        "csi")
            components_to_process=("shared-resource-webhook" "shared-resource")
            echo "CSI mode: Only fetching shared-resource component digests"
            ;;
        "operator")
            components_to_process=("controller" "git-cloner" "image-processing" "image-bundler" "waiters" "webhook" "shared-resource-webhook" "shared-resource")
            echo "Operator mode: Fetching all 8 component digests"
            ;;
        *)
            echo "Error: Invalid mode '$mode' specified"
            exit 1
            ;;
    esac

    for internal_name in "${components_to_process[@]}"; do
        local component_name_in_konflux="${KONFLUX_COMPONENT_NAME_MAPPING[$internal_name]}"
        
        if [ -z "$component_name_in_konflux" ]; then
            echo "  - Warning: No Konflux component name mapping found for internal name: '${internal_name}'. Skipping."
            continue
        fi
        
        # Add version pattern suffix to component name
        component_name_in_konflux="${component_name_in_konflux}-${version_pattern}"
        
        echo "  - Fetching component: ${component_name_in_konflux}"
        
        # Fetch component YAML
        local component_yaml
        if ! component_yaml=$(oc get component "$component_name_in_konflux" -o yaml 2>/dev/null); then
            echo "    -> Error: Could not fetch component '${component_name_in_konflux}'. Skipping."
            continue
        fi
        
        # Extract lastPromotedImage using yq
        local last_promoted_image=$(echo "$component_yaml" | yq eval '.status.lastPromotedImage' -)
        
        if [ -n "$last_promoted_image" ] && [ "$last_promoted_image" != "null" ]; then
            local current_digest=$(echo "$last_promoted_image" | awk -F'@' '{print $2}')
            
            if [ -n "$current_digest" ]; then
                local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
                eval "${digest_var_name}='${current_digest}'"
                
                # For bundle-url mode, also save the full URL
                if [ "$mode" = "bundle-url" ] && [ "$internal_name" = "bundle" ]; then
                    local url_var_name="BUNDLE_FULL_URL"
                    eval "${url_var_name}='${last_promoted_image}'"
                fi
                
                found_any_digest=true
                echo "    -> Found '${internal_name}' digest: ${current_digest}"
            else
                echo "    -> Warning: Could not extract digest from lastPromotedImage for '${internal_name}'. Skipping."
            fi
        else
            echo "    -> Warning: No lastPromotedImage found for '${internal_name}'. Skipping."
        fi
    done

    # Validate that required components were found
    case "$mode" in
        "bundle")
            local operator_digest_var_name="OPERATOR_DIGEST"
            local operator_digest="${!operator_digest_var_name}"
            if [ -z "$operator_digest" ]; then
                echo "Error: Bundle mode requires operator digest, but operator digest was not found."
                exit 1
            fi
            echo "✓ Bundle mode: Successfully found operator digest: $operator_digest"
            ;;
        "bundle-url")
            local bundle_digest_var_name="BUNDLE_DIGEST"
            local bundle_digest="${!bundle_digest_var_name}"
            if [ -z "$bundle_digest" ]; then
                echo "Error: Bundle URL mode requires bundle digest, but bundle digest was not found."
                exit 1
            fi
            echo "    -> Found 'bundle' URL: ${BUNDLE_FULL_URL}"
            ;;
        "csi")
            local shared_resource_webhook_digest_var_name="SHARED_RESOURCE_WEBHOOK_DIGEST"
            local shared_resource_digest_var_name="SHARED_RESOURCE_DIGEST"
            local shared_resource_webhook_digest="${!shared_resource_webhook_digest_var_name}"
            local shared_resource_digest="${!shared_resource_digest_var_name}"
            
            local missing_components=()
            if [ -z "$shared_resource_webhook_digest" ]; then
                missing_components+=("shared-resource-webhook")
            fi
            if [ -z "$shared_resource_digest" ]; then
                missing_components+=("shared-resource")
            fi
            
            if [ ${#missing_components[@]} -gt 0 ]; then
                echo "Error: CSI mode requires both shared-resource-webhook and shared-resource digests, but the following were not found:"
                for component in "${missing_components[@]}"; do
                    echo "  - $component"
                done
                exit 1
            fi
            
            echo "✓ CSI mode: Successfully found shared-resource-webhook digest: $shared_resource_webhook_digest"
            echo "✓ CSI mode: Successfully found shared-resource digest: $shared_resource_digest"
            ;;
        "operator")
            local required_components=("controller" "git-cloner" "image-processing" "image-bundler" "waiters" "webhook" "shared-resource-webhook" "shared-resource")
            local missing_components=()
            
            for component in "${required_components[@]}"; do
                local digest_var_name=$(echo "$component" | tr '[:lower:]-' '[:upper:]_')_DIGEST
                local current_digest="${!digest_var_name}"
                if [ -z "$current_digest" ]; then
                    missing_components+=("$component")
                fi
            done
            
            if [ ${#missing_components[@]} -gt 0 ]; then
                echo "Error: Operator mode requires all 8 component digests, but the following were not found:"
                for component in "${missing_components[@]}"; do
                    echo "  - $component"
                done
                exit 1
            fi
            
            echo "✓ Operator mode: Successfully found all ${#required_components[@]} required component digests"
            ;;
    esac

    if ! $found_any_digest; then
        echo "Error: No digests could be fetched from components. Please check the component names and versions."
        exit 1
    else
        echo "Digest fetching from components complete."
    fi
}

# Help message
function show_help {
    echo "Usage: $0 <version-pattern> <flag>"
    echo ""
    echo "Release Helper script to fetch digests from Konflux Components and update relevant images in various yaml files."
    echo ""
    echo "Arguments:"
    echo "  <version-pattern>  Version pattern to search for in component names (e.g., '1-5', '1-6'). REQUIRED."
    echo "                     Note: Z-stream versions (e.g., '1-5-1') are not supported yet."
    echo ""
    echo "Flags:"
    echo "  --pipeline-operator  Update Tekton pipeline files for the specified version for operator repo"
    echo "  --operator           Update all 8 components' digests from latest components in operator and bundle files"
    echo "  --operator-csi       Update only CSI components' digests from latest components in operator and bundle files"
    echo "  --bundle             Update bundle files with operator digest from latest component"
    echo "  --bundle-url         Get bundle URL from latest component"
    echo ""
    echo "Examples:"
    echo "  $0 1-5 --pipeline-operator       # Updates Tekton pipeline files for builds-1.5"
    echo "  $0 1-5 --operator                # Finds latest components with '1-5' and updates operator and bundle files"
    echo "  $0 1-6 --operator                # Finds latest components with '1-6' and updates operator and bundle files"
    echo "  $0 2-0 --operator                # Finds latest components with '2-0' and updates operator and bundle files"
    echo "  $0 1-5 --operator-csi            # Updates only CSI components"
    echo "  $0 1-5 --bundle                  # Finds latest components with '1-5' and updates bundle files"
    echo "  $0 1-5 --bundle-url              # Shows bundle URL from latest component with '1-5'"
    echo ""
    echo "Error: Please provide a valid version pattern and flag."
    exit 1
}

# Update manager.yaml
function update_manager_yaml {
    local manager_file=$1
    echo "Updating images in $manager_file..."
    if [ ! -f "$manager_file" ]; then
        echo "Warning: $manager_file not found. Skipping..."
        return
    fi

    local updated_count=0
    local total_count=0
    
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
            # Only update the digest part, preserving the existing image URL
            sed "${SED_INPLACE[@]}" -E "/- name: ${env_var_name}/ {n; s#@sha256:[a-f0-9]+#@${current_digest}#;}" "$manager_file"
            echo "  - Updated ${env_var_name} digest in manager.yaml to ${current_digest}"
            updated_count=$((updated_count + 1))
        fi
        total_count=$((total_count + 1))
    done
    
    if [ $updated_count -eq 0 ]; then
        echo "  - No relevant components found for manager.yaml updates in current mode"
    else
        echo "  - Updated $updated_count out of $total_count components in manager.yaml"
    fi
}

# Update CSV deployment environment variables (only SHA digests)
function update_csv_deployment_env_vars {
    local csv_file=$1
    echo "Updating deployment environment variables in $csv_file..."
    if [ ! -f "$csv_file" ]; then
        echo "Warning: $csv_file not found. Skipping..."
        return
    fi

    local updated_count=0
    local total_count=0
    
    # Update images specified in environment variables (IMAGE_SHIPWRIGHT_*)
    for internal_name in "${!INTERNAL_TO_MANAGER_ENV[@]}"; do
        env_var_name="${INTERNAL_TO_MANAGER_ENV[$internal_name]}"
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        
        if [ -z "$env_var_name" ]; then
            echo "  - Info: Internal name '${internal_name}' does not map to an environment variable in CSV. Skipping."
            continue
        fi
        
        if [ -n "$current_digest" ]; then
            # Update only the digest part, preserving the existing image URL
            sed "${SED_INPLACE[@]}" -E "/- name: ${env_var_name}/ {n; s#@sha256:[a-f0-9]+#@${current_digest}#;}" "$csv_file"
            echo "  - Updated ${env_var_name} digest in CSV to ${current_digest}"
            updated_count=$((updated_count + 1))
        fi
        total_count=$((total_count + 1))
    done
    
    if [ $updated_count -eq 0 ]; then
        echo "  - No relevant components found for CSV deployment environment variable updates in current mode"
    else
        echo "  - Updated $updated_count out of $total_count deployment environment variables in CSV"
    fi
}

# Update shared resource files (node_daemonset.yaml and webhook_deployment.yaml)
function update_shared_resource_files {
    local resource_file=$1
    local internal_image_name=$2
    
    echo "Updating images in $resource_file..."
    if [ ! -f "$resource_file" ]; then
        echo "Warning: $resource_file not found. Skipping..."
        return
    fi
    
    if [[ -z "${IMAGE_MAPPING[$internal_image_name]}" ]]; then
        echo "Warning: Image key '$internal_image_name' not found in IMAGE_MAPPING. Skipping for $resource_file."
        return
    fi
    
    local digest_var_name=$(echo "$internal_image_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
    local current_digest="${!digest_var_name}"
    
    if [ -n "$current_digest" ]; then
        local image_path="${IMAGE_MAPPING[$internal_image_name]}"
        # Extract just the image name (after last /)
        local image_name=$(basename "$image_path")
        
        # Update only the digest part for the specific image, preserving the existing image URL
        # This targets only images that contain our specific image name in the path
        sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*${image_name}[^@]*)@sha256:[a-f0-9]+#\\1@${current_digest}#g" "$resource_file"
        echo "  - Updated ${internal_image_name} digest in $resource_file to ${current_digest}"
    else
        echo "  - Info: No digest found for ${internal_image_name}. Skipping update for this image in $resource_file."
    fi
}

# Update relatedImages in CSV
function update_csv_without_operator {
    local csv_file=$1
    echo "Updating relatedImages in $csv_file..."
    if ! grep -qE '^\s*relatedImages:' "$csv_file"; then
        echo "Warning: 'relatedImages' block not found in $csv_file. Skipping CSV update."
        return 0
    fi
    
    local updated_count=0
    
    for internal_name in "${!IMAGE_MAPPING[@]}"; do
        local digest_var_name=$(echo "$internal_name" | tr '[:lower:]-' '[:upper:]_')_DIGEST
        local current_digest="${!digest_var_name}"
        if [ -n "$current_digest" ]; then
            local image_path="${IMAGE_MAPPING[$internal_name]}"
            # Extract just the image name (after last /)
            image_name=$(basename "$image_path")
            # Only update the digest part for this specific image, preserving the existing image URL
            # Target the specific image by name within the relatedImages section
            sed "${SED_INPLACE[@]}" -E "/^  relatedImages:/,/^[^ ]/s#(image: [^@]*${image_name})@sha256:[a-f0-9]+#\\1@${current_digest}#g" "$csv_file"
            echo "  - Updated ${internal_name} digest in CSV relatedImages to ${current_digest}"
            updated_count=$((updated_count + 1))
        fi
    done
    
    if [ $updated_count -eq 0 ]; then
        echo "  - No relevant components found for CSV relatedImages updates in current mode"
    else
        echo "  - Updated $updated_count components in CSV relatedImages"
    fi
}

# Update operator SHA in bundle files
function update_operator_sha {
    local operator_digest_var_name="OPERATOR_DIGEST"
    local operator_digest="${!operator_digest_var_name}"
    
    if [ -z "$operator_digest" ]; then
        echo "Error: No operator digest found. Cannot update operator SHA."
        return 1
    fi
    
    echo "Updating operator SHA ($operator_digest) in bundle files..."
    
    # File 1: bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml
    local bundle_csv_file="./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml"
    if [ -f "$bundle_csv_file" ]; then
        echo "  - Updating $bundle_csv_file..."
        
        # Update OPENSHIFT_BUILDS_OPERATOR image in relatedImages section (only the image field)
        sed "${SED_INPLACE[@]}" -E "/name: OPENSHIFT_BUILDS_OPERATOR@sha256:[a-f0-9]+/{n; s#(image: [^@]+)@sha256:[a-f0-9]+#\\1@${operator_digest}#;}" "$bundle_csv_file"
        
        # Update operator image in deployment section (line 737)
        sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*openshift-builds-rhel9-operator)@sha256:[a-f0-9]+#\\1@${operator_digest}#g" "$bundle_csv_file"
        
        echo "    -> Updated operator digest in $bundle_csv_file"
    else
        echo "  - Warning: $bundle_csv_file not found. Skipping..."
    fi
    
    # File 2: config/manager/kustomization.yaml
    local kustomization_file="./config/manager/kustomization.yaml"
    if [ -f "$kustomization_file" ]; then
        echo "  - Updating $kustomization_file..."
        
        # Check what format the file currently uses and apply appropriate update
        if grep -q "newTag:" "$kustomization_file"; then
            echo "    -> Found newTag, converting to digest"
            sed "${SED_INPLACE[@]}" -E "s#newTag: [0-9]+\.[0-9]+\.[0-9]+#digest: ${operator_digest}#g" "$kustomization_file"
        elif grep -q "digest:" "$kustomization_file"; then
            echo "    -> Found existing digest, updating"
            sed "${SED_INPLACE[@]}" -E "s#digest: sha256:.*#digest: ${operator_digest}#g" "$kustomization_file"
        else
            echo "    -> Warning: Neither newTag nor digest found in $kustomization_file"
        fi
        
        echo "    -> Updated kustomization.yaml"
    else
        echo "  - Warning: $kustomization_file not found. Skipping..."
    fi
    
    # File 3: config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml
    local manifest_bases_csv_file="./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
    if [ -f "$manifest_bases_csv_file" ]; then
        echo "  - Updating $manifest_bases_csv_file..."
        
        # Update OPENSHIFT_BUILDS_OPERATOR image in relatedImages section (directly target the operator image line)
        sed "${SED_INPLACE[@]}" -E "s#(image: [^@]*openshift-builds-rhel9-operator)@sha256:[a-f0-9]+#\\1@${operator_digest}#g" "$manifest_bases_csv_file"
        
        echo "    -> Updated operator digest in $manifest_bases_csv_file"
    else
        echo "  - Warning: $manifest_bases_csv_file not found. Skipping..."
    fi
    
    echo "  - Operator SHA update completed successfully!"
}

# Update Tekton pipeline files for the specified version pattern
function update_tekton_pipelines {
    local version_pattern=$1
    local branch_version=$(echo "$version_pattern" | tr "-" ".")
    
    echo "Updating Tekton pipeline files for version pattern: $version_pattern"
    
    # Check if yq is available
    if ! command -v yq &> /dev/null; then
        echo "Error: 'yq' not found. Please install yq to use this feature."
        exit 1
    fi
    
    # ===== Operator Bundle Pull Request File Changes =====
    echo "Updating openshift-builds-operator-bundle-pull-request.yaml..."
    yq eval -i '.metadata.annotations."pipelinesascode.tekton.dev/on-cel-expression" |= sub("target_branch == \"main\"", "target_branch == \"builds-'$branch_version'\"")' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/application" = "openshift-builds-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/component" = "openshift-builds-operator-bundle-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    yq eval -i '.metadata.name = "openshift-builds-operator-bundle-'$version_pattern'-on-pull-request"' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    yq eval -i '(.spec.params[] | select(.name == "output-image") | .value) = "quay.io/redhat-user-workloads/rh-openshift-builds-tenant/openshift-builds-operator-bundle-'$version_pattern':on-pr-{{revision}}"' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    yq eval -i '.spec.taskRunTemplate.serviceAccountName = "build-pipeline-openshift-builds-operator-bundle-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-pull-request.yaml
    echo "✅ Updated openshift-builds-operator-bundle-pull-request.yaml"
    
    # ===== Operator Bundle Push File Changes =====
    echo "Updating openshift-builds-operator-bundle-push.yaml..."
    yq eval -i '.metadata.annotations."pipelinesascode.tekton.dev/on-cel-expression" |= sub("target_branch == \"main\"", "target_branch == \"builds-'$branch_version'\"")' .tekton/openshift-builds-operator-bundle-push.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/application" = "openshift-builds-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-push.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/component" = "openshift-builds-operator-bundle-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-push.yaml
    yq eval -i '.metadata.name = "openshift-builds-operator-bundle-'$version_pattern'-on-push"' .tekton/openshift-builds-operator-bundle-push.yaml
    yq eval -i '(.spec.params[] | select(.name == "output-image") | .value) = "quay.io/redhat-user-workloads/rh-openshift-builds-tenant/openshift-builds-operator-bundle-'$version_pattern':{{revision}}"' .tekton/openshift-builds-operator-bundle-push.yaml
    yq eval -i '.spec.taskRunTemplate.serviceAccountName = "build-pipeline-openshift-builds-operator-bundle-'$version_pattern'"' .tekton/openshift-builds-operator-bundle-push.yaml
    echo "✅ Updated openshift-builds-operator-bundle-push.yaml"
    
    # ===== Operator Pull Request File Changes =====
    echo "Updating openshift-builds-operator-pull-request.yaml..."
    yq eval -i '.metadata.annotations."pipelinesascode.tekton.dev/on-cel-expression" |= sub("target_branch == \"main\"", "target_branch == \"builds-'$branch_version'\"")' .tekton/openshift-builds-operator-pull-request.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/application" = "openshift-builds-'$version_pattern'"' .tekton/openshift-builds-operator-pull-request.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/component" = "openshift-builds-operator-'$version_pattern'"' .tekton/openshift-builds-operator-pull-request.yaml
    yq eval -i '.metadata.name = "openshift-builds-operator-'$version_pattern'-on-pull-request"' .tekton/openshift-builds-operator-pull-request.yaml
    yq eval -i '(.spec.params[] | select(.name == "output-image") | .value) = "quay.io/redhat-user-workloads/rh-openshift-builds-tenant/openshift-builds-operator-'$version_pattern':on-pr-{{revision}}"' .tekton/openshift-builds-operator-pull-request.yaml
    yq eval -i '.spec.taskRunTemplate.serviceAccountName = "build-pipeline-openshift-builds-operator-'$version_pattern'"' .tekton/openshift-builds-operator-pull-request.yaml
    echo "✅ Updated openshift-builds-operator-pull-request.yaml"
    
    # ===== Operator Push File Changes =====
    echo "Updating openshift-builds-operator-push.yaml..."
    yq eval -i '.metadata.annotations."pipelinesascode.tekton.dev/on-cel-expression" |= sub("target_branch == \"main\"", "target_branch == \"builds-'$branch_version'\"")' .tekton/openshift-builds-operator-push.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/application" = "openshift-builds-'$version_pattern'"' .tekton/openshift-builds-operator-push.yaml
    yq eval -i '.metadata.labels."appstudio.openshift.io/component" = "openshift-builds-operator-'$version_pattern'"' .tekton/openshift-builds-operator-push.yaml
    yq eval -i '.metadata.name = "openshift-builds-operator-'$version_pattern'-on-push"' .tekton/openshift-builds-operator-push.yaml
    yq eval -i '(.spec.params[] | select(.name == "output-image") | .value) = "quay.io/redhat-user-workloads/rh-openshift-builds-tenant/openshift-builds-operator-'$version_pattern':{{revision}}"' .tekton/openshift-builds-operator-push.yaml
    yq eval -i '.spec.taskRunTemplate.serviceAccountName = "build-pipeline-openshift-builds-operator-'$version_pattern'"' .tekton/openshift-builds-operator-push.yaml
    echo "✅ Updated openshift-builds-operator-push.yaml"
    
    echo "🎉 All Tekton pipeline files have been successfully updated for builds-$branch_version!"
}

# Main execution - Parse flags and arguments
if [ $# -lt 2 ]; then
    echo "Error: Both version pattern and flag are required."
    show_help
fi

# Parse the version pattern (first argument) and flag (second argument)
VERSION_PATTERN="$1"
FLAG="$2"

# Check for z-stream version pattern (not supported yet)
if [[ "$VERSION_PATTERN" =~ ^[0-9]+-[0-9]+-[0-9]+$ ]]; then
    echo "Error: Z-stream version pattern '$VERSION_PATTERN' is not supported yet."
    echo "This script currently supports only major.minor versions (e.g., '1-5', '1-6')."
    echo "Please reach out to the script author to discuss how to support z-stream versions."
    exit 1
fi

# Validate flag
case "$FLAG" in
    --pipeline-operator) 
        echo "=========================================="
        echo "Pipeline-Operator Mode: Creating Tekton pipeline files for version pattern: $VERSION_PATTERN"
        echo "=========================================="

        # Check if we're in the correct repository for pipeline-operator operations
        check_operator_repo

        # Update Tekton pipeline files with the specified version pattern
        update_tekton_pipelines "$VERSION_PATTERN"

        echo "=========================================="
        echo "Pipeline-Operator creation completed successfully!"
        echo "=========================================="
        ;;
    --operator) 
        # Check if OpenShift CLI is installed
        check_oc_installed
        
        echo "=========================================="
        echo "Operator Mode: Fetching digests from components with '$VERSION_PATTERN'"
        echo "=========================================="

        # Fetch digests from components
        fetch_digests_from_components "$VERSION_PATTERN" "operator"

        echo "=========================================="
        echo "Updating manager.yaml with fetched digests"
        echo "=========================================="

        # Update manager.yaml with the fetched digests
        update_manager_yaml "$MANAGER_FILE"

        echo "=========================================="
        echo "Updating CSV relatedImages with fetched digests"
        echo "=========================================="

        # Update CSV files with the fetched digests
        update_csv_without_operator "$CSV_FILE"
        update_csv_without_operator "$MANIFEST_BASES_FILE"

        echo "=========================================="
        echo "Updating CSV deployment environment variables with fetched digests"
        echo "=========================================="

        # Update CSV deployment environment variables with the fetched digests
        update_csv_deployment_env_vars "$CSV_FILE"
        update_csv_deployment_env_vars "$MANIFEST_BASES_FILE"

        echo "=========================================="
        echo "Updating shared resource files with fetched digests"
        echo "=========================================="

        # Update shared resource files with the fetched digests
        update_shared_resource_files "$SHARED_RESOURCE_DAEMONSET_FILE" "shared-resource"
        update_shared_resource_files "$SHARED_RESOURCE_WEBHOOK_FILE" "shared-resource-webhook"

        echo "=========================================="
        echo "Operator mode completed successfully!"
        echo "=========================================="
        ;;
    --operator-csi) 
        # Check if OpenShift CLI is installed
        check_oc_installed
        
        echo "=========================================="
        echo "Operator CSI Mode: Fetching digests from CSI components with '$VERSION_PATTERN'"
        echo "=========================================="

        # Fetch digests from components (CSI mode)
        fetch_digests_from_components "$VERSION_PATTERN" "csi"

        echo "=========================================="
        echo "Updating manager.yaml with fetched digests"
        echo "=========================================="

        # Update manager.yaml with the fetched digests
        update_manager_yaml "$MANAGER_FILE"

        echo "=========================================="
        echo "Updating CSV relatedImages with fetched digests"
        echo "=========================================="

        # Update CSV files with the fetched digests
        update_csv_without_operator "$CSV_FILE"
        update_csv_without_operator "$MANIFEST_BASES_FILE"

        echo "=========================================="
        echo "Updating CSV deployment environment variables with fetched digests"
        echo "=========================================="

        # Update CSV deployment environment variables with the fetched digests
        update_csv_deployment_env_vars "$CSV_FILE"
        update_csv_deployment_env_vars "$MANIFEST_BASES_FILE"

        echo "=========================================="
        echo "Updating shared resource files with fetched digests"
        echo "=========================================="

        # Update shared resource files with the fetched digests
        update_shared_resource_files "$SHARED_RESOURCE_DAEMONSET_FILE" "shared-resource"
        update_shared_resource_files "$SHARED_RESOURCE_WEBHOOK_FILE" "shared-resource-webhook"

        echo "=========================================="
        echo "Operator CSI mode completed successfully!"
        echo "=========================================="
        ;;
    --bundle) 
        # Check if OpenShift CLI is installed
        check_oc_installed
        
        echo "=========================================="
        echo "Bundle mode: Fetching operator digest from component with '$VERSION_PATTERN'"
        echo "=========================================="

        # Fetch digests from components (bundle mode - operator only)
        fetch_digests_from_components "$VERSION_PATTERN" "bundle"

        echo "=========================================="
        echo "Updating operator SHA in bundle files"
        echo "=========================================="

        # Update operator SHA in bundle files
        update_operator_sha

        echo "=========================================="
        echo "Bundle update completed successfully!"
        echo "=========================================="
        ;;
    --bundle-url) 
        # Check if OpenShift CLI is installed
        check_oc_installed
        
        echo "=========================================="
        echo "Bundle URL Mode: Fetching bundle URL from component with '$VERSION_PATTERN'"
        echo "=========================================="

        # Fetch bundle URL from components
        fetch_digests_from_components "$VERSION_PATTERN" "bundle-url"

        echo "=========================================="
        echo "Bundle URL:"
        echo "=========================================="
        
        # Display bundle URL
        bundle_url_var_name="BUNDLE_FULL_URL"
        bundle_url="${!bundle_url_var_name}"
        if [ -n "$bundle_url" ]; then
            echo "$bundle_url"
        else
            echo "Bundle URL not found"
            exit 1
        fi
        
        echo "=========================================="
        echo "✓ Bundle URL extraction completed successfully!"
        echo "=========================================="
        ;;
    --help|-h)
        show_help
        ;;
    *) 
        echo "Error: Unknown flag '$FLAG'"
        echo ""
        show_help
        ;;
esac