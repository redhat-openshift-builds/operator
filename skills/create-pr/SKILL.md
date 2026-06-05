---
name: create-pr
description: Commit, push, and create or amend a PR with repo conventions enforced. Trigger on "create pr", "create-pr", "land pr", "push this", "ship this".
allowed-tools: [Bash, Read, AskUserQuestion]
user_invocable: true
---

# Create PR

Commit, push, and create (or amend) a pull request.

## Repo Conventions

- **Jira prefix:** All commit subjects and PR titles start with `[BUILD-XXXX]`
- **Commit flags:** Always use `-s` (sign-off) and `-S` (GPG sign)
- **Commit style:** Conventional commits — `[BUILD-XXXX] scope: description`
- **PR title:** `[BUILD-XXXX] builds-X.Y: scope: description` (include branch version prefix when targeting a release branch)
- **PR body:** `## Summary` header, changes as bullet points, `### Jira Issues` with `Resolves: BUILD-XXXX`, ends with `Co-Authored-By: Claude Code`

## Arguments

The user may pass a Jira key (e.g., `BUILD-2046`). If not provided, ask for it before committing.

The skill auto-detects new PR vs amend mode from the current branch state.

## Step 1 — Pre-flight

### 1a. Clear staged files

```bash
git reset HEAD
```

### 1b. Verify GitHub auth

Discover the currently logged-in GitHub user:

```bash
gh auth status 2>&1
```

Extract the active account from the output. Store as `GH_USER`. If auth fails entirely, stop and tell the user to run `gh auth login`.

### 1c. Update main

```bash
git checkout main
git pull --ff-only origin main
```

If `git pull` fails, stop and report the error.

## Step 2 — Detect Mode

```bash
BRANCH=$(git branch --show-current)
```

**If `$BRANCH` is `main`** → new PR mode. Continue to Step 3.

**If `$BRANCH` is not `main`** → check for an open PR on this branch:

```bash
gh pr list --head "$BRANCH" --state open --json number,title,url
```

- **Open PR found** → amend mode (default). Show the user and ask to confirm via AskUserQuestion. User can override to add a separate commit if needed.
- **No open PR** → new PR mode on this existing branch. Skip Step 3.

**Amend mode principle:** When amending, rewrite the commit message, PR title, and PR body to cover the entire diff from main as if everything was done in one shot. Never reference "added later", "fixed after review", or incremental changes — the final message should read as a single coherent unit of work.

## Step 3 — Create Branch (new PR mode only)

Analyze changes to auto-generate a descriptive kebab-case branch name (3-5 words, no prefixes like `feat/`).

```bash
git checkout -b <generated-branch-name>
```

## Step 4 — Stage Files

Run `git status --short` to list all changed files.

Present files to the user via AskUserQuestion (multiSelect):
- **Suggested** — files changed in this conversation
- **Other changes** — additional files the user can opt-in to

After confirmation:

```bash
git add <file1> <file2> ...
```

## Step 5 — Commit

### 5a. Analyze the diff

In amend mode, analyze full PR diff:

```bash
git diff main...HEAD
```

In new PR mode, analyze staged changes:

```bash
git diff --cached
```

### 5b. Generate commit message

Write a commit message following repo conventions:
- **Subject line**: `[BUILD-XXXX] scope: description` — under 72 chars
- **Body**: 2-4 lines describing what changed and why
- **End with**: `Co-Authored-By: Claude Code`

Show the commit message to the user for review before committing.

**New PR mode:**

```bash
git commit -s -S -m "$(cat <<'EOF'
[BUILD-XXXX] scope: description

<body>

Co-Authored-By: Claude Code
EOF
)"
```

**Amend mode:**

```bash
git commit --amend -s -S -m "$(cat <<'EOF'
[BUILD-XXXX] scope: description covering full diff from main

<body covering ALL changes in the PR>

Co-Authored-By: Claude Code
EOF
)"
```

## Step 6 — Push

**New PR mode:**

```bash
git push -u origin <branch>
```

**Amend mode:**

```bash
git push --force-with-lease
```

## Step 7 — Create or Update PR

Detect fork vs upstream for PR creation:

```bash
FETCH_URL=$(git remote get-url origin)
PUSH_URL=$(git remote get-url --push origin)
```

If FETCH_URL != PUSH_URL (fork), extract upstream slug and fork owner for `--repo` and `--head` flags.

Determine base branch: use `builds-X.Y` if on a release branch, otherwise `main`.

**New PR mode:**

```bash
gh pr create --title "[BUILD-XXXX] scope: description" --body "$(cat <<'EOF'
## Summary
- First change
- Second change

### Jira Issues
Resolves: BUILD-XXXX

Co-Authored-By: Claude Code
EOF
)"
```

The title matches the commit subject line. The body is `## Summary` with bullet points, `### Jira Issues` with resolves line.

**Amend mode:**

Update title and body to reflect the current full diff:

```bash
PR_NUMBER=$(gh pr list --head "$(git branch --show-current)" --state open --json number --jq '.[0].number')
gh pr edit "$PR_NUMBER" --title "[BUILD-XXXX] scope: updated description" --body "$(cat <<'EOF'
## Summary
- Updated bullet points covering ALL changes from main

### Jira Issues
Resolves: BUILD-XXXX

Co-Authored-By: Claude Code
EOF
)"
```

## Step 8 — Report

Print a summary:

> **PR landed!**
>
> - **URL:** <pr-url>
> - **Branch:** `<branch>`
> - **Commit:** `<short-sha>`
> - **Mode:** New PR / Amended
