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

**Check if docs/todo.md needs updates:**

- If the issue introduces new features, update docs/todo.md
- If the issue changes existing behavior, update relevant documentation
- If the issue is a bug fix, consider adding notes to docs/todo.md if it affects user-facing behavior
- Keep documentation concise and relevant

### 5. Create Commit in New Branch

**Branch naming convention:**

- Use descriptive branch name: `feature/issue-<number>-<short-description>`
- Or: `fix/issue-<number>-<short-description>` for bug fixes
- Or: `chore/issue-<number>-<short-description>` for maintenance tasks

**Commit process:**

- Create new branch from main/master: `git checkout -b <branch-name>`
- Stage changes: `git add <files>`
- Commit with conventional commit message: `git commit -m "<type>: <description> (#<issue-number>)"`
- Push branch: `git push -u origin <branch-name>`

### 6. Create Pull Request

**PR creation:**

- Use GitHub CLI: `gh pr create`
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
4. Update docs/todo.md with new authentication feature
5. git checkout -b feature/issue-123-user-authentication
6. git add . && git commit -m "feat: add user authentication (#123)"
7. git push -u origin feature/issue-123-user-authentication
8. gh pr create --title "Add user authentication" --body "Closes #123"
```

## Important Guidelines

- **Always assign the issue first** - this prevents duplicate work
- **Read the issue thoroughly** - understand requirements before coding
- **Update documentation** - keep docs/todo.md in sync with changes
- **Use conventional commits** - follow commit message conventions
- **Reference the issue** - always link PR to the original issue
- **Test your changes** - ensure tests pass before creating PR
- **Keep PRs focused** - one issue per PR when possible

## Tools Used

- `gh` (GitHub CLI) for issue and PR management
- `git` for branching and commits
- Standard text editors for code changes
