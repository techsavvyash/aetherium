#!/bin/bash
# =============================================================================
# Aetherium Rootfs Build and Upload Script
# =============================================================================
# Builds the rootfs with tools and uploads to cloud storage for auto-scaling.
#
# Usage:
#   ./scripts/build-and-upload-rootfs.sh [provider] [bucket]
#
# Providers: aws, azure, gcp
# Examples:
#   ./scripts/build-and-upload-rootfs.sh aws my-bucket
#   ./scripts/build-and-upload-rootfs.sh azure mystorageaccount/container
#   ./scripts/build-and-upload-rootfs.sh gcp my-gcs-bucket
# =============================================================================

set -euo pipefail

PROVIDER="${1:-}"
BUCKET="${2:-}"
ROOTFS_PATH="/var/firecracker/rootfs.ext4"
ROOTFS_NAME="aetherium-rootfs-$(date +%Y%m%d).ext4"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

usage() {
    cat <<EOF
Usage: $0 [provider] [bucket]

Providers:
  aws     - Upload to S3
  azure   - Upload to Azure Blob Storage
  gcp     - Upload to Google Cloud Storage
  local   - Just build, don't upload

Examples:
  $0 aws my-aetherium-bucket
  $0 azure mystorageaccount/mycontainer
  $0 gcp my-gcs-bucket
  $0 local
EOF
    exit 1
}

# =============================================================================
# Build Rootfs
# =============================================================================
build_rootfs() {
    log "Building rootfs with tools..."

    if [ ! -f ./scripts/prepare-rootfs-with-tools.sh ]; then
        error "prepare-rootfs-with-tools.sh not found. Run from aetherium root."
    fi

    # Build rootfs
    sudo ./scripts/prepare-rootfs-with-tools.sh

    if [ ! -f "$ROOTFS_PATH" ]; then
        error "Rootfs not found at $ROOTFS_PATH after build"
    fi

    log "Rootfs built successfully: $ROOTFS_PATH"
    log "Size: $(du -h $ROOTFS_PATH | cut -f1)"
}

# =============================================================================
# Upload to AWS S3
# =============================================================================
upload_aws() {
    local bucket="$1"

    log "Uploading to S3: s3://${bucket}/${ROOTFS_NAME}"

    if ! command -v aws &>/dev/null; then
        error "AWS CLI not installed. Install with: pip install awscli"
    fi

    aws s3 cp "$ROOTFS_PATH" "s3://${bucket}/${ROOTFS_NAME}" \
        --acl private

    # Also upload as 'latest'
    aws s3 cp "$ROOTFS_PATH" "s3://${bucket}/aetherium-rootfs-latest.ext4" \
        --acl private

    log "Upload complete!"
    log "URL: https://${bucket}.s3.amazonaws.com/${ROOTFS_NAME}"
    log "Latest: https://${bucket}.s3.amazonaws.com/aetherium-rootfs-latest.ext4"

    # Generate presigned URL for Helm values
    PRESIGNED_URL=$(aws s3 presign "s3://${bucket}/${ROOTFS_NAME}" --expires-in 604800)
    log ""
    log "Add to values.yaml:"
    log "  nodeProvisioner:"
    log "    rootfsUrl: \"${PRESIGNED_URL}\""
}

# =============================================================================
# Upload to Azure Blob Storage
# =============================================================================
upload_azure() {
    local storage_path="$1"
    local account="${storage_path%%/*}"
    local container="${storage_path#*/}"

    log "Uploading to Azure: ${account}/${container}/${ROOTFS_NAME}"

    if ! command -v az &>/dev/null; then
        error "Azure CLI not installed. Install with: curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash"
    fi

    az storage blob upload \
        --account-name "$account" \
        --container-name "$container" \
        --name "$ROOTFS_NAME" \
        --file "$ROOTFS_PATH" \
        --auth-mode login

    # Also upload as 'latest'
    az storage blob upload \
        --account-name "$account" \
        --container-name "$container" \
        --name "aetherium-rootfs-latest.ext4" \
        --file "$ROOTFS_PATH" \
        --auth-mode login \
        --overwrite

    log "Upload complete!"
    log "URL: https://${account}.blob.core.windows.net/${container}/${ROOTFS_NAME}"

    # Generate SAS URL
    EXPIRY=$(date -u -d "+7 days" '+%Y-%m-%dT%H:%MZ')
    SAS_URL=$(az storage blob generate-sas \
        --account-name "$account" \
        --container-name "$container" \
        --name "$ROOTFS_NAME" \
        --permissions r \
        --expiry "$EXPIRY" \
        --auth-mode login \
        --as-user \
        --full-uri \
        -o tsv)

    log ""
    log "Add to values.yaml:"
    log "  nodeProvisioner:"
    log "    rootfsUrl: \"${SAS_URL}\""
}

# =============================================================================
# Upload to Google Cloud Storage
# =============================================================================
upload_gcp() {
    local bucket="$1"

    log "Uploading to GCS: gs://${bucket}/${ROOTFS_NAME}"

    if ! command -v gsutil &>/dev/null; then
        error "Google Cloud SDK not installed. Install with: curl https://sdk.cloud.google.com | bash"
    fi

    gsutil cp "$ROOTFS_PATH" "gs://${bucket}/${ROOTFS_NAME}"

    # Also upload as 'latest'
    gsutil cp "$ROOTFS_PATH" "gs://${bucket}/aetherium-rootfs-latest.ext4"

    log "Upload complete!"
    log "URL: https://storage.googleapis.com/${bucket}/${ROOTFS_NAME}"

    # Generate signed URL
    SIGNED_URL=$(gsutil signurl -d 7d "$ROOTFS_PATH" "gs://${bucket}/${ROOTFS_NAME}" 2>/dev/null | tail -1 | awk '{print $NF}')

    log ""
    log "Add to values.yaml:"
    log "  nodeProvisioner:"
    log "    rootfsUrl: \"${SIGNED_URL}\""
}

# =============================================================================
# Main
# =============================================================================
main() {
    if [ -z "$PROVIDER" ]; then
        usage
    fi

    log "=============================================="
    log "  Aetherium Rootfs Build & Upload"
    log "=============================================="
    log "Provider: $PROVIDER"
    log "Bucket: ${BUCKET:-N/A}"
    log "=============================================="

    # Check if running as root (needed for rootfs build)
    if [ "$EUID" -ne 0 ] && [ "$PROVIDER" != "upload-only" ]; then
        error "This script must be run as root (for rootfs build)"
    fi

    # Build rootfs
    build_rootfs

    # Upload based on provider
    case "$PROVIDER" in
        aws)
            [ -z "$BUCKET" ] && error "Bucket name required for AWS"
            upload_aws "$BUCKET"
            ;;
        azure)
            [ -z "$BUCKET" ] && error "Storage account/container required for Azure"
            upload_azure "$BUCKET"
            ;;
        gcp)
            [ -z "$BUCKET" ] && error "Bucket name required for GCP"
            upload_gcp "$BUCKET"
            ;;
        local)
            log "Rootfs built locally. Not uploading."
            log "Path: $ROOTFS_PATH"
            ;;
        *)
            error "Unknown provider: $PROVIDER"
            ;;
    esac

    log ""
    log "=============================================="
    log "  Done!"
    log "=============================================="
}

main "$@"
