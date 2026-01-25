# Postmortem: PR #25 Harness Framework Implementation

**Date:** 2025-01-25
**Duration of issue:** ~5 review cycles over multiple sessions
**Author:** Claude (with tedks)
**Status:** Resolved

## Summary

PR #25 introduced a harness framework for multi-agent support, consisting of ~15 commits across 4 implementation phases. The PR went through 5+ rounds of code review, with several rounds addressing similar classes of issues (concurrency bugs, resource leaks, security gaps) that appeared across multiple components. This postmortem examines why similar issues recurred and how to prevent this pattern.

## Impact

- **Developer time:** Estimated 3-4x more review cycles than necessary
- **Risk exposure:** Several security and reliability bugs made it to review stage
- **Complexity:** Each fix round added defensive code that could have been designed in from the start

## Timeline

| Commit | Description | Issues Addressed |
|--------|-------------|------------------|
| `ba86e88` | Initial harness framework | - |
| `1b27cb7` | Fix resource management | nil ctx panic, pipe leaks, goroutine coordination, bash parser false positives |
| `52ae4e0` | Fix concurrency & security | mutex for `running`, pool goroutine leaks, env var validation, SafeFilePath |
| `8546c5a` | Fix goroutine leak | RunAdvisorAsync goroutine tracking, parse warnings |
| `0f12806` | Fix blocking issues | harness selection, error events, crash monitor, thread-safety docs |
| `2e45868` | Round 4 fixes | SafeFilePath tests, process exit errors, pipe cleanup |
| `97aabe3` | Self-review fixes | nested locks, error context, constants extraction |
| `19f35e3` | Final race condition | cmd.Wait() called twice, SafeFilePath absolute path bypass |

## Root Causes

### RC1: No Concurrency Design Document

**What happened:** Concurrency bugs appeared in multiple components (BaseHarness, ClaudeHarness, Pool, GameScene) because there was no upfront design for goroutine lifecycle, channel ownership, and synchronization strategy.

**Evidence:**
- `running` field had no mutex initially
- `cmd.Wait()` was called from both `monitorProcess()` and `Stop()`
- Events channel could be closed while writers were still active
- `done` channel, `WaitGroup`, and `closeOnce` were added incrementally across 4 commits

**Why it wasn't caught earlier:** Each component was implemented in isolation. Concurrency patterns weren't established as a framework-wide concern.

### RC2: Incomplete Security Threat Model

**What happened:** Security measures were added reactively rather than designed systematically:
- `SafeFilePath()` initially only checked for `..` prefix, missing absolute paths
- Environment variable validation (LD_PRELOAD, PATH) was added in a later commit
- File path handling was spread across multiple methods without centralized validation

**Evidence:**
- Commit `52ae4e0`: Added SafeFilePath and env var validation
- Commit `19f35e3`: Fixed SafeFilePath to reject absolute paths outside baseDir

**Why it wasn't caught earlier:** No threat model was created before implementation. Security was treated as a feature rather than a cross-cutting concern.

### RC3: Missing Integration Test Strategy

**What happened:** Unit tests passed but integration scenarios revealed bugs:
- Concurrent Start/Stop revealed double-close panics
- Full lifecycle tests revealed goroutine leaks
- Process crash scenarios revealed consumer deadlocks

**Evidence:**
- Commit `3869e99`: Added lifecycle integration tests *after* initial implementation
- These tests immediately found the double-close bug

**Why it wasn't caught earlier:** Testing strategy focused on unit tests. Complex state interactions weren't tested until problems appeared in review.

### RC4: Incremental Fixes Without Pattern Recognition

**What happened:** When a bug was found in one component, similar bugs in other components weren't proactively identified:
- Mutex added to BaseHarness.running, but similar races existed in Pool
- Goroutine leak fixed in Pool.broadcastEvent, but RunAdvisorAsync had same issue
- Channel close protection added to ClaudeHarness, but GameScene needed it too

**Evidence:** Multiple commits fixing "same pattern, different location":
- `1b27cb7`: Fixed ClaudeHarness goroutine coordination
- `52ae4e0`: Fixed Pool goroutine leaks
- `8546c5a`: Fixed RunAdvisorAsync goroutine leaks
- `fa77bcc`: Fixed GameScene event processor cleanup

**Why it wasn't caught earlier:** Fixes were applied locally to where the bug was reported, without searching for the same pattern elsewhere.

## Lessons Learned

### What went well

1. **Initial architecture was sound** - The Harness interface, Registry pattern, and Event system didn't require fundamental redesign
2. **Test coverage caught regressions** - New tests prevented re-introducing fixed bugs
3. **Code review caught real bugs** - Every round found legitimate issues, not just style nits

### What went wrong

1. **Concurrency was retrofitted** - Goroutine lifecycle, channel ownership, and shutdown coordination were added incrementally instead of designed upfront
2. **Security was reactive** - Path traversal and env var validation were added after initial implementation
3. **Pattern fixes were local** - Same bug class was fixed multiple times across different files
4. **Integration tests came late** - Lifecycle tests written after problems found in review

## Corrective Actions

### Immediate (this PR)

| Action | Owner | Status |
|--------|-------|--------|
| Fix cmd.Wait() race condition | Claude | Done |
| Complete SafeFilePath protection | Claude | Done |
| Add monitorWg for goroutine tracking | Claude | Done |

### Short-term (next PRs)

| Action | Owner | Due |
|--------|-------|-----|
| Create CONCURRENCY.md documenting goroutine patterns | tedks/Claude | Next PR |
| Add go vet and race detector to CI | tedks | Next PR |
| Audit all channel close sites for sync.Once | Claude | Next PR |

### Long-term (process changes)

| Action | Owner | Due |
|--------|-------|-----|
| Require concurrency design doc for new async code | tedks | Ongoing |
| Add threat model template to PR checklist | tedks | Ongoing |
| Run `-race` in CI for all tests | tedks | CI update |
| Add pattern search step to bug fixes | Claude | Ongoing |

## Preventive Measures

### For Claude (agent)

1. **When implementing concurrency:** Before writing code, explicitly state:
   - Which goroutines exist and who owns them
   - Which channels exist, their ownership, and close responsibility
   - What mutex protects which fields
   - Shutdown ordering and coordination mechanism

2. **When fixing a bug:** Before committing, search for the same pattern:
   ```
   grep -r "similar_pattern" --include="*.go"
   ```
   Ask: "Does this bug class exist elsewhere?"

3. **When handling file paths:** Always validate against a base directory. Never trust cleaned paths alone.

4. **When adding security measures:** Create a threat model first:
   - What are the untrusted inputs?
   - What could a malicious input do?
   - Where are all the validation points?

### For tedks (reviewer)

1. **First review pass:** Check for concurrency design before diving into logic
2. **Pattern audit:** When one bug is found, request audit of similar code
3. **Security checklist:** Verify path handling, env vars, command construction
4. **Integration test requirement:** Require lifecycle tests for state machines

## Metrics for Success

Future PRs with async code should:
- Pass `-race` detector in CI
- Have concurrency documented before implementation
- Have lifecycle integration tests before first review
- Have security threat model in PR description

## Appendix: Issue Categories

| Category | Count | Examples |
|----------|-------|----------|
| Race conditions | 4 | running field, cmd.Wait(), channel close, Pool map |
| Goroutine leaks | 3 | readOutput, RunAdvisorAsync, broadcastEvent |
| Resource leaks | 2 | stdout/stderr pipes, events channel |
| Security gaps | 2 | SafeFilePath absolute paths, env var injection |
| Missing error handling | 2 | process exit errors, parse failures |
