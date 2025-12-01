#!/bin/bash
# =============================================================================
# Aetherium Kubernetes Deployment Script
# =============================================================================
# Usage:
#   ./scripts/deploy-k8s.sh [environment] [action]
#
# Environments: dev, staging, prod
# Actions: deploy, upgrade, rollback, delete
#
# Examples:
#   ./scripts/deploy-k8s.sh dev deploy
#   ./scripts/deploy-k8s.sh prod upgrade
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="${1:-dev}"
ACTION="${2:-deploy}"
NAMESPACE="aetherium"
RELEASE_NAME="aetherium"
CHART_PATH="./helm/aetherium"
TIMEOUT="10m"

# Validate environment
case "$ENVIRONMENT" in
    dev|development)
        VALUES_FILE="values.yaml"
        NAMESPACE="aetherium-dev"
        ;;
    staging)
        VALUES_FILE="values.yaml"
        NAMESPACE="aetherium-staging"
        ;;
    prod|production)
        VALUES_FILE="values-production.yaml"
        NAMESPACE="aetherium"
        ;;
    *)
        echo -e "${RED}Invalid environment: $ENVIRONMENT${NC}"
        echo "Valid environments: dev, staging, prod"
        exit 1
        ;;
esac

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed"
    fi

    if ! command -v helm &> /dev/null; then
        error "helm is not installed"
    fi

    # Check kubectl connection
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi

    success "All prerequisites met"
}

# Create namespace if it doesn't exist
ensure_namespace() {
    log "Ensuring namespace $NAMESPACE exists..."

    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        kubectl create namespace "$NAMESPACE"
        kubectl label namespace "$NAMESPACE" \
            app.kubernetes.io/managed-by=helm \
            environment="$ENVIRONMENT"
        success "Created namespace $NAMESPACE"
    else
        log "Namespace $NAMESPACE already exists"
    fi
}

# Label nodes for worker scheduling
label_worker_nodes() {
    log "Checking for KVM-enabled nodes..."

    # Get nodes with KVM
    KVM_NODES=$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$KVM_NODES" ]; then
        warn "No nodes found. Make sure to label KVM-enabled nodes with:"
        warn "  kubectl label nodes <node-name> aetherium.io/kvm-enabled=true"
    else
        log "Found nodes: $KVM_NODES"
        log "Ensure KVM-enabled nodes are labeled with: aetherium.io/kvm-enabled=true"
    fi
}

# Update Helm dependencies
update_dependencies() {
    log "Updating Helm dependencies..."

    cd "$CHART_PATH"
    helm dependency update
    cd - > /dev/null

    success "Dependencies updated"
}

# Deploy or upgrade
deploy() {
    log "Deploying Aetherium to $NAMESPACE (environment: $ENVIRONMENT)..."

    ensure_namespace
    update_dependencies

    # Check if release exists
    if helm status "$RELEASE_NAME" -n "$NAMESPACE" &> /dev/null; then
        log "Upgrading existing release..."
        helm upgrade "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$CHART_PATH/$VALUES_FILE" \
            --timeout "$TIMEOUT" \
            --wait \
            --atomic
    else
        log "Installing new release..."
        helm install "$RELEASE_NAME" "$CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$CHART_PATH/$VALUES_FILE" \
            --timeout "$TIMEOUT" \
            --wait \
            --atomic \
            --create-namespace
    fi

    success "Deployment complete!"

    # Show status
    log "Deployment status:"
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance="$RELEASE_NAME"
}

# Rollback
rollback() {
    log "Rolling back Aetherium in $NAMESPACE..."

    REVISION="${3:-}"

    if [ -z "$REVISION" ]; then
        log "Rolling back to previous revision..."
        helm rollback "$RELEASE_NAME" -n "$NAMESPACE" --wait
    else
        log "Rolling back to revision $REVISION..."
        helm rollback "$RELEASE_NAME" "$REVISION" -n "$NAMESPACE" --wait
    fi

    success "Rollback complete!"
}

# Delete
delete() {
    log "Deleting Aetherium from $NAMESPACE..."

    read -p "Are you sure you want to delete the deployment? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        helm uninstall "$RELEASE_NAME" -n "$NAMESPACE"
        success "Deployment deleted"

        read -p "Delete namespace $NAMESPACE as well? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kubectl delete namespace "$NAMESPACE"
            success "Namespace deleted"
        fi
    else
        log "Aborted"
    fi
}

# Show status
status() {
    log "Aetherium status in $NAMESPACE:"

    echo ""
    echo "=== Helm Release ==="
    helm status "$RELEASE_NAME" -n "$NAMESPACE" 2>/dev/null || echo "No release found"

    echo ""
    echo "=== Pods ==="
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/instance="$RELEASE_NAME" -o wide

    echo ""
    echo "=== Services ==="
    kubectl get svc -n "$NAMESPACE" -l app.kubernetes.io/instance="$RELEASE_NAME"

    echo ""
    echo "=== Recent Events ==="
    kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' | tail -10
}

# Main
main() {
    echo "=============================================="
    echo "  Aetherium Kubernetes Deployment"
    echo "=============================================="
    echo "Environment: $ENVIRONMENT"
    echo "Namespace:   $NAMESPACE"
    echo "Action:      $ACTION"
    echo "=============================================="
    echo ""

    check_prerequisites
    label_worker_nodes

    case "$ACTION" in
        deploy|install|upgrade)
            deploy
            ;;
        rollback)
            rollback "$@"
            ;;
        delete|uninstall)
            delete
            ;;
        status)
            status
            ;;
        *)
            error "Invalid action: $ACTION. Valid actions: deploy, upgrade, rollback, delete, status"
            ;;
    esac
}

main "$@"
