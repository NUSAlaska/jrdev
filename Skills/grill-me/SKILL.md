---
name: grill-me
description: Stress-test a plan or design through sequential Q&A—one question per turn—walking depth-first down each decision branch until resolved. Use when user wants to stress-test a plan, get grilled on their design, or mentions "grill me".
---

# Grill me

Explore the plan or design until you and the user share a clear picture. Work **depth-first**: choose **one** branch of the decision tree, drill into it until every detail that matters for that branch is settled, then move to the **next** branch. **Never** hop between unrelated branches in parallel; finish the current branch (or explicitly agree it is deferred) before starting another.

## Interaction rules (non-negotiable)

1. Ask **exactly one** question per reply. No multi-part "Question 1 / Question 2", no bullet lists of questions, no trailing "also, ...?" in the same message.
2. **Wait** for the user's answer before asking the next question.
3. Use the answer to pick the next question along the **same** branch until that branch is nailed down; only then switch to a sibling or parent branch.
4. For that single question, you **may** give your **recommended answer** (briefly—why you lean that way). Do not let that become a second question in the same turn.
5. If the question can be answered by exploring the codebase, **explore first**, then ask **one** follow-up only if something is still ambiguous.

End the message with one clear question the user can answer in their next reply.
