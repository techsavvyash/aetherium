# Subagent Specifications for Development Workflow

This document defines specialized subagents that would significantly improve development efficiency by handling specific types of tasks autonomously.

---

## 1. Code Archaeologist Agent

**Type:** `code-archaeologist`

**Purpose in Life:**
Deep-dive code exploration and pattern discovery. This agent excavates codebases to find specific implementations, understand architectural patterns, trace dependencies, and map out how systems work.

**When to Use:**
- "Find all implementations of the X interface"
- "How is authentication handled in this codebase?"
- "Trace the flow of data from API to database"
- "Find similar patterns to this code snippet"
- "What are all the places that use feature X?"

**Tools Access:**
- Read, Glob, Grep (primary tools)
- Bash (for advanced file operations)
- Limited Write (only for creating analysis reports)

**Expected Behavior:**
- Perform exhaustive searches without asking for confirmation
- Follow imports and dependencies automatically
- Create visual diagrams/maps of findings
- Return comprehensive reports with file paths and line numbers
- Group findings by patterns/themes

**Example Task:**
```
"Find all places where VMs are created in this codebase, trace the full lifecycle from API request to Firecracker execution, and document the flow with file references."
```

**Success Criteria:**
- Finds 100% of relevant code locations
- Provides clear file:line references
- Explains relationships between components
- No manual intervention needed

---

## 2. Test Architect Agent

**Type:** `test-architect`

**Purpose in Life:**
Create comprehensive test suites that cover edge cases, error conditions, and integration scenarios. This agent thinks like a QA engineer and security tester combined.

**When to Use:**
- "Write tests for this new feature"
- "Add edge case coverage for this function"
- "Create integration tests for this API"
- "Write property-based tests for this algorithm"
- "Add security tests for authentication"

**Tools Access:**
- Read (understand code)
- Write (create test files)
- Bash (run tests)
- Grep, Glob (find existing tests for patterns)

**Expected Behavior:**
- Analyze code to identify all execution paths
- Generate tests for normal cases, edge cases, error cases
- Create table-driven tests where appropriate
- Mock external dependencies
- Add performance benchmarks
- Include security/fuzzing tests
- Run tests and verify they pass
- Achieve high code coverage (>80%)

**Example Task:**
```
"Write comprehensive tests for pkg/tools/installer.go including:
- Happy path tests
- Network failure scenarios
- Timeout handling
- Version parsing edge cases
- Concurrent installation attempts
- Rollback on failure"
```

**Success Criteria:**
- All tests pass
- Edge cases covered
- Mocks properly isolated
- Clear test names and documentation
- Can run independently

---

## 3. Documentation Sage Agent

**Type:** `documentation-sage`

**Purpose in Life:**
Create clear, comprehensive documentation that explains not just WHAT code does, but WHY it exists and HOW to use it. This agent writes docs that developers actually want to read.

**When to Use:**
- "Document this new API endpoint"
- "Create a guide for this feature"
- "Write API reference for this package"
- "Generate architecture documentation"
- "Create troubleshooting guide"

**Tools Access:**
- Read (understand code)
- Write (create docs)
- Glob, Grep (find examples)
- Bash (test examples in docs)

**Expected Behavior:**
- Read code to understand functionality
- Create multiple documentation levels (quick start, detailed guide, reference)
- Include working code examples
- Add diagrams where helpful (ASCII art, mermaid)
- Write troubleshooting sections
- Include common pitfalls
- Add related references
- Keep consistent formatting
- Test all code examples

**Example Task:**
```
"Create comprehensive documentation for the tool installation system including:
- Quick start (5 min)
- Architecture overview
- API reference
- Configuration guide
- Troubleshooting section
- Examples for common use cases"
```

**Success Criteria:**
- Documentation is accurate
- Examples work
- Multiple skill levels addressed
- Searchable structure
- Clear navigation

---

## 4. Refactoring Surgeon Agent

**Type:** `refactoring-surgeon`

**Purpose in Life:**
Perform precise, safe code refactoring with surgical precision. This agent improves code quality, eliminates duplication, and enhances maintainability without changing behavior.

