# Root Directory File Archive

This document indexes all files that were consolidated from the root directory into the `docs/` folder for better organization.

## File Organization

### Setup & Installation
- `SETUP.md` → Copied to docs/ as comprehensive setup guide
- `QUICKSTART.md` → Merged into existing docs/QUICKSTART.md
- `QUICK-START.md` → Redundant, consolidated
- `QUICKSTART_GUIDE.md` → Redundant, consolidated
- `PRE_EXECUTION_CHECKLIST.md` → Pre-flight verification checklist

### Execution & Testing
- `EXECUTION_SUMMARY.md` → E2E testing guide and documentation index
- `EXECUTION_INDEX.md` → Navigation guide for execution docs
- `MANUAL_RUN_STEPS.md` → Step-by-step execution walkthrough
- `MANUAL_TEST_RESULTS.md` → Test result documentation
- `VISUAL_WALKTHROUGH.md` → Visual guide with expected outputs
- `WEBSOCKET_TEST_PLAN.md` → WebSocket testing documentation

### Bug Fixes & Solutions
- `ERROR_MESSAGE_FIX.md` → Error handling and fixes
- `VSOCK_FIX.md` → vsock communication fixes
- `VM_LOGIN_ISSUE.md` → VM login troubleshooting
- `WEBSOCKET_FIXES.md` → WebSocket implementation fixes
- `WEBSOCKET_IMPLEMENTATION.md` → WebSocket feature implementation
- `WEBSOCKET_STREAMING_COMPLETE.md` → WebSocket streaming completion notes
- `PROXY_IMPLEMENTATION.md` → Proxy implementation details
- `TUI_STREAMING_FIX.md` → TUI streaming fixes

### Project Documentation
- `IMPLEMENTATION-SUMMARY.md` → Implementation summary and status
- `FIX_PLAN.md` → Historical fix planning document
- `SUBAGENT-SPECIFICATIONS.md` → Sub-agent specifications
- `WELCOME-BACK.md` → Welcome back documentation
- `WALKTHROUGH.md` → System walkthrough
- `README_EXECUTION.md` → Execution-specific README
- `COMMANDS_REFERENCE.md` → Reference of all commands
- `POSTGRES_CREDS.md` → Database credential reference
- `CORRECT_CREDENTIALS.md` → Credential verification

### Development Files
- `AGENTS.md` → AI agent guidelines (SHOULD NOT BE DELETED)
- `CLAUDE.md` → Claude-specific instructions (SHOULD NOT BE DELETED)
- `TODO.md` → Project TODO list
- `Makefile` → Build automation
- `go.mod` / `go.sum` → Go dependencies

### Scripts
- `fix-rootfs.sh` → rootfs fixing script
- `start-services.sh` → Service startup script
- `stop-services.sh` → Service shutdown script
- `RUN_NOW.sh` → Quick run script

### System Files
- `docker-compose.yml` → Docker service definitions
- `api-gateway` → Binary (built)
- `worker` → Binary (built)
- `api-gateway.log` → Log file
- `worker.log` → Log file
- `.git/` → Git repository
- `.gitignore` → Git ignore rules

## Migration Status

✅ **Recommended for docs/** (Good candidates):
- SETUP.md
- EXECUTION_SUMMARY.md
- EXECUTION_INDEX.md
- MANUAL_RUN_STEPS.md
- VISUAL_WALKTHROUGH.md
- MANUAL_TEST_RESULTS.md
- PRE_EXECUTION_CHECKLIST.md
- COMMANDS_REFERENCE.md
- FIX_PLAN.md
- IMPLEMENTATION-SUMMARY.md
- SUBAGENT-SPECIFICATIONS.md
- WEBSOCKET_TEST_PLAN.md
- PROXY_IMPLEMENTATION.md

❌ **Keep in root** (Essential):
- README.md
- Makefile
- docker-compose.yml
- go.mod / go.sum
- .gitignore
- .git/
- AGENTS.md (guidelines)
- CLAUDE.md (AI agent instructions)

⚠️ **Cleanup** (Redundant/Outdated):
- QUICKSTART.md (duplicate)
- QUICK-START.md (duplicate)
- QUICKSTART_GUIDE.md (duplicate)
- WELCOME-BACK.md (outdated)
- WALKTHROUGH.md (outdated)
- README_EXECUTION.md (covered in other docs)
- POSTGRES_CREDS.md (security - should be in env)
- CORRECT_CREDENTIALS.md (security - should be in env)
- VM_LOGIN_ISSUE.md (specific bug, resolved)
- ERROR_MESSAGE_FIX.md (specific fix, applied)
- WEBSOCKET_FIXES.md (applied)
- WEBSOCKET_IMPLEMENTATION.md (covered in design docs)
- WEBSOCKET_STREAMING_COMPLETE.md (completed feature)
- VSOCK_FIX.md (applied)
- PROXY_IMPLEMENTATION.md (design doc)

🗑️ **Delete** (Build artifacts/logs):
- api-gateway (binary)
- worker (binary)
- api-gateway.log
- worker.log
- bin/ (build artifacts)
- RUN_NOW.sh (temporary script)
- fix-rootfs.sh (temporary script)

## Next Steps

1. Move selected docs to `docs/` folder
2. Update README.md to reference docs/ folder
3. Delete redundant/outdated files from root
4. Keep AGENTS.md and CLAUDE.md for AI guidelines
5. Verify all references are updated
6. Update navigation in EXECUTION_INDEX.md if moved

## Notes

- All confidential credentials should move to environment variables or .env files
- Binary files should not be committed - ensure .gitignore covers them
- Log files are not needed in repo - add to .gitignore
- Keep the root directory clean with only essential files
- Documentation structure should follow docs/ organization
