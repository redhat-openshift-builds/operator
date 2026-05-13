#!/usr/bin/env bash
#
# setup-release-branch.sh - Configure repository for a release branch
#
# This script updates Tekton pipelines, Dockerfiles, and Makefile for a new
# release branch. It can be run locally or via GitHub Actions.
#
# Usage:
#   ./hack/setup-release-branch.sh [OPTIONS]
#
# Options:
#   -b, --branch BRANCH    Branch name (e.g., builds-1.8). If not provided,
#                          attempts to detect from current git branch.
#   -d, --dry-run          Show what would be changed without making changes.
#   -s, --skip-bundle      Skip running 'make bundle' at the end.
#   -h, --help             Show this help message.
#
# Examples:
#   ./hack/setup-release-branch.sh --branch builds-1.8
#   ./hack/setup-release-branch.sh --dry-run
#   ./hack/setup-release-branch.sh --skip-bundle
#   ./hack/setup-release-branch.sh  # Uses current git branch

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
BRANCH=""
DRY_RUN=false

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Show usage
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Configure repository for a release branch by updating Tekton pipelines,
Dockerfiles, and Makefile.

Options:
    -b, --branch BRANCH    Branch name (e.g., builds-1.8). If not provided,
                           attempts to detect from current git branch.
    -d, --dry-run          Show what would be changed without making changes.
    -h, --help             Show this help message.

Examples:
    $(basename "$0") --branch builds-1.8
    $(basename "$0") --branch builds-1.8 --dry-run
    $(basename "$0")  # Uses current git branch

Note: Run 'make bundle' separately after this script to regenerate bundle manifests.
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--branch)
                BRANCH="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Detect branch from git if not provided
detect_branch() {
    if [[ -z "$BRANCH" ]]; then
        if git rev-parse --is-inside-work-tree &>/dev/null; then
            BRANCH=$(git rev-parse --abbrev-ref HEAD)
            log_info "Detected branch from git: $BRANCH"
        else
            log_error "Not in a git repository and no branch specified"
            exit 1
        fi
    fi
}

# Validate branch name format
validate_branch() {
    if [[ ! "$BRANCH" =~ ^builds-[0-9]+\.[0-9]+$ ]]; then
        log_error "Branch name '$BRANCH' does not match expected format 'builds-x.y'"
        log_error "Example valid branch names: builds-1.8, builds-2.0, builds-1.10"
        exit 1
    fi
    log_success "Branch name validated: $BRANCH"
}

# Extract version components from branch name
extract_version() {
    VERSION="${BRANCH#builds-}"
    MAJOR="${VERSION%%.*}"
    MINOR="${VERSION#*.}"
    VERSION_SUFFIX="${MAJOR}-${MINOR}"

    log_info "Version: $VERSION (major=$MAJOR, minor=$MINOR)"
    log_info "Version suffix for names: $VERSION_SUFFIX"
}

# Run sed command (respects dry-run mode)
run_sed() {
    local pattern="$1"
    local file="$2"

    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY-RUN] Would apply: sed -i '$pattern' $file"
    else
        sed -i "$pattern" "$file"
    fi
}

# Move file (respects dry-run mode)
run_mv() {
    local src="$1"
    local dst="$2"

    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY-RUN] Would rename: $src -> $dst"
    else
        mv "$src" "$dst"
        log_info "Renamed: $src -> $dst"
    fi
}