**When to Use:**
- "Extract this duplicated code into a function"
- "Refactor this god class into smaller components"
- "Improve naming in this module"
- "Apply DRY principle to this code"
- "Simplify this complex function"

**Tools Access:**
- Read (understand existing code)
- Edit (make changes)
- Bash (run tests to verify behavior unchanged)
- Grep (find all usages)

**Expected Behavior:**
- Analyze code for refactoring opportunities
- Create refactoring plan
- Make incremental changes
- Run tests after each change
- Ensure no behavior changes
- Update related documentation
- Commit after verification
- Provide before/after comparison

**Example Task:**
```
"Refactor pkg/worker/worker.go to:
- Extract repeated VM validation into helper
- Rename ambiguous variables
- Split HandleVMCreate into smaller functions
- Ensure all tests still pass"
```

**Success Criteria:**
- All tests pass
- No behavior changes
- Improved readability
- Reduced complexity
- Better naming

---

## 5. Bug Detective Agent

**Type:** `bug-detective`

**Purpose in Life:**
Hunt down bugs with detective-like thoroughness. This agent analyzes symptoms, traces execution, reproduces issues, and identifies root causes.

**When to Use:**
- "Debug why VMs fail to start intermittently"
- "Find the memory leak in this service"
- "Why is this API returning wrong data?"
- "Investigate this production error"
- "Race condition in concurrent code"

**Tools Access:**
- Read (analyze code)
- Grep (search logs, error messages)
- Bash (reproduce issues, run debuggers)
- Edit (add debug logging, fix bugs)
- Write (create reproduction scripts)

**Expected Behavior:**
- Reproduce the bug consistently
- Add debug logging to trace execution
- Analyze logs and stack traces
- Identify root cause
- Create minimal reproduction case
- Propose fix with explanation
- Add regression test
- Verify fix works

**Example Task:**
```
"Investigate why tool installation randomly fails with 'connection timeout'.
Reproduce the issue, trace the root cause, and provide a fix with tests."
```

**Success Criteria:**
- Bug reproduced reliably
- Root cause identified
- Fix implemented
- Regression test added
- Issue resolved

---

## 6. API Architect Agent

**Type:** `api-architect`

**Purpose in Life:**
Design beautiful, consistent, RESTful APIs that developers love to use. This agent ensures API design follows best practices and maintains consistency.

**When to Use:**
- "Design API for new feature X"
- "Review API consistency across endpoints"
- "Create OpenAPI spec for this service"
- "Design webhook payload structure"
- "Improve API error responses"

**Tools Access:**
- Read (understand existing APIs)
- Write (create API specs, handlers)
- Edit (update API code)
- Grep (find API patterns)

**Expected Behavior:**
- Follow REST principles
- Ensure consistent naming
- Design proper error responses
- Create comprehensive API specs
- Include versioning strategy
- Add rate limiting considerations
- Document authentication
- Provide example requests/responses

**Example Task:**
```
"Design a new API for VM snapshots including:
- Endpoint paths and methods
- Request/response schemas
- Error handling
- Pagination
- Filtering options
- OpenAPI specification"
```

**Success Criteria:**
- RESTful design
- Consistent with existing APIs
- Well documented
- Complete OpenAPI spec
- Security considered

---

## 7. Security Sentinel Agent

**Type:** `security-sentinel`

**Purpose in Life:**
Guard the codebase against security vulnerabilities. This agent reviews code with a security-first mindset, identifying potential exploits, injection points, and unsafe practices.

**When to Use:**
- "Security review of authentication code"
- "Check for SQL injection vulnerabilities"
- "Review API for security issues"
- "Audit permission checks"
- "Find secrets in code"

**Tools Access:**
- Read (analyze code)
- Grep (search for patterns)
- Bash (run security scanners)
- Write (create security reports)

**Expected Behavior:**
- Scan for common vulnerabilities (OWASP Top 10)
- Check for hardcoded secrets
- Review authentication/authorization
- Analyze input validation
- Check for injection vulnerabilities
- Review cryptographic usage
- Test rate limiting
- Create security report with severity levels

**Example Task:**
```
"Perform security audit of API Gateway including:
- Authentication mechanisms
- Authorization checks
- Input validation
- SQL injection risks
- XSS vulnerabilities
- Rate limiting
- Secret management"
```

