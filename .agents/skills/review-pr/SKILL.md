---
name: "review-pr"
description: "Reviews a GitHub pull request end-to-end: lands the code in a local worktree, judges whether the problem is real and the fix stays in scope, verifies by running builds and tests, then reports graded findings with a merge recommendation. Invoke when the user asks to review a PR, check a pull request, or decide whether a PR should be merged."
---

# Review PR

## Principle

Judge the change before the code. A PR that fixes a non-existent problem, or rewrites unrelated modules, is a bad PR however clean the diff looks. Every conclusion must be backed by files on disk and commands actually run — not by remote file blobs alone.

Phases 1-4 answer the four judgment questions (intent, reality, solution, scope), Phase 5 reviews details, Phase 7 reports. Phases 0 and 6 are the non-negotiable additions: land the code locally, and verify by running it.

---

## Phase 0 — Land the PR locally

Never review by fetching remote files one by one. Read the real tree so surrounding code, sibling modules and tests are available.

```bash
git fetch origin refs/pull/<N>/head:refs/pr/<N>
git worktree add <local-path> refs/pr/<N>
gh pr view <N> --json title,body,baseRefName,headRefName,files
git log --oneline main..refs/pr/<N>
git diff --name-status main...refs/pr/<N>
```

For stacked PRs, first confirm how they sit on top of each other. Compare `--name-only` outputs to isolate this PR's own increment, and use `git log --oneline main..refs/pr/<N>` to confirm linear stacking:

```bash
git diff --name-only main...refs/pr/<A> | sort > /tmp/pr-a.txt
git diff --name-only main...refs/pr/<B> | sort > /tmp/pr-b.txt
comm -13 /tmp/pr-a.txt /tmp/pr-b.txt   # files introduced by B only
```

Do not attribute another PR's files to this one.

---

## Phase 1 — What problem does the PR claim to solve?

- Restate it in one sentence, from the title, body, linked issues, commit messages and changed tests.
- Write down the acceptance criteria the author implies.
- If the description and the diff disagree, flag it as blocking until clarified.

## Phase 2 — Is the problem real, and worth solving?

Evidence, not intuition:

1. Reproduce on `main` before looking at the fix — a failing command, test, or export output.
2. Search the repo: is the capability already implemented elsewhere (another collector or formatter)?
3. Check the linked issue state, existing tests and docs — is this by design, already solved, or cosmetic?
4. Decide: `real` / `not reproducible` / `by design` / `already solved` / `cosmetic`.

If the answer is anything but `real`, say so and stop. Do not review the details of a change that should not exist.

## Phase 3 — Is the solution reasonable?

- Is it the smallest change that fixes the problem? Does it reuse existing capabilities instead of adding new layers?
- Respect the contracts in `CONTRIBUTING.md`:
  - collectors return `nil, nil` when no endpoints are found
  - formatters return valid empty output for an empty endpoints slice
  - `api-collector` and `api-formatter` are the only shared contracts; no module imports above its layer
  - external plugins use the stdin/stdout JSON protocol
- Does it fit the `go.work` module boundaries, without new cross-module dependencies or cycles?
- Name one simpler alternative. If the PR chose a heavier one, ask why.

## Phase 4 — Does the change exceed the stated problem?

Build a file-level change matrix and classify **every** changed file:

| Path | Status | Lines | In scope? | Note |
|------|--------|-------|-----------|------|
| `path/to/file` | M | +12/-3 | in scope | ... |

- Labels: `in scope` / `supporting` (tests, docs, CI) / `unrelated`.
- Watch for whole-file reformatting, drive-by refactors, dependency bumps, symbol renames, generated files, and unexplained golden file churn.
- Out-of-scope changes are not automatically wrong, but call them out explicitly and ask for a split when they obscure the fix.

## Phase 5 — Review the details

Each finding must carry `file:line`, why it matters, and a concrete suggested change.

- Correctness: edge cases, nil/empty values, error paths, concurrency, platform differences (CI covers Linux/macOS/Windows).
- Regression: existing callers, output stability, golden files under `apilot-cli/testdata/goproject/`.
- Tests: is the new behavior covered? Do tests assert behavior rather than implementation? Would they fail on `main`?
- Docs: README, CONTRIBUTING and CHANGELOG consistent with the new behavior.
- Conventions: branch name (`<type>/<slug>`) and every commit message follow the always-on constraints in `AGENTS.md` and the full rules in `CONTRIBUTING.md`; commit types are in the `auto-label.yml` whitelist; no merge commits in the PR branch; English-only artifacts; no stray build output or temporary files.
- Squash merge: the **PR title** follows `<type>[(scope)][!]: <subject>` too, because it becomes the commit subject on `main`. For release PRs the title must be `release: cli vX.Y.Z` or `release: plugin vX.Y.Z`, otherwise `release.yml` does not trigger. Check that the PR is one logical change, since the squashed commit cannot be split later.

## Phase 6 — Verify by running

```bash
bash scripts/test.sh                 # go test ./... per module
(cd <module> && go vet ./...)        # mirrors .github/workflows/ci.yml
(cd apilot-cli && go test ./...)     # golden files
npm test                            # scripts/install.test.js, when JS changed
```

- CI runs `go test ./... && go vet ./...` per module on Linux/macOS/Windows with Go 1.22/1.23; replicate at least the affected modules.
- If a golden file changed, confirm the export change is intended and the `-update` refresh was deliberate, not a silent baseline rewrite.
- Report the exact commands and their results. A review without a run is an opinion, not a review.

---

## Phase 7 — Report

Conclusion first, then numbered reasons.

```markdown
## Verdict: Approve / Request changes / Comment
<one sentence>

### Problem
Is it real? Worth solving? What is the evidence?

### Solution and scope
Approach assessment + file-level change matrix.

### Findings
| Level | Location | Issue | Suggested change |
|-------|----------|-------|------------------|
| Blocking | `path:line` | ... | ... |

### Verification
Commands run and pass/fail.

### Open questions
```

Rules:

- Three levels only: **Blocking** (must fix before merge), **Non-blocking** (should fix), **Nit** (optional).
- Never bury a blocking issue among nits.
- Never say "LGTM" without stating what was verified.
- Report only — do not edit the PR branch unless the user explicitly asks.

## Notes

- Remove the temporary worktree when done if you created it: `git worktree remove <local-path>`.
- Scope is a correctness property here: a well-written but out-of-scope PR still needs an explicit scope finding.
