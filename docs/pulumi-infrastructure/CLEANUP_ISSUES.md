# Pulumi Infrastructure Cleanup Issues

## Issues Found

### 1. **aetherium.ts Issues**

#### Issue 1.1: Incorrect Helm Chart Path
- **Line 30**: `chart: "../helm/aetherium"` - relative path won't work in production
- **Fix**: Use absolute path or proper Pulumi asset reference

#### Issue 1.2: Missing Service Type Specification Error
- **Lines 146-154**: Trying to get Deployment/DaemonSet by ID immediately after creation
- **Problem**: K8s resources aren't created synchronously, IDs don't exist yet
- **Fix**: Return the created resources directly from helm chart, not by re-querying

#### Issue 1.3: Incorrect Function Parameter Type
- **Line 20**: `namespace: pulumi.Output<string>` - passed from index.ts as metadata.name
- **Line 34**: Same issue in deployInfrastructure
- **Fix**: Proper handling of namespace as Output

### 2. **infrastructure.ts Issues**

#### Issue 2.1: PostgreSQL Secret Type Issue
- **Line 52**: Using `pulumi.secret()` but it needs to be a string
- **Fix**: Use proper secret reference or string password

#### Issue 2.2: Missing Service Return Type
- **Line 381**: `postgresService.metadata.name` returns `Output<string>` not `string`
- **Line 386**: Same for Redis service
- **Fix**: Return Output types or interpolate properly

#### Issue 2.3: Consul Service Reference Issue
- **Line 312**: `consulService.metadata.name` type mismatch
- **Fix**: Handle Output types properly

#### Issue 2.4: Loki StatefulSet Not Exported in Interface
- **Line 30**: Interface defines `loki?: k8s.apps.v1.StatefulSet`
- **But returned at line 390** without proper reference
- **Fix**: Ensure consistent typing

### 3. **index.ts Issues**

#### Issue 3.1: Async Function Not Awaited
- **Line 59**: `export const outputs = main();` - async function returns Promise
- **Fix**: Handle promise properly with Pulumi

#### Issue 3.2: Status Property on Deployment
- **Lines 48-49**: `.status` doesn't exist on Deployment type
- **Fix**: Return the resource itself, not status

#### Issue 3.3: Missing Configuration Handling
- **Line 14**: `provider` read but never used
- **Fix**: Route based on provider or remove

### 4. **namespace.ts Issues**
- **Line 14**: `name: name` - redundant but harmless
- Minor: Could use optional parameters better

### 5. **bare-metal.ts Issues**

#### Issue 5.1: Incorrect Node Query
- **Line 236**: `k8s.core.v1.Node.get("worker-nodes", "")` - invalid parameters
- **Fix**: Use proper node selector with provider

#### Issue 5.2: Hard-coded Worker Node Count
- **Line 237**: `pulumi.output(1)` - doesn't reflect actual cluster state
- **Fix**: Query cluster for actual node count

### 6. **node-pools.ts Issues**

#### Issue 6.1: Complex Cloud-Init String Interpolation
- **Lines 89-95**: Template string interpolation may fail
- **Fix**: Cleaner string interpolation

#### Issue 6.2: Missing Configuration Validation
- **Line 43-45**: rootfsUrl is empty by default
- **Fix**: Document required parameters or provide defaults

### 7. **Cross-Module Issues**

#### Issue 7.1: Inconsistent Type Handling
- Some functions return `Output<T>`, others return `T`
- **Fix**: Standardize on Output types throughout

#### Issue 7.2: Missing Error Handling
- No try-catch or error handling in any deployment functions
- **Fix**: Add error handling and validation

#### Issue 7.3: Missing Documentation
- Functions lack JSDoc comments on complex parameters
- **Fix**: Add comprehensive documentation

#### Issue 7.4: Configuration Not Passed Through
- index.ts reads config but doesn't pass to all functions
- **Fix**: Create unified config object passed to all functions

## Priority Fixes

### High Priority (Breaks Deployment)
1. Fix namespace parameter type handling in deployAetherium
2. Fix Helm chart path reference
3. Fix postgres secret type
4. Fix deployment resource retrieval (not by ID after creation)
5. Fix async/promise handling in index.ts

### Medium Priority (May Cause Runtime Errors)
1. Fix node query in bare-metal.ts
2. Fix worker node count query
3. Standardize Output type handling
4. Add error handling

### Low Priority (Code Quality)
1. Add comprehensive documentation
2. Add validation for required config
3. Improve string interpolation
4. Code cleanup and refactoring

## Next Steps

1. Create fixed versions of each file
2. Add proper error handling
3. Test locally with `pulumi preview`
4. Document configuration requirements
5. Add validation layer