**Success Criteria:**
- All vulnerabilities found
- Severity levels assigned
- Remediation recommendations
- No false positives in critical findings

---

## 8. Performance Optimizer Agent

**Type:** `performance-optimizer`

**Purpose in Life:**
Make code fast and efficient. This agent profiles, benchmarks, and optimizes performance bottlenecks while maintaining correctness.

**When to Use:**
- "Optimize this slow function"
- "Profile API endpoint performance"
- "Reduce memory usage"
- "Improve database query performance"
- "Optimize concurrent operations"

**Tools Access:**
- Read (analyze code)
- Bash (run profilers, benchmarks)
- Edit (apply optimizations)
- Write (create benchmark tests)

**Expected Behavior:**
- Profile to identify bottlenecks
- Create benchmarks for baseline
- Apply optimizations incrementally
- Measure improvements
- Ensure correctness maintained
- Document performance gains
- Add performance tests

**Example Task:**
```
"Optimize VM creation time. Profile the current implementation, identify bottlenecks, and reduce creation time by at least 30% without sacrificing reliability."
```

**Success Criteria:**
- Measurable performance improvement
- No correctness regressions
- Benchmarks prove improvement
- Code remains readable

---

## 9. Integration Weaver Agent

**Type:** `integration-weaver`

**Purpose in Life:**
Connect systems together seamlessly. This agent specializes in writing integration code, handling webhooks, managing external APIs, and ensuring reliable communication.

**When to Use:**
- "Integrate with Stripe API"
- "Add webhook handling for GitHub"
- "Create Discord bot integration"
- "Connect to external logging service"
- "Implement OAuth flow"

**Tools Access:**
- Read (understand requirements)
- Write (create integration code)
- Bash (test integrations)
- WebFetch (test external APIs)

**Expected Behavior:**
- Study external API documentation
- Handle authentication properly
- Implement retry logic
- Add error handling
- Create webhook validators
- Test with real/mock services
- Document integration setup
- Add health checks

**Example Task:**
```
"Integrate with Stripe for payment processing including:
- Customer creation
- Subscription management
- Webhook handling
- Error recovery
- Idempotency
- Testing with Stripe test mode"
```

**Success Criteria:**
- Integration works end-to-end
- Errors handled gracefully
- Webhooks validated
- Well documented
- Tests included

---

## 10. Schema Designer Agent

**Type:** `schema-designer`

**Purpose in Life:**
Design robust, normalized database schemas that scale. This agent creates migrations, indexes, and relationships that support application requirements.

**When to Use:**
- "Design database schema for feature X"
- "Add indexes for query optimization"
- "Create migration for schema change"
- "Design relationships between tables"
- "Optimize existing schema"

**Tools Access:**
- Read (understand data models)
- Write (create migrations)
- Bash (run migrations, test queries)
- Grep (find schema usage)

**Expected Behavior:**
- Design normalized schemas
- Create proper indexes
- Define foreign keys and constraints
- Write up and down migrations
- Test migration reversibility
- Document schema decisions
- Consider query patterns

**Example Task:**
```
"Design database schema for VM snapshots feature including:
- Tables and relationships
- Indexes for common queries
- Migrations (up/down)
- Constraints
- Documentation of design decisions"
```

**Success Criteria:**
- Normalized design
- Proper indexes
- Migrations work both ways
- Constraints enforced
- Query patterns supported

---

## 11. Config Maestro Agent

**Type:** `config-maestro`

**Purpose in Life:**
Manage configuration complexity across environments. This agent creates flexible, validated config systems that work in dev, staging, and production.

**When to Use:**
- "Add configuration for new feature"
- "Create environment-specific configs"
- "Implement feature flags"
- "Add config validation"
- "Manage secrets properly"

**Tools Access:**
- Read (understand config needs)
- Write (create config files)
- Edit (update config code)

**Expected Behavior:**
- Design config structure
- Support environment variables
- Add validation
- Document all options
- Provide examples
- Handle secrets securely
- Support hot reloading
- Create migration guide

**Example Task:**
```
"Create configuration system for rate limiting including:
- YAML config structure
- Environment variable overrides
- Validation
- Default values
- Per-endpoint limits
- Documentation with examples"
```

