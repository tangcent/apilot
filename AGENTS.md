# AGENTS.md

APilot is a Go multi-module workspace (`go.work`) that collects API endpoints
from source code and formats them into Markdown, Postman, YApi, cURL and JSON.
`api-collector` and `api-formatter` are the only shared contracts.

This file is the always-on layer: the few constraints that must never be missed,
plus an index of where the detail lives. Skills are step-by-step workflows;
references are long or situational detail. Keep this file short — anything long,
occasional, or task-specific belongs behind the index, not here.

## Non-negotiable constraints

- Read `CONTRIBUTING.md` before creating a branch, committing, or opening a PR.
  Branch `<type>/<slug>` from `main`; commit `<type>[(scope)][!]: <subject>`.
  The commit type must be one of the 12 in `auto-label.yml` — never `ci:`,
  `doc:`, or `bugfix:`.
- PRs are squash-merged by default. The **PR title** must follow the commit
  format, because it becomes the commit subject on `main`. Release PR titles
  must be `release: cli vX.Y.Z` or `release: plugin vX.Y.Z`.
- Never merge `main` into a PR branch; rebase. GitHub's "Update branch" button
  creates a merge commit whose subject fails `auto-label.yml`.
- `docs:` adds the `type: doc` label and skips the build and coverage jobs —
  documentation-only PRs only.
- Respect the module layers: no module imports above `api-collector` or
  `api-formatter`. Collectors return `nil, nil` when no endpoints are found;
  formatters return valid empty output for an empty endpoint slice.
- Never silently refresh golden files with `-update`. When an export change is
  intentional, refresh deliberately and review the diff.
- Verify by running, not by reading: `bash scripts/test.sh`, then `go vet ./...`
  in each touched module. CI covers Linux/macOS/Windows on Go 1.22 and 1.23.
- Every artifact that enters the repository — code, comments, commit messages,
  PR titles and descriptions — is written in English.

## Index — read on demand

| When you need... | Read |
|---|---|
| Branch, commit, PR and squash rules in full | `CONTRIBUTING.md` → "Branch and commit conventions", "Merging pull requests" |
| Build, test and golden file commands | `CONTRIBUTING.md` → "Development Setup" |
| Module layering and design | `docs/architecture.md` |
| API model and field semantics | `docs/api-model.md` |
| Implementing or extending a collector | `docs/contributing-collectors.md` |
| Implementing or extending a formatter | `docs/contributing-formatters.md` |
| External plugin stdin/stdout protocol | `docs/plugin-protocol.md` |
| CI labels, release triggers, version conventions | `.github/workflows/auto-label.yml`, `release.yml`, `plugin-release.yml`, `pr-release.yml` |
| A step-by-step workflow (issue, bug, PR review, tests) | `.agents/skills/*/SKILL.md` |

## Where new information goes

- **Short, always-true, costly-to-miss constraint** → this file.
- **Step-by-step workflow for a specific task** → `.agents/skills/<name>/SKILL.md`.
- **Long or situational detail** → a reference. The existing references are
  `CONTRIBUTING.md` and `docs/`; put agent-specific detail under
  `.agents/references/` only when it does not belong in human-facing docs, and
  register it in the index above so it can actually be found.

Do not duplicate reference content here or inside a skill. `scripts/init.sh
--all` distributes the skills and generates the tool-specific pointers to this
file.
