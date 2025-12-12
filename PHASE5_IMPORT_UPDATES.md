# Phase 5: Import Path Updates - Execution Guide

**Status**: In Progress  
**Duration**: 2-3 hours  
**Effort**: Systematic, low risk  
**Branch**: `refactor/monorepo-restructure` (already created)

---

## Overview

Phase 5 requires updating all Go import paths throughout the codebase from old locations (in flat `/pkg`) to new locations (in service/lib structure).

This must happen in all:
- Source files (`.go`)
- Test files (`*_test.go`)
- Command files (`cmd/*/main.go`)

---

## Import Path Mapping

### Services

#### Core Service
```go
// OLD → NEW

"github.com/techsavvyash/aetherium/pkg/vmm" 
→ "github.com/techsavvyash/aetherium/services/core/pkg/vmm"

"github.com/techsavvyash/aetherium/pkg/network"
→ "github.com/techsavvyash/aetherium/services/core/pkg/network"

"github.com/techsavvyash/aetherium/pkg/storage"
→ "github.com/techsavvyash/aetherium/services/core/pkg/storage"

"github.com/techsavvyash/aetherium/pkg/queue"
→ "github.com/techsavvyash/aetherium/services/core/pkg/queue"

"github.com/techsavvyash/aetherium/pkg/tools"
→ "github.com/techsavvyash/aetherium/services/core/pkg/tools"

"github.com/techsavvyash/aetherium/pkg/service"
→ "github.com/techsavvyash/aetherium/services/core/pkg/service"

"github.com/techsavvyash/aetherium/pkg/worker"
→ "github.com/techsavvyash/aetherium/services/core/pkg/worker"

"github.com/techsavvyash/aetherium/pkg/mcp"
→ "github.com/techsavvyash/aetherium/services/core/pkg/mcp"
```

#### Gateway Service
```go
"github.com/techsavvyash/aetherium/pkg/api"
→ "github.com/techsavvyash/aetherium/services/gateway/pkg/api"

"github.com/techsavvyash/aetherium/pkg/integrations"
→ "github.com/techsavvyash/aetherium/services/gateway/pkg/integrations"

"github.com/techsavvyash/aetherium/pkg/websocket"
→ "github.com/techsavvyash/aetherium/services/gateway/pkg/websocket"

"github.com/techsavvyash/aetherium/pkg/discovery"
→ "github.com/techsavvyash/aetherium/services/gateway/pkg/discovery"
```

### Shared Libraries

#### Common Library
```go
"github.com/techsavvyash/aetherium/pkg/logging"
→ "github.com/techsavvyash/aetherium/libs/common/pkg/logging"

"github.com/techsavvyash/aetherium/pkg/config"
→ "github.com/techsavvyash/aetherium/libs/common/pkg/config"

"github.com/techsavvyash/aetherium/pkg/container"
→ "github.com/techsavvyash/aetherium/libs/common/pkg/container"

"github.com/techsavvyash/aetherium/pkg/events"
→ "github.com/techsavvyash/aetherium/libs/common/pkg/events"
```

#### Types Library
```go
"github.com/techsavvyash/aetherium/pkg/types"
→ "github.com/techsavvyash/aetherium/libs/types/pkg/domain"
```

---

## Step 1: Find All Import Statements

Before making changes, identify all files that need updating:

```bash
# Find all Go files with old import paths
grep -r "github.com/techsavvyash/aetherium/pkg/" --include="*.go" . | head -50

# Count files affected
find . -name "*.go" -type f -exec grep -l "github.com/techsavvyash/aetherium/pkg/" {} \; | wc -l
```

---

## Step 2: Automated Import Updates

Use sed to update imports in batches. **Run these commands from the repository root.**

### Core Service Imports

```bash
# vmm package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/vmm|github.com/techsavvyash/aetherium/services/core/pkg/vmm|g' {} \;

# network package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/network|github.com/techsavvyash/aetherium/services/core/pkg/network|g' {} \;

# storage package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/storage|github.com/techsavvyash/aetherium/services/core/pkg/storage|g' {} \;

# queue package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/queue|github.com/techsavvyash/aetherium/services/core/pkg/queue|g' {} \;

# tools package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/tools|github.com/techsavvyash/aetherium/services/core/pkg/tools|g' {} \;

# service package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/service|github.com/techsavvyash/aetherium/services/core/pkg/service|g' {} \;

# worker package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/worker|github.com/techsavvyash/aetherium/services/core/pkg/worker|g' {} \;

# mcp package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/mcp|github.com/techsavvyash/aetherium/services/core/pkg/mcp|g' {} \;
```

### Gateway Service Imports

```bash
# api package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/api|github.com/techsavvyash/aetherium/services/gateway/pkg/api|g' {} \;

# integrations package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/integrations|github.com/techsavvyash/aetherium/services/gateway/pkg/integrations|g' {} \;

# websocket package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/websocket|github.com/techsavvyash/aetherium/services/gateway/pkg/websocket|g' {} \;

# discovery package
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/discovery|github.com/techsavvyash/aetherium/services/gateway/pkg/discovery|g' {} \;
```

### Shared Library Imports