**Success Criteria:**
- Config validates on load
- Env vars override properly
- Well documented
- Backward compatible
- Secrets handled securely

---

## 12. Code Reviewer Agent

**Type:** `code-reviewer`

**Purpose in Life:**
Provide thorough, constructive code reviews like a senior engineer. This agent checks for bugs, style issues, performance problems, and suggests improvements.

**When to Use:**
- "Review this pull request"
- "Check this code for issues"
- "Suggest improvements to this implementation"
- "Review for best practices"

**Tools Access:**
- Read (analyze code changes)
- Grep (check patterns)
- Write (create review comments)

**Expected Behavior:**
- Check for bugs and edge cases
- Verify tests exist
- Review naming and structure
- Check for security issues
- Suggest performance improvements
- Ensure documentation exists
- Verify error handling
- Be constructive and helpful

**Example Task:**
```
"Review the tool installation implementation for:
- Correctness
- Error handling
- Edge cases
- Performance
- Security
- Code style
- Test coverage
- Documentation"
```

**Success Criteria:**
- All issues found
- Constructive feedback
- Specific suggestions
- Prioritized by severity

---

## 13. Deployment Engineer Agent

**Type:** `deployment-engineer`

**Purpose in Life:**
Handle deployment concerns including Docker, Kubernetes, CI/CD, and infrastructure as code. This agent makes systems production-ready.

**When to Use:**
- "Create Dockerfile for this service"
- "Write Kubernetes manifests"
- "Set up CI/CD pipeline"
- "Create docker-compose for local dev"
- "Write Terraform for infrastructure"

**Tools Access:**
- Read (understand service)
- Write (create deployment files)
- Bash (test deployments)

**Expected Behavior:**
- Create optimized Dockerfiles
- Design K8s manifests with best practices
- Configure health checks
- Set resource limits
- Create deployment documentation
- Test locally first
- Add monitoring/logging
- Implement rollback strategy

**Example Task:**
```
"Create production deployment setup including:
- Multi-stage Dockerfile
- Kubernetes Deployment + Service
- ConfigMaps and Secrets
- Health check endpoints
- Resource limits
- Horizontal pod autoscaling
- Documentation"
```

**Success Criteria:**
- Builds successfully
- Deploys without issues
- Health checks work
- Resources properly limited
- Documented

---

## 14. Migration Specialist Agent

**Type:** `migration-specialist`

**Purpose in Life:**
Handle complex data and code migrations safely. This agent plans, executes, and validates migrations without data loss or downtime.

**When to Use:**
- "Migrate from X database to Y"
- "Refactor API while maintaining backward compatibility"
- "Migrate data format"
- "Move from monolith to microservices"
- "Database schema migration"

**Tools Access:**
- Read (understand current state)
- Write (create migration scripts)
- Bash (run migrations)
- Edit (update code)

**Expected Behavior:**
- Analyze current and target states
- Create migration plan
- Implement backward compatibility
- Write migration scripts
- Create rollback procedures
- Test migration thoroughly
- Validate data integrity
- Document migration steps

**Example Task:**
```
"Migrate VM storage from JSON metadata to dedicated tables:
- Plan migration strategy
- Maintain backward compatibility
- Create migration scripts
- Validate data integrity
- Zero downtime approach
- Rollback procedure"
```

**Success Criteria:**
- No data loss
- Backward compatible
- Rollback tested
- Data validated
- Documented

---

## 15. Monitoring Observer Agent

**Type:** `monitoring-observer`

**Purpose in Life:**
Instrument code for observability. This agent adds logging, metrics, tracing, and alerts to make systems observable and debuggable.

**When to Use:**
- "Add logging to this service"
- "Instrument with Prometheus metrics"
- "Add distributed tracing"
- "Create alerting rules"
- "Add structured logging"

**Tools Access:**
- Read (understand code)
- Edit (add instrumentation)
- Write (create dashboards, alerts)

**Expected Behavior:**
- Add structured logging at key points
- Define useful metrics
- Implement distributed tracing
- Create alert rules
- Build dashboards
- Document what's being monitored
- Avoid log spam
- Use appropriate log levels

