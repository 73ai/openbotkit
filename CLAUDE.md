## Project Guidelines

- Do not call the task done until it is fully complete and tested.
- Always write tests for new features and bug fixes.
- Do not dismiss bug as a pre-existing" issue even if it was present before your change. It does not matter, it's still your responsibility to fix it. When you see a bug, fix it. Don't ignore it.

## Coding Guidelines

  - Keep code simple and easy to read.
  - Avoid excessive comments — only comment when absolutely necessary.

## Git Branch Rules

  - No slashes in branch names (e.g., use `fix-something` not `fix/something`).

## Git Commit Rules

  - Commit after every small, atomic change. Each commit should touch 1-3 files max.
  - Use conventional commit format: `feat|fix|refactor|docs|test|chore|ci(scope): message`
  - Never use `git add .` or `git add -A`. Always stage specific files by name.
  - Keep commits small: aim for under 20 lines changed per commit.
  - Don't batch multiple unrelated changes into one commit.
  - Commit early and often — a working 5-line change is better than a pending 200-line change.