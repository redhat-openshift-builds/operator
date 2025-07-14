# Release Helper Script

A shell script 
  - update Tekton pipeline configuration for a new version
  - to update components iamges in operator and bundle from Konflux snapshots 

## Setup & Configuration

### Prerequisites
- **OpenShift CLI (`oc`)**: Required for all operations
- **yq**: Required only for `--pipeline-operator` flag
- **jq**: Used internally for JSON parsing
- **Active OpenShift login**: Run `oc login` before using the script

### Configuration Variables

Edit these variables at the top of `release_helper.sh`:

```bash
# Timezone configuration for age calculations of Konflux snapshots
TIMEZONE_OFFSET="+0530"  # Format: "+HHMM" or "-HHMM"

# Number of snapshots to display in listings
MAX_SNAPSHOTS_TO_DISPLAY=5
```

## Usage

```bash
./release_helper.sh <version-pattern> <flag> [--snapshot-name <snapshot-name>]
```

**Note**: Z-stream versions (e.g., `1-5-1`) are not supported yet. Please reach out to the script author to discuss how to support z-stream versions.

### Available Flags

| Flag | Description | Snapshots Shown | Repository Requirement |
|------|-------------|-----------------|------------------------|
| `--pipeline-operator` | Update Tekton pipeline files for specified version | N/A | Must be operator repo |
| `--snapshot [N]` | Show N snapshots (default: 5) and prompt for selection | N (default: 5) | Any |
| `--operator` | Update all build & CSI component digests | 2 | Any |
| `--operator-csi` | Update only CSI component digests | 2 | Any |
| `--bundle` | Update bundle files with operator digest | 2 | Any |
| `--bundle-url` | Get bundle URL from latest snapshot | 2 | Any |

### Optional Parameters

| Parameter | Description | Compatible With |
|-----------|-------------|-----------------|
| `--snapshot-name <name>` | Use specific snapshot instead of auto-finding latest | All flags except `--pipeline-operator` |
| `--all` | Show ALL snapshots for version pattern | Only with `--snapshot` |
| `--skip N` | Skip N latest snapshots | Only with `--snapshot --all` |
| `--component` | Show component names in 4th column | Only with `--snapshot` |

### Examples

```bash
# Show 5 snapshots and prompt for selection
./release_helper.sh 1-5 --snapshot

# Show 10 snapshots and prompt for selection
./release_helper.sh 1-5 --snapshot 10

# Show ALL snapshots and prompt for selection
./release_helper.sh 1-5 --snapshot --all

# Show ALL snapshots except 3 latest and prompt for selection
./release_helper.sh 1-5 --snapshot --all --skip 3

# Show snapshots with component names
./release_helper.sh 1-5 --snapshot --component

# Show all snapshots with components, skipping 2 latest
./release_helper.sh 1-5 --snapshot --all --component --skip 2

# Update all operator components
./release_helper.sh 1-5 --operator

# Update only CSI components
./release_helper.sh 1-5 --operator-csi

# Update bundle files
./release_helper.sh 1-5 --bundle

# Get bundle URL
./release_helper.sh 1-5 --bundle-url

# Create pipeline files for new version
./release_helper.sh 1-6 --pipeline-operator

# Using specific snapshot instead of auto-finding:
./release_helper.sh 1-5 --operator --snapshot-name openshift-builds-1-5-abc123
./release_helper.sh 1-5 --operator-csi --snapshot-name openshift-builds-1-5-xyz789
./release_helper.sh 1-5 --bundle --snapshot-name openshift-builds-1-5-def456
./release_helper.sh 1-5 --bundle-url --snapshot-name openshift-builds-1-5-ghi012
./release_helper.sh 1-5 --snapshot --snapshot-name openshift-builds-1-5-jkl345
```

## Code Logic Flow

### 1. Validation & Setup
- Validate OpenShift CLI installation
- Check repository requirements (for `--pipeline-operator`)
- Verify OpenShift login status
- Parse and validate command-line arguments
- Validate snapshot name contains version pattern (if provided)

### 2. Snapshot Discovery
- **Interactive mode**: Query OpenShift for snapshots matching version pattern, sort by creation timestamp (newest first), display snapshot list with age and PR information, prompt user for selection (index, name, or Enter for latest)
- **Manual mode**: Use provided snapshot name (with `--snapshot-name`) after validation
- **Skip mode**: With `--skip N`, skip N latest snapshots before displaying
- **Component mode**: With `--component`, show available component names (operator, bundle, webhook, etc.) in additional column

### 3. Digest Extraction
- Fetch snapshot JSON from OpenShift
- Map internal component names to Konflux component names
- Extract container image digests based on operation mode:
  - **Bundle mode**: operator only
  - **CSI mode**: shared-resource-webhook, shared-resource
  - **Operator mode**: all components except operator
  - **Snapshot mode**: all components

### 4. File Updates

#### Updated Files by Mode
- **Manager.yaml**: Environment variables for build components
- **CSV files**: RelatedImages and deployment environment variables
- **Shared resource files**: CSI component images
- **Bundle files**: Operator digest in bundle manifests
- **Tekton files**: Pipeline configurations (pipeline-operator only)

#### Component Mappings
- `controller` → `IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD`
- `git-cloner` → `IMAGE_SHIPWRIGHT_GIT_CONTAINER_IMAGE`
- `image-processing` → `IMAGE_SHIPWRIGHT_IMAGE_PROCESSING_CONTAINER_IMAGE`
- `image-bundler` → `IMAGE_SHIPWRIGHT_BUNDLE_CONTAINER_IMAGE`
- `waiters` → `IMAGE_SHIPWRIGHT_WAITER_CONTAINER_IMAGE`
- `webhook` → `IMAGE_SHIPWRIGHT_SHIPWRIGHT_BUILD_WEBHOOK`

## Component Architecture

### Internal vs Konflux Names
The script maintains mappings between:
- **Internal names**: Used in code logic (e.g., `controller`, `git-cloner`)
- **Konflux names**: Used in snapshot queries (e.g., `openshift-builds-controller-1-5`)
- **Image names**: Used in file updates (e.g., `openshift-builds-controller-rhel9`)

### Operation Modes
- **Bundle operations**: Focus on operator image for bundle generation
- **CSI operations**: Target shared-resource components only
- **Operator operations**: Update all build system components
- **Pipeline operations**: Configure Tekton pipelines for new versions

## Files Modified

### Always Updated
- `./config/manager/manager.yaml` - Environment variables
- `./bundle/manifests/openshift-builds-operator.clusterserviceversion.yaml` - Bundle CSV
- `./config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml` - Base CSV

### CSI-Specific
- `./config/sharedresource/node_daemonset.yaml` - CSI daemon
- `./config/sharedresource/webhook_deployment.yaml` - CSI webhook

### Bundle-Specific
- `./config/manager/kustomization.yaml` - Operator image reference

### Pipeline-Specific
- `.tekton/openshift-builds-operator-*.yaml` - Pipeline configurations

## Error Handling

- **Snapshot not found**: Exits with error message
- **Login required**: Prompts for `oc login`
- **Missing components**: Lists missing components and exits
- **Old snapshots**: Rejects snapshots older than 30 days
- **Repository mismatch**: Validates operator repo for pipeline operations

## Security & Validation

- Validates OpenShift authentication before operations
- Checks repository module name for pipeline operations
- Validates all required components exist in snapshots
- Preserves existing image URLs while updating only digests
- Cross-platform compatibility (Linux/macOS) 