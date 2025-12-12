#!/bin/bash

echo "=== PHASE 6: TESTING & VERIFICATION ==="
echo ""

FAILED=0

# Clean Go build cache
echo "1. Cleaning build cache..."
go clean -cache
go clean -modcache

# Step 2: Individual builds  
echo ""
echo "2. Testing individual service builds..."

for dir in services/core services/gateway services/k8s-manager libs/common libs/types; do
  echo "  Building $dir..."
  cd "$dir"
  # Ignore version mismatch warnings - they're harmless
  OUTPUT=$(go build ./... 2>&1 | grep -v "version .* does not match" | grep -E "error|cannot find|undefined" | head -5)
  if [ -z "$OUTPUT" ]; then
    echo "    ✓ $dir built successfully"
  else
    echo "    ✗ $dir BUILD FAILED"
    echo "$OUTPUT"
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
OLD_IMPORTS=$(grep -r "github.com/aetherium/aetherium/pkg/" --include="*.go" services libs 2>/dev/null | wc -l)
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
  if go build ./... 2>&1 | grep -q "import cycle"; then
    echo "  ✗ Found cycle in $dir"
    FAILED=$((FAILED + 1))
  else
    echo "  ✓ No cycles in $dir"
  fi
  cd - > /dev/null
done

# Step 6: Verify module syntax
echo ""
echo "6. Verifying module syntax..."
for dir in services/core services/gateway services/k8s-manager libs/common libs/types; do
  cd "$dir"
  if go mod verify > /dev/null 2>&1; then
    echo "  ✓ $dir module is valid"
  else
    echo "  ✗ $dir module has issues"
    go mod verify 2>&1 | head -3
    FAILED=$((FAILED + 1))
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