```bash
# logging package (common)
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/logging|github.com/techsavvyash/aetherium/libs/common/pkg/logging|g' {} \;

# config package (common)
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/config|github.com/techsavvyash/aetherium/libs/common/pkg/config|g' {} \;

# container package (common)
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/container|github.com/techsavvyash/aetherium/libs/common/pkg/container|g' {} \;

# events package (common)
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/events|github.com/techsavvyash/aetherium/libs/common/pkg/events|g' {} \;

# types package → types/domain
find . -name "*.go" -type f -exec sed -i 's|github.com/techsavvyash/aetherium/pkg/types|github.com/techsavvyash/aetherium/libs/types/pkg/domain|g' {} \;
```

---

## Step 3: Verify Changes

After running sed commands, verify the updates:

```bash
# Check if any old imports remain
grep -r "github.com/techsavvyash/aetherium/pkg/" --include="*.go" . | grep -v "Binary file" | head -20

# If empty output, all imports are updated ✓
```

---

## Step 4: Update go.mod and go.sum Files

After imports are updated, update module files:

```bash
# In core service
cd services/core
go mod tidy

# In gateway service
cd ../../services/gateway
go mod tidy

# In common library
cd ../../libs/common
go mod tidy

# In types library
cd ../types
go mod tidy

# From root, sync workspace
cd ../..
go work sync
```

---

## Step 5: Fix Import Cycles (if any)

Check for circular dependencies:

```bash
# From root
go work vendor

# Check for specific issues
cd services/core && go mod verify
cd ../gateway && go mod verify
cd ../..
```

If you see "cycle" errors:
1. Review the imports in the conflicting packages
2. Move circular dependencies to a shared library
3. Re-run `go mod tidy`

Common solutions:
- Move shared types to `libs/types`
- Move shared utilities to `libs/common`
- Avoid importing from higher-level services

---

## Step 6: Syntax Validation

Ensure no syntax errors were introduced:

```bash
# Check all Go files for syntax errors
find . -name "*.go" -type f -exec gofmt -l {} \; | head -20

# If output, files need formatting:
gofmt -w .
```

---

## Step 7: Build Verification (Quick)

Before moving to Phase 6, do a quick check:

```bash
# Core service
cd services/core
go build ./...

# Gateway service
cd ../gateway
go build ./...

# Root level (if go.work set up correctly)
cd ../..
go work sync
```

If builds succeed, imports are correct.

---

## Step 8: Commit Changes

Once verified, commit the import updates:

```bash
# From root
git add -A
git commit -m "Phase 5: Update import paths for monorepo structure

- Updated all imports from /pkg to /services/{service}/pkg
- Updated shared library imports to /libs/{library}/pkg
- Ran go mod tidy in all services
- Verified no import cycles or syntax errors
- Builds succeed for all services"
```

---

## Common Issues & Solutions

### Issue: "could not locate package X"
**Solution**: Check the import path matches exactly:
```bash
# Find where the package actually is
find services libs -type d -name "your_package"

# Update import to match the found location
```

### Issue: "import cycle not allowed"
**Solution**:
1. Identify the circular dependency
2. Extract shared code to `libs/common` or `libs/types`
3. Import from the shared library instead

### Issue: sed command didn't work
**Solution**:
```bash
# Use more conservative approach
grep -r "OLD_IMPORT" --include="*.go" | cut -d: -f1 | sort -u | while read f; do
  sed -i 's|OLD_IMPORT|NEW_IMPORT|g' "$f"
done
```

### Issue: Accidental replacement in comments
**Solution**: Revert and use more specific grep:
```bash
git checkout -- .
# Only replace in actual import statements:
grep -r 'import (' --include="*.go" -A 50 | grep "OLD_IMPORT"
# Then update those manually or with better regex
```

---

## Verification Checklist

- [ ] All old import paths replaced
- [ ] No old imports remain (`grep` returns nothing)
- [ ] `go mod tidy` runs without errors
- [ ] `go work sync` succeeds
- [ ] No import cycles (`go mod verify` clean)
- [ ] All services build: `go build ./...`
- [ ] No syntax errors in updated files
- [ ] Changes committed with descriptive message

---

## When to Move to Phase 6

Once ALL of the above checklist items are complete, you're ready for **Phase 6: Testing & Verification**.

Phase 6 will:
- Test each service builds independently
- Run full test suite
- Verify no functional regressions
- Check production readiness

---

## Time Estimate

- Finding imports: 10 minutes
- Running sed commands: 5 minutes
- Verifying changes: 20 minutes
- Running go mod tidy: 10 minutes
- Build verification: 10 minutes
- **Total: ~55 minutes**

---

## Parallel Execution

If you have team members, this can be parallelized:

**Person 1**: Core service imports
```bash
cd services/core
# Update imports in cmd/ and pkg/
# Run go mod tidy
```

**Person 2**: Gateway service imports
```bash
cd services/gateway
# Update imports in cmd/ and pkg/
# Run go mod tidy
```

**Person 3**: Shared libraries imports
```bash
cd libs/common && go mod tidy
cd ../types && go mod tidy
```

Then sync everything from root:
```bash
go work sync
```

---

## Next Steps After Phase 5

Once complete:
1. ✅ Commit your changes
2. → Move to **Phase 6: Testing & Verification**
3. → Then **Phase 7: Deployment**

See `RESTRUCTURING_COMPLETED.md` for Phase 6 details.

---

**Status**: Ready to execute  
**Last Updated**: December 12, 2025  
**Next**: Phase 6 (Testing & Verification)
