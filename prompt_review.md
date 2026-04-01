# jrdev — Review phase

You are in the **issue git worktree** on **`{{.IssueBranch}}`**. **jrdev** only runs this phase when **implement** produced at least one commit.

## Scope

- Focus on **this issue only** (#{{.IssueNumber}} {{.IssueTitle}}): correctness, tests, rule compliance, and consistency with existing style.
- Prefer **small, actionable** follow-up edits on the same branch; no scope creep to unrelated issues.
- Do **not** close issues, change labels, or merge to **`main`**.

## Context
Here are the last 10 commits:
{{.CommitHistory}}

Here is the current Diff:
{{.GitDiff}}

Integration branch: **`{{.IntegrationBranch}}`**

## Issue body

{{.IssueBody}}

### Linked issues and PRDs (private GitHub)

If the body links to other issues (e.g. a PRD) by URL. read them with **`gh issue view`** from the repo root, not unauthenticated HTTP/Cursor **WebFetch**

## Review Process

### 1. Read the diff and look for anything dodgy  
Read the diff carefully. For anything that looks suspicious — fragile logic, unchecked assumptions, tricky conditions, implicit type coercions, missing guards — write a test that exercises it. Try to actually break it. If you can break it, fix it.

### 2. Stress Test Edge Cases  
Go beyond the happy path. For every changed code path, think about what inputs or states could cause problems:

- Empty arrays, empty strings, zero, negative numbers
- Missing optional fields, null values, undefined properties
- Rapid repeated calls, race conditions, state that changes mid-operation
- Off-by-one errors in loops or slice/substring operations
- Regressions in adjacent functionality

Write tests for anything that isn't already covered.

### 3. Analize for Code Quality Improvements  
Look for opportunities to:

- Reduce unnecessary complexity and nesting
- Eliminate redundant code and abstractions
- Improve readability through clear variable and function names
- Consolidate related logic
- Remove unnecessary comments that describe obvious code
- Avoid nested ternary operators - prefer switch statements or if/else chains
- Choose clarity over brevity - explicit code is often better than overly compact code

### 4. Maintain Balance   
Avoid over-simplification that could:

- Reduce code clarity or maintainability
- Create overly clever solutions that are hard to understand
- Combine too many concerns into single functions or components
- Remove helpful abstractions that improve code organization
- Make the code harder to debug or extend

### 5. Apply Project Standards  
Follow the established coding standards in the project

### 6. Preserve Functionality  
Never change what the code does, only how it does it. All original features, outputs, and behaviors must remain intact.

## Execution  
1. Run `go vet ./..` and `go test ./..` first to confirm the current state passes
2. Attempt to reproduce the original bug with new test cases - if you can, fix it.
3. Write edge case tests that stress the implementation
4. make any code quality improvements directly on this branch
5. run `go vet ./..` and `go test ./..` again to verify nothing has broken
6. Commit with a message starting with `JRDEV: Review -` describing the refinements. 

If the code is already clean, well-tested, and handles edge cases properly, do nothing.

Once complete, output COMPLETE.