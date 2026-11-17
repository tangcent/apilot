---
name: "github-issue-resolver"
description: "Resolves GitHub issues end-to-end: assignment, implementation, documentation updates, branching, and PR creation. Invoke when user wants to work on a GitHub issue or mentions issue handling workflow."
---

# GitHub Issue Resolver

This skill provides a systematic workflow for handling GitHub issues from assignment to PR creation.

## Workflow

### 1. Assign the Issue to Yourself

- Use GitHub CLI or web interface to assign the issue
- Command: `gh issue edit <issue-number> --add-assignee @me`
- Verify assignment was successful

### 2. Read and Understand the Issue

- Read the issue description carefully
- Understand the requirements and acceptance criteria
- Identify affected code areas
- Ask clarifying questions if needed

### 3. Implement the Solution

- Analyze the codebase to understand current implementation
- Design the solution approach
- Implement the required changes
- Follow existing code patterns and conventions
- Write/update tests as needed

### 4. Update Documentation

**Check whether README.md or a file under docs/ needs an update:**

- If the issue introduces new features, update README.md
- If the issue changes existing behavior, update the relevant file under docs/
- If the issue is a bug fix, note user-facing changes in the relevant docs
- Keep documentation concise and relevant

### 5. Create Commit in New Branch

**Branch naming convention:**

Follow the always-on constraints in `AGENTS.md` and the full rules in
`CONTRIBUTING.md` → "Branch and commit conventions". Use
`<type>/<issue>-<short-description>`, for example
`feature/196-required-fields-mode` or `fix/190-implicit-binding`.
Never use a tool or personal name as the prefix.

**Commit process:**

- Create the branch from `main`: `git checkout -b <branch-name> main`
- Stage changes by path: `git add <files>`
- Commit with a type from the CI whitelist: `git commit -m "<type>: <description> (#<issue-number>)"`
- Do not merge `main` into the branch; rebase on `main` instead
- Push branch: `git push -u origin <branch-name>`

### 6. Create Pull Request

**PR creation:**

- Use GitHub CLI: `gh pr create`
- PRs are squash-merged by default, so the **PR title must follow the commit
  format** `<type>[(scope)][!]: <subject>` — it becomes the commit subject on
  `main`. See `CONTRIBUTING.md` → "Merging pull requests"
- Reference the original issue in PR description
- Include "Closes #<issue-number>" or "Fixes #<issue-number>" in PR description
- Provide clear description of changes
- List any breaking changes or migration steps

**PR template:**

```markdown
## Summary
Brief description of changes

## Changes
- Change 1
- Change 2

## Testing
How to test these changes

## Related Issue
Closes #<issue-number>
```

## Example Workflow

```
User: "Handle issue #123"

1. gh issue edit 123 --add-assignee @me
2. Read issue #123: "Add user authentication feature"
3. Implement authentication in UserService
4. Update README.md with the new behavior
5. git checkout -b feature/123-user-authentication main
6. git add <files> && git commit -m "feat: add user authentication (#123)"
7. git push -u origin feature/123-user-authentication
8. gh pr create --title "feat: add user authentication (#123)" --body "Closes #123"
```

## Important Guidelines

- **Always assign the issue first** - this prevents duplicate work
- **Read the issue thoroughly** - understand requirements before coding
- **Update documentation** - keep README.md and docs/ in sync with changes
- **Follow the git rules** - see `AGENTS.md` and `CONTRIBUTING.md` for branch names, commit types and PR titles
- **Reference the issue** - always link PR to the original issue
- **Test your changes** - ensure tests pass before creating PR
- **Keep PRs focused** - one issue per PR when possible

## Tools Used

- `gh` (GitHub CLI) for issue and PR management
- `git` for branching and commits
- Standard text editors for code changes
