# Phase 6: Testing & Verification - Execution Guide

**Status**: Ready to Execute  
**Duration**: 1-2 hours  
**Effort**: Systematic testing and verification  
**Branch**: `refactor/monorepo-restructure` (already on this branch)

---

## Overview

Phase 6 verifies that all services build independently, the workspace integrates correctly, and no functional regressions have occurred. This is the final verification before Phase 7 (Deployment).

---

## Step 1: Clean Build Environment

Start fresh to ensure repeatable builds:

```bash
# From root
go clean -cache
go mod verify
go work sync
```

---

## Step 2: Individual Service Builds

Test each service builds completely in isolation:

### 2.1 Core Service Build

```bash
cd services/core

# Build all packages
go build ./...

# Verify no import errors
go mod verify

# Check for any build warnings
go build -v ./... 2>&1 | grep -i "warning\|error"
```

**Expected Output**: Clean build, no errors or warnings

### 2.2 Gateway Service Build

```bash
cd ../gateway

# Build all packages
go build ./...

# Verify no import errors
go mod verify

# Check for any build warnings
go build -v ./... 2>&1 | grep -i "warning\|error"
```

**Expected Output**: Clean build, no errors or warnings

### 2.3 K8s Manager Service Build

```bash
cd ../k8s-manager

# Build all packages
go build ./...

# Verify no import errors
go mod verify
```

**Expected Output**: Clean build (may have minimal dependencies)

### 2.4 Common Library Build

```bash
cd ../../libs/common

# Build all packages
go build ./...

# Verify module
go mod verify
```

**Expected Output**: Clean build

### 2.5 Types Library Build

```bash
cd ../types

# Build all packages
go build ./...

# Verify module
go mod verify
```

**Expected Output**: Clean build

---

## Step 3: Workspace Integration

Test the go.work workspace integration:

```bash
# From root
cd /to/root

# Sync workspace
go work sync

# Verify no errors
echo $?  # Should output 0
```

**Expected Output**: No errors, exit code 0

---

## Step 4: Run Test Suite

Execute all tests to ensure no regressions:

### 4.1 Core Service Tests

```bash
cd services/core

# Run tests
go test ./... -v

# Or with coverage
go test ./... -cover
```

**Expected**: All tests pass

### 4.2 Gateway Service Tests

```bash
cd ../gateway

# Run tests
go test ./... -v
```

**Expected**: All tests pass

### 4.3 Common Library Tests

```bash
cd ../../libs/common

# Run tests
go test ./... -v
```

**Expected**: All tests pass (if any)

---

## Step 5: Integration Tests

Run integration tests from the tests/ directory:

```bash
# From root
cd tests/integration

# Run integration tests (may need special setup)
go test -v -timeout 30m ./...
```

**Expected**: All integration tests pass

---

## Step 6: Verify No Import Cycles

Check for circular import dependencies:

```bash
# From root
cd /to/root

# Check for cycles in core
cd services/core
go build ./... 2>&1 | grep -i "cycle"

# Check for cycles in gateway
cd ../gateway
go build ./... 2>&1 | grep -i "cycle"

# Check for cycles in common
cd ../../libs/common
go build ./... 2>&1 | grep -i "cycle"
```

**Expected Output**: No cycle errors (empty output)

---

## Step 7: Verify Build System

Test the Makefile build system:

```bash
# From root
cd /to/root

# Test Makefile.monorepo
make -f Makefile.monorepo build 2>&1 | tail -20

# Should complete successfully
```

**Expected**: Build succeeds, produces binaries in bin/

---

## Step 8: Check for Old Import Paths

Final verification that no old imports remain:

```bash
# From root
cd /to/root

# Search for old import paths
grep -r "github.com/aetherium/aetherium/pkg/" --include="*.go" . | grep -v "Binary"

# Should output nothing
```

**Expected Output**: Empty (no matches)

---

## Step 9: Verify Directory Structure

Ensure files are in correct locations:

```bash
# From root
cd /to/root

# Check services structure
tree services/ -d -L 2

# Check libs structure
tree libs/ -d -L 2

# Check infrastructure structure
tree infrastructure/ -d -L 2
```

**Expected Output**: Clean hierarchy with appropriate subdirectories

---

## Step 10: Go Module Health Check

Verify all go.mod files are well-formed:

```bash
# From root
cd /to/root

# Verify each module
for dir in services/* libs/*; do
  echo "Checking $dir..."
  cd "$dir"
  go mod graph > /dev/null 2>&1
  if [ $? -eq 0 ]; then
    echo "  ✓ OK"
  else
    echo "  ✗ FAILED"
  fi
  cd - > /dev/null
done
```

**Expected Output**: All modules OK

---

## Automated Verification Script

Run this comprehensive verification script:

```bash
#!/bin/bash

echo "=== PHASE 6: TESTING & VERIFICATION ==="
echo ""

FAILED=0

# Step 1: Clean environment
echo "1. Cleaning build environment..."
go clean -cache
go mod verify

# Step 2: Individual builds
echo ""
echo "2. Testing individual service builds..."

for dir in services/core services/gateway services/k8s-manager libs/common libs/types; do
  echo "  Building $dir..."
  cd "$dir"
  if go build ./... > /dev/null 2>&1; then
    echo "    ✓ $dir built successfully"
  else
    echo "    ✗ $dir BUILD FAILED"
    FAILED=$((FAILED + 1))
  fi
  cd - > /dev/null
done

# Step 3: Workspace sync
echo ""
echo "3. Syncing workspace..."
if go work sync > /dev/null 2>&1; then
  echo "  ✓ Workspace synced successfully"
else
  echo "  ✗ Workspace sync FAILED"
  FAILED=$((FAILED + 1))
fi

# Step 4: Check for old imports
echo ""
echo "4. Checking for old import paths..."
OLD_IMPORTS=$(grep -r "github.com/aetherium/aetherium/pkg/" --include="*.go" . 2>/dev/null | wc -l)
if [ "$OLD_IMPORTS" -eq 0 ]; then
  echo "  ✓ No old imports found"
else
  echo "  ✗ Found $OLD_IMPORTS old imports"
  FAILED=$((FAILED + 1))
fi

# Step 5: Check for import cycles
echo ""
echo "5. Checking for import cycles..."
for dir in services/core services/gateway libs/common; do
  cd "$dir"
  if go build ./... 2>&1 | grep -q "cycle"; then
    echo "  ✗ Found cycle in $dir"
    FAILED=$((FAILED + 1))
  else
    echo "  ✓ No cycles in $dir"
  fi
  cd - > /dev/null
done

# Results
echo ""
echo "=== VERIFICATION COMPLETE ==="
if [ $FAILED -eq 0 ]; then
  echo "✅ All checks passed! Ready for Phase 7."
  exit 0
else
  echo "❌ $FAILED check(s) failed. Review errors above."
  exit 1
fi
```

Save this as `scripts/verify-phase6.sh` and run:

```bash
chmod +x scripts/verify-phase6.sh
./scripts/verify-phase6.sh
```

---

## Checklist: Phase 6 Verification

- [ ] **Build System**
  - [ ] Core service builds independently
  - [ ] Gateway service builds independently
  - [ ] K8s-manager service builds independently
  - [ ] Common library builds independently
  - [ ] Types library builds independently
  - [ ] Root level build (go.work) succeeds

- [ ] **Tests**
  - [ ] Core service tests pass
  - [ ] Gateway service tests pass
  - [ ] Common library tests pass
  - [ ] Integration tests pass (if applicable)

- [ ] **Code Quality**
  - [ ] No old import paths remaining
  - [ ] No import cycles detected
  - [ ] go mod verify passes for all modules
  - [ ] No syntax errors or warnings

- [ ] **Structure**
  - [ ] Services in correct locations
  - [ ] Libraries in correct locations
  - [ ] Infrastructure code organized
  - [ ] All directories present

- [ ] **Module Configuration**
  - [ ] All go.mod files well-formed
  - [ ] Replace directives correct
  - [ ] Dependencies properly declared
  - [ ] go.work includes all modules

---

## Troubleshooting Phase 6

### Problem: "Module not found" error

**Cause**: Replace directives in go.mod may be incorrect

**Solution**:
```bash
# Verify relative paths are correct
cat services/core/go.mod | grep "replace"

# Should see:
# replace github.com/aetherium/aetherium/libs/common => ../../libs/common
# replace github.com/aetherium/aetherium/libs/types => ../../libs/types
```

### Problem: "import cycle not allowed" error

**Cause**: Service imports create circular dependency

**Solution**:
1. Identify the circular import
2. Move shared code to a library
3. Import from library instead
4. Re-run go mod tidy

### Problem: Build fails with version mismatch

**Cause**: Go version mismatch between go.mod and environment

**Solution**:
```bash
# Ensure consistent Go versions
go version

# Update all go.mod files if needed
for f in $(find . -name "go.mod"); do
  sed -i 's/go 1.25/go 1.25.3/g' "$f"
done

# Re-tidy
go mod tidy
```

### Problem: Tests fail unexpectedly

**Cause**: Import path changes may affect test behavior

**Solution**:
1. Run tests with verbose output: `go test -v`
2. Check test file imports are updated
3. Verify test dependencies resolve correctly
4. Run test in isolation: `go test ./path/to/test -v`

---

## What Gets Verified

After Phase 6, you can be confident:

✅ **All services build independently** - Can develop and deploy individually  
✅ **No import cycles** - Module structure is clean  
✅ **No old references** - Migration is complete  
✅ **Tests pass** - No functional regressions  
✅ **Workspace integrates** - go.work correctly coordinates modules  

---

## Next Steps

Once ALL Phase 6 checks pass:

1. ✅ Review checklist above
2. → Commit final verification results
3. → Move to **Phase 7: Deployment**

---

## Success Criteria

Phase 6 is complete when:

- ✅ All services build cleanly
- ✅ All tests pass
- ✅ No import errors or cycles
- ✅ No old import paths remain
- ✅ Workspace syncs successfully
- ✅ Team confirms no regressions

---

**Status**: Ready to Execute  
**Last Updated**: December 12, 2025  
**Next**: Phase 7 (Deployment & CI/CD)