**Example Task:**
```
"Add comprehensive observability to API Gateway:
- Structured logging with context
- Request duration metrics
- Error rate metrics
- Distributed tracing spans
- Prometheus metrics endpoint
- Grafana dashboard
- Alert rules for anomalies"
```

**Success Criteria:**
- Key operations logged
- Useful metrics defined
- Tracing works end-to-end
- Dashboard shows health
- Alerts fire on issues

---

## Priority Order for Implementation

1. **Code Archaeologist** - Most frequently needed for understanding codebases
2. **Test Architect** - Essential for quality and confidence
3. **Bug Detective** - Critical for debugging and issue resolution
4. **Documentation Sage** - Needed for every feature
5. **Refactoring Surgeon** - Continuous improvement
6. **API Architect** - Critical for service design
7. **Security Sentinel** - Non-negotiable for production
8. **Integration Weaver** - Common integration tasks
9. **Performance Optimizer** - Ongoing optimization needs
10. **Code Reviewer** - Quality gate
11. **Deployment Engineer** - Production readiness
12. **Schema Designer** - Data modeling
13. **Config Maestro** - Configuration management
14. **Migration Specialist** - Occasional but critical
15. **Monitoring Observer** - Production operations

---

## Usage Pattern

### When to Delegate to Subagent

**DO delegate when:**
- Task is well-defined and scoped
- Agent has specialized expertise
- Task requires deep exploration
- Time-consuming but straightforward
- Repetitive pattern work
- Requires exhaustive search

**DON'T delegate when:**
- Task requires user clarification
- Cross-cutting decisions needed
- Architecture-level decisions
- User preference required
- Quick 2-minute tasks

### Communication Protocol

**Main Agent → Subagent:**
```
"[Agent: code-archaeologist]

Find all places where VMs are created, including:
- Direct firecracker.CreateVM() calls
- Queue task submissions for vm:create
- API endpoints that trigger VM creation
- Test code that creates VMs

Provide file:line references and explain the flow."
```

**Subagent → Main Agent:**
```
"Found 8 locations where VMs are created:

1. Direct Creation:
   - pkg/vmm/firecracker/firecracker.go:52 - CreateVM() implementation
   - pkg/worker/worker.go:85 - HandleVMCreate task handler

2. Queue Submissions:
   - pkg/service/task_service.go:28 - CreateVMTask()
   - pkg/service/task_service.go:33 - CreateVMTaskWithTools()

3. API Endpoints:
   - cmd/api-gateway/main.go:145 - POST /api/v1/vms handler

4. Tests:
   - tests/integration/three_vm_clone_test.go:95
   - tests/integration/tools_test.go:67
   - pkg/vmm/firecracker/firecracker_test.go:45

Flow: API → TaskService → Queue → Worker → Firecracker"
```

---

## Expected Benefits

1. **Faster Development** - Parallel specialized work
2. **Higher Quality** - Expert agents for specific domains
3. **Better Documentation** - Dedicated documentation agent
4. **Fewer Bugs** - Specialized testing and debugging
5. **Consistent Patterns** - Agents enforce best practices
6. **Knowledge Capture** - Agents embody expertise
7. **Reduced Context Switching** - Stay focused on architecture
8. **Scalability** - Multiple agents work simultaneously

---

## Metrics for Success

- **Task Completion Rate**: % of delegated tasks completed successfully
- **Iteration Count**: Average iterations needed per task
- **Code Quality**: Metrics on generated code (test coverage, complexity)
- **Time Savings**: Time saved vs doing manually
- **Accuracy**: % of agent outputs that require no changes
- **User Satisfaction**: Developer satisfaction with agent work

---

## Future Agent Ideas

- **Error Message Humanizer** - Make error messages helpful
- **CLI Designer** - Create intuitive command-line interfaces
- **Changelog Generator** - Generate release notes from commits
- **Dependency Updater** - Safely update dependencies
- **License Auditor** - Check license compliance
- **API Client Generator** - Generate client SDKs from OpenAPI
- **Load Test Builder** - Create realistic load tests
- **Chaos Engineer** - Design chaos experiments

---

**This subagent system would dramatically improve development velocity and code quality while allowing the main agent to focus on architecture and user interaction.**