# Update Tekton pipeline files
update_tekton_pipelines() {
    log_info "Updating Tekton pipeline files..."

    local tekton_dir="${REPO_ROOT}/.tekton"

    if [[ ! -d "$tekton_dir" ]]; then
        log_warning "Tekton directory not found: $tekton_dir"
        return
    fi

    # First, update contents of all yaml files
    for file in "$tekton_dir"/*.yaml; do
        [[ -f "$file" ]] || continue
        log_info "Processing: $(basename "$file")"

        # Update target_branch reference in CEL expression (from main to builds-x.y)
        run_sed "s/target_branch == \"main\"/target_branch == \"$BRANCH\"/g" "$file"

        # Update pipelineRef revision (from main to builds-x.y)
        run_sed "s/value: main$/value: $BRANCH/g" "$file"

        # Update output image names - add version suffix before the colon
        # Order matters: bundle first (more specific), then operator
        run_sed "s|openshift-builds-operator-bundle:|openshift-builds-operator-bundle-$VERSION_SUFFIX:|g" "$file"
        run_sed "s|openshift-builds-operator:|openshift-builds-operator-$VERSION_SUFFIX:|g" "$file"

        # Update pipeline names in metadata.name
        run_sed "s/name: openshift-builds-operator-bundle-on-\(push\|pull-request\)/name: openshift-builds-operator-bundle-$VERSION_SUFFIX-on-\1/g" "$file"
        run_sed "s/name: openshift-builds-operator-on-\(push\|pull-request\)/name: openshift-builds-operator-$VERSION_SUFFIX-on-\1/g" "$file"

        # Update file references in CEL expressions
        run_sed "s|\.tekton/openshift-builds-operator-bundle-\(push\|pull-request\)\.yaml|.tekton/openshift-builds-operator-bundle-$VERSION_SUFFIX-\1.yaml|g" "$file"
        run_sed "s|\.tekton/openshift-builds-operator-\(push\|pull-request\)\.yaml|.tekton/openshift-builds-operator-$VERSION_SUFFIX-\1.yaml|g" "$file"
    done

    # Then rename the files
    log_info "Renaming Tekton pipeline files..."

    # Rename bundle files first (more specific pattern)
    for file in "$tekton_dir"/openshift-builds-operator-bundle-*.yaml; do
        [[ -f "$file" ]] || continue
        if [[ ! "$file" =~ $VERSION_SUFFIX ]]; then
            local newname
            newname=$(echo "$file" | sed "s/openshift-builds-operator-bundle-/openshift-builds-operator-bundle-$VERSION_SUFFIX-/")
            run_mv "$file" "$newname"
        fi
    done

    # Rename operator files (exclude bundle files)
    for file in "$tekton_dir"/openshift-builds-operator-*.yaml; do
        [[ -f "$file" ]] || continue
        if [[ ! "$file" =~ $VERSION_SUFFIX && ! "$file" =~ "bundle" ]]; then
            local newname
            newname=$(echo "$file" | sed "s/openshift-builds-operator-/openshift-builds-operator-$VERSION_SUFFIX-/")
            run_mv "$file" "$newname"
        fi
    done

    log_success "Tekton pipeline files updated"
}

# Update Dockerfiles
update_dockerfiles() {
    log_info "Updating Dockerfiles with version $VERSION..."

    # Update Dockerfile
    local dockerfile="${REPO_ROOT}/Dockerfile"
    if [[ -f "$dockerfile" ]]; then
        log_info "Processing: Dockerfile"

        # Update version label (e.g., v1.7.0 -> v1.8.0)
        run_sed "s/version=\"v[0-9]\+\.[0-9]\+\.[0-9]\+\"/version=\"v${VERSION}.0\"/g" "$dockerfile"

        # Update CPE (e.g., openshift_builds:1.7 -> openshift_builds:1.8)
        run_sed "s/openshift_builds:[0-9]\+\.[0-9]\+/openshift_builds:${VERSION}/g" "$dockerfile"

        # Ensure release is set to 1
        run_sed 's/release="[0-9]\+"/release="1"/g' "$dockerfile"

        log_success "Updated Dockerfile"
    else
        log_warning "Dockerfile not found: $dockerfile"
    fi

    # Update bundle.Dockerfile
    local bundle_dockerfile="${REPO_ROOT}/bundle.Dockerfile"
    if [[ -f "$bundle_dockerfile" ]]; then
        log_info "Processing: bundle.Dockerfile"

        # Update version label
        run_sed "s/version=\"v[0-9]\+\.[0-9]\+\.[0-9]\+\"/version=\"v${VERSION}.0\"/g" "$bundle_dockerfile"

        # Update CPE
        run_sed "s/openshift_builds:[0-9]\+\.[0-9]\+/openshift_builds:${VERSION}/g" "$bundle_dockerfile"

        # Ensure release is set to 1
        run_sed 's/release="[0-9]\+"/release="1"/g' "$bundle_dockerfile"

        log_success "Updated bundle.Dockerfile"
    else
        log_warning "bundle.Dockerfile not found: $bundle_dockerfile"
    fi
}

# Update Makefile
update_makefile() {
    log_info "Updating Makefile with version $VERSION..."

    local makefile="${REPO_ROOT}/Makefile"
    if [[ -f "$makefile" ]]; then
        log_info "Processing: Makefile"

        # Update VERSION (e.g., VERSION ?= 1.6.1 -> VERSION ?= 1.8.0)
        run_sed "s/^VERSION ?= [0-9]\+\.[0-9]\+\.[0-9]\+/VERSION ?= ${VERSION}.0/g" "$makefile"

        # Update CHANNELS (e.g., "latest,openshift-builds-1.6" -> "latest,openshift-builds-1.8")
        run_sed "s/CHANNELS ?= \"latest,openshift-builds-[0-9]\+\.[0-9]\+\"/CHANNELS ?= \"latest,openshift-builds-${VERSION}\"/g" "$makefile"

        log_success "Updated Makefile"
    else
        log_warning "Makefile not found: $makefile"
    fi
}

# Update config manifests
update_config_manifests() {
    log_info "Updating config manifests..."

    local csv_base="${REPO_ROOT}/config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"
    if [[ -f "$csv_base" ]]; then
        log_info "Processing: config/manifests/bases/openshift-builds-operator.clusterserviceversion.yaml"

        # Update documentation URL (e.g., /1.4 -> /1.8)
        run_sed "s|documentation/builds_for_red_hat_openshift/[0-9]\+\.[0-9]\+|documentation/builds_for_red_hat_openshift/${VERSION}|g" "$csv_base"

        log_success "Updated CSV base"
    else
        log_warning "CSV base not found: $csv_base"
    fi
}

# Show summary of changes
show_summary() {
    echo ""
    log_info "=========================================="
    log_info "Release Branch Setup Summary"
    log_info "=========================================="
    echo ""
    echo "Branch:         $BRANCH"
    echo "Version:        $VERSION"
    echo "Version Suffix: $VERSION_SUFFIX"
    echo ""

    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY-RUN MODE: No changes were made"
        echo ""
        echo "Run without --dry-run to apply changes."
    else
        log_success "All changes applied successfully!"
        echo ""
        echo "Changed files:"
        if git rev-parse --is-inside-work-tree &>/dev/null; then
            git status --short
        else
            ls -la "${REPO_ROOT}/.tekton/"
        fi
        echo ""
        echo "Next steps:"
        echo "  1. Review the changes: git diff"
        echo "  2. Stage the changes: git add -A"
        echo "  3. Commit: git commit -m 'chore: configure release branch $BRANCH'"
        echo "  4. Push: git push origin $BRANCH"
    fi
}

# Main function
main() {
    parse_args "$@"

    echo ""
    log_info "=========================================="
    log_info "Release Branch Setup Script"
    log_info "=========================================="
    echo ""

    # Change to repo root
    cd "$REPO_ROOT"

    detect_branch
    validate_branch
    extract_version

    echo ""
    update_tekton_pipelines
    echo ""
    update_makefile
    echo ""
    update_config_manifests
    echo ""
    update_dockerfiles
    echo ""
    show_summary
}

main "$@"
