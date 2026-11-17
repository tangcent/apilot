# Contributing to APilot

Thank you for your interest in contributing to APilot! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions with the community.

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates. When creating a bug report, include:

- A clear and descriptive title
- Steps to reproduce the issue
- Expected behavior and actual behavior
- Your environment (OS, Go version)
- Any relevant output or error messages

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. Include:

- A clear and descriptive title
- A detailed description of the proposed functionality
- Why this enhancement would be useful
- Examples of how it would be used

### Pull Requests

1. Fork the repository and create your branch from `main`
2. Follow the project's architecture principles (see `docs/architecture.md`)
3. Follow the [branch and commit conventions](#branch-and-commit-conventions) — they are enforced by CI
4. Include tests for new functionality
5. Ensure all tests pass
6. Update documentation as needed
7. PRs are squash-merged by default — see [merging pull requests](#merging-pull-requests)

## Development Setup

### Prerequisites

- Go 1.22 or higher

### Building

```bash
# Build the apilot-cli binary
go build -o bin/apilot ./apilot-cli

# Cross-platform builds
GOOS=linux   GOARCH=amd64 go build -o bin/apilot-linux-amd64   ./apilot-cli
GOOS=darwin  GOARCH=arm64 go build -o bin/apilot-darwin-arm64  ./apilot-cli
GOOS=windows GOARCH=amd64 go build -o bin/apilot-windows-amd64.exe ./apilot-cli
```

### Running Tests

```bash
go test ./...
```

### Golden Files

`apilot-cli` diffs its exported output against the golden files in
`apilot-cli/testdata/goproject/`, so a field that vanishes from an export fails
the test instead of passing silently. When an export change is intended, refresh
the baseline deliberately and review the diff:

```bash
cd apilot-cli && go test ./... -update
```

## Architecture Guidelines

See [docs/architecture.md](docs/architecture.md) for the full breakdown.

### Key Principles

1. `api-collector` and `api-formatter` are the only shared contracts — no module may import above its layer
2. Collectors return `nil, nil` (not an error) when no endpoints are found
3. Formatters must return valid empty output for an empty endpoints slice
4. External plugins use the stdin/stdout JSON protocol — see [docs/plugin-protocol.md](docs/plugin-protocol.md)

## Project Structure

```
apilot/
├── api-collector/                  # Collector interface + ApiEndpoint model
├── api-formatter/                   # Formatter interface + FormatOptions
├── api-master/                     # Core engine: registry, plugin loader, orchestration
├── apilot-cli/                     # Bundled CLI: statically links all collectors + formatters
├── api-collector-{lang}/   # Language-specific collectors
├── api-formatter-{name}/            # Output formatters
├── vscode-plugin/                  # VSCode extension (TypeScript)
└── docs/                           # Architecture and protocol documentation
```

## AI Agent Instructions

Agent-facing content has one always-on entry point and one source of truth per
layer:

| Layer | Location | Holds |
|-------|----------|-------|
| Always-on constraints and index | `AGENTS.md` at the repository root | The few rules that must never be missed, plus pointers to everything else |
| Step-by-step workflows | `.agents/skills/<name>/SKILL.md` | Task-specific skills such as `review-pr` |
| Long or situational detail | `CONTRIBUTING.md`, `docs/`, and `.agents/references/` when needed | Full conventions and reference material |

Keep `AGENTS.md` short. Add a workflow to `.agents/skills/`, not to
`AGENTS.md`, and do not copy reference text into either place. Register any new
reference in the `AGENTS.md` index so it can actually be found.

`.claude` is a symlink to `.agents` and `CLAUDE.md` is a symlink to
`AGENTS.md`, so Claude Code reads the same skills and rules without a second
copy. Do not replace either link with a regular file.

After cloning, run `bash scripts/init.sh --all` once to distribute the skills
to Trae, Qoder, WorkBuddy, Kiro and Cursor and to generate their pointers to
`AGENTS.md`. Those directories are generated output and are gitignored; edit
only `AGENTS.md` and the files under `.agents/skills/`.

## Branch and Commit Conventions

Branches and commits share one type vocabulary. The separator is the only
difference: a slash for branches, a colon for commits.

### Branch names

Format: `<type>/<slug>`, or `<type>/<issue>-<slug>` when an issue exists.

- Create branches from `main`.
- Lowercase, kebab-case, English only; keep the slug short and descriptive.
- Never use a tool name or a personal name as the prefix (for example `workbuddy/`).

| Branch prefix | Commit type | Use for |
|---------------|-------------|---------|
| `feature/` | `feat` | New functionality |
| `fix/` | `fix` | Bug fixes |
| `docs/` | `docs` | Documentation only |
| `test/` | `test` | Tests only |
| `chore/` | `chore` | Maintenance, tooling, repository chores |
| `refactor/` | `refactor` | Restructuring without a behavior change |
| `perf/` | `perf` | Performance improvements |
| `style/` | `style` | Formatting and whitespace, no behavior change |
| `build/` | `build` | Build system and dependency changes |
| `enhance/` | `enhance` | Improvements to existing functionality |
| `amend/` | `amend` | Small amendments to a previous change |
| `release/` | `release` | Release preparation |

Two release branch shapes are reserved by the release workflows and must be
kept verbatim:

- `cli-release/vX.Y.Z`
- `plugin-release/vX.Y.Z`

Examples: `feature/196-required-fields-mode`, `fix/190-implicit-binding`,
`docs/codecov-badge-default-branch`, `chore/agents-skills-dir`.

Use `feature/` (not `feat/`), `fix/` (not `bugfix/`), and `docs/` (not `doc/`).

### Commit messages

Format: `<type>[(scope)][!]: <subject>`

| Field | Rule |
|-------|------|
| `type` | One of the 12 types below, lowercase |
| `scope` | Optional; a module name such as `apilot-cli` or `api-collector-java` |
| `!` | Optional; marks a breaking change |
| `subject` | English, imperative mood, lowercase first letter, no trailing period, at most 72 characters |

`type` is enforced by `.github/workflows/auto-label.yml` on every commit in a
PR. The whitelist is: `feat`, `fix`, `docs`, `test`, `chore`, `enhance`,
`amend`, `style`, `refactor`, `perf`, `build`, `release`. Any other prefix makes
the workflow post a failure comment on the PR. In particular, do not use
`ci:`, `doc:`, or `bugfix:`; use `chore:` or `build:`, `docs:`, and `fix:`
instead.

A `docs:` commit adds the `type: doc` label, which makes the build and coverage
workflows skip the PR. Use it only for documentation-only changes.

Release commits use their own format and trigger the release workflows:

```text
release: cli v1.2.3
release: plugin v1.2.3
```

Examples:

```text
feat(yapi): add a YApi formatter that reads EasyYapi rules files (#197)
fix(kotlin): recover from grammar errors instead of aborting the collection
docs(readme): fix Codecov badge to follow default branch
chore(skills): move .agent to .agents for native discovery
```

Do not merge `main` into a PR branch. The auto-label workflow validates every
commit in the PR, and a `Merge branch 'main' into ...` subject does not match
the format. Rebase on `main` instead.

## Merging Pull Requests

**Squash merge is the default.** Each PR lands on `main` as a single commit.
Use "Create a merge commit" or "Rebase and merge" only when a workflow
explicitly needs the branch history.

GitHub derives the squashed commit subject from the PR title (or, depending on
the repository setting, from the subject of a single-commit PR), and derives
the commit body either from the PR body or from the PR's commit messages.
**Both the PR title and the PR's commits must therefore follow
`<type>[(scope)][!]: <subject>`.** Keep both readable — either can end up in
the history.

Consequences specific to this repository:

- `release.yml` and `plugin-release.yml` trigger on `head_commit.message`
  containing `release: cli` / `release: plugin`, or containing the
  `cli-release/` / `plugin-release/` branch name from a merge-commit subject.
  Squash merging removes that merge-commit subject, so a **release PR title
  must be exactly `release: cli vX.Y.Z` or `release: plugin vX.Y.Z`**. Any
  other title merges cleanly and silently skips the release.
- `release.yml` builds release notes from `git log --pretty=format:"* %s"`,
  filtered to `refactor|feat|fix|perf|docs`. One conventional commit per PR
  produces a readable changelog; a free-form PR title drops out of it.
- `auto-label.yml` validates **every commit in the PR before** squashing.
  Squashing does not relax that check. Rebase on `main` locally; do not use
  GitHub's "Update branch" button, which adds a merge commit whose subject
  fails the workflow.
- `docs:` still adds the `type: doc` label to the PR and skips the build and
  coverage jobs. A documentation-only PR should use `docs:` for both its title
  and its commits.

Keep one logical change per PR — a squashed commit cannot be split later — and
delete the source branch after merging.

One-time repository setting (GitHub → Settings → General → Pull Requests):
allow squash merging, make it the default, set the squash commit title to the
PR title and the squash commit body to the PR body. Disabling "Allow merge
commits" and "Allow rebase merging" is the only way to enforce squash merging
instead of relying on convention.

## Questions?

Feel free to open an issue for any questions about contributing.
